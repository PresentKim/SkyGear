package core

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

type bigEvent struct {
	A [12]uint64 // 96 bytes
}

// bigCancel is bigEvent as an event that can be cancelled.
type bigCancel struct {
	Actor
	A [12]uint64
}

// newContext returns a context nobody cancelled, as Dragonfly passes one.
func newContext() *player.Context { return &player.Context{Context: &world.Context{}} }

func newBigCancel() bigCancel { return bigCancel{Actor: Actor{Ctx: newContext()}} }

func mustPanic(t *testing.T, want ...string) {
	t.Helper()
	r := recover()
	if r == nil {
		t.Fatal("expected a panic")
	}
	msg := fmt.Sprint(r)
	for _, w := range want {
		if !strings.Contains(msg, w) {
			t.Fatalf("panic %q lacks %q", msg, w)
		}
	}
}

func traceInto(got *[]string) {
	Trace = func(name string) { *got = append(*got, name) }
}

// TestRunsInTierOrder: the tiers run in the order of the constants, whatever
// the order of registration, on both kinds of hook and with the option.
func TestRunsInTierOrder(t *testing.T) {
	ResetHooks()
	want := []string{"lowest", "low", "normal", "high", "highest", "monitor"}
	h := NewHook[int]("order")
	var got []string
	rec := func(name string) func(int) { return func(int) { got = append(got, name) } }
	h.On(Monitor, "monitor", rec("monitor"))
	h.On(Highest, "highest", rec("highest"))
	h.On(High, "high", rec("high"))
	h.On(Normal, "normal", rec("normal"))
	h.On(Low, "low", rec("low"))
	h.On(Lowest, "lowest", rec("lowest"))
	h.Emit(1)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}

	c := NewCancelHook[bigCancel]("order-cancel")
	got = nil
	recC := func(name string) func(bigCancel) { return func(bigCancel) { got = append(got, name) } }
	c.On(Monitor, "monitor", recC("monitor"), HandleCancelled)
	c.On(Highest, "highest", recC("highest"))
	c.On(High, "high", recC("high"), HandleCancelled)
	c.On(Normal, "normal", recC("normal"))
	c.On(Low, "low", recC("low"))
	c.On(Lowest, "lowest", recC("lowest"), HandleCancelled)
	c.Emit(newBigCancel())
	if !slices.Equal(got, want) {
		t.Fatalf("cancel hook: got %v want %v", got, want)
	}
}

// TestSameTierRunsInRegistrationOrder: what happens inside a tier is the
// order of registration, which makes a run repeatable (no subscriber may be
// written against it); a later registration in an earlier tier still runs
// before them.
func TestSameTierRunsInRegistrationOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("tier")
	var got []string
	rec := func(name string) func(int) { return func(int) { got = append(got, name) } }
	h.On(Normal, "b.x", rec("b.x"))
	h.On(Normal, "a.x", rec("a.x"))
	h.On(Monitor, "m.x", rec("m.x"))
	h.On(Normal, "c.x", rec("c.x"))
	h.On(Low, "l.x", rec("l.x"))
	h.Emit(0)
	want := []string{"l.x", "b.x", "a.x", "c.x", "m.x"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	var listed []string
	for _, s := range Subscriptions() {
		if s.Hook == "tier" {
			listed = append(listed, s.Name)
		}
	}
	if !slices.Equal(listed, want) {
		t.Fatalf("Subscriptions lists %v, want %v", listed, want)
	}
}

func TestPriorityNames(t *testing.T) {
	var got []string
	for p := Lowest; p <= Monitor; p++ {
		got = append(got, p.String())
	}
	if want := []string{"LOWEST", "LOW", "NORMAL", "HIGH", "HIGHEST", "MONITOR"}; !slices.Equal(got, want) {
		t.Fatalf("got %v", got)
	}
	if got := Priority(9).String(); got != "Priority(9)" {
		t.Fatalf("got %q", got)
	}
}

func TestNotATierPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("notier")
	defer mustPanic(t, "notier", "a.x", "Priority(6)", "not a tier")
	h.On(Monitor+1, "a.x", func(int) {})
}

// TestUnknownOptionPanics: HandleCancelled is the only option.
func TestUnknownOptionPanics(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("option")
	defer mustPanic(t, "option", "a.x", "unknown option")
	h.On(Normal, "a.x", func(bigCancel) {}, Option(7))
}

