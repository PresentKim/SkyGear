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
		// DayLengthMinutes is the length of an instance day in real minutes.
		DayLengthMinutes int
		// LoadRadius is the radius in chunks kept loaded and ticked around
		// the island centre of an active instance.
		LoadRadius int
		// MaxActive limits active instances. 0 means no limit.
		MaxActive int
		// AutosaveSeconds is how often active instances store their state.
		AutosaveSeconds int
		// WorldSaveSeconds is how often active instances write their world
		// chunks to disk.
		WorldSaveSeconds int
	}
	Backup struct {
		// Enabled turns the backups on. Dir is where they go. The database
		// is copied every DatabaseMinutes (the newest DatabaseKeep stay), and
		// a save's world folder is archived each time its instance closes
		// (the newest WorldKeep per save stay).
		Enabled         bool
		Dir             string
		DatabaseMinutes int
		DatabaseKeep    int
		WorldKeep       int
	}
	ResourcePack struct {
		// Minify strips comments and white space from the packs' JSON, and
		// Encrypt encrypts the packs. Both are on
		// by default; turn them off to read what the client receives.
		Minify, Encrypt bool
		// SlotDeco adds the slotdeco pack: slot backgrounds by item rarity
		// and tier badges. It changes the vanilla
		// item slot template of every container screen; turn it off if the
		// screens look wrong.
		SlotDeco bool
	}
	ChestUI struct {
		// Ride links the chest entity to the player so the chest stays open
		// while moving. WaitLatency and Retry switch on further InvMenu-style
		// steps of opening entity chests. Debug logs the open/close packets.
		Ride, WaitLatency, Retry, Debug bool
	}
	Experimental struct {
		// VoxelShapes sends a minimal voxel shape registry on spawn.
		VoxelShapes bool
		// BlockActions logs the block breaking actions clients send.
		BlockActions bool
	}
	Admin struct {
		// Operators are the XUIDs (digits) of the players allowed to use
		// operator tools such as /sgadmin. An entry that is not a number is a
		// player name; names can pass to another player when an Xbox gamertag
		// changes hands, so the server warns when one is used.
		Operators []string
	}
}

// IsOperator reports whether the player with the XUID and name is an operator.
func (c Config) IsOperator(name, xuid string) bool {
	ok, _ := c.operator(name, xuid)
	return ok
}

// OperatorByNameOnly reports whether the player is an operator only because
// of a name entry, so that the config should list the XUID instead.
func (c Config) OperatorByNameOnly(name, xuid string) bool {
	ok, byXUID := c.operator(name, xuid)
	return ok && !byXUID
}

func (c Config) operator(name, xuid string) (ok, byXUID bool) {
	for _, op := range c.Admin.Operators {
		if xuid != "" && op == xuid {
			return true, true
		}
		if !isNumber(op) && strings.EqualFold(op, name) {
			ok = true
		}
	}
	return ok, false
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
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
	c.Instance.WorldSaveSeconds = 60
	c.Backup.Enabled = true
	c.Backup.Dir = "data/backups"
	c.Backup.DatabaseMinutes = 60
	c.Backup.DatabaseKeep = 48
	c.Backup.WorldKeep = 5
	c.ChestUI.Ride = true
	c.ResourcePack.Minify = true
	c.ResourcePack.Encrypt = true
	c.ResourcePack.SlotDeco = true
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
	if c.Backup.Enabled && (c.Backup.DatabaseMinutes <= 0 || c.Backup.Dir == "") {
		return c, errors.New("platform config: Backup.DatabaseMinutes must be positive and Backup.Dir set")
	}
	return c, nil
}
