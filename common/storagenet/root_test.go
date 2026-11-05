package storagenet

import (
	"math/rand"
	"testing"
)

func newPair() (*Root, *Disk, *Disk) {
	r := NewRoot()
	a, b := NewDisk(10), NewDisk(10)
	r.Add("A", a, 0, nil)
	r.Add("B", b, 5, nil)
	return r, a, b
}

func TestRootExamples(t *testing.T) {
	if got := NewRoot().Totals(); len(got) != 0 {
		t.Fatal(got)
	}
	if NewDisk(10).Stored() != 0 {
		t.Fatal("stored")
	}
	r, a, b := newPair()
	if got := r.Insert("a", 15, false); got != 15 {
		t.Fatal(got)
	}
	if b.Count("a") != 10 || a.Count("a") != 5 {
		t.Fatal(b.Count("a"), a.Count("a"))
	}
	if got := r.Extract("a", 3, false); got != 3 || a.Count("a") != 2 {
		t.Fatal(got, a.Count("a"))
	}
	if r.Totals()["a"] != 12 {
		t.Fatal(r.Totals())
	}
	// free space is 8 after the extract (spec e.g. says 5, which holds before it)
	if got := r.Insert("b", 30, true); got != 8 {
		t.Fatal(got)
	}
	r3, _, _ := newPair()
	r3.Insert("a", 15, false)
	if got := r3.Insert("b", 30, true); got != 5 {
		t.Fatal(got)
	}
	if r.Totals()["b"] != 0 || a.Stored() != 2 {
		t.Fatal("sim changed state")
	}
}

func TestHolderFirst(t *testing.T) {
	r := NewRoot()
	a, c := NewDisk(10), NewDisk(10)
	r.Add("A", a, 1, nil)
	r.Add("C", c, 1, nil)
	a.Insert("x", 1, false)
	r.Resync("A")
	// C is added after A; put C first by holder rule check with reversed order too.
	r.Insert("x", 3, false)
	if a.Count("x") != 4 || c.Count("x") != 0 {
		t.Fatal(a.Count("x"), c.Count("x"))
	}
	// holder wins even over higher-priority empty source
	r2 := NewRoot()
	hi, lo := NewDisk(10), NewDisk(10)
	r2.Add("hi", hi, 9, nil)
	r2.Add("lo", lo, 1, nil)
	lo.Insert("x", 1, false)
	r2.Resync("lo")
	r2.Insert("x", 2, false)
	if lo.Count("x") != 3 || hi.Count("x") != 0 {
		t.Fatal(lo.Count("x"), hi.Count("x"))
	}
}

func TestFilterSkips(t *testing.T) {
	r := NewRoot()
	a, b := NewDisk(10), NewDisk(10)
	r.Add("A", a, 9, func(k Key) bool { return k == "only" })
	r.Add("B", b, 1, nil)
	if got := r.Insert("x", 4, false); got != 4 || a.Count("x") != 0 || b.Count("x") != 4 {
		t.Fatal(got, a.Count("x"), b.Count("x"))
	}
	if got := r.Insert("only", 4, false); got != 4 || a.Count("only") != 4 {
		t.Fatal(got)
	}
	r2 := NewRoot()
	r2.Add("A", NewDisk(10), 0, func(Key) bool { return false })
	if r2.Insert("x", 1, false) != 0 {
		t.Fatal("filtered insert accepted")
	}
}

func TestListenDeltas(t *testing.T) {
	r, _, _ := newPair()
	type ev struct {
		k Key
		d int64
	}
	var evs []ev
	off := r.Listen(func(k Key, d int64) { evs = append(evs, ev{k, d}) })
	r.Insert("a", 15, false)
	if len(evs) != 2 { // spans two disks
		t.Fatal(evs)
	}
	var sum int64
	for _, e := range evs {
		if e.k != "a" {
			t.Fatal(e)
		}
		sum += e.d
	}
	if sum != 15 {
		t.Fatal(sum)
	}
	evs = nil
	r.Remove("B")
	if len(evs) != 1 || evs[0] != (ev{"a", -10}) {
		t.Fatal(evs)
	}
	off()
	evs = nil
	r.Insert("a", 1, false)
	if len(evs) != 0 {
		t.Fatal("fired after unsubscribe")
	}
}

func TestListenSingleDisk(t *testing.T) {
	r := NewRoot()
	r.Add("A", NewDisk(100), 0, nil)
	n := 0
	r.Listen(func(k Key, d int64) {
		n++
		if k != "a" || d != 15 {
			t.Fatal(k, d)
		}
	})
	r.Insert("a", 15, false)
	if n != 1 {
		t.Fatal(n)
	}
}

func TestResyncDiff(t *testing.T) {
	r := NewRoot()
	d := NewDisk(100)
	d.Insert("a", 5, false)
	d.Insert("b", 7, false)
	r.Add("D", d, 0, nil)
	got := map[Key]int64{}
	r.Listen(func(k Key, delta int64) { got[k] += delta })
	r.Resync("D")
	if len(got) != 0 {
		t.Fatal("no change should emit nothing", got)
	}
	d.Insert("a", 3, false)
	d.Extract("b", 7, false)
	d.Insert("c", 2, false)
	r.Resync("D")
	if len(got) != 3 || got["a"] != 3 || got["b"] != -7 || got["c"] != 2 {
		t.Fatal(got)
	}
	tot := r.Totals()
	if tot["a"] != 8 || tot["c"] != 2 || len(tot) != 2 {
		t.Fatal(tot)
	}
	if r.Extract("b", 1, false) != 0 {
		t.Fatal("stale holder")
	}
}

func TestTotalsMatchFresh(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	r := NewRoot()
	disks := []*Disk{NewDisk(50), NewDisk(30), NewDisk(80)}
	for i, d := range disks {
		r.Add(string(rune('A'+i)), d, i%2, nil)
	}
	keys := []Key{"a", "b", "c", "d"}
	for i := 0; i < 1000; i++ {
		k := keys[rng.Intn(len(keys))]
		n := int64(rng.Intn(20))
		sim := rng.Intn(4) == 0
		if rng.Intn(2) == 0 {
			r.Insert(k, n, sim)
		} else {
			r.Extract(k, n, sim)
		}
	}
	want := map[Key]int64{}
	for _, d := range disks {
		for k, v := range d.Contents() {
			want[k] += v
		}
	}
	got := r.Totals()
	if len(got) != len(want) {
		t.Fatal(got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatal(k, got[k], v)
		}
	}
}

func TestTotalsCopy(t *testing.T) {
	r, _, _ := newPair()
	r.Insert("a", 1, false)
	r.Totals()["a"] = 99
	if r.Count("a") != 1 {
		t.Fatal("totals aliased")
	}
}

func TestDiskLoad(t *testing.T) {
	d := NewDisk(10)
	d.Load(map[Key]int64{"a": 6, "z": 0})
	if d.Stored() != 6 || len(d.Contents()) != 1 {
		t.Fatal(d.Contents())
	}
	if d.Insert("b", 9, false) != 4 {
		t.Fatal("cap")
	}
}
