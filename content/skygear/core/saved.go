package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// savedPart is one Saved part as the session's state document sees it.
type savedPart struct {
	id int
	// new makes the data of a new save, zero the data a save's document is
	// read into.
	new, zero func() any
}

var (
	saved      []savedPart
	savedField = map[string]string{} // JSON field -> type of the part that owns it
)

// Saved is a feature's saved data in each session: a struct whose JSON
// fields are fields of the save's state document. Two parts may not have a
// field of the same name. Declare one as a package variable.
type Saved[T any] struct{ id int }

// NewSaved declares a saved part. mk makes the data of a new save (nil: the
// zero T); a save that has a document reads it into a zero T, so a field the
// document lacks stays zero.
func NewSaved[T any](mk func() *T) *Saved[T] {
	if mk == nil {
		mk = func() *T { return new(T) }
	}
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("core.NewSaved: %s is not a struct", t))
	}
	partsMu.Lock()
	defer partsMu.Unlock()
	for _, f := range jsonFields(t) {
		if other, ok := savedField[f]; ok {
			panic(fmt.Sprintf("core.NewSaved: field %q of %s is also %s's", f, t, other))
		}
		savedField[f] = t.String()
	}
	nParts++
	p := &Saved[T]{id: nParts - 1}
	saved = append(saved, savedPart{id: p.id, new: func() any { return mk() }, zero: func() any { return new(T) }})
	return p
}

// Of is the part's data in s.
func (p *Saved[T]) Of(s *Session) *T { return s.parts[p.id].(*T) }

// jsonFields are the JSON names of the exported fields of struct type t.
func jsonFields(t reflect.Type) []string {
	var out []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		out = append(out, name)
	}
	return out
}

// loadSaved makes every saved part of s and fills it from doc (nil: a new
// save).
func (s *Session) loadSaved(doc []byte) error {
	for _, p := range saved {
		if doc == nil {
			s.parts[p.id] = p.new()
			continue
		}
		v := p.zero()
		if err := json.Unmarshal(doc, v); err != nil {
			return err
		}
		s.parts[p.id] = v
	}
	return nil
}

// marshalSaved is the state document of s: the fields of every saved part
// in one object.
func (s *Session) marshalSaved() ([]byte, error) {
	doc := map[string]json.RawMessage{}
	for _, p := range saved {
		b, err := json.Marshal(s.parts[p.id])
		if err != nil {
			return nil, err
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(b, &fields); err != nil {
			return nil, err
		}
		for k, v := range fields {
			doc[k] = v
		}
	}
	return json.Marshal(doc)
}
