package game

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

// Config is the game specific configuration, stored in game.toml.
type Config struct {
	Island struct {
		// Spacing is the distance in blocks between island centres.
		Spacing int
		// Radius is the protected (buildable) radius around the OneBlock.
		Radius int
		// Height is the Y level of the OneBlock.
		Height int
		// AutoCreate creates an island for players joining for the first time.
		AutoCreate bool
	}
	Farming struct {
		// RandomTickSpeed controls crop growth speed. Vanilla is 3.
		RandomTickSpeed int
	}
	Data struct {
		// File is where island and profile data is stored.
		File string
		// SaveIntervalSeconds is how often data is saved to disk.
		SaveIntervalSeconds int
	}
}

// DefaultConfig returns the default game configuration.
func DefaultConfig() Config {
	var c Config
	c.Island.Spacing = 512
	c.Island.Radius = 100
	c.Island.Height = 64
	c.Island.AutoCreate = true
	c.Farming.RandomTickSpeed = 6
	c.Data.File = "data/game.json"
	c.Data.SaveIntervalSeconds = 120
	return c
}

// LoadConfig reads the config at path, writing the defaults if it is missing.
func LoadConfig(path string) (Config, error) {
	c := DefaultConfig()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		b, err = toml.Marshal(c)
		if err != nil {
			return c, fmt.Errorf("encode game config: %w", err)
		}
		return c, os.WriteFile(path, b, 0o644)
	} else if err != nil {
		return c, fmt.Errorf("read game config: %w", err)
	}
	if err := toml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("decode game config: %w", err)
	}
	if c.Island.Radius*2 >= c.Island.Spacing {
		return c, fmt.Errorf("island radius %d is too large for spacing %d", c.Island.Radius, c.Island.Spacing)
	}
	return c, nil
}
