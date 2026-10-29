package skygear

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/platform/db"
)

const stateKey = "skygear.state"

// stateVersion is increased when State changes incompatibly.
const stateVersion = 1

// State is the shared progress of one save. It is owned by the instance
// world: it may only be read or changed from that world's transactions.
type State struct {
	Version int `json:"version"`
	// Chunks are the owned island chunks.
	Chunks []ChunkPos `json:"chunks"`
	// Home is where the island's players start (x, z); zero for a save made
	// when the island was around the origin.
	Home [2]float64 `json:"home,omitempty"`
	// Researched are the done researches (research.go).
	Researched map[string]bool `json:"researched,omitempty"`
	// ResearchPaid counts the items delivered for researches not done yet,
	// by research and item key.
	ResearchPaid map[string]map[string]int `json:"research_paid,omitempty"`
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
	// Lost is the lost-and-found box, most recent first.
	Lost []*LostItem `json:"lost,omitempty"`
	// Crops are the planted crops by position ("x,y,z", crops.go).
	Crops map[string]*CropRecord `json:"crops,omitempty"`
	// Advancements are the done advancements (advancement.go).
	Advancements map[string]bool `json:"advancements,omitempty"`
	// Tracked is the advancement the quest panel follows instead of the
	// first open one (advancement.go).
	Tracked string `json:"tracked,omitempty"`
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

// ChunkPos is a chunk coordinate.
type ChunkPos [2]int32

// LostItem is a stack type in the lost-and-found box.
type LostItem struct {
	// Item is the little endian NBT of the item with count 1. Drops are
	// merged when the decoded stacks are Comparable (same item and NBT).
	Item  []byte `json:"item"`
	Count int    `json:"count"`
	// LastDropped is the instance time of the most recent drop.
	LastDropped time.Duration `json:"last_dropped"`
	// stack is Item decoded, once it was needed (decoded).
	stack item.Stack
}

// decoded is the stack of the entry, with count 1.
func (l *LostItem) decoded() item.Stack {
	if l.stack.Empty() {
		l.stack = decodeItem(l.Item)
	}
	return l.stack
}

func newState() *State {
	return &State{
		Version: stateVersion,
		// The starting island covers one chunk, with the 3×3×3 island in its
		// centre.
		Chunks: []ChunkPos{{0, 0}},
		Home:   [2]float64{islandCentre + 0.5, islandCentre + 0.5},
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

// stateWriter writes the marshalled states of one save outside the world,
// keeping only the newest one not written yet.
type stateWriter struct {
	write   func([]byte)
	mu      sync.Mutex
	pending []byte
	wake    chan struct{}
	quit    chan struct{}
	done    chan struct{}
}

func newStateWriter(write func([]byte)) *stateWriter {
	w := &stateWriter{write: write, wake: make(chan struct{}, 1), quit: make(chan struct{}), done: make(chan struct{})}
	go w.run()
	return w
}

func (w *stateWriter) put(b []byte) {
	w.mu.Lock()
	w.pending = b
	w.mu.Unlock()
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *stateWriter) run() {
	defer close(w.done)
	for {
		select {
		case <-w.wake:
			w.mu.Lock()
			b := w.pending
			w.pending = nil
			w.mu.Unlock()
			if b != nil {
				w.write(b)
			}
		case <-w.quit:
			return
		}
	}
}

// stop ends the writer once the write in progress is done, dropping the
// pending state.
func (w *stateWriter) stop() {
	close(w.quit)
	<-w.done
}
