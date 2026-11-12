package core

import (
	"slices"
	"testing"
)

// TestReorderChangesTheRunOrder: Reorder hands every hook's subscribers in
// run order and the hook then runs them in the order of the returned indexes.
func TestReorderChangesTheRunOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("reorder")
	empty := NewHook[int]("reorder-empty")
	var ran []string
	h.On(Normal, "a", func(int) { ran = append(ran, "a") })
	h.On(Normal, "b", func(int) { ran = append(ran, "b") })
	h.On(Normal, "c", func(int) { ran = append(ran, "c") })
	var asked []string
	Reorder(func(hook string, subs []Subscription) []int {
		asked = append(asked, hook)
		if hook != "reorder" {
			t.Fatalf("asked for the order of %q, a hook without subscribers", hook)
		}
		if len(subs) != 3 || subs[0].Name != "a" || subs[2].Name != "c" {
			t.Fatalf("subscribers %+v", subs)
		}
		return []int{2, 0, 1}
	})
	h.Emit(0)
	empty.Emit(0)
	if !slices.Equal(ran, []string{"c", "a", "b"}) {
		t.Fatalf("ran %v", ran)
	}
	if !slices.Contains(asked, "reorder") {
		t.Fatalf("asked %v", asked)
	}
	var names []string
	for _, s := range Subscriptions() {
		if s.Hook == "reorder" {
			names = append(names, s.Name)
		}
	}
	if !slices.Equal(names, []string{"c", "a", "b"}) {
		t.Fatalf("Subscriptions lists %v", names)
	}
}

func TestReorderAfterEmitPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("reorder-late")
	h.On(Normal, "a", func(int) {})
	h.Emit(0)
	defer mustPanic(t, "Reorder", "reorder-late", "emitted")
	Reorder(func(string, []Subscription) []int { return []int{0} })
}

func TestReorderWantsEveryIndexOnce(t *testing.T) {
	for _, bad := range [][]int{{0}, {0, 0}, {0, 2}, {1, 0, 0}} {
		ResetHooks()
		h := NewHook[int]("reorder-bad")
		h.On(Normal, "a", func(int) {})
		h.On(Normal, "b", func(int) {})
		func() {
			defer mustPanic(t, "Reorder", "reorder-bad")
			Reorder(func(string, []Subscription) []int { return bad })
		}()
	}
}

// TestReorderKeepsTheTiers: a subscriber moves inside its tier only.
func TestReorderKeepsTheTiers(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("reorder-tier")
	h.On(Low, "a", func(int) {})
	h.On(Normal, "b", func(int) {})
	defer mustPanic(t, "Reorder", "reorder-tier", "a", "b", "inside its tier")
	Reorder(func(string, []Subscription) []int { return []int{1, 0} })
}

// TestReorderKeepsAlways: after a reordering the Always subscribers still
// receive a cancelled event, wherever they stand now.
func TestReorderKeepsAlways(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("reorder-always")
	var ran []string
	h.Always(Normal, "w", func(_ int, cancelled bool) {
		if cancelled {
			ran = append(ran, "w")
		}
	})
	h.Handle(Normal, "stop", func(int) bool { return true })
	h.On(Normal, "on", func(int) { ran = append(ran, "on") })
	Reorder(func(string, []Subscription) []int { return []int{1, 2, 0} })
	if !h.Emit(0) || !slices.Equal(ran, []string{"w"}) {
		t.Fatalf("ran %v", ran)
	}
}
