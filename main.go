// Command skygear runs the Sky Gear island server.
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/pelletier/go-toml"
	"github.com/presentkim/skygear/common/backup"
	"github.com/presentkim/skygear/common/chestui"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/packs"
	"github.com/presentkim/skygear/common/resourcepack"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/account"
	"github.com/presentkim/skygear/internal/db"
	"github.com/presentkim/skygear/internal/instance"
	_ "github.com/presentkim/skygear/internal/locales" // this server's texts
	"github.com/presentkim/skygear/internal/nsmigrate"
	"github.com/presentkim/skygear/internal/platform"
	"github.com/presentkim/skygear/internal/skygear"
	"github.com/presentkim/skygear/internal/skygear/naming"
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
	pconf, err := platform.LoadConfig("platform.toml")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pconf.Data.Dir, 0o755); err != nil {
		return err
	}
	// Saves from before the rename to SkyGear are converted once, before
	// the lobby world opens.
	if err := nsmigrate.Run(log, pconf.Data.Dir, pconf.Backup.Dir); err != nil {
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
	d, err := db.Open(filepath.Join(pconf.Data.Dir, "platform.db"))
	if err != nil {
		return err
	}
	defer d.Close()

	// Every world is an empty void; the lobby and islands are built on demand.
	conf.Generator = func(world.Dimension) world.Generator { return world.NopGenerator{} }
	// The disconnect screen shows text as it is (no translation keys, no
	// colour codes), so the shutdown message is written out in plain text.
	conf.ShutdownKick = func(p *player.Player) string { return i18n.T(i18n.Of(p), "main.shutdown") }
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
	// This server's pack, then the common packs (form and HUD theme,
	// chest windows, WAILA, slot decorations). The first pack is the top of
	// the stack: this server's pack fills the hooks of the sgui theme.
	sgPack, err := fs.Sub(packFS, "assets/resourcepack/skygear")
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
	// The game takes its in-game Ore UI page from an encrypted pack only, so
	// the menus are drawn there (DDUI) when the packs are encrypted.
	if packOpt.Encrypt {
		packNames = append(packNames, "oreui")
		ui.OreUI = true
	}
	for _, name := range packNames {
		dir := sgPack
		if name != "skygear" {
			if dir, err = packs.Sub(name); err != nil {
				return fmt.Errorf("resource pack %s: %w", name, err)
			}
		}
		pack, err := resourcepack.LoadFS(dir, name, packOpt)
		if err != nil {
			return fmt.Errorf("load resource pack: %w", err)
		}
		conf.Resources = append(conf.Resources, pack)
	}
	// Textures, models and names of custom blocks and items come from the
	// packs above; Dragonfly does not build its own.
	conf.DisableResourceBuilding = true
	// Chest windows, item holders and custom blocks look wrong without the packs.
	conf.ResourcesRequired = true
	// Chest windows reach the sessions through hooked connections.
	sghook.Wrap(&conf)
	srv := conf.New()
	skygear.EditItemEntries(srv)
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
		AfterClose:        afterClose,
		Dir:               filepath.Join(pconf.Data.Dir, "saves"),
		DayLength:         pconf.DayLength(),
		LoadRadius:        pconf.Instance.LoadRadius,
		MaxActive:         pconf.Instance.MaxActive,
		AutosaveInterval:  time.Duration(pconf.Instance.AutosaveSeconds) * time.Second,
		WorldSaveInterval: time.Duration(pconf.Instance.WorldSaveSeconds) * time.Second,
		Entities:          conf.Entities,
	}, platform.Lobby(lobby, &pf), skygear.New(log, d, isOperator))
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
		if err := srv.Close(); err != nil {
			log.Error("close server", "err", err)
		}
	}()

	srv.Listen()
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
