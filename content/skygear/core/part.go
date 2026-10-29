package core

import (
	"sync"

	"github.com/df-mc/dragonfly/server/world"
)

var (
	partsMu sync.Mutex
	nParts  int
	// leaving are the Players parts, cleared by Session.Leave.
	leaving []func(s *Session, h *world.EntityHandle)
)

func newPartID() int {
	partsMu.Lock()
	defer partsMu.Unlock()
	nParts++
	return nParts - 1
}

// Part is a feature's runtime data in each session, made by its new function
// the first time Of asks for it. Declare one as a package variable.
type Part[T any] struct {
	id  int
	new func() *T
}

// NewPart declares a part whose data mk makes (nil: the zero T).
func NewPart[T any](mk func() *T) *Part[T] {
	if mk == nil {
		mk = func() *T { return new(T) }
	}
	return &Part[T]{id: newPartID(), new: mk}
}

// Of is the part's data in s.
func (p *Part[T]) Of(s *Session) *T {
	if v, ok := s.parts[p.id]; ok {
		return v.(*T)
	}
	v := p.new()
	s.parts[p.id] = v
	return v
}

// Players is a feature's data per player in each session. A player's entry
// goes when the player leaves the session (Session.Leave).
type Players[T any] struct {
	part *Part[map[*world.EntityHandle]T]
}

// NewPlayers declares a per-player part.
func NewPlayers[T any]() *Players[T] {
	p := &Players[T]{part: NewPart(func() *map[*world.EntityHandle]T {
		m := map[*world.EntityHandle]T{}
		return &m
	})}
	partsMu.Lock()
	leaving = append(leaving, func(s *Session, h *world.EntityHandle) {
		if _, ok := s.parts[p.part.id]; ok {
			delete(*p.part.Of(s), h)
		}
	})
	partsMu.Unlock()
	return p
}

// Of is the map of every player's entry in s.
func (p *Players[T]) Of(s *Session) map[*world.EntityHandle]T { return *p.part.Of(s) }
