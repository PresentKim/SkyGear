package core

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type bigEvent struct {
	A [12]uint64 // 96 bytes
}

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
// the order of registration, with every kind of subscriber.
func TestRunsInTierOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("order")
	var got []string
	rec := func(name string) func(int) { return func(int) { got = append(got, name) } }
	h.On(Monitor, "monitor", rec("monitor"))
	h.Always(Highest, "highest", func(int, bool) { got = append(got, "highest") })
	h.On(High, "high", rec("high"))
	h.Handle(Normal, "normal", func(int) bool { got = append(got, "normal"); return false })
	h.On(Low, "low", rec("low"))
	h.Handle(Lowest, "lowest", func(int) bool { got = append(got, "lowest"); return false })
	if h.Emit(1) {
		t.Fatal("Emit reported a cancel")
	}
	if want := []string{"lowest", "low", "normal", "high", "highest", "monitor"}; !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// TestSameTierRunsInRegistrationOrder: inside a tier the subscribers run in
// the order they were registered, and a later registration in an earlier tier
// still runs before them.
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

// TestHandleAtMonitorPanics: a Monitor subscriber does not cancel, so it is
// registered with On or Always.
func TestHandleAtMonitorPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("monitor")
	h.On(Monitor, "a.x", func(int) {})
	h.Always(Monitor, "b.x", func(int, bool) {})
	defer mustPanic(t, "monitor", "c.x", "Monitor", "does not cancel")
	h.Handle(Monitor, "c.x", func(int) bool { return false })
}

func TestStopSkipsLaterSteps(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("stop")
	var traced, ran []string
	traceInto(&traced)
	defer func() { Trace = nil }()
	h.Handle(Normal, "a.x", func(int) bool { ran = append(ran, "a.x"); return true })
	h.On(Normal, "b.x", func(int) { ran = append(ran, "b.x") })
	h.Handle(High, "c.x", func(int) bool { ran = append(ran, "c.x"); return false })
	h.On(Monitor, "d.x", func(int) { ran = append(ran, "d.x") })
	if !h.Emit(1) {
		t.Fatal("Emit did not report the stop")
	}
	if !slices.Equal(ran, []string{"a.x"}) || !slices.Equal(traced, []string{"a.x"}) {
		t.Fatalf("ran %v traced %v", ran, traced)
	}
}

// alwaysCase is one arrangement of TestAlways: the tier of the subscriber
// that may cancel, and the tier of the two subscribers under test (one
// registered with On, one with Always), registered before or after it.
type alwaysCase struct {
	stopTier, tier Priority
	before         bool
}