// TestNoSubscriberStopsAnother: every subscriber of a hook is called; on a
// hook whose event cannot be cancelled there is nothing that skips one.
func TestNoSubscriberStopsAnother(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("all")
	var traced, ran []string
	traceInto(&traced)
	defer func() { Trace = nil }()
	for i, tier := range []Priority{Lowest, Low, Normal, Normal, High, Highest, Monitor} {
		name := fmt.Sprint("s", i)
		h.On(tier, name, func(int) { ran = append(ran, name) })
	}
	h.Emit(1)
	want := []string{"s0", "s1", "s2", "s3", "s4", "s5", "s6"}
	if !slices.Equal(ran, want) || !slices.Equal(traced, want) {
		t.Fatalf("ran %v traced %v", ran, traced)
	}
}

// cancelCase is one arrangement of TestCancelSkipsDefaultSubscribers: the
// tier of the subscriber that may cancel, and the tier of the two
// subscribers under test (one default, one with HandleCancelled), registered
// before or after it.
type cancelCase struct {
	cancelTier, tier Priority
	before           bool
}

// TestCancelSkipsDefaultSubscribers: a default subscriber and one with
// HandleCancelled, in every tier, before and after a subscriber that may
// cancel in every tier but Monitor, for an event nobody cancels, one that
// subscriber cancels and one that was cancelled before Emit.
//
//   - a live event: both run and read Cancelled() == false;
//   - an event cancelled in the walk: a subscriber that stands before the
//     cancelling one ran already (and read false); one that stands after it
//     runs only with HandleCancelled, and reads true;
//   - an event cancelled before the walk: only the HandleCancelled
//     subscriber runs, and reads true; the cancelling subscriber, a default
//     one, is not called either.
func TestCancelSkipsDefaultSubscribers(t *testing.T) {
	var cases []cancelCase
	for cancel := Lowest; cancel <= Highest; cancel++ {
		for tier := Lowest; tier <= Monitor; tier++ {
			if tier != cancel {
				cases = append(cases, cancelCase{cancel, tier, tier < cancel})
				continue
			}
			cases = append(cases, cancelCase{cancel, tier, true}, cancelCase{cancel, tier, false})
		}
	}
	for _, c := range cases {
		for _, mode := range []string{"live", "cancelled", "before"} {
			withFix(t, nil, func(x *fix) {
				var ran []string
				under := func() {
					Sneak.On(c.tier, "t.default", func(e PlayerEvent) {
						ran = append(ran, fmt.Sprint("default=", e.Cancelled()))
					})
					Sneak.On(c.tier, "t.cancelled", func(e PlayerEvent) {
						ran = append(ran, fmt.Sprint("cancelled=", e.Cancelled()))
					}, HandleCancelled)
				}
				if c.before {
					under()
				}
				Sneak.On(c.cancelTier, "t.cancel", func(e PlayerEvent) {
					ran = append(ran, "cancel")
					if mode == "cancelled" {
						e.Cancel()
					}
				})
				if !c.before {
					under()
				}
				if mode == "before" {
					x.ctx.Cancel()
				}
				Sneak.Emit(PlayerEvent{x.h.actor(x.ctx)})
				var want []string
				switch {
				case mode == "before":
					want = []string{"cancelled=true"}
				case c.before:
					want = []string{"default=false", "cancelled=false", "cancel"}
				case mode == "cancelled":
					want = []string{"cancel", "cancelled=true"}
				default:
					want = []string{"cancel", "default=false", "cancelled=false"}
				}
				if !slices.Equal(ran, want) {
					t.Errorf("%+v %s: ran %v, want %v", c, mode, ran, want)
				}
				if got := x.ctx.Cancelled(); got != (mode != "live") {
					t.Errorf("%+v %s: the context is cancelled = %t", c, mode, got)
				}
			})
		}
	}
	if len(cases) != 35 {
		t.Fatalf("%d arrangements", len(cases))
	}
}

