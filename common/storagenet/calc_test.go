package storagenet

import (
	"errors"
	"reflect"
	"testing"
)

func pat(id string, out Stack, ins ...Ingredient) Pattern {
	return Pattern{ID: id, Inputs: ins, Outputs: []Stack{out}}
}

func in(n int64, opts ...Key) Ingredient { return Ingredient{Count: n, Options: opts} }

func stockOf(m map[Key]int64) func(Key) int64 { return func(k Key) int64 { return m[k] } }

func woodRepo() *Repo {
	r := NewRepo()
	r.Add(pat("plank", Stack{"plank", 4}, in(1, "log")))
	r.Add(pat("stick", Stack{"stick", 4}, in(2, "plank")))
	return r
}

func TestRepo(t *testing.T) {
	r := woodRepo()
	if p, ok := r.Get("plank"); !ok || p.ID != "plank" {
		t.Fatal(p, ok)
	}
	if p, ok := r.Get("nope"); ok || p.ID != "" {
		t.Fatal(p, ok)
	}
	if !reflect.DeepEqual(r.Outputs(), []Key{"plank", "stick"}) {
		t.Fatal(r.Outputs())
	}
	r.Remove("plank")
	if len(r.ByOutput("plank")) != 0 || len(r.Outputs()) != 1 {
		t.Fatal("not removed")
	}
}

func TestCalcWood(t *testing.T) {
	p, err := Calculate(woodRepo(), stockOf(map[Key]int64{"log": 1}), "stick", 4, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Steps, []Step{{"plank", 1}, {"stick", 1}}) ||
		!reflect.DeepEqual(p.Use, map[Key]int64{"log": 1}) || len(p.Missing) != 0 || !p.OK() {
		t.Fatalf("%+v", p)
	}
	p, err = Calculate(woodRepo(), stockOf(map[Key]int64{"log": 1}), "stick", 12, 100)
	if err != nil || !reflect.DeepEqual(p.Missing, map[Key]int64{"log": 1}) || p.OK() {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestCycle(t *testing.T) {
	r := NewRepo()
	r.Add(pat("a", Stack{"a", 1}, in(1, "b")))
	r.Add(pat("b", Stack{"b", 1}, in(1, "a")))
	if _, err := Calculate(r, stockOf(nil), "a", 1, 100); !errors.Is(err, ErrCycle) {
		t.Fatal(err)
	}
}

func TestOptionCycle(t *testing.T) {
	r := NewRepo()
	r.Add(pat("w", Stack{"w", 1}, in(1, "x", "y")))
	p, err := Calculate(r, stockOf(map[Key]int64{"y": 1}), "w", 1, 100)
	if err != nil || !p.OK() || !reflect.DeepEqual(p.Use, map[Key]int64{"y": 1}) {
		t.Fatalf("%+v %v", p, err)
	}
	// all options missing: first option is reported
	p, err = Calculate(r, stockOf(nil), "w", 1, 100)
	if err != nil || !reflect.DeepEqual(p.Missing, map[Key]int64{"x": 1}) {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestSecondPatternFallback(t *testing.T) {
	r := NewRepo()
	r.Add(pat("p1", Stack{"o", 1}, in(1, "scarce")))
	r.Add(pat("p2", Stack{"o", 1}, in(1, "common")))
	p, err := Calculate(r, stockOf(map[Key]int64{"common": 2}), "o", 2, 100)
	if err != nil || !p.OK() || !reflect.DeepEqual(p.Steps, []Step{{"p2", 2}}) {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestLeftoverReuse(t *testing.T) {
	r := woodRepo()
	// two sticks crafts need 2 planks each (4 total... one craft yields 4 sticks)
	r.Add(pat("chest", Stack{"chest", 1}, in(1, "plank"), in(1, "stick")))
	p, err := Calculate(r, stockOf(map[Key]int64{"log": 1}), "chest", 1, 100)
	if err != nil || !p.OK() {
		t.Fatalf("%+v %v", p, err)
	}
	// plank crafted once (4), 1 used by chest, 2 by stick craft, 1 left over
	want := []Step{{"plank", 1}, {"stick", 1}, {"chest", 1}}
	if !reflect.DeepEqual(p.Steps, want) || !reflect.DeepEqual(p.Use, map[Key]int64{"log": 1}) {
		t.Fatalf("%+v", p)
	}
}

func TestOverflow(t *testing.T) {
	r := NewRepo()
	r.Add(pat("a", Stack{"a", 1}, in(4, "b")))
	r.Add(pat("b", Stack{"b", 1}, in(4, "c")))
	r.Add(pat("c", Stack{"c", 1}, in(4, "d")))
	if _, err := Calculate(r, stockOf(nil), "a", 1<<62, 100); !errors.Is(err, ErrOverflow) {
		t.Fatal(err)
	}
}

func TestBudget(t *testing.T) {
	r := NewRepo()
	ins := make([]Ingredient, 20)
	for i := range ins {
		ins[i] = in(1, Key(string(rune('a'+i))))
	}
	r.Add(pat("big", Stack{"big", 1}, ins...))
	if _, err := Calculate(r, stockOf(nil), "big", 1, 10); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
	if p, err := Calculate(r, stockOf(nil), "big", 1, 100); err != nil || len(p.Missing) != 20 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestOptionCycleFallsThrough(t *testing.T) {
	r := NewRepo()
	r.Add(pat("a", Stack{"a", 1}, in(1, "a")))
	r.Add(pat("w", Stack{"w", 1}, in(1, "a", "y")))
	p, err := Calculate(r, stockOf(map[Key]int64{"y": 1}), "w", 1, 100)
	if err != nil || !p.OK() || !reflect.DeepEqual(p.Use, map[Key]int64{"y": 1}) {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestSiblingMissingDoesNotMaskFallback(t *testing.T) {
	r := NewRepo()
	r.Add(pat("p1", Stack{"o", 1}, in(1, "scarce")))
	r.Add(pat("p2", Stack{"o", 1}, in(1, "common")))
	r.Add(Pattern{ID: "top", Inputs: []Ingredient{in(1, "nothing"), in(1, "o")}, Outputs: []Stack{{"top", 1}}})
	p, err := Calculate(r, stockOf(map[Key]int64{"common": 1}), "top", 1, 100)
	if err != nil || !reflect.DeepEqual(p.Missing, map[Key]int64{"nothing": 1}) || p.Use["common"] != 1 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestDuplicateOutputLeftover(t *testing.T) {
	r := NewRepo()
	r.Add(Pattern{ID: "d", Inputs: []Ingredient{in(1, "src")}, Outputs: []Stack{{"o", 1}, {"o", 1}}})
	p, err := Calculate(r, stockOf(map[Key]int64{"src": 10}), "o", 2, 100)
	if err != nil || !reflect.DeepEqual(p.Steps, []Step{{"d", 1}}) {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestDuplicateOutputLeftoverCount(t *testing.T) {
	r := NewRepo()
	r.Add(Pattern{ID: "d", Inputs: []Ingredient{in(1, "src")}, Outputs: []Stack{{"o", 1}, {"o", 1}}})
	r.Add(Pattern{ID: "top", Inputs: []Ingredient{in(3, "o"), in(2, "o")}, Outputs: []Stack{{"top", 1}}})
	p, err := Calculate(r, stockOf(map[Key]int64{"src": 10}), "top", 1, 100)
	if err != nil || !reflect.DeepEqual(p.Steps, []Step{{"d", 3}, {"top", 1}}) {
		t.Fatalf("%+v %v", p, err)
	}
}
