package skygear

import (
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
}

func newState() *State {
	return &State{
		Version: stateVersion,
	}
}