// TestAfterCancelOnlyHandleCancelledRuns: after a cancel the subscribers
// with HandleCancelled still to come run in order, traced like any other,
// whatever stands between them; a default subscriber that would cancel too is
// not called, and a skipped subscriber is not traced.
func TestAfterCancelOnlyHandleCancelledRuns(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("after")
	var traced, ran []string
	traceInto(&traced)
	defer func() { Trace = nil }()
	rec := func(name string) func(bigCancel) {
		return func(e bigCancel) { ran = append(ran, fmt.Sprint(name, "=", e.Cancelled())) }
	}
	h.On(Lowest, "a", rec("a"), HandleCancelled)
	h.On(Low, "cancel", func(e bigCancel) { ran = append(ran, "cancel"); e.Cancel() })
	h.On(Low, "b", rec("b"), HandleCancelled)
	h.On(Normal, "on", rec("on"))
	h.On(Normal, "cancel2", func(e bigCancel) { ran = append(ran, "cancel2"); e.Cancel() })
	h.On(High, "c", rec("c"), HandleCancelled)
	h.On(Monitor, "monitor", rec("monitor"))
	h.On(Monitor, "d", rec("d"), HandleCancelled)
	e := newBigCancel()
	h.Emit(e)
	if !e.Cancelled() {
		t.Fatal("the event is not cancelled")
	}
	if want := []string{"a=false", "cancel", "b=true", "c=true", "d=true"}; !slices.Equal(ran, want) {
		t.Fatalf("ran %v, want %v", ran, want)
	}
	if want := []string{"a", "cancel", "b", "c", "d"}; !slices.Equal(traced, want) {
		t.Fatalf("traced %v, want %v", traced, want)
	}
	var opted []string
	for _, s := range Subscriptions() {
		if s.Hook == "after" && s.HandleCancelled {
			opted = append(opted, s.Name)
		}
	}
	if want := []string{"a", "b", "c", "d"}; !slices.Equal(opted, want) {
		t.Fatalf("Subscriptions marks %v as HandleCancelled, want %v", opted, want)
	}
}

// TestEventWithoutContextIsNeverCancelled: an event built without a context
// (a test that emits by hand) reaches every subscriber; Cancel does nothing.
func TestEventWithoutContextIsNeverCancelled(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("nocontext")
	n := 0
	h.On(Low, "a", func(e bigCancel) { n++; e.Cancel() })
	h.On(Normal, "b", func(e bigCancel) {
		if n++; e.Cancelled() {
			t.Error("an event without a context is cancelled")
		}
	})
	h.On(Monitor, "c", func(bigCancel) { n++ }, HandleCancelled)
	h.Emit(bigCancel{})
	if n != 3 {
		t.Fatalf("%d subscribers ran, want 3", n)
	}
}

// TestMonitorCancelPanicsWhileTraced: a Monitor subscriber does not cancel.
// While Trace is set (the tests that trace) Emit panics when one did; a
// Monitor subscriber that receives an event cancelled before it is fine.
func TestMonitorCancelPanicsWhileTraced(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("monitor")
	var traced []string
	cancel := false
	h.On(Normal, "first", func(e bigCancel) {
		if cancel {
			e.Cancel()
		}
	})
	h.On(Monitor, "bad.x", func(e bigCancel) { e.Cancel() }, HandleCancelled)
	h.Emit(newBigCancel()) // not traced: no check
	traceInto(&traced)
	defer func() { Trace = nil }()
	cancel = true
	h.Emit(newBigCancel()) // cancelled before the Monitor subscriber: its Cancel changes nothing
	cancel = false
	defer mustPanic(t, "monitor", "bad.x", "Monitor subscriber", "cancelled the event")
	h.Emit(newBigCancel())
}

// TestOnlyActorEventsCancel: Cancel and Cancelled exist exactly on the
// events that carry the Dragonfly context (they embed Actor); the events of
// Jump, Death, Respawn and of the session and world hooks have neither, so a
// subscriber that tried to cancel one would not compile.
func TestOnlyActorEventsCancel(t *testing.T) {
	cancellable := []any{StartBreakEvent{}, BlockBreakEvent{}, BlockPlaceEvent{}, UseOnBlockEvent{}, EntityEvent{}, UseEvent{}, MoveEvent{}, PlayerEvent{}, ItemEvent{}, ItemDamageEvent{}, HurtEvent{}}
	for _, e := range cancellable {
		if _, ok := e.(Cancellable); !ok {
			t.Errorf("%T cannot be cancelled", e)
		}
	}
	for _, e := range []any{NoticeEvent{}, RespawnEvent{}, Tick{}, DayEvent{}, Presence{}, ColumnEvent{}, DespawnEvent{}, Who{}} {
		typ := reflect.TypeOf(e)
		for _, m := range []string{"Cancel", "Cancelled"} {
			if _, has := typ.MethodByName(m); has {
				t.Errorf("%v has %s: an event that cannot be cancelled must not have it", typ, m)
			}
		}
		if _, ok := typ.FieldByName("Ctx"); ok {
			t.Errorf("%v has a Ctx field", typ)
		}
	}
}

