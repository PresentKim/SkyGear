package storagenet

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"
)

type fakeEnv struct {
	crafts map[string]int
	refuse bool
	pushed []Stack
}

func (e *fakeEnv) Crafts(id string) int { return e.crafts[id] }
func (e *fakeEnv) Push(id string, in []Stack) bool {
	if e.refuse {
		return false
	}
	e.pushed = append(e.pushed, in...)
	return true
}

func taskRepo() *Repo {
	r := NewRepo()
	r.Add(Pattern{ID: "plank", Inputs: []Ingredient{{1, []Key{"log"}}}, Outputs: []Stack{{"plank", 4}}})
	r.Add(Pattern{ID: "stick", Inputs: []Ingredient{{2, []Key{"plank"}}}, Outputs: []Stack{{"stick", 4}}})
	r.Add(Pattern{ID: "smelt", Processing: true, Inputs: []Ingredient{{1, []Key{"ore"}}}, Outputs: []Stack{{"ingot", 1}}})
	return r
}

func rootWith(k Key, n int64, capacity int64) *Root {
	r := NewRoot()
	r.Add("d", NewDisk(capacity), 0, nil)
	r.Insert(k, n, false)
	return r
}

func sticksTask(t *testing.T, root *Root) (*Task, *Repo) {
	repo := taskRepo()
	plan, err := Calculate(repo, root.Count, "stick", 4, 100)
	if err != nil || !plan.OK() {
		t.Fatal(err, plan)
	}
	tk, err := NewTask("t1", plan, repo, root)
	if err != nil {
		t.Fatal(err)
	}
	return tk, repo
}

func TestTaskCraftFlow(t *testing.T) {
	root := rootWith("log", 1, 100)
	tk, _ := sticksTask(t, root)
	if root.Count("log") != 0 {
		t.Fatal("not reserved")
	}
	env := &fakeEnv{crafts: map[string]int{"plank": 1, "stick": 1}}
	tk.Tick(env)
	st := tk.Status()
	if st.Steps[0].Done != 1 || st.Steps[1].Done != 0 || tk.Done() {
		t.Fatal(st)
	}
	tk.Tick(env)
	if !tk.Done() {
		t.Fatal(tk.Status())
	}
	if left := tk.Finish(root); len(left) != 0 {
		t.Fatal(left)
	}
	if root.Count("stick") != 4 || root.Count("plank") != 2 {
		t.Fatal(root.Totals())
	}
}

func TestNewTaskShort(t *testing.T) {
	root := rootWith("log", 1, 100)
	repo := taskRepo()
	plan, _ := Calculate(repo, root.Count, "stick", 4, 100)
	root.Extract("log", 1, false)
	if _, err := NewTask("t", plan, repo, root); !errors.Is(err, ErrShort) {
		t.Fatal(err)
	}
	if root.Count("log") != 0 {
		t.Fatal("changed")
	}
	// partial: second key short must not take the first
	root.Insert("a", 1, false)
	p := &Plan{Use: map[Key]int64{"a": 1, "b": 1}}
	if _, err := NewTask("t", p, repo, root); !errors.Is(err, ErrShort) || root.Count("a") != 1 {
		t.Fatal(err, root.Count("a"))
	}
}

func procTask(t *testing.T) (*Task, *Root) {
	root := rootWith("ore", 1, 100)
	repo := taskRepo()
	plan := &Plan{Want: Stack{"ingot", 1}, Steps: []Step{{"smelt", 1}}, Use: map[Key]int64{"ore": 1}}
	tk, err := NewTask("p", plan, repo, root)
	if err != nil {
		t.Fatal(err)
	}
	return tk, root
}

func TestProcessingReceive(t *testing.T) {
	tk, root := procTask(t)
	env := &fakeEnv{crafts: map[string]int{"smelt": 1}}
	tk.Tick(env)
	if w := tk.Status().Waiting; !reflect.DeepEqual(w, map[Key]int64{"ingot": 1}) {
		t.Fatal(w)
	}
	if tk.Done() {
		t.Fatal("done while waiting")
	}
	if got := tk.Receive("ingot", 3); got != 1 {
		t.Fatal(got)
	}
	if tk.Receive("ingot", 1) != 0 || tk.Receive("zzz", 1) != 0 {
		t.Fatal("over receive")
	}
	if !tk.Done() {
		t.Fatal("not done")
	}
	tk.Finish(root)
	if root.Count("ingot") != 1 {
		t.Fatal(root.Totals())
	}
}

