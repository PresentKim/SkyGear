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
	// Level is the save level, starting at 1.
	Level int `json:"level"`
	// Delivered counts items delivered for the current level's mission.
	Delivered map[string]int `json:"delivered,omitempty"`
	// CorePoints are unspent core upgrade points.
	CorePoints int `json:"core_points"`
	// TierSpeed[1] is the core speed upgrade level (once by core tier).
	TierSpeed map[int]int `json:"tier_speed,omitempty"`
	// Researched are the done researches (research.go).
	Researched map[string]bool `json:"researched,omitempty"`
	// ResearchPaid counts the items delivered for researches not done yet,
	// by research and item key.
	ResearchPaid map[string]map[string]int `json:"research_paid,omitempty"`
	// NextFairy is the instance time of the next gift fairy visit.
	NextFairy time.Duration `json:"next_fairy,omitempty"`
	// Shop is the stock of shop products by product ID.
	Shop map[string]*ShopStock `json:"shop,omitempty"`
	// Buyback is today's buyback budget and sold amounts.
	Buyback *Buyback `json:"buyback,omitempty"`
	// Lost is the lost-and-found box, most recent first.
	Lost []*LostItem `json:"lost,omitempty"`
	// Urgent is the running urgent mission; NextUrgent is the instance time
	// when the next one starts (0: not scheduled yet).
	Urgent     *UrgentMission `json:"urgent,omitempty"`
	NextUrgent time.Duration  `json:"next_urgent,omitempty"`
	// Crops are the planted crops by position ("x,y,z", crops.go);
	// CropLevel is the crop growth upgrade level.
	Crops     map[string]*CropRecord `json:"crops,omitempty"`
	CropLevel int                    `json:"crop_level,omitempty"`
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
		Level:   1,
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
	// States saved before a field existed get its starting value.
	if s.Level < 1 {
		s.Level = 1
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
