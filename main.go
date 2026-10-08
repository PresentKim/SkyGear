// Command dragonfactory runs the multi game island server (docs/design).
package main

import (
	"fmt"
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
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/pelletier/go-toml"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/platform"
	"github.com/presentkim/dragonfactory/internal/resourcepack"
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
	session.EntityContainerOptions.Ride = pconf.ChestUI.Ride
	session.EntityContainerOptions.WaitLatency = pconf.ChestUI.WaitLatency
	session.EntityContainerOptions.Retry = pconf.ChestUI.Retry
	session.EntityContainerOptions.Debug = pconf.ChestUI.Debug
	d, err := db.Open(filepath.Join(pconf.Data.Dir, "platform.db"))
	if err != nil {
		return err
	}
	defer d.Close()

	// Every world is an empty void; the lobby and islands are built on demand.
	conf.Generator = func(world.Dimension) world.Generator { return world.NopGenerator{} }
	conf.Entities = entity.DefaultRegistry.Config().New(dragonfactory.EntityTypes())
	for _, dir := range []string{"assets/resourcepack/dragonfactory", "assets/resourcepack/inventoryui"} {
		pack, err := resourcepack.Load(dir)
		if err != nil {
			return fmt.Errorf("load resource pack: %w", err)
		}
		conf.Resources = append(conf.Resources, pack)
	}
	// Chest windows and item holders look wrong without the packs.
	conf.ResourcesRequired = true
	srv := conf.New()
	lobby := srv.World()
	lobby.SetSpawn(platform.LobbySpawn)
	lobby.SetDefaultGameMode(world.GameModeAdventure)
	lobby.StopTime()
	<-lobby.Do(platform.BuildLobby).Done()

	var pf *platform.Platform
	m := instance.NewManager(log, d, instance.Config{
		Dir:              filepath.Join(pconf.Data.Dir, "saves"),
		DayLength:        pconf.DayLength(),
		LoadRadius:       pconf.Instance.LoadRadius,
		MaxActive:        pconf.Instance.MaxActive,
		AutosaveInterval: time.Duration(pconf.Instance.AutosaveSeconds) * time.Second,
		Entities:         conf.Entities,
	}, platform.Lobby(lobby, &pf))
	pf = platform.New(log, d, m, dragonfactory.New(log, d, func(p *player.Player) bool { return pconf.IsOperator(p.Name()) }), srv.Player)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		// Store every instance before players are disconnected.
		m.CloseAll()
		if err := srv.Close(); err != nil {
			log.Error("close server", "err", err)
		}
	}()

	srv.Listen()
	for p := range srv.Accept() {
		pf.Join(p)
	}
	m.CloseAll()
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
