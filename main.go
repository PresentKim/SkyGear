// Command dragonfactory runs a OneBlock x Skyblock farming server on Dragonfly.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/pelletier/go-toml"
	"github.com/presentkim/dragonfactory/internal/game"
	"github.com/presentkim/dragonfactory/internal/store"
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
	conf, err := readConfig(log)
	if err != nil {
		return err
	}
	gconf, err := game.LoadConfig("game.toml")
	if err != nil {
		return err
	}
	st, err := store.Open(gconf.Data.File)
	if err != nil {
		return err
	}

	// Every dimension is an empty void: islands are built on demand.
	conf.Generator = func(world.Dimension) world.Generator { return world.NopGenerator{} }
	conf.RandomTickSpeed = gconf.Farming.RandomTickSpeed

	srv := conf.New()
	g := game.New(log, srv, gconf, st)

	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(time.Duration(gconf.Data.SaveIntervalSeconds) * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				g.Save()
			case <-stop:
				return
			}
		}
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		g.Save()
		if err := srv.Close(); err != nil {
			log.Error("close server", "err", err)
		}
	}()

	srv.Listen()
	for p := range srv.Accept() {
		g.Join(p)
	}
	close(stop)
	g.Save()
	return nil
}

// readConfig reads the Dragonfly configuration from config.toml, or creates
// the file with defaults if it does not yet exist.
func readConfig(log *slog.Logger) (server.Config, error) {
	c := server.DefaultConfig()
	c.Server.Name = "OneBlock Farm"
	if _, err := os.Stat("config.toml"); os.IsNotExist(err) {
		data, err := toml.Marshal(c)
		if err != nil {
			return server.Config{}, fmt.Errorf("encode default config: %w", err)
		}
		if err := os.WriteFile("config.toml", data, 0o644); err != nil {
			return server.Config{}, fmt.Errorf("create default config: %w", err)
		}
		return c.Config(log)
	}
	data, err := os.ReadFile("config.toml")
	if err != nil {
		return server.Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := toml.Unmarshal(data, &c); err != nil {
		return server.Config{}, fmt.Errorf("decode config: %w", err)
	}
	return c.Config(log)
}
