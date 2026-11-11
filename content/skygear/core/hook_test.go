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

func TestRunsInPriorityOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("order")
	var got []string
	h.On(30, "c", func(int) { got = append(got, "c") })
	h.Handle(10, "a", func(int) bool { got = append(got, "a"); return false })
	h.On(20, "b", func(int) { got = append(got, "b") })
	if h.Emit(1) {
		t.Fatal("Emit reported a stop")
	}
	if want := []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestStopSkipsLaterSteps(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("stop")
	var traced, ran []string
	traceInto(&traced)
	defer func() { Trace = nil }()
	h.Handle(10, "a.x", func(int) bool { ran = append(ran, "a.x"); return true })
	h.On(20, "b.x", func(int) { ran = append(ran, "b.x") })
	if !h.Emit(1) {
		t.Fatal("Emit did not report the stop")
	}
	if !slices.Equal(ran, []string{"a.x"}) || !slices.Equal(traced, []string{"a.x"}) {
		t.Fatalf("ran %v traced %v", ran, traced)
	}
}

func TestOnNeverStops(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("onstop")
	n := 0
	h.On(1, "a", func(int) { n++ })
	h.Handle(2, "b", func(int) bool { n++; return false })
	h.On(3, "c", func(int) { n++ })
	if h.Emit(0) || n != 3 {
		t.Fatalf("stopped=%v n=%d", h.Emit(0), n)
	}
}

func TestSamePriorityPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("prio")
	h.On(10, "a.x", func(int) {})
	defer mustPanic(t, "prio", "10", "a.x", "b.x")
	h.On(10, "b.x", func(int) {})
}

func TestSameNamePanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("name")
	h.On(10, "a.x", func(int) {})
	defer mustPanic(t, "name", "a.x")
	h.Handle(20, "a.x", func(int) bool { return false })
}

func TestNilFuncPanics(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("nilf")
	defer mustPanic(t, "nilf", "a")
	h.On(1, "a", nil)
}

func TestInstallingPrefix(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("pfx")
	SetInstalling("feat")
	h.On(1, "feat.ok", func(int) {})
	func() {
		defer mustPanic(t, "pfx", "other.bad", "feat")
		h.On(2, "other.bad", func(int) {})
	}()
	found := false
	for _, s := range Subscriptions() {
		if s.Hook == "pfx" {
			found = true
			if s.Feature != "feat" || s.Name != "feat.ok" || s.Prio != 1 {
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
	h.On(1, "a", func(int) {})
	h.Emit(0)
	defer mustPanic(t, "late", "b")
	h.On(2, "b", func(int) {})
}

func TestEmptyEmitStillSealsRegistration(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("early")
	h.Emit(0) // no steps: an empty emit still counts as emission
	defer mustPanic(t, "early")
	h.On(1, "a", func(int) {})
}

func TestSubscriptionsRunOrder(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("T")
	h.On(20, "b.x", func(int) {})
	h.On(10, "a.x", func(int) {})
	var got []Subscription
	for _, s := range Subscriptions() {
		if s.Hook == "T" {
			got = append(got, s)
		}
	}
	want := []Subscription{{Hook: "T", Prio: 10, Name: "a.x"}, {Hook: "T", Prio: 20, Name: "b.x"}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestSubscriptionsHookDeclarationOrder(t *testing.T) {
	ResetHooks()
	h1 := NewHook[int]("decl1")
	h2 := NewHook[int]("decl2")
	h2.On(1, "z", func(int) {})
	h1.On(5, "y", func(int) {})
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
	h.On(2, "b", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":b")) })
	h.On(1, "a", func(int) { ran = append(ran, fmt.Sprint(len(traced), ":a")) })
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
	h.On(1, "a", func(int) { panic("boom!") })
	h.On(2, "b", func(int) { later = true })
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
		h.On(i, fmt.Sprint("s", i), func(e int) { n.Add(int64(e)) })
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

func fillHook(h *Hook[bigEvent], n int, handle bool) {
	for i := 0; i < n; i++ {
		name := fmt.Sprint("s", i)
		if handle {
			h.Handle(i, name, func(e bigEvent) bool { return e.A[0] == 99 })
		} else {
			h.On(i, name, func(e bigEvent) { _ = e.A[0] })
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
