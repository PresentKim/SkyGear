package core

import (
	"encoding/json"
	"testing"
)

type savedA struct {
	A int    `json:"a,omitempty"`
	B string `json:"b"`
}

type savedB struct {
	C []int `json:"c,omitempty"`
}

var (
	partA = NewSaved(func() *savedA { return &savedA{A: 1} })
	partB = NewSaved[savedB](nil)
	runA  = NewPart[int](nil)
)

// TestSavedOneDocument: the saved parts are fields of one document, a new
// save gets the parts' new data and a saved one reads its fields, leaving
// the ones it lacks zero.
func TestSavedOneDocument(t *testing.T) {
	var stored []byte
	s := New(Config{Store: func(b []byte) error { stored = b; return nil }})
	if partA.Of(s).A != 1 {
		t.Fatalf("new save: A = %d, want 1", partA.Of(s).A)
	}
	partA.Of(s).B = "x"
	partB.Of(s).C = []int{2}
	*runA.Of(s) = 5
	s.Close()
	var doc map[string]any
	if err := json.Unmarshal(stored, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc) != 3 || doc["b"] != "x" {
		t.Fatalf("document %s", stored)
	}

	s2 := New(Config{State: []byte(`{"b":"y","other":true}`)})
	if a := partA.Of(s2); a.A != 0 || a.B != "y" {
		t.Errorf("read part A %+v", a)
	}
	if *runA.Of(s2) != 0 {
		t.Error("a runtime part was carried over")
	}
}

// TestSavedFieldsOwnedOnce: a field of two parts is refused.
func TestSavedFieldsOwnedOnce(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a second part with field b was accepted")
		}
	}()
	type clash struct {
		B string `json:"b"`
	}
	NewSaved[clash](nil)
}
