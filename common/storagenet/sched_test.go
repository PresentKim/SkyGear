package storagenet

import (
	"reflect"
	"testing"
)

func TestSchedulerBackoff(t *testing.T) {
	s := NewScheduler()
	var ran []int64
	s.Add("i", 4, 64, 0, func(now int64) bool { ran = append(ran, now); return false })
	for now := int64(0); now <= 200; now++ {
		s.Run(now)
	}
	want := []int64{0, 8, 24, 56, 120, 184}
	if !reflect.DeepEqual(ran, want) {
		t.Fatalf("ran %v want %v", ran, want)
	}
}

func TestSchedulerBusyResets(t *testing.T) {
	s := NewScheduler()
	var ran []int64
	s.Add("i", 4, 64, 0, func(now int64) bool { ran = append(ran, now); return true })
	for now := int64(0); now <= 12; now++ {
		s.Run(now)
	}
	if !reflect.DeepEqual(ran, []int64{0, 4, 8, 12}) {
		t.Fatalf("ran %v", ran)
	}
}

func TestWakeRunsNow(t *testing.T) {
	s := NewScheduler()
	n := 0
	s.Add("i", 4, 64, 0, func(int64) bool { n++; return false })
	s.Run(0)
	s.Run(1)
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	s.Wake("i", 1)
	if got := s.Run(1); got != 1 || n != 2 {
		t.Fatalf("got %d n=%d", got, n)
	}
	// wake reset the interval to min (4); the idle run doubled it: next at 1+8
	s.Run(8)
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	s.Run(9)
	if n != 3 {
		t.Fatalf("n=%d", n)
	}
	s.Wake("nope", 5) // unknown id is a no-op
}

func TestRemoveDuringRun(t *testing.T) {
	s := NewScheduler()
	a, b := 0, 0
	s.Add("a", 1, 1, 0, func(int64) bool { a++; s.Remove("b"); return true })
	s.Add("b", 1, 1, 0, func(int64) bool { b++; return true })
	s.Run(0)
	s.Run(1)
	if a != 2 || b != 0 {
		t.Fatalf("a=%d b=%d", a, b)
	}
	c := 0
	s.Add("c", 1, 1, 2, func(int64) bool { c++; s.Remove("c"); return true })
	s.Run(2)
	s.Run(3)
	if c != 1 {
		t.Fatalf("c=%d", c)
	}
}
