package core

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/world"
)

// Trace, when set, is called with a subscriber's name just before Emit runs
// it, and with every Mark. It is nil outside tests and costs one nil check per
// Emit. Set it only while no Emit is running.
//
// While Trace is set, a CancelHook also checks its Monitor subscribers: Emit
// panics when one of them cancelled the event.
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

// Option changes how a subscriber of a CancelHook is called.
type Option uint8

// HandleCancelled makes a subscriber also receive a cancelled event (the
// handleCancelled option of other event systems). Without it a subscriber is
// not called once the event is cancelled. The subscriber reads the state from
// the event: e.Cancelled().
const HandleCancelled Option = 1

// Subscription is one registered subscriber, as Subscriptions lists it.
// HandleCancelled is set for a subscriber that also receives a cancelled
// event (the HandleCancelled option).
type Subscription struct {
	Hook            string
	Name            string
	Feature         string
	Tier            Priority
	HandleCancelled bool
}

// A step is one subscriber. cancelled is set when it also receives a
// cancelled event.
type step[E any] struct {
	tier      Priority
	cancelled bool
	name      string
	feature   string
	f         func(E)
}

// Hook is a typed event with subscribers in tiers, for an event that cannot
// be cancelled: every subscriber is called for every event. The event is
// passed by value, so Emit allocates nothing. CancelHook is the hook of an
// event that can be cancelled.
//
// The tiers run in the order of the Priority constants. The subscribers of
// one tier have no order among themselves: nothing may be written against
// the order they happen to run in (it is the order of registration, which
// only makes a run repeatable, and the tests run them in others). What must
// come before or after belongs in another tier, by its role.
//
// No subscriber can keep another from being called. A subscriber acts when
// its own conditions hold (its state, the item in hand, the block, the mode
// the player is in), not because no subscriber before it acted.
//
// Subscribers are registered at startup, before the first Emit; registering
// afterwards panics. Once registered, Emit only reads the hook, so any number
// of goroutines may Emit at once without a lock. Emit does not recover: a
// panic in a subscriber propagates to the caller unchanged and later
// subscribers do not run.
type Hook[E any] struct {
	name   string
	steps  []step[E] // ascending tier, registration order inside a tier
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

// On subscribes f in tier prio. f is called for every event.
func (h *Hook[E]) On(prio Priority, name string, f func(E)) {
	h.add(step[E]{tier: prio, name: name, f: f})
}

// add registers s.
func (h *Hook[E]) add(s step[E]) {
	if s.f == nil {
		panic(fmt.Sprintf("core: hook %q: subscriber %q has a nil func", h.name, s.name))
	}
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
	// After the last subscriber of its tier.
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
}

// Emit calls every subscriber, tier by tier. It is safe to call from many
// goroutines at once.
func (h *Hook[E]) Emit(e E) {
	if !h.sealed.Load() {
		h.sealed.Store(true)
	}
	tr := Trace
	for i := range h.steps {
		s := &h.steps[i]
		if tr != nil {
			tr(s.name)
		}
		s.f(e)
	}
}

func (h *Hook[E]) subscriptions() []Subscription {
	out := make([]Subscription, len(h.steps))
	for i, s := range h.steps {
		out[i] = Subscription{Hook: h.name, Name: s.name, Feature: s.feature, Tier: s.tier, HandleCancelled: s.cancelled}
	}
	return out
}

func (h *Hook[E]) reset() {
	h.steps = nil
	h.sealed.Store(false)
}

// Cancellable is an event that can be cancelled: one that carries the
// Dragonfly context of the event, which every event that embeds Actor does.
// Cancel and Cancelled exist on these events only, so cancelling an event
// that cannot be cancelled does not compile.
type Cancellable interface {
	Cancel()
	Cancelled() bool
	cancelState() *world.Context
}

// CancelHook is the hook of an event that can be cancelled. It is a Hook
// whose subscribers are not called once the event is cancelled, unless they
// asked for cancelled events with the HandleCancelled option.
//
// An event is cancelled when its Dragonfly context is: a subscriber called
// e.Cancel(), in this walk or before it. Cancelling says two things at once:
// the action the player asked for does not happen in Dragonfly, and the
// subscribers that follow are not called for it. There is no other way to
// keep a subscriber from being called. A subscriber cancels only when that is
// what it means ("in this state I handle this, and nothing else may"); when
// the action may go on, it does its work and leaves the event alone.
//
// Because cancelling skips the subscribers of the same tier that happen to
// run later, two subscribers of one tier that could act on the same event,
// one of them cancelling, must exclude each other by their own conditions or
// stand in different tiers.
type CancelHook[E Cancellable] struct {
	hook Hook[E]
}

// NewCancelHook returns a hook for a cancellable event, named name, and lists
// it for Subscriptions like NewHook.
func NewCancelHook[E Cancellable](name string) *CancelHook[E] {
	h := &CancelHook[E]{}
	h.hook.name = name
	hooksMu.Lock()
	hooks = append(hooks, &h.hook)
	hooksMu.Unlock()
	return h
}

// On subscribes f in tier prio. f is not called for a cancelled event, unless
// the HandleCancelled option is given: then it is called for every event and
// reads e.Cancelled() if it cares. A Monitor subscriber does not cancel.
func (h *CancelHook[E]) On(prio Priority, name string, f func(E), opts ...Option) {
	s := step[E]{tier: prio, name: name, f: f}
	for _, o := range opts {
		if o != HandleCancelled {
			panic(fmt.Sprintf("core: hook %q: subscriber %q: unknown option %d", h.hook.name, name, o))
		}
		s.cancelled = true
	}
	h.hook.add(s)
}

// Emit calls the subscribers tier by tier. Once the event is cancelled, only
// the subscribers registered with HandleCancelled are still called. It is
// safe to call from many goroutines at once.
func (h *CancelHook[E]) Emit(e E) {
	c := e.cancelState()
	if c == nil {
		// An event built without a context (a test) is never cancelled.
		h.hook.Emit(e)
		return
	}
	if !h.hook.sealed.Load() {
		h.hook.sealed.Store(true)
	}
	tr := Trace
	steps := h.hook.steps
	for i := range steps {
		s := &steps[i]
		if c.Cancelled() && !s.cancelled {
			continue
		}
		if tr == nil {
			s.f(e)
			continue
		}
		tr(s.name)
		was := c.Cancelled()
		s.f(e)
		if s.tier == Monitor && !was && c.Cancelled() {
			panic(fmt.Sprintf("core: hook %q: Monitor subscriber %q cancelled the event", h.hook.name, s.name))
		}
	}
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
