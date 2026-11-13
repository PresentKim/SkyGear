// Command skygear runs the Sky Gear island server.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // serves /debug/pprof on Debug.Pprof only
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/pelletier/go-toml"
	"github.com/presentkim/skygear/assets"
	"github.com/presentkim/skygear/common/backup"
	"github.com/presentkim/skygear/common/chestui"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/netcompress"
	"github.com/presentkim/skygear/common/resourcepack"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/intro"
	_ "github.com/presentkim/skygear/content/skygear/locales" // this server's texts
	"github.com/presentkim/skygear/content/skygear/locator"
	"github.com/presentkim/skygear/content/skygear/migrate"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/platform"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

func main() {
	log := slog.Default()
	chat.Global.Subscribe(chat.StdoutSubscriber{})
	if err := run(log); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	log.Info("Sky Gear", "version", Version)
	pconf, err := platform.LoadConfig("platform.toml")
	if err != nil {
		return err
	}
	if _, set := os.LookupEnv("GOMEMLIMIT"); !set && pconf.Runtime.MemoryLimitMB > 0 {
		debug.SetMemoryLimit(int64(pconf.Runtime.MemoryLimitMB) << 20)
	}
	if _, set := os.LookupEnv("GOGC"); !set && pconf.Runtime.GCPercent > 0 {
		debug.SetGCPercent(pconf.Runtime.GCPercent)
	}
	if addr := pconf.Debug.Pprof; addr != "" {
		go func() {
			log.Info("pprof listening", "addr", addr)
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Error("pprof", "err", err)
			}
		}()
	}
	if err := os.MkdirAll(pconf.Data.Dir, 0o755); err != nil {
		return err
	}
	// Saves from before the rename to SkyGear are converted once, before
	// the lobby world opens.
	if err := migrate.Namespace(log, pconf.Data.Dir, pconf.Backup.Dir); err != nil {
		return err
	}
	conf, err := readServerConfig(log)
	if err != nil {
		return err
	}
	chestui.Options.Ride = pconf.ChestUI.Ride
	chestui.Options.WaitLatency = pconf.ChestUI.WaitLatency
	chestui.Options.Retry = pconf.ChestUI.Retry
	chestui.Options.Debug = pconf.ChestUI.Debug
	sghook.SendBasicVoxelShapes = pconf.Experimental.VoxelShapes
	sghook.BlockActionDebug = pconf.Experimental.BlockActions
	locator.Enabled = pconf.Experimental.LocatorBar
	platform.CameraSpline = pconf.Experimental.CameraSpline
	intro.Spline = pconf.Experimental.CameraSpline
	d, err := db.Open(filepath.Join(pconf.Data.Dir, "platform.db"))
	if err != nil {
		return err
	}
	defer d.Close()
	if err := migrate.Run(log, filepath.Join(pconf.Data.Dir, "platform.db"), filepath.Join(pconf.Data.Dir, "saves"), pconf.Backup.Dir, Version); err != nil {
		return err
	}

	// Every world is an empty void; the lobby and islands are built on demand.
	conf.Generator = func(world.Dimension) world.Generator { return world.NopGenerator{} }
	// No server-wide join and quit lines: the lobby has no chat, and a save's
	// players hear of each other only (instance.Manager).
	conf.JoinMessage, conf.QuitMessage = chat.Translation{}, chat.Translation{}
	conf.Entities = entity.DefaultRegistry.Config().New(skygear.EntityTypes())
	packOpt := resourcepack.Options{Minify: pconf.ResourcePack.Minify, Encrypt: pconf.ResourcePack.Encrypt}
	if packOpt.Encrypt {
		if packOpt.Secret, err = resourcepack.LoadSecret(filepath.Join(pconf.Data.Dir, "resourcepack.secret")); err != nil {
			return fmt.Errorf("resource pack secret: %w", err)
		}
	}
	// This server's pack, then the common packs (HUD and menu textures,
	// chest windows, WAILA, slot decorations, the Ore UI page of the menus).
	// The first pack is the top of the stack.
	sgPack, err := assets.Pack("skygear")
	if err != nil {
		return fmt.Errorf("resource pack skygear: %w", err)
	}
	if err := naming.LoadNames(sgPack); err != nil {
		return fmt.Errorf("resource pack skygear names: %w", err)
	}
	packNames := []string{"skygear", "sgui", "inventoryui", "waila"}
	if pconf.ResourcePack.SlotDeco {
		packNames = append(packNames, "slotdeco")
	}
	packNames = append(packNames, "oreui")
	packOpt.CacheDir = filepath.Join(pconf.Data.Dir, "packcache")
	// The packs build concurrently; conf.Resources keeps the list order.
	base := len(conf.Resources)
	conf.Resources = append(conf.Resources, make([]*resource.Pack, len(packNames))...)
	packErrs := make([]error, len(packNames))
	var packWG sync.WaitGroup
	for i, name := range packNames {
		dir := sgPack
		if name != "skygear" {
			if dir, err = assets.Pack(name); err != nil {
				return fmt.Errorf("resource pack %s: %w", name, err)
			}
		}
		packWG.Add(1)
		go func() {
			defer packWG.Done()
			conf.Resources[base+i], packErrs[i] = resourcepack.LoadFS(dir, name, packOpt)
		}()
	}
	packWG.Wait()
	for _, perr := range packErrs {
		if perr != nil {
			return fmt.Errorf("load resource pack: %w", perr)
		}
	}
	// Textures, models and names of custom blocks and items come from the
	// packs above; Dragonfly does not build its own.
	conf.DisableResourceBuilding = true
	// Chest windows, item holders and custom blocks look wrong without the packs.
	conf.ResourcesRequired = true
	// Same Flate ID as vanilla, at the configured level. Installed so incoming
	// batches with that ID decode with it too.
	conf.Compression = netcompress.Install(pconf.Network.CompressionLevel)
	// Chest windows reach the sessions through hooked connections.
	sghook.Wrap(&conf)
	sghook.JoinLog = log
	skygear.LogCustomBlockCounts(log)
	srv := conf.New()
	core.EditEntries(srv)
	lobby := srv.World()
	lobby.SetSpawn(platform.LobbySpawn)
	lobby.SetDefaultGameMode(world.GameModeAdventure)
	lobby.StopTime()
	<-lobby.Do(platform.BuildLobby).Done()

	var afterClose func(int64, string)
	stopBackup := make(chan struct{})
	if b := pconf.Backup; b.Enabled {
		go backup.RunDatabase(log, d, b.Dir, time.Duration(b.DatabaseMinutes)*time.Minute, b.DatabaseKeep, stopBackup)
		afterClose = func(id int64, dir string) {
			if p, err := backup.Directory(dir, b.Dir, backup.WorldPrefix(id), b.WorldKeep, time.Now()); err != nil {
				log.Error("world backup", "save", id, "err", err)
			} else {
				log.Info("world backup", "file", p)
			}
		}
	}

	isOperator := func(p *player.Player) bool { return pconf.IsOperator(p.Name(), p.XUID()) }
	var pf *platform.Platform
	m := instance.NewManager(log, d, instance.Config{
		AfterClose:         afterClose,
		Dir:                filepath.Join(pconf.Data.Dir, "saves"),
		DayLength:          pconf.DayLength(),
		LoadRadius:         pconf.Instance.LoadRadius,
		MaxActive:          pconf.Instance.MaxActive,
		AutosaveInterval:   time.Duration(pconf.Instance.AutosaveSeconds) * time.Second,
		StateFlushInterval: time.Duration(pconf.Instance.StateFlushSeconds) * time.Second,
		WorldSaveInterval:  time.Duration(pconf.Instance.WorldSaveSeconds) * time.Second,
		Entities:           conf.Entities,
	}, platform.Lobby(lobby, &pf), skygear.New(log, d, isOperator, Version))
	pf = platform.New(log, d, m, srv.Player, isOperator)
	// Operators who turn it on (/sgadmin tps) see the server load.
	stopLoad := make(chan struct{})
	go platform.ShowLoad(lobby, isOperator, stopLoad)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		close(stopLoad)
		// Store every instance before players are disconnected.
		m.CloseAll()
		account.Flush()
		close(stopBackup)
		// The disconnect screen shows text as it is, so the shutdown message
		// is written out in plain text.
		platform.DisconnectAll(srv.Players(nil), func(p *player.Player) string { return i18n.T(i18n.Of(p), "platform.shutdown") })
		if err := srv.Close(); err != nil {
			log.Error("close server", "err", err)
		}
	}()

	srv.Listen()
	// The edits of the client entries run now, not at the first join: a bad
	// block or item declaration stops the start.
	if err := core.CheckEntries(); err != nil {
		return err
	}
	for p := range srv.Accept() {
		sghook.Attach(p)
		if pconf.OperatorByNameOnly(p.Name(), p.XUID()) {
			log.Warn("operator listed by name; list the XUID in Admin.Operators", "name", p.Name(), "xuid", p.XUID())
		}
		pf.Join(p)
	}
	m.CloseAll()
	account.Flush()
	if b := pconf.Backup; b.Enabled {
		if _, err := backup.Database(d, b.Dir, b.DatabaseKeep, time.Now()); err != nil {
			log.Error("database backup", "err", err)
		}
	}
	return nil
}

// readServerConfig reads config.toml, creating it with defaults if missing.
// The platform stores player data per save, so Dragonfly's player provider is
// disabled.
func readServerConfig(log *slog.Logger) (server.Config, error) {
	c := server.DefaultConfig()
	c.Server.Name = "Sky Gear"
	if _, err := os.Stat("config.toml"); os.IsNotExist(err) {
		data, err := toml.Marshal(c)
		if err != nil {
			return server.Config{}, fmt.Errorf("encode default config: %w", err)
		}
		if err := os.WriteFile("config.toml", data, 0o644); err != nil {
			return server.Config{}, fmt.Errorf("create default config: %w", err)
		}
	} else {
		data, err := os.ReadFile("config.toml")
		if err != nil {
			return server.Config{}, fmt.Errorf("read config: %w", err)
		}
		if err := toml.Unmarshal(data, &c); err != nil {
			return server.Config{}, fmt.Errorf("decode config: %w", err)
		}
	}
	c.Players.SaveData = false
	return c.Config(log)
}
