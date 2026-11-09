package sghook

import (
	"slices"
	"testing"
	"time"
)

func attrMap(t *testing.T, a []any) map[string]any {
	t.Helper()
	m := map[string]any{}
	for i := 0; i+1 < len(a); i += 2 {
		m[a[i].(string)] = a[i+1]
	}
	return m
}

func TestJoinMarksInOrder(t *testing.T) {
	t0 := time.Now()
	m := &joinMarks{login: t0, packs: t0.Add(1500 * time.Millisecond), start: t0.Add(1600 * time.Millisecond), loaded: t0.Add(9600 * time.Millisecond)}
	got := attrMap(t, m.attrs("Steve", "perm"))
	want := map[string]any{
		"player": "Steve", "probe": "perm",
		"total_ms": 9600.0, "connect_to_packs_ms": 1500.0, "to_start_game_ms": 100.0, "client_load_ms": 8000.0,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
}

func TestJoinMarksMissingFieldOmitted(t *testing.T) {
	t0 := time.Now()
	// No login mark (the Allower did not see it): the durations from it go.
	m := &joinMarks{packs: t0, start: t0.Add(time.Millisecond), loaded: t0.Add(time.Second)}
	got := attrMap(t, m.attrs("Alex", "off"))
	for _, k := range []string{"total_ms", "connect_to_packs_ms"} {
		if _, ok := got[k]; ok {
			t.Errorf("%s logged without its mark", k)
		}
	}
	for _, k := range []string{"to_start_game_ms", "client_load_ms"} {
		if _, ok := got[k]; !ok {
			t.Errorf("%s missing", k)
		}
	}
}

func TestJoinDisconnectBeforeEnd(t *testing.T) {
	t0 := time.Now()
	m := &joinMarks{login: t0, packs: t0.Add(time.Second), start: t0.Add(2 * time.Second)}
	if a := m.attrs("Gone", "off"); a != nil {
		t.Fatalf("line for a join that never finished loading: %v", a)
	}
}

func TestPendingLogins(t *testing.T) {
	var l logins
	t0 := time.Now()
	l.add("a", t0)
	l.add("b", t0.Add(time.Second))
	if got := l.take("a"); !got.Equal(t0) {
		t.Fatalf("take(a) = %v", got)
	}
	if got := l.take("a"); !got.IsZero() {
		t.Fatal("a mark is taken once")
	}
	// b never gets accepted (disconnect during login); a later login purges it.
	l.add("c", t0.Add(pendingLoginTTL+2*time.Second))
	if keys := mapKeys(l.m); !slices.Equal(keys, []string{"c"}) {
		t.Fatalf("pending logins = %v, want only c", keys)
	}
	l.take("c")
	if len(l.m) != 0 {
		t.Fatalf("leaked %v", l.m)
	}
}

func mapKeys(m map[string]time.Time) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
