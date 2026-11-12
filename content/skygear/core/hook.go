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

// Priority is the tier a subscriber runs in. A hook runs its tiers in the
// order of the constants. Each tier has one role; a subscriber takes the tier
// of what it does, not a place before or after another subscriber.
type Priority uint8

const (
	// Lowest is preprocessing: for a subscriber that must shape or block the
	// event itself before anything else.
	Lowest Priority = iota
	// Low is for shaping or blocking the event that may happen after the
	// preprocessing.
	Low
	// Normal is the general tier, for what may happen after the
	// preprocessing.
	Normal
	// High is for work of low priority that is not postprocessing.
	High
	// Highest is the last postprocessing that may still shape the event.
	Highest
	// Monitor is for handling the result without shaping or cancelling the
	// event: records, achievements and the like.
	Monitor
)

var priorityNames = [...]string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}

// String returns the tier's name in capitals, as the order pin writes it.
func (p Priority) String() string {
	if int(p) < len(priorityNames) {
		return priorityNames[p]
	}
	return fmt.Sprintf("Priority(%d)", uint8(p))
}

// Subscription is one registered subscriber, as Subscriptions lists it.
// Always is set for a subscriber that also receives a cancelled event
// (Hook.Always).
type Subscription struct {
	Hook    string
	Name    string
	Feature string
	Tier    Priority
	Always  bool
}

// A step has exactly one of on and handle. An Always step has on (the
// subscriber told that the event is live) and always set; what it is called
// with for a cancelled event is in Hook.after.
type step[E any] struct {
	tier    Priority
	always  bool
	name    string
	feature string
	on      func(E)
	handle  func(E) bool
}

// afterStep is an Always subscriber as the walk after a cancel calls it: at is
// its index in Hook.steps.
type afterStep[E any] struct {
	at   int
	name string
	f    func(E, bool)
}

// Hook is a typed event with subscribers in tiers. The event is passed by
// value, so Emit allocates nothing.
//
// The tiers run in the order of the Priority constants. Inside a tier the
// subscribers run in the order they were registered (the features in
// installation order, each feature's calls in the order it made them). That
// order only makes a run repeatable: a subscriber must not depend on its
// place inside its tier.
//
// An event is cancelled, as far as its hook is concerned, when a Handle
// subscriber returned true. The subscribers after it do not run, except those
// registered with Always.
//
// Subscribers are registered at startup, before the first Emit; registering
// afterwards panics. Once registered, Emit only reads the hook, so any number
// of goroutines may Emit at once without a lock. Emit does not recover: a
// panic in a subscriber propagates to the caller unchanged and later
// subscribers do not run.
type Hook[E any] struct {
	name  string
	steps []step[E] // ascending tier, registration order inside a tier
	// alwaysF are the functions of the Always subscribers by name; after are
	// the same subscribers in the order they run, and lastAlways is the index
	// in steps of the last of them, -1 when there is none (index keeps the two
	// up to date).
	alwaysF    map[string]func(E, bool)
	after      []afterStep[E]
	lastAlways int
	sealed     atomic.Bool
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
	h := &Hook[E]{name: name, lastAlways: -1}
	hooksMu.Lock()
	hooks = append(hooks, h)
	hooksMu.Unlock()
	return h
}

// On subscribes f in tier prio. A subscriber added with On never cancels the
// event, and does not run for an event an earlier subscriber cancelled.
func (h *Hook[E]) On(prio Priority, name string, f func(E)) {
	if f == nil {
		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
	}
	h.add(step[E]{tier: prio, name: name, on: f}, nil)
}

// Handle subscribes f in tier prio. When f returns true the event is
// cancelled: later subscribers do not run, except those registered with
// Always, and Emit returns true. Handle panics for the Monitor tier, whose
// subscribers do not cancel.
func (h *Hook[E]) Handle(prio Priority, name string, f func(E) bool) {
	if f == nil {
		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
	}
	if prio == Monitor {
		panic(fmt.Sprintf("core: hook %q: subscriber %q: a Monitor subscriber does not cancel the event; use On or Always", h.name, name))
	}
	h.add(step[E]{tier: prio, name: name, handle: f}, nil)
}