func TestSameNamePanics(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("name")
	h.On(Normal, "a.x", func(bigCancel) {})
	defer mustPanic(t, "name", "a.x")
	h.On(High, "a.x", func(bigCancel) {}, HandleCancelled)
}

func TestNilFuncPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("nilf")
	c := NewCancelHook[bigCancel]("nilc")
	func() {
		defer mustPanic(t, "nilc", "h")
		c.On(Normal, "h", nil)
	}()
	func() {
		defer mustPanic(t, "nilc", "w")
		c.On(Normal, "w", nil, HandleCancelled)
	}()
	defer mustPanic(t, "nilf", "a")
	h.On(Normal, "a", nil)
}

func TestInstallingPrefix(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("pfx")
	SetInstalling("feat")
	h.On(Normal, "feat.ok", func(int) {})
	func() {
		defer mustPanic(t, "pfx", "other.bad", "feat")
		h.On(Normal, "other.bad", func(int) {})
	}()
	found := false
	for _, s := range Subscriptions() {
		if s.Hook == "pfx" {
			found = true
			if s.Feature != "feat" || s.Name != "feat.ok" || s.Tier != Normal || s.HandleCancelled {
				t.Fatalf("subscription %+v", s)
			}
		}
	}
	if !found {
		t.Fatal("subscription missing")
	}
}

func TestRegisterAfterEmitPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("late")
	h.On(Normal, "a", func(int) {})
	h.Emit(0)
	defer mustPanic(t, "late", "b")
	h.On(Normal, "b", func(int) {})
}

// TestRegisterAfterEmitPanicsOnACancelHook: after the first Emit, of a live
// event, a cancelled one or one without a context, a CancelHook refuses a
// subscriber with and without the option, and refuses neither before it.
func TestRegisterAfterEmitPanicsOnACancelHook(t *testing.T) {
	kinds := map[string]func(h *CancelHook[bigCancel], name string){
		"default":   func(h *CancelHook[bigCancel], name string) { h.On(Normal, name, func(bigCancel) {}) },
		"cancelled": func(h *CancelHook[bigCancel], name string) { h.On(Monitor, name, func(bigCancel) {}, HandleCancelled) },
	}
	events := map[string]func() bigCancel{
		"live":      newBigCancel,
		"cancelled": func() bigCancel { e := newBigCancel(); e.Cancel(); return e },
		"nocontext": func() bigCancel { return bigCancel{} },
	}
	for _, event := range events {
		for kind, add := range kinds {
			ResetHooks()
			h := NewCancelHook[bigCancel]("late")
			for other, add := range kinds {
				add(h, "before."+other)
			}
			h.Emit(event())
			func() {
				defer mustPanic(t, "late", "after."+kind, "after the first Emit")
				add(h, "after."+kind)
			}()
		}
	}
}

