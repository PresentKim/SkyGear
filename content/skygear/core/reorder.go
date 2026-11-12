package core

import "fmt"

// Reorder is a seam for the tests that prove no subscriber depends on its
// place inside its tier. It changes the order the subscribers of every hook
// run in: order is called once per hook that has subscribers, with the hook's
// name and its subscribers in the order they run, and returns the new order
// as indexes into subs. The new order may only move a subscriber inside its
// tier.
//
// Call it after Install and before the first Emit. It panics when a hook has
// emitted already, when order does not return every index once or when the
// new order does not keep the tiers in order. Nothing outside tests calls it.
func Reorder(order func(hook string, subs []Subscription) []int) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	for _, h := range hooks {
		h.reorder(order)
	}
}

func (h *Hook[E]) reorder(order func(hook string, subs []Subscription) []int) {
	if len(h.steps) == 0 {
		return
	}
	if h.sealed.Load() {
		panic(fmt.Sprintf("core: Reorder: hook %q has emitted", h.name))
	}
	idx := order(h.name, h.subscriptions())
	if len(idx) != len(h.steps) {
		panic(fmt.Sprintf("core: Reorder: hook %q: %d indexes for %d subscribers", h.name, len(idx), len(h.steps)))
	}
	seen := make([]bool, len(h.steps))
	steps := make([]step[E], len(h.steps))
	for to, from := range idx {
		if from < 0 || from >= len(h.steps) || seen[from] {
			panic(fmt.Sprintf("core: Reorder: hook %q: index %d is out of range or repeated", h.name, from))
		}
		seen[from] = true
		steps[to] = h.steps[from]
		if to > 0 && steps[to].tier < steps[to-1].tier {
			panic(fmt.Sprintf("core: Reorder: hook %q: %q (%v) after %q (%v): a subscriber moves inside its tier only", h.name, steps[to].name, steps[to].tier, steps[to-1].name, steps[to-1].tier))
		}
	}
	h.steps = steps
	h.index()
}
