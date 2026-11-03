package core

import (
	"encoding/json"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// TestSavedRoundTrip: marshal then load gives equal parts, also with empty
// and nil parts, and the spliced document is valid JSON.
func TestSavedRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    savedA
		c    []int
	}{
		{"filled", savedA{A: 3, B: "x"}, []int{1, 2}},
		{"empty parts", savedA{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New(Config{})
			*partA.Of(s) = tc.a
			partB.Of(s).C = tc.c
			b, err := s.marshalSaved()
			if err != nil {
				t.Fatal(err)
			}
			if !json.Valid(b) {
				t.Fatalf("invalid document %s", b)
			}
			s2 := New(Config{State: b})
			if !reflect.DeepEqual(*partA.Of(s2), tc.a) || !reflect.DeepEqual(partB.Of(s2).C, tc.c) {
				t.Fatalf("loaded %+v %v from %s", *partA.Of(s2), partB.Of(s2).C, b)
			}
		})
	}
}

// TestLoadIgnoresUnknownAndOtherFields: fields no part owns are skipped, a
// part whose omitempty fields are absent keeps its zero values, and a null
// document loads nothing.
func TestLoadIgnoresUnknownAndOtherFields(t *testing.T) {
	s := New(Config{State: []byte(`{"a":7,"zzz":{"x":[1,2]},"c":[4]}`)})
	if a := partA.Of(s); a.A != 7 || a.B != "" {
		t.Fatalf("part A = %+v", *a)
	}
	if c := partB.Of(s).C; len(c) != 1 || c[0] != 4 {
		t.Fatalf("part B = %v", c)
	}
	s = New(Config{State: []byte(`null`)})
	if a := partA.Of(s); a.A != 0 {
		t.Fatalf("null document loaded %+v", *a)
	}
}

// TestFlushEveryInterval: with an interval of 10s and a change every second,
// FlushIfDue flushes once per 10 calls.
func TestFlushEveryInterval(t *testing.T) {
	s := New(Config{})
	s.FlushInterval = 10 * time.Second
	s.Flush()
	defer s.Close()
	base := time.Now()
	n := 0
	for i := 1; i <= 30; i++ {
		s.MarkDirty()
		if s.FlushIfDue(base.Add(time.Duration(i) * time.Second)) {
			n++
		}
	}
	if n != 3 {
		t.Fatalf("flushed %d times in 30 seconds, want 3", n)
	}
	if s.FlushIfDue(base.Add(time.Hour)) {
		t.Fatal("a clean session flushed")
	}
}

// TestAutosaveFlushes: Flush stores even a clean session.
func TestAutosaveFlushes(t *testing.T) {
	var n atomic.Int32
	s := New(Config{Store: func([]byte) error { n.Add(1); return nil }})
	s.Flush()
	deadline := time.Now().Add(2 * time.Second)
	for n.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if n.Load() == 0 {
		t.Fatal("Flush of a clean session stored nothing")
	}
	s.Close()
}

type benchCrop struct {
	X, Y, Z int
	Kind    string
	Age     int
}

type benchPart struct {
	Crops []benchCrop `json:"benchCrops,omitempty"`
}

var benchSaved = NewSaved[benchPart](nil)

func BenchmarkMarshalSaved(b *testing.B) {
	s := New(Config{})
	for i := 0; i < 2000; i++ {
		benchSaved.Of(s).Crops = append(benchSaved.Of(s).Crops, benchCrop{i, 64, i * 2, "wheat", i % 8})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.marshalSaved(); err != nil {
			b.Fatal(err)
		}
	}
}