// TestEmitConcurrentWithCancel: many goroutines emit events of their own
// that a subscriber cancels; for every event the subscribers before the
// cancel run once, the HandleCancelled subscribers after it run once and
// read true, the default subscribers after it never run.
func TestEmitConcurrentWithCancel(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("conccancel")
	var before, optedBeforeLive, optedBeforeCancelled, after, optedAfterLive, optedAfterCancelled atomic.Int64
	count := func(live, cancelled *atomic.Int64) func(bigCancel) {
		return func(e bigCancel) {
			if e.Cancelled() {
				cancelled.Add(1)
			} else {
				live.Add(1)
			}
		}
	}
	h.On(Lowest, "on.before", func(bigCancel) { before.Add(1) })
	h.On(Low, "opted.before", count(&optedBeforeLive, &optedBeforeCancelled), HandleCancelled)
	h.On(Normal, "cancel", func(e bigCancel) { e.Cancel() })
	h.On(Normal, "on.after", func(bigCancel) { after.Add(1) })
	h.On(High, "opted.after", count(&optedAfterLive, &optedAfterCancelled), HandleCancelled)
	h.On(Monitor, "monitor.after", func(bigCancel) { after.Add(1) })
	const goroutines, emits = 16, 1000
	var notCancelled atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < emits; i++ {
				e := newBigCancel()
				if h.Emit(e); !e.Cancelled() {
					notCancelled.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	const n = goroutines * emits
	if notCancelled.Load() != 0 || before.Load() != n || optedBeforeLive.Load() != n || optedBeforeCancelled.Load() != 0 ||
		after.Load() != 0 || optedAfterLive.Load() != 0 || optedAfterCancelled.Load() != n {
		t.Fatalf("not cancelled %d, before %d, opted before %d/%d (live/cancelled), default after %d, opted after %d/%d",
			notCancelled.Load(), before.Load(), optedBeforeLive.Load(), optedBeforeCancelled.Load(),
			after.Load(), optedAfterLive.Load(), optedAfterCancelled.Load())
	}
}

func TestEmptyEmitStillSealsRegistration(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("early")
	h.Emit(0) // no steps: an empty emit still counts as emission
	defer mustPanic(t, "early")
	h.On(Normal, "a", func(int) {})
}

func TestSubscriptionsRunOrder(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("T")
	h.On(High, "b.x", func(bigCancel) {}, HandleCancelled)
	h.On(Low, "a.x", func(bigCancel) {})
	var got []Subscription
	for _, s := range Subscriptions() {
		if s.Hook == "T" {
			got = append(got, s)
		}
	}
	want := []Subscription{{Hook: "T", Tier: Low, Name: "a.x"}, {Hook: "T", Tier: High, Name: "b.x", HandleCancelled: true}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestSubscriptionsHookDeclarationOrder(t *testing.T) {
	ResetHooks()
	h1 := NewHook[int]("decl1")
	h2 := NewCancelHook[bigCancel]("decl2")
	h2.On(Lowest, "z", func(bigCancel) {})
	h1.On(Monitor, "y", func(int) {})
	var names []string
	for _, s := range Subscriptions() {
		if s.Hook == "decl1" || s.Hook == "decl2" {
			names = append(names, s.Name)
		}
	}
	if !slices.Equal(names, []string{"y", "z"}) {
		t.Fatalf("got %v", names)
	}
}

func TestTraceNamesInOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("trace")
	var traced, ran []string
	h.On(Normal, "b", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":b")) })
	h.On(Low, "a", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":a")) })
	traceInto(&traced)
	defer func() { Trace = nil }()
	h.Emit(0)
	if !slices.Equal(traced, []string{"a", "b"}) {
		t.Fatalf("traced %v", traced)
	}
	// Trace ran just before each step: 1 name seen by a, 2 by b.
	if !slices.Equal(ran, []string{"1:a", "2:b"}) {
		t.Fatalf("ran %v", ran)
	}
}

func TestMarkCallsTrace(t *testing.T) {
	ResetHooks()
	Mark("nobody listens") // nil Trace is fine
	var got []string
	traceInto(&got)
	defer func() { Trace = nil }()
	Mark("m1")
	Mark("m2")
	if !slices.Equal(got, []string{"m1", "m2"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSubscriberPanicPropagates(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("boom")
	later := false
	h.On(Normal, "a", func(int) { panic("boom!") })
	h.On(Normal, "b", func(int) { later = true })
	func() {
		defer func() {
			if r := recover(); r != "boom!" {
				t.Fatalf("recovered %v", r)
			}
		}()
		h.Emit(0)
	}()
	if later {
		t.Fatal("later step ran after a panic")
	}
}

func TestEmitConcurrent(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("conc")
	var n atomic.Int64
	for i := 0; i < 8; i++ {
		h.On(Priority(i%6), fmt.Sprint("s", i), func(e int) { n.Add(int64(e)) })
	}
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				h.Emit(1)
			}
		}()
	}
	wg.Wait()
	if n.Load() != 16*1000*8 {
		t.Fatalf("n=%d", n.Load())
	}
}

// fillHook gives h n subscribers spread over the tiers.
func fillHook(h *Hook[bigEvent], n int) {
	for i := 0; i < n; i++ {
		h.On(Priority(i%6), fmt.Sprint("s", i), func(e bigEvent) { _ = e.A[0] })
	}
}

// fillCancelHook gives h n subscribers spread over the tiers that may cancel,
// none of which cancels the events the tests emit, every fourth with
// HandleCancelled.
func fillCancelHook(h *CancelHook[bigCancel], n int) {
	for i := 0; i < n; i++ {
		f := func(e bigCancel) {
			if e.A[0] == 99 {
				e.Cancel()
			}
		}
		if i%4 == 3 {
			h.On(Priority(i%5), fmt.Sprint("s", i), f, HandleCancelled)
		} else {
			h.On(Priority(i%5), fmt.Sprint("s", i), f)
		}
	}
}

