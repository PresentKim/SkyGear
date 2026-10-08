package platform

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
)

// Config is the platform configuration, stored in platform.toml.
type Config struct {
	Data struct {
		// Dir holds the database and the save worlds.
		Dir string
	}
	Instance struct {
		// DayLengthMinutes is the length of an instance day in real minutes
		// (docs/design/open-questions.md: decided during balancing).
		DayLengthMinutes int
		// LoadRadius is the radius in chunks kept loaded and ticked around
		// the island centre of an active instance.
		LoadRadius int
		// MaxActive limits active instances. 0 means no limit.
		MaxActive int
		// AutosaveSeconds is how often active instances store their state.
		AutosaveSeconds int
	}
	ChestUI struct {
		// Ride, WaitLatency and Retry switch on the InvMenu-style steps of
		// opening entity chests (docs/tech/chestui.md); all off is the plain
		// open that is known to work. Debug logs the open/close packets.
		Ride, WaitLatency, Retry, Debug bool
	}
	Admin struct {
		// Operators are player names allowed to use operator tools such as
		// /dfadmin (testing only).
		Operators []string
	}
}

// IsOperator reports whether name is an operator.
func (c Config) IsOperator(name string) bool {
	for _, op := range c.Admin.Operators {
		if strings.EqualFold(op, name) {
			return true
		}
	}
	return false
}

// DayLength returns the instance day length.
func (c Config) DayLength() time.Duration {
	return time.Duration(c.Instance.DayLengthMinutes) * time.Minute
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	var c Config
	c.Data.Dir = "data"
	c.Instance.DayLengthMinutes = 20
	c.Instance.LoadRadius = 4
	c.Instance.AutosaveSeconds = 30
	return c
}

// LoadConfig reads path, writing the defaults first if it does not exist.
func LoadConfig(path string) (Config, error) {
	c := DefaultConfig()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if b, err = toml.Marshal(c); err != nil {
			return c, fmt.Errorf("encode platform config: %w", err)
		}
		return c, os.WriteFile(path, b, 0o644)
	} else if err != nil {
		return c, fmt.Errorf("read platform config: %w", err)
	}
	if err := toml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("decode platform config: %w", err)
	}
	if c.Instance.DayLengthMinutes <= 0 || c.Instance.LoadRadius <= 0 {
		return c, errors.New("platform config: DayLengthMinutes and LoadRadius must be positive")
	}
	return c, nil
}
