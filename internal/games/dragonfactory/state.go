package dragonfactory

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/presentkim/dragonfactory/internal/db"
)

const stateKey = "dragonfactory.state"

// stateVersion is increased when State changes incompatibly.
const stateVersion = 1

// State is the shared progress of one save. It is owned by the instance
// world: it may only be read or changed from that world's transactions.
type State struct {
	Version int `json:"version"`
	// Chunks are the owned island chunks.
	Chunks []ChunkPos `json:"chunks"`
	// Lost is the lost-and-found box, most recent first.
	Lost []*LostItem `json:"lost,omitempty"`
}

// ChunkPos is a chunk coordinate.
type ChunkPos [2]int32

// LostItem is a stack type in the lost-and-found box (docs/design/02-dragonfactory.md §9.3).
type LostItem struct {
	// Item is the little endian NBT of the item with count 1. Drops are
	// merged when the decoded stacks are Comparable (same item and NBT).
	Item  []byte `json:"item"`
	Count int    `json:"count"`
	// LastDropped is the instance time of the most recent drop.
	LastDropped time.Duration `json:"last_dropped"`
}

func newState() *State {
	return &State{
		Version: stateVersion,
		// The starting island covers the four chunks around the origin.
		Chunks: []ChunkPos{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}},
	}
}

func loadState(d *db.DB, saveID int64) (*State, error) {
	b, err := d.SaveState(saveID, stateKey)
	if errors.Is(err, db.ErrNotFound) {
		return newState(), nil
	} else if err != nil {
		return nil, err
	}
	s := &State{}
	if err := json.Unmarshal(b, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *State) save(d *db.DB, saveID int64) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return d.SetSaveState(saveID, stateKey, b)
}
