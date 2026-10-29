package skygear

import (
	"time"

	"github.com/presentkim/skygear/content/skygear/core"
)

const stateKey = "skygear.state"

// savedState is State in each session (core.Saved).
var savedState = core.NewSaved(newState)

// stateVersion is increased when State changes incompatibly.
const stateVersion = 1

// State is the shared progress of one save. It is owned by the instance
// world: it may only be read or changed from that world's transactions.
type State struct {
	Version int `json:"version"`
	// NextFairy is the instance time of the next gift fairy visit.
	NextFairy time.Duration `json:"next_fairy,omitempty"`
	// FairyDust is whether the fairy has given the save fairy dust; until
	// then a joining player gets the dust of the nights before
	// (backfillFairyDust).
	FairyDust bool `json:"fairy_dust,omitempty"`
	// Weather is the weather a fairy spell brought ("", rain or snow) and
	// WeatherUntil the instance time it ends (fairyspell.go).
	Weather      string        `json:"weather,omitempty"`
	WeatherUntil time.Duration `json:"weather_until,omitempty"`
	// Sun is the fixed sun ("" flows with the instance clock).
	Sun string `json:"sun,omitempty"`
	// GrowClock is the growth clock the crops and magic plants count on
	// (growNow).
	GrowClock time.Duration `json:"grow_clock,omitempty"`
	// BaseBiome is the island's biome an old save's snow changed, given back
	// at the next start (restoreBiome).
	BaseBiome *uint32 `json:"base_biome,omitempty"`
	// Crops are the planted crops by position ("x,y,z", crops.go).
	Crops map[string]*CropRecord `json:"crops,omitempty"`
	// Composters are the plant items in each composter by position
	// ("x,y,z", composter.go).
	Composters map[string]int `json:"composters,omitempty"`
	// ComposterReady is the instance time the dirt of each full composter
	// is ready (composter.go).
	ComposterReady map[string]time.Duration `json:"composter_ready,omitempty"`
	// Barrels are the barrels' contents by position (barrel.go).
	Barrels map[string]*BarrelRecord `json:"barrels,omitempty"`
	// Plants are the magic plants by position (magic.go).
	Plants map[string]*PlantRecord `json:"plants,omitempty"`
	// GateTier is the highest level of the dimension gate cleared (wave.go).
	GateTier int `json:"gate_tier,omitempty"`
	// Produced counts the items the machines made, by item key
	// (stats.go).
	Produced map[string]int64 `json:"produced,omitempty"`
	// AlertsOn are the accounts that turned the stall alerts on; they are
	// off until then (monitor.go).
	AlertsOn map[string]bool `json:"alerts_on,omitempty"`
	// Favourites are each account's favourite item keys in the codex,
	// oldest first (codexgoal.go).
	Favourites map[string][]string `json:"favourites,omitempty"`
	// LastSapling is when /묘목 last gave a sapling (real time).
	LastSapling time.Time `json:"last_sapling,omitempty"`
}

func newState() *State {
	return &State{
		Version: stateVersion,
	}
}
