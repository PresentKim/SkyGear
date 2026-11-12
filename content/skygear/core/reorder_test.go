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
	h.On(10, "a", func(int) { ran = append(ran, "a") })
	h.On(20, "b", func(int) { ran = append(ran, "b") })
	h.On(30, "c", func(int) { ran = append(ran, "c") })
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
	h.On(10, "a", func(int) {})
	h.Emit(0)
	defer mustPanic(t, "Reorder", "reorder-late", "emitted")
	Reorder(func(string, []Subscription) []int { return []int{0} })
}

func TestReorderWantsEveryIndexOnce(t *testing.T) {
	for _, bad := range [][]int{{0}, {0, 0}, {0, 2}, {1, 0, 0}} {
		ResetHooks()
		h := NewHook[int]("reorder-bad")
		h.On(10, "a", func(int) {})
		h.On(20, "b", func(int) {})
		func() {
			defer mustPanic(t, "Reorder", "reorder-bad")
			Reorder(func(string, []Subscription) []int { return bad })
		}()
	}
}