func TestEmitAllocatesNothing(t *testing.T) {
	for _, n := range []int{0, 1, 4, 8, 32} {
		ResetHooks()
		h := NewHook[bigEvent](fmt.Sprintf("alloc-%d", n))
		fillHook(h, n)
		var e bigEvent
		e.A[0] = 1
		if got := testing.AllocsPerRun(200, func() { h.Emit(e) }); got != 0 {
			t.Errorf("hook, %d subscribers: %v allocs", n, got)
		}
		c := NewCancelHook[bigCancel](fmt.Sprintf("alloc-cancel-%d", n))
		fillCancelHook(c, n)
		ce := newBigCancel()
		ce.A[0] = 1
		if got := testing.AllocsPerRun(200, func() { c.Emit(ce) }); got != 0 {
			t.Errorf("cancel hook, %d subscribers: %v allocs", n, got)
		}
	}
}

// TestEmitAllocatesNothingAfterCancel: the walk over a cancelled event, in
// which the default subscribers are skipped, allocates nothing either.
func TestEmitAllocatesNothingAfterCancel(t *testing.T) {
	ResetHooks()
	h := NewCancelHook[bigCancel]("alloc-cancelled")
	seen, skipped := 0, 0
	h.On(Lowest, "cancel", func(e bigCancel) { e.Cancel() })
	h.On(Normal, "on", func(bigCancel) { skipped++ })
	h.On(Normal, "a", func(e bigCancel) {
		if e.Cancelled() {
			seen++
		}
	}, HandleCancelled)
	h.On(Monitor, "b", func(e bigCancel) {
		if e.Cancelled() {
			seen++
		}
	}, HandleCancelled)
	e := newBigCancel()
	if got := testing.AllocsPerRun(200, func() { h.Emit(e) }); got != 0 {
		t.Errorf("%v allocs", got)
	}
	if seen == 0 || seen%2 != 0 || skipped != 0 {
		t.Fatalf("the HandleCancelled subscribers saw %d cancelled events, the default one ran %d times", seen, skipped)
	}
}

// BenchmarkEmit emits to n subscribers: of a hook whose event cannot be
// cancelled (hook=plain) and of a CancelHook whose event nobody cancels
// (hook=cancel), which asks the context before every default subscriber.
func BenchmarkEmit(b *testing.B) {
	for _, n := range []int{0, 1, 4, 8, 32} {
		b.Run(fmt.Sprintf("hook=plain/subs=%d", n), func(b *testing.B) {
			ResetHooks()
			h := NewHook[bigEvent](fmt.Sprintf("bench-%d", n))
			fillHook(h, n)
			var e bigEvent
			e.A[0] = 1
			b.ReportAllocs()
			for b.Loop() {
				h.Emit(e)
			}
		})
	}
	for _, n := range []int{0, 1, 4, 8, 32} {
		b.Run(fmt.Sprintf("hook=cancel/subs=%d", n), func(b *testing.B) {
			ResetHooks()
			h := NewCancelHook[bigCancel](fmt.Sprintf("bench-cancel-%d", n))
			fillCancelHook(h, n)
			e := newBigCancel()
			e.A[0] = 1
			b.ReportAllocs()
			for b.Loop() {
				h.Emit(e)
			}
		})
	}
}

var moveSink float64

// BenchmarkEmitMove emits a MoveEvent to four subscribers, the shape of the
// move hook: one that may cancel first, one that may cancel last, a record,
// and a record with HandleCancelled.
func BenchmarkEmitMove(b *testing.B) {
	ResetHooks()
	h := NewCancelHook[MoveEvent]("bench-move")
	h.On(Monitor, "a", func(e MoveEvent) { moveSink += e.Rot[0] + e.Pos[0] }, HandleCancelled)
	h.On(Lowest, "b", func(e MoveEvent) {
		if e.P == nil && e.Pos[1] > 1e9 {
			e.Cancel()
		}
	})
	h.On(Monitor, "c", func(e MoveEvent) { moveSink += e.Pos[2] })
	h.On(Highest, "d", func(e MoveEvent) {
		if e.Pos[1] < -1e9 {
			e.Cancel()
		}
	})
	e := MoveEvent{Actor: Actor{Ctx: newContext()}, Pos: [3]float64{1, 64, 2}}
	b.ReportAllocs()
	for b.Loop() {
		h.Emit(e)
	}
}
