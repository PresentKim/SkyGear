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

// newParts makes the part slots of a session, one per declared part.
func newParts() []any {
	partsMu.Lock()
	defer partsMu.Unlock()
	return make([]any, nParts)
}

// setPart stores v as part id, growing the slots if id was declared after s.
func (s *Session) setPart(id int, v any) {
	if id >= len(s.parts) {
		s.parts = append(s.parts, make([]any, id+1-len(s.parts))...)
	}
	s.parts[id] = v
}

func (s *Session) hasPart(id int) bool { return id < len(s.parts) && s.parts[id] != nil }

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
	if p.id < len(s.parts) {
		if v := s.parts[p.id]; v != nil {
			return v.(*T)
		}
	}
	v := p.new()
	s.setPart(p.id, v)
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
		if s.hasPart(p.part.id) {
			delete(*p.part.Of(s), h)
		}
	})
	partsMu.Unlock()
	return p
}

// Of is the map of every player's entry in s.
func (p *Players[T]) Of(s *Session) map[*world.EntityHandle]T { return *p.part.Of(s) }
