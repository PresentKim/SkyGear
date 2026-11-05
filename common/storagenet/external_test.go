package storagenet

import "testing"

type fakeSrc struct {
	m   map[Key]int64
	cap int64
}

func (f *fakeSrc) List() map[Key]int64 {
	out := map[Key]int64{}
	for k, n := range f.m {
		out[k] = n
	}
	return out
}
func (f *fakeSrc) Insert(k Key, n int64, sim bool) int64 {
	if n > f.cap {
		n = f.cap
	}
	if !sim {
		f.m[k] += n
	}
	return n
}
func (f *fakeSrc) Extract(k Key, n int64, sim bool) int64 {
	if n > f.m[k] {
		n = f.m[k]
	}
	if !sim {
		f.m[k] -= n
	}
	return n
}

func TestExternalCount(t *testing.T) {
	e := NewExternal(&fakeSrc{m: map[Key]int64{"a": 2}, cap: 10})
	if e.Count("a") != 2 {
		t.Fatal(e.Count("a"))
	}
}

func TestExternalPoll(t *testing.T) {
	src := &fakeSrc{m: map[Key]int64{"a": 2}, cap: 10}
	e := NewExternal(src)
	if e.Poll() {
		t.Fatal("unchanged poll reported change")
	}
	src.m["a"] += 3
	if !e.Poll() || e.Count("a") != 5 {
		t.Fatalf("count %d", e.Count("a"))
	}
	delete(src.m, "a")
	if !e.Poll() || e.Count("a") != 0 {
		t.Fatal("removal not seen")
	}
}

func TestExternalInsertUpdatesCache(t *testing.T) {
	src := &fakeSrc{m: map[Key]int64{"a": 2}, cap: 4}
	e := NewExternal(src)
	if got := e.Insert("a", 9, true); got != 4 || e.Count("a") != 2 {
		t.Fatalf("sim got %d count %d", got, e.Count("a"))
	}
	if got := e.Insert("a", 9, false); got != 4 || e.Count("a") != 6 {
		t.Fatalf("got %d count %d", got, e.Count("a"))
	}
	if got := e.Extract("a", 10, false); got != 6 || e.Count("a") != 0 {
		t.Fatalf("got %d count %d", got, e.Count("a"))
	}
	if e.Poll() {
		t.Fatal("cache drifted from source")
	}
	n := 0
	e.Each(func(Key, int64) { n++ })
	if n != 0 {
		t.Fatal(n)
	}
}
