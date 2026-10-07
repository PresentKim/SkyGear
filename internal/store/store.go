// Package store persists island and player profile data as JSON files.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Island is a single OneBlock island. Its world position is derived from Index.
type Island struct {
	Index     int         `json:"index"`
	Owner     uuid.UUID   `json:"owner"`
	OwnerName string      `json:"owner_name"`
	Members   []uuid.UUID `json:"members,omitempty"`
	// Broken is the total amount of times the OneBlock has been broken.
	Broken int `json:"broken"`
	// Home is the relative home offset from the island centre. Nil means the OneBlock itself.
	Home *[3]float64 `json:"home,omitempty"`
}

// HasMember reports if id is the owner or a member of the island.
func (i *Island) HasMember(id uuid.UUID) bool {
	if i.Owner == id {
		return true
	}
	for _, m := range i.Members {
		if m == id {
			return true
		}
	}
	return false
}

// Profile holds per-player progression.
type Profile struct {
	Name   string `json:"name"`
	Island int    `json:"island"` // -1 when the player has no island.
	Money  int64  `json:"money"`
	FarmXP int    `json:"farm_xp"`
	// Harvested counts fully grown crops harvested, keyed by crop name.
	Harvested map[string]int `json:"harvested,omitempty"`
	// AutoReplant toggles automatic replanting once the perk is unlocked.
	AutoReplant bool `json:"auto_replant"`
}

type data struct {
	NextIndex int                    `json:"next_index"`
	Islands   map[int]*Island        `json:"islands"`
	Profiles  map[uuid.UUID]*Profile `json:"profiles"`
}

// Store is a concurrency-safe container of all persistent game data. Callers
// must hold the lock (via Do) while reading or mutating returned pointers.
type Store struct {
	mu   sync.Mutex
	path string
	d    data
}

// Open loads the store from path, creating an empty one if it does not exist.
func Open(path string) (*Store, error) {
	s := &Store{path: path, d: data{Islands: map[int]*Island{}, Profiles: map[uuid.UUID]*Profile{}}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	} else if err != nil {
		return nil, fmt.Errorf("read store: %w", err)
	}
	if err := json.Unmarshal(b, &s.d); err != nil {
		return nil, fmt.Errorf("decode store: %w", err)
	}
	if s.d.Islands == nil {
		s.d.Islands = map[int]*Island{}
	}
	if s.d.Profiles == nil {
		s.d.Profiles = map[uuid.UUID]*Profile{}
	}
	return s, nil
}

// Tx gives access to the store's data while holding its lock.
type Tx struct{ s *Store }

// Do runs f while holding the store lock.
func (s *Store) Do(f func(tx Tx)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(Tx{s: s})
}

// Profile returns the profile of id, creating it if needed.
func (tx Tx) Profile(id uuid.UUID, name string) *Profile {
	p, ok := tx.s.d.Profiles[id]
	if !ok {
		p = &Profile{Name: name, Island: -1, AutoReplant: true}
		tx.s.d.Profiles[id] = p
	}
	if name != "" {
		p.Name = name
	}
	return p
}

// LookupProfile returns the profile of id if it exists.
func (tx Tx) LookupProfile(id uuid.UUID) (*Profile, bool) {
	p, ok := tx.s.d.Profiles[id]
	return p, ok
}

// Island returns the island at index.
func (tx Tx) Island(index int) (*Island, bool) {
	i, ok := tx.s.d.Islands[index]
	return i, ok
}

// Islands returns all islands.
func (tx Tx) Islands() map[int]*Island { return tx.s.d.Islands }

// NewIsland allocates a new island for owner.
func (tx Tx) NewIsland(owner uuid.UUID, name string) *Island {
	i := &Island{Index: tx.s.d.NextIndex, Owner: owner, OwnerName: name}
	tx.s.d.NextIndex++
	tx.s.d.Islands[i.Index] = i
	return i
}

// Save writes the store to disk atomically.
func (s *Store) Save() error {
	s.mu.Lock()
	b, err := json.MarshalIndent(s.d, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write store: %w", err)
	}
	return os.Rename(tmp, s.path)
}

// DeleteIsland removes the island at index. Its area becomes unclaimed.
func (tx Tx) DeleteIsland(index int) {
	delete(tx.s.d.Islands, index)
}

// ProfileByName finds a profile by player name, ignoring case.
func (tx Tx) ProfileByName(name string) (uuid.UUID, *Profile, bool) {
	for id, p := range tx.s.d.Profiles {
		if strings.EqualFold(p.Name, name) {
			return id, p, true
		}
	}
	return uuid.Nil, nil, false
}