func TestPushRefused(t *testing.T) {
	tk, _ := procTask(t)
	env := &fakeEnv{crafts: map[string]int{"smelt": 1}, refuse: true}
	tk.Tick(env)
	if tk.buffer["ore"] != 1 || tk.Status().Steps[0].Done != 0 || len(tk.Status().Waiting) != 0 {
		t.Fatal(tk.Data())
	}
	env.refuse = false
	tk.Tick(env)
	if tk.buffer["ore"] != 0 || tk.Status().Waiting["ingot"] != 1 {
		t.Fatal(tk.Data())
	}
}

func TestCancelReturnsBuffer(t *testing.T) {
	root := rootWith("log", 1, 100)
	tk, _ := sticksTask(t, root)
	tk.Tick(&fakeEnv{crafts: map[string]int{"plank": 1}})
	if left := tk.Cancel(root); len(left) != 0 {
		t.Fatal(left)
	}
	if root.Count("plank") != 4 || root.Count("log") != 0 {
		t.Fatal(root.Totals())
	}
	if tk.Done() || tk.Receive("plank", 1) != 0 {
		t.Fatal("cancelled task active")
	}
}

func TestCancelRootFull(t *testing.T) {
	root := rootWith("log", 1, 1)
	tk, _ := sticksTask(t, root) // frees the log slot: capacity 1
	tk.Tick(&fakeEnv{crafts: map[string]int{"plank": 1}})
	left := tk.Cancel(root)
	if root.Count("plank") != 1 {
		t.Fatal(root.Totals())
	}
	if len(left) != 1 || left[0] != (Stack{"plank", 3}) {
		t.Fatal(left)
	}
}

func TestDataRoundTrip(t *testing.T) {
	root := rootWith("log", 1, 100)
	tk, repo := sticksTask(t, root)
	env := &fakeEnv{crafts: map[string]int{"plank": 1, "stick": 1}}
	tk.Tick(env)
	raw, err := json.Marshal(tk.Data())
	if err != nil {
		t.Fatal(err)
	}
	var d TaskData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	tk2, err := LoadTask(d, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tk.Data(), tk2.Data()) {
		t.Fatal(tk.Data(), tk2.Data())
	}
	tk2.Tick(env)
	if !tk2.Done() {
		t.Fatal("loaded task did not finish")
	}
	tk2.Finish(root)
	if root.Count("stick") != 4 || root.Count("plank") != 2 {
		t.Fatal(root.Totals())
	}
	repo.Remove("stick")
	if _, err := LoadTask(d, repo); !errors.Is(err, ErrNoPattern) {
		t.Fatal(err)
	}
}

func TestSurplusReturned(t *testing.T) {
	root := rootWith("log", 3, 100)
	repo := taskRepo()
	plan, _ := Calculate(repo, root.Count, "plank", 5, 100) // 2 crafts -> 8 planks
	tk, err := NewTask("s", plan, repo, root)
	if err != nil {
		t.Fatal(err)
	}
	env := &fakeEnv{crafts: map[string]int{"plank": 5}}
	tk.Tick(env)
	if !tk.Done() {
		t.Fatal(tk.Status())
	}
	left := tk.Finish(root)
	got := root.Count("plank")
	sort.Slice(left, func(i, j int) bool { return left[i].Key < left[j].Key })
	if len(left) != 0 || got != 8 || root.Count("log") != 1 {
		t.Fatal(left, root.Totals())
	}
}

func TestLoadTaskDropsNonPositive(t *testing.T) {
	tk, err := LoadTask(TaskData{ID: "x", Buffer: map[Key]int64{"a": -1, "b": 2}, Waiting: map[Key]int64{"w": 0}}, taskRepo())
	if err != nil || !tk.Done() || len(tk.Data().Buffer) != 1 {
		t.Fatal(err, tk.Data())
	}
}