// TestAlways: an On subscriber and an Always subscriber, in every tier,
// before and after a Handle subscriber in every tier but Monitor, for an
// event that is not cancelled, one that the Handle subscriber cancels and one
// whose Dragonfly context a subscriber cancelled while the walk went on.
//
//   - a live event: both run, the Always subscriber is told cancelled=false;
//   - a cancelled event: a subscriber that stands before the cancelling one
//     ran already (told false); one that stands after it runs only when it
//     was registered with Always, and is told true;
//   - a cancelled context without a cancelled walk is a live event for the
//     hook: both run and are told false; Actor.Cancelled tells them.
func TestAlways(t *testing.T) {
	var cases []alwaysCase
	for stop := Lowest; stop <= Highest; stop++ {
		for tier := Lowest; tier <= Monitor; tier++ {
			if tier != stop {
				cases = append(cases, alwaysCase{stop, tier, tier < stop})
				continue
			}
			cases = append(cases, alwaysCase{stop, tier, true}, alwaysCase{stop, tier, false})
		}
	}
	for _, c := range cases {
		for _, mode := range []string{"live", "cancelled", "context"} {
			withFix(t, nil, func(x *fix) {
				var ran []string
				var onSaw, alwaysSaw bool
				under := func() {
					Sneak.On(c.tier, "t.on", func(e PlayerEvent) {
						ran = append(ran, "on")
						onSaw = e.Cancelled()
					})
					Sneak.Always(c.tier, "t.always", func(e PlayerEvent, cancelled bool) {
						ran = append(ran, fmt.Sprint("always=", cancelled))
						alwaysSaw = e.Cancelled()
					})
				}
				if c.before {
					under()
				}
				Sneak.Handle(c.stopTier, "t.stop", func(e PlayerEvent) bool {
					ran = append(ran, "stop")
					if mode == "context" {
						e.Cancel()
					}
					return mode == "cancelled"
				})
				if !c.before {
					under()
				}
				stopped := Sneak.Emit(PlayerEvent{Actor{S: x.s, P: x.p, Tx: x.tx, Ctx: x.ctx}})
				if stopped != (mode == "cancelled") {
					t.Errorf("%+v %s: Emit = %t", c, mode, stopped)
				}
				var want []string
				switch {
				case c.before:
					want = []string{"on", "always=false", "stop"}
				case mode == "cancelled":
					want = []string{"stop", "always=true"}
				default:
					want = []string{"stop", "on", "always=false"}
				}
				if !slices.Equal(ran, want) {
					t.Errorf("%+v %s: ran %v, want %v", c, mode, ran, want)
				}
				// The context is cancelled for those that run after the cancel.
				if wantSaw := mode == "context" && !c.before; alwaysSaw != wantSaw || (onSaw != wantSaw) {
					t.Errorf("%+v %s: Cancelled() was %t for On and %t for Always, want %t", c, mode, onSaw, alwaysSaw, wantSaw)
				}
			})
		}
	}
	if len(cases) != 35 {
		t.Fatalf("%d arrangements", len(cases))
	}
}

// TestAlwaysAfterCancelRunsOnlyAlways: after a cancel the Always subscribers
// still to come run in order, traced like any other, whatever stands between
// them; a second Handle subscriber is not asked, and Emit still reports the
// cancel.
func TestAlwaysAfterCancelRunsOnlyAlways(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("always")
	var traced, ran []string
	traceInto(&traced)
	defer func() { Trace = nil }()
	rec := func(name string) func(int, bool) {
		return func(_ int, cancelled bool) { ran = append(ran, fmt.Sprint(name, "=", cancelled)) }
	}
	h.Always(Lowest, "a", rec("a"))
	h.Handle(Low, "stop", func(int) bool { ran = append(ran, "stop"); return true })
	h.Always(Low, "b", rec("b"))
	h.On(Normal, "on", func(int) { ran = append(ran, "on") })
	h.Handle(Normal, "handle", func(int) bool { ran = append(ran, "handle"); return true })
	h.Always(High, "c", rec("c"))
	h.On(Monitor, "monitor", func(int) { ran = append(ran, "monitor") })
	h.Always(Monitor, "d", rec("d"))
	if !h.Emit(0) {
		t.Fatal("Emit did not report the cancel")
	}
	if want := []string{"a=false", "stop", "b=true", "c=true", "d=true"}; !slices.Equal(ran, want) {
		t.Fatalf("ran %v, want %v", ran, want)
	}
	if want := []string{"a", "stop", "b", "c", "d"}; !slices.Equal(traced, want) {
		t.Fatalf("traced %v, want %v", traced, want)
	}
	var always []string
	for _, s := range Subscriptions() {
		if s.Hook == "always" && s.Always {
			always = append(always, s.Name)
		}
	}
	if want := []string{"a", "b", "c", "d"}; !slices.Equal(always, want) {
		t.Fatalf("Subscriptions marks %v as Always, want %v", always, want)
	}
}

func TestOnNeverStops(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("onstop")
	n := 0
	h.On(Lowest, "a", func(int) { n++ })
	h.Handle(Normal, "b", func(int) bool { n++; return false })
	h.On(High, "c", func(int) { n++ })
	if h.Emit(0) || n != 3 {
		t.Fatalf("stopped=%v n=%d", h.Emit(0), n)
	}
}

func TestSameNamePanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("name")
	h.On(Normal, "a.x", func(int) {})
	defer mustPanic(t, "name", "a.x")
	h.Handle(High, "a.x", func(int) bool { return false })
}

func TestNilFuncPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("nilf")
	func() {
		defer mustPanic(t, "nilf", "h")
		h.Handle(Normal, "h", nil)
	}()
	func() {
		defer mustPanic(t, "nilf", "w")
		h.Always(Normal, "w", nil)
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
			if s.Feature != "feat" || s.Name != "feat.ok" || s.Tier != Normal || s.Always {
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

// TestRegisterAfterEmitPanicsForEveryKind: after the first Emit, live or
// cancelled, a hook refuses a subscriber of every kind (On, Handle, Always),
// and refuses none of them before it.
func TestRegisterAfterEmitPanicsForEveryKind(t *testing.T) {
	kinds := map[string]func(h *Hook[int], name string){
		"on":     func(h *Hook[int], name string) { h.On(Normal, name, func(int) {}) },
		"handle": func(h *Hook[int], name string) { h.Handle(Normal, name, func(int) bool { return false }) },
		"always": func(h *Hook[int], name string) { h.Always(Monitor, name, func(int, bool) {}) },
	}
	for _, cancel := range []bool{false, true} {
		for kind, add := range kinds {
			ResetHooks()
			h := NewHook[int]("late")
			h.Handle(Low, "stop", func(int) bool { return cancel })
			for other, add := range kinds {
				add(h, "before."+other)
			}
			if h.Emit(0) != cancel {
				t.Fatalf("cancel=%t: Emit did not report it", cancel)
			}
			func() {
				defer mustPanic(t, "late", "after."+kind, "after the first Emit")
				add(h, "after."+kind)
			}()
		}
	}
}

// TestEmitConcurrentWithCancelAndAlways: many goroutines emit an event that a
// Handle subscriber cancels; every Emit reports the cancel, the On and Always
// subscribers before it run once with false, the Always subscribers after it
// run once with true, the On subscribers after it never.
func TestEmitConcurrentWithCancelAndAlways(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("conccancel")
	var onBefore, alwaysBeforeLive, alwaysBeforeCancelled, onAfter, alwaysAfterLive, alwaysAfterCancelled atomic.Int64
	count := func(live, cancelled *atomic.Int64) func(int, bool) {
		return func(_ int, c bool) {
			if c {
				cancelled.Add(1)
			} else {
				live.Add(1)
			}
		}
	}
	h.On(Lowest, "on.before", func(int) { onBefore.Add(1) })
	h.Always(Low, "always.before", count(&alwaysBeforeLive, &alwaysBeforeCancelled))
	h.Handle(Normal, "stop", func(int) bool { return true })
	h.On(Normal, "on.after", func(int) { onAfter.Add(1) })
	h.Always(High, "always.after", count(&alwaysAfterLive, &alwaysAfterCancelled))
	h.On(Monitor, "monitor.after", func(int) { onAfter.Add(1) })
	const goroutines, emits = 16, 1000
	var notCancelled atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < emits; i++ {
				if !h.Emit(1) {
					notCancelled.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	const n = goroutines * emits
	if notCancelled.Load() != 0 || onBefore.Load() != n || alwaysBeforeLive.Load() != n || alwaysBeforeCancelled.Load() != 0 ||
		onAfter.Load() != 0 || alwaysAfterLive.Load() != 0 || alwaysAfterCancelled.Load() != n {
		t.Fatalf("not cancelled %d, on before %d, always before %d/%d (live/cancelled), on after %d, always after %d/%d",
			notCancelled.Load(), onBefore.Load(), alwaysBeforeLive.Load(), alwaysBeforeCancelled.Load(),
			onAfter.Load(), alwaysAfterLive.Load(), alwaysAfterCancelled.Load())
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
	h := NewHook[int]("T")
	h.Always(High, "b.x", func(int, bool) {})
	h.On(Low, "a.x", func(int) {})
	var got []Subscription
	for _, s := range Subscriptions() {
		if s.Hook == "T" {
			got = append(got, s)
		}
	}
	want := []Subscription{{Hook: "T", Tier: Low, Name: "a.x"}, {Hook: "T", Tier: High, Name: "b.x", Always: true}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestSubscriptionsHookDeclarationOrder(t *testing.T) {
	ResetHooks()
	h1 := NewHook[int]("decl1")
	h2 := NewHook[int]("decl2")
	h2.On(Lowest, "z", func(int) {})
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

// fillHook gives h n subscribers spread over the tiers (Handle subscribers
// over the five tiers that may cancel).
func fillHook(h *Hook[bigEvent], n int, handle bool) {
	for i := 0; i < n; i++ {
		name := fmt.Sprint("s", i)
		if handle {
			h.Handle(Priority(i%5), name, func(e bigEvent) bool { return e.A[0] == 99 })
		} else {
			h.On(Priority(i%6), name, func(e bigEvent) { _ = e.A[0] })
		}
	}
}

func TestEmitAllocatesNothing(t *testing.T) {
	for _, handle := range []bool{false, true} {
		for _, n := range []int{0, 1, 4, 8, 32} {
			ResetHooks()
			h := NewHook[bigEvent](fmt.Sprintf("alloc-%v-%d", handle, n))
			fillHook(h, n, handle)
			var e bigEvent
			e.A[0] = 1
			got := testing.AllocsPerRun(200, func() { h.Emit(e) })
			t.Logf("handle=%v subscribers=%d allocs/op=%v", handle, n, got)
			if got != 0 {
				t.Errorf("handle=%v n=%d: %v allocs", handle, n, got)
			}
		}
	}
}

// TestEmitAllocatesNothingAfterCancel: the walk of the Always subscribers
// after a cancel allocates nothing either.
func TestEmitAllocatesNothingAfterCancel(t *testing.T) {
	ResetHooks()
	h := NewHook[bigEvent]("alloc-always")
	seen := 0
	h.Handle(Lowest, "stop", func(e bigEvent) bool { return e.A[0] == 1 })
	h.On(Normal, "on", func(bigEvent) {})
	h.Always(Normal, "a", func(_ bigEvent, cancelled bool) {
		if cancelled {
			seen++
		}
	})
	h.Always(Monitor, "b", func(_ bigEvent, cancelled bool) {
		if cancelled {
			seen++
		}
	})
	var e bigEvent
	e.A[0] = 1
	if got := testing.AllocsPerRun(200, func() { h.Emit(e) }); got != 0 {
		t.Errorf("%v allocs", got)
	}
	if seen == 0 || seen%2 != 0 {
		t.Fatalf("the Always subscribers saw %d cancelled events", seen)
	}
}

func BenchmarkEmit(b *testing.B) {
	for _, handle := range []bool{false, true} {
		for _, n := range []int{0, 1, 4, 8, 32} {
			b.Run(fmt.Sprintf("handle=%v/subs=%d", handle, n), func(b *testing.B) {
				ResetHooks()
				h := NewHook[bigEvent](fmt.Sprintf("bench-%v-%d", handle, n))
				fillHook(h, n, handle)
				var e bigEvent
				e.A[0] = 1
				b.ReportAllocs()
				for b.Loop() {
					h.Emit(e)
				}
			})
		}
	}
}

var moveSink float64

// BenchmarkEmitMove emits a MoveEvent to four mixed subscribers, the shape of
// the move hook: Handle (not cancelling), On, On, Always.
func BenchmarkEmitMove(b *testing.B) {
	ResetHooks()
	h := NewHook[MoveEvent]("bench-move")
	h.Always(Monitor, "a", func(e MoveEvent, _ bool) { moveSink += e.Rot[0] + e.Pos[0] })
	h.Handle(Lowest, "b", func(e MoveEvent) bool { return e.P == nil && e.Pos[1] > 1e9 })
	h.On(Monitor, "c", func(e MoveEvent) { moveSink += e.Pos[2] })
	h.On(Highest, "d", func(e MoveEvent) {
		if e.Pos[1] < -1e9 {
			e.Cancel()
		}
	})
	e := MoveEvent{Pos: [3]float64{1, 64, 2}}
	b.ReportAllocs()
	for b.Loop() {
		h.Emit(e)
	}
}
