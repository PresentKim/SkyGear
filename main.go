// Command dragonfactory runs the multi game island server (docs/design).
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
	"github.com/presentkim/dragonfactory/common/backup"
	"github.com/presentkim/dragonfactory/common/chestui"
	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/packs"
	"github.com/presentkim/dragonfactory/common/resourcepack"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory"
	"github.com/presentkim/dragonfactory/internal/instance"
	_ "github.com/presentkim/dragonfactory/internal/locales" // this server's texts
	"github.com/presentkim/dragonfactory/internal/platform"
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
	conf, err := readServerConfig(log)
	if err != nil {
		return err
	}
	pconf, err := platform.LoadConfig("platform.toml")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pconf.Data.Dir, 0o755); err != nil {
		return err
	}
	chestui.Options.Ride = pconf.ChestUI.Ride
	chestui.Options.WaitLatency = pconf.ChestUI.WaitLatency
	chestui.Options.Retry = pconf.ChestUI.Retry
	chestui.Options.Debug = pconf.ChestUI.Debug
	dfhook.SendBasicVoxelShapes = pconf.Experimental.VoxelShapes
	dfhook.BlockActionDebug = pconf.Experimental.BlockActions
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
	conf.Entities = entity.DefaultRegistry.Config().New(dragonfactory.EntityTypes())
	packOpt := resourcepack.Options{Minify: pconf.ResourcePack.Minify, Encrypt: pconf.ResourcePack.Encrypt}
	if packOpt.Encrypt {
		if packOpt.Secret, err = resourcepack.LoadSecret(filepath.Join(pconf.Data.Dir, "resourcepack.secret")); err != nil {
			return fmt.Errorf("resource pack secret: %w", err)
		}
	}
	// This server's pack, then the common packs (form and HUD theme,
	// chest windows, WAILA, slot decorations). The first pack is the top of
	// the stack: this server's pack fills the hooks of the dfui theme.
	dfPack, err := fs.Sub(packFS, "assets/resourcepack/dragonfactory")
	if err != nil {
		return fmt.Errorf("resource pack dragonfactory: %w", err)
	}
	if err := dragonfactory.LoadNames(dfPack); err != nil {
		return fmt.Errorf("resource pack dragonfactory names: %w", err)
	}
	packDirs := map[string]fs.FS{"dragonfactory": dfPack}
	packNames := []string{"dragonfactory", "dfui", "inventoryui", "waila"}
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
		dir, ok := packDirs[name]
		if !ok {
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
	dfhook.Wrap(&conf)
	srv := conf.New()
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
	}, platform.Lobby(lobby, &pf))
	isOperator := func(p *player.Player) bool { return pconf.IsOperator(p.Name(), p.XUID()) }
	pf = platform.New(log, d, m, dragonfactory.New(log, d, isOperator), srv.Player, isOperator)
	// Operators who turn it on (/dfadmin tps) see the server load.
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
		dfhook.Attach(p)
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

// readServerConfig reads the Dragonfly configuration from config.toml, or
// creates the file with defaults if it does not yet exist. Player data is
// always stored by the platform per save, so Dragonfly's player provider is
// disabled.
func readServerConfig(log *slog.Logger) (server.Config, error) {
	c := server.DefaultConfig()
	c.Server.Name = "DragonFactory"
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
