package core

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Trace, when set, is called with a subscriber's name just before Emit runs
// it, and with every Mark. It is nil outside tests and costs one nil check per
// Emit. Set it only while no Emit is running.
var Trace func(name string)

// Mark reports a step that is not a subscriber to Trace. It does nothing when
// Trace is nil.
func Mark(name string) {
	if t := Trace; t != nil {
		t(name)
	}
}

// installing is the feature whose subscriptions are being registered. While it
// is set, every subscriber name must start with it plus a dot.
var installing string

// Subscription is one registered subscriber, as Subscriptions lists it.
type Subscription struct {
	Hook    string
	Name    string
	Feature string
	Prio    int
}

type step[E any] struct {
	prio    int
	name    string
	feature string
	on      func(E)
	handle  func(E) bool
}

// Hook is a typed event with subscribers in explicit priority order. The event
// is passed by value, so Emit allocates nothing.
//
// Subscribers are registered at startup, before the first Emit; registering
// afterwards panics. Once registered, Emit only reads the hook, so any number
// of goroutines may Emit at once without a lock. Emit does not recover: a
// panic in a subscriber propagates to the caller unchanged and later
// subscribers do not run.
type Hook[E any] struct {
	name   string
	steps  []step[E] // ascending prio
	sealed atomic.Bool
}

type anyHook interface {
	subscriptions() []Subscription
	reorder(order func(hook string, subs []Subscription) []int)
	reset()
}

var (
	hooksMu sync.Mutex
	hooks   []anyHook
)

// NewHook returns a hook named name and lists it for Subscriptions. Call it
// from package-level variable declarations, so hooks list in declaration
// order.
func NewHook[E any](name string) *Hook[E] {
	h := &Hook[E]{name: name}
	hooksMu.Lock()
	hooks = append(hooks, h)
	hooksMu.Unlock()
	return h
}

// On subscribes f at priority prio. A subscriber added with On never stops the
// event.
func (h *Hook[E]) On(prio int, name string, f func(E)) {
	if f == nil {
		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
	}
	h.add(step[E]{prio: prio, name: name, on: f})
}

// Handle subscribes f at priority prio. When f returns true the event is
// handled: later subscribers do not run and Emit returns true.
func (h *Hook[E]) Handle(prio int, name string, f func(E) bool) {
	if f == nil {
		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
	}
	h.add(step[E]{prio: prio, name: name, handle: f})
}

func (h *Hook[E]) add(s step[E]) {
	if h.sealed.Load() {
		panic(fmt.Sprintf("core: hook %q: subscriber %q registered after the first Emit", h.name, s.name))
	}
	if installing != "" {
		if !strings.HasPrefix(s.name, installing+".") {
			panic(fmt.Sprintf("core: hook %q: subscriber %q of feature %q must be named %q.<step>", h.name, s.name, installing, installing))
		}
		s.feature = installing
	}
	i := 0
	for ; i < len(h.steps); i++ {
		o := &h.steps[i]
		if o.name == s.name {
			panic(fmt.Sprintf("core: hook %q: subscriber name %q registered twice", h.name, s.name))
		}
		if o.prio == s.prio {
			panic(fmt.Sprintf("core: hook %q: priority %d taken by %q, cannot add %q", h.name, s.prio, o.name, s.name))
		}
	}
	// Names were all checked above; find the sorted position.
	at := len(h.steps)
	for i := range h.steps {
		if h.steps[i].prio > s.prio {
			at = i
			break
		}
	}
	h.steps = append(h.steps, step[E]{})
	copy(h.steps[at+1:], h.steps[at:])
	h.steps[at] = s
}

// Emit runs the subscribers in ascending priority and reports whether a Handle
// subscriber stopped the walk. It is safe to call from many goroutines at
// once.
func (h *Hook[E]) Emit(e E) bool {
	if !h.sealed.Load() {
		h.sealed.Store(true)
	}
	tr := Trace
	for i := range h.steps {
		s := &h.steps[i]
		if tr != nil {
			tr(s.name)
		}
		if s.on != nil {
			s.on(e)
		} else if s.handle(e) {
			return true
		}
	}
	return false
}

func (h *Hook[E]) subscriptions() []Subscription {
	out := make([]Subscription, len(h.steps))
	for i, s := range h.steps {
		out[i] = Subscription{Hook: h.name, Name: s.name, Feature: s.feature, Prio: s.prio}
	}
	return out
}

func (h *Hook[E]) reset() {
	h.steps = nil
	h.sealed.Store(false)
}

// Subscriptions lists every subscriber: hooks in declaration order, then each
// hook's subscribers in the order they run.
func Subscriptions() []Subscription {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	var out []Subscription
	for _, h := range hooks {
		out = append(out, h.subscriptions()...)
	}
	return out
}