// Always subscribes f in tier prio like On, and f also receives the event
// when an earlier subscriber cancelled it (the handleCancelled option of
// other event systems): cancelled then is true, and false for a live event.
// Like On, f never cancels the event itself.
func (h *Hook[E]) Always(prio Priority, name string, f func(e E, cancelled bool)) {
	if f == nil {
		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, name))
	}
	h.add(step[E]{tier: prio, always: true, name: name, on: func(e E) { f(e, false) }}, f)
}

// add registers s; always is the function of an Always subscriber, nil for
// the others.
func (h *Hook[E]) add(s step[E], always func(E, bool)) {
	if h.sealed.Load() {
		panic(fmt.Sprintf("core: hook %q: subscriber %q registered after the first Emit", h.name, s.name))
	}
	if s.tier > Monitor {
		panic(fmt.Sprintf("core: hook %q: subscriber %q: %v is not a tier", h.name, s.name, s.tier))
	}
	if installing != "" {
		if !strings.HasPrefix(s.name, installing+".") {
			panic(fmt.Sprintf("core: hook %q: subscriber %q of feature %q must be named %q.<step>", h.name, s.name, installing, installing))
		}
		s.feature = installing
	}
	// After the last subscriber of its tier: registration order inside a tier.
	at := len(h.steps)
	for i := range h.steps {
		o := &h.steps[i]
		if o.name == s.name {
			panic(fmt.Sprintf("core: hook %q: subscriber name %q registered twice", h.name, s.name))
		}
		if o.tier > s.tier && at == len(h.steps) {
			at = i
		}
	}
	h.steps = append(h.steps, step[E]{})
	copy(h.steps[at+1:], h.steps[at:])
	h.steps[at] = s
	if always != nil {
		if h.alwaysF == nil {
			h.alwaysF = map[string]func(E, bool){}
		}
		h.alwaysF[s.name] = always
	}
	h.index()
}

// index lists the Always steps in the order they run.
func (h *Hook[E]) index() {
	h.after, h.lastAlways = nil, -1
	for i := range h.steps {
		if s := &h.steps[i]; s.always {
			h.after = append(h.after, afterStep[E]{at: i, name: s.name, f: h.alwaysF[s.name]})
			h.lastAlways = i
		}
	}
}

// Emit runs the subscribers tier by tier and reports whether a Handle
// subscriber cancelled the event. After a cancel only the Always subscribers
// still to come run. It is safe to call from many goroutines at once.
func (h *Hook[E]) Emit(e E) bool {
	if !h.sealed.Load() {
		h.sealed.Store(true)
	}
	tr := Trace
	i := 0
	for ; i < len(h.steps); i++ {
		s := &h.steps[i]
		if tr != nil {
			tr(s.name)
		}
		if s.on != nil {
			s.on(e)
		} else if s.handle(e) {
			break
		}
	}
	if i == len(h.steps) {
		return false
	}
	if i < h.lastAlways {
		h.emitCancelled(e, i)
	}
	return true
}

// emitCancelled runs the Always subscribers that stand after the step at
// index stop, for an event that step cancelled.
//
//go:noinline
func (h *Hook[E]) emitCancelled(e E, stop int) {
	tr := Trace
	for i := range h.after {
		s := &h.after[i]
		if s.at <= stop {
			continue
		}
		if tr != nil {
			tr(s.name)
		}
		s.f(e, true)
	}
}

func (h *Hook[E]) subscriptions() []Subscription {
	out := make([]Subscription, len(h.steps))
	for i, s := range h.steps {
		out[i] = Subscription{Hook: h.name, Name: s.name, Feature: s.feature, Tier: s.tier, Always: s.always}
	}
	return out
}

func (h *Hook[E]) reset() {
	h.steps, h.alwaysF, h.after = nil, nil, nil
	h.lastAlways = -1
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
