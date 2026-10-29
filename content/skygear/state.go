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
	// GateTier is the highest level of the dimension gate cleared (wave.go).
	GateTier int `json:"gate_tier,omitempty"`
	// Favourites are each account's favourite item keys in the codex,
	// oldest first (codexgoal.go).
	Favourites map[string][]string `json:"favourites,omitempty"`
}

func newState() *State {
	return &State{
		Version: stateVersion,
	}
}
