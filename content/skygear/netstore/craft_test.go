package netstore

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// fakeCraftEnv is the fake of the machines: it takes every push (unless told
// to refuse) and crafts as many as asked.
type fakeCraftEnv struct {
	crafts int
	refuse bool
}

func (f *fakeCraftEnv) Crafts(string) int { return f.crafts }
func (f *fakeCraftEnv) Push(string, []storagenet.Stack) bool {
	return !f.refuse
}

// craftNet builds a powered network with a controller and a drive-less Root
// holding a big disk.
func craftNet(t *testing.T, tx *world.Tx, s *core.Session) (*Network, *craftState) {
	t.Helper()
	put(tx, p(0), machine.KindNetController)
	put(tx, p(1), machine.KindNetAutocrafter)
	powered(tx, p(0), 1<<30)
	n := At(tx, p(0))
	n.Root.Add("test", storagenet.NewDisk(1000), 0, nil)
	cs := craftOf(n)
	if cs == nil {
		t.Fatal("no craft state")
	}
	tickNet(s, tx, n, cs)
	cs.nextScan = 1 << 40 // keep the patterns set by the test
	return n, cs
}

func tickNet(s *core.Session, tx *world.Tx, n *Network, cs *craftState) {
	runController(s, stateAt(tx, n.Controller), n.Controller, tx)
	cs.tick(tx)
}

func stickRepo(cs *craftState) {
	r := storagenet.NewRepo()
	r.Add(storagenet.Pattern{ID: "plank", Inputs: []storagenet.Ingredient{{Count: 1, Options: []storagenet.Key{"log"}}}, Outputs: []storagenet.Stack{{Key: "plank", Count: 4}}})
	r.Add(storagenet.Pattern{ID: "stick", Inputs: []storagenet.Ingredient{{Count: 2, Options: []storagenet.Key{"plank"}}}, Outputs: []storagenet.Stack{{Key: "stick", Count: 4}}})
	cs.repo = r
	cs.envFor = func(*world.Tx) storagenet.Env { return &fakeCraftEnv{crafts: 1} }
}

// waitJob ticks until cb ran or the timeout passes.
func waitJob(t *testing.T, s *core.Session, tx *world.Tx, n *Network, cs *craftState, done *bool) {
	t.Helper()
	for end := time.Now().Add(2 * time.Second); !*done && time.Now().Before(end); {
		tickNet(s, tx, n, cs)
		time.Sleep(time.Millisecond)
	}
	if !*done {
		t.Fatal("calculation never applied")
	}
}

func TestRequestCraftsSticks(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, cs := craftNet(t, tx, s)
		stickRepo(cs)
		n.Root.Insert("log", 1, false)
		var got *storagenet.Task
		var done bool
		if err := cs.Request("stick", 4, true, func(_ *world.Tx, plan *storagenet.Plan, task *storagenet.Task, err error) {
			done, got = true, task
			if err != nil || !plan.OK() {
				t.Error(err, plan)
			}
		}); err != nil {
			t.Fatal(err)
		}
		waitJob(t, s, tx, n, cs, &done)
		if got == nil || n.Root.Count("log") != 0 {
			t.Fatalf("task %v, log %d", got, n.Root.Count("log"))
		}
		for i := 0; i < 5 && len(cs.Tasks()) > 0; i++ {
			tickNet(s, tx, n, cs)
		}
		if len(cs.Tasks()) != 0 || n.Root.Count("stick") != 4 || n.Root.Count("plank") != 2 {
			t.Fatalf("tasks %d, totals %v", len(cs.Tasks()), n.Root.Totals())
		}
	})
}

func TestProcessingReceive(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, cs := craftNet(t, tx, s)
		r := storagenet.NewRepo()
		r.Add(storagenet.Pattern{ID: "smelt", Processing: true, Inputs: []storagenet.Ingredient{{Count: 1, Options: []storagenet.Key{"ore"}}}, Outputs: []storagenet.Stack{{Key: "ingot", Count: 1}}})
		cs.repo = r
		cs.envFor = func(*world.Tx) storagenet.Env { return &fakeCraftEnv{crafts: 1} }
		n.Root.Insert("ore", 1, false)
		var done bool
		_ = cs.Request("ingot", 1, true, func(*world.Tx, *storagenet.Plan, *storagenet.Task, error) { done = true })
		waitJob(t, s, tx, n, cs, &done)
		tickNet(s, tx, n, cs)
		if w := cs.Tasks()[0].Status().Waiting["ingot"]; w != 1 {
			t.Fatalf("waiting %d", w)
		}
		if got := Intercept(n, "ingot", 3); got != 1 {
			t.Fatalf("intercepted %d", got)
		}
		if Intercept(n, "ingot", 1) != 0 || Intercept(n, "other", 1) != 0 {
			t.Fatal("took unawaited items")
		}
		tickNet(s, tx, n, cs)
		if len(cs.Tasks()) != 0 || n.Root.Count("ingot") != 1 {
			t.Fatalf("tasks %d totals %v", len(cs.Tasks()), n.Root.Totals())
		}
	})
}

func TestTasksSurviveReload(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, cs := craftNet(t, tx, s)
		ore, ingot := item.NewStack(item.RawIron{}, 1), item.NewStack(item.IronIngot{}, 1)
		pat := EncodeProcessing(item.NewStack(Pattern{}, 1), []item.Stack{ore}, []item.Stack{ingot})
		_ = stateAt(tx, p(1)).Inv().SetItem(0, pat)
		cs.rescan(tx)
		cs.nextScan = 1 << 40
		cs.envFor = func(*world.Tx) storagenet.Env { return &fakeCraftEnv{crafts: 1} }
		n.Root.Insert(KeyOf(ore), 1, false)
		var done bool
		_ = cs.Request(KeyOf(ingot), 1, true, func(_ *world.Tx, _ *storagenet.Plan, task *storagenet.Task, err error) {
			done = task != nil
			if err != nil {
				t.Error(err)
			}
		})
		waitJob(t, s, tx, n, cs, &done)
		tickNet(s, tx, n, cs)
		before := cs.Tasks()[0].Data()
		if len(before.Waiting) != 1 {
			t.Fatalf("not waiting: %+v", before)
		}
		Invalidate(tx, p(0)) // saves the tasks and drops the network
		n2 := At(tx, p(0))
		if n2 == n {
			t.Fatal("network not rebuilt")
		}
		cs2 := craftOf(n2)
		if len(cs2.Tasks()) != 1 {
			t.Fatalf("tasks after reload %d", len(cs2.Tasks()))
		}
		if after := cs2.Tasks()[0].Data(); after.ID != before.ID || after.Waiting[KeyOf(ingot)] != 1 || len(after.Steps) != 1 {
			t.Fatalf("after %+v before %+v", after, before)
		}
		if Intercept(n2, KeyOf(ingot), 1) != 1 {
			t.Fatal("reloaded task does not wait")
		}
	})
}

func TestCalcOffThread(t *testing.T) {
	gate := make(chan struct{})
	calcWait = func() { <-gate }
	t.Cleanup(func() { calcWait = nil })
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, cs := craftNet(t, tx, s)
		stickRepo(cs)
		n.Root.Insert("log", 1, false)
		var plan *storagenet.Plan
		var done bool
		_ = cs.Request("stick", 4, false, func(_ *world.Tx, pl *storagenet.Plan, _ *storagenet.Task, _ error) { done, plan = true, pl })
		n.Root.Insert("log", 5, false) // after the snapshot: the plan must not see it
		for i := 0; i < 3; i++ {
			tickNet(s, tx, n, cs)
		}
		if done {
			t.Fatal("applied before the goroutine finished")
		}
		close(gate)
		waitJob(t, s, tx, n, cs, &done)
		if plan.Use["log"] != 1 || !plan.OK() {
			t.Fatalf("plan %+v", plan)
		}
		// a reservation that fails is calculated once more: the log is gone
		// by the time the first result is applied
		calcWait = nil
		done = false
		var task *storagenet.Task
		n.Root.Extract("log", 6, false)
		n.Root.Insert("log", 1, false)
		_ = cs.Request("stick", 4, true, func(_ *world.Tx, _ *storagenet.Plan, tk *storagenet.Task, _ error) { done, task = true, tk })
		n.Root.Extract("log", 1, false) // reservation will fail
		waitJob(t, s, tx, n, cs, &done)
		if task != nil {
			t.Fatal("task started without materials")
		}
	})
}

func TestTaskLimit8(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, cs := craftNet(t, tx, s)
		stickRepo(cs)
		cs.envFor = func(*world.Tx) storagenet.Env { return &fakeCraftEnv{crafts: 0} }
		n.Root.Insert("log", 100, false)
		for i := 0; i < MaxTasks; i++ {
			var done bool
			if err := cs.Request("plank", 4, true, func(_ *world.Tx, _ *storagenet.Plan, task *storagenet.Task, _ error) { done = task != nil }); err != nil {
				t.Fatal(i, err)
			}
			waitJob(t, s, tx, n, cs, &done)
		}
		if len(cs.Tasks()) != MaxTasks {
			t.Fatalf("tasks %d", len(cs.Tasks()))
		}
		if err := cs.Request("plank", 4, true, nil); err != ErrTaskLimit {
			t.Fatalf("9th task: %v", err)
		}
		if !cs.Cancel(tx, cs.Tasks()[0].ID()) || len(cs.Tasks()) != MaxTasks-1 {
			t.Fatal("cancel failed")
		}
		if n.Root.Count("log") != 100-7 {
			t.Fatalf("log %d", n.Root.Count("log"))
		}
	})
}

// startOreTask starts a processing task that has reserved one ore.
func startOreTask(t *testing.T, s *core.Session, tx *world.Tx, n *Network, cs *craftState) (ore, ingot item.Stack) {
	t.Helper()
	ore, ingot = item.NewStack(item.RawIron{}, 1), item.NewStack(item.IronIngot{}, 1)
	pat := EncodeProcessing(item.NewStack(Pattern{}, 1), []item.Stack{ore}, []item.Stack{ingot})
	_ = stateAt(tx, p(1)).Inv().SetItem(0, pat)
	cs.rescan(tx)
	cs.nextScan = 1 << 40
	cs.envFor = func(*world.Tx) storagenet.Env { return &fakeCraftEnv{crafts: 1, refuse: true} }
	n.Root.Insert(KeyOf(ore), 1, false)
	var done bool
	_ = cs.Request(KeyOf(ingot), 1, true, func(_ *world.Tx, _ *storagenet.Plan, task *storagenet.Task, _ error) { done = task != nil })
	waitJob(t, s, tx, n, cs, &done)
	return ore, ingot
}

func TestTerminalDepositGoesToTask(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		if InsertHook == nil || CraftableHook == nil || RequestHook == nil || MonitorHook == nil {
			t.Fatal("craft hooks not set")
		}
		old := InsertHook
		InsertHook = Intercept
		defer func() { InsertHook = old }()
		n, cs := craftNet(t, tx, s)
		_, ingot := startOreTask(t, s, tx, n, cs)
		cs.envFor = func(*world.Tx) storagenet.Env { return &fakeCraftEnv{crafts: 1} }
		tickNet(s, tx, n, cs)
		if cs.Tasks()[0].Status().Waiting[KeyOf(ingot)] != 1 {
			t.Fatal("task not waiting")
		}
		if got := netInsert(n, ingot.Grow(2)); got != 3 {
			t.Fatalf("inserted %d", got)
		}
		if n.Root.Count(KeyOf(ingot)) != 2 {
			t.Fatalf("root has %d ingots, want 2 (1 went to the task)", n.Root.Count(KeyOf(ingot)))
		}
		tickNet(s, tx, n, cs)
		if len(cs.Tasks()) != 0 || n.Root.Count(KeyOf(ingot)) != 3 {
			t.Fatalf("tasks %d, ingots %d", len(cs.Tasks()), n.Root.Count(KeyOf(ingot)))
		}
	})
}

func droppedItems(tx *world.Tx) map[storagenet.Key]int64 {
	out := map[storagenet.Key]int64{}
	for e := range tx.Entities() {
		if ent, ok := e.(*entity.Ent); ok {
			if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
				out[KeyOf(ib.Item())] += int64(ib.Item().Count())
			}
		}
	}
	return out
}

func TestBrokenControllerDropsBuffer(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, cs := craftNet(t, tx, s)
		ore, _ := startOreTask(t, s, tx, n, cs)
		key := configKey(p(0))
		// a rebuild (cable break) keeps the task
		Invalidate(tx, p(0))
		sweepLost(tx, s, true)
		if len(craftDoc.Of(s).Tasks[key]) != 1 || len(droppedItems(tx)) != 0 {
			t.Fatal("rebuild cancelled the task")
		}
		// the controller goes for good
		clr(tx, p(0))
		sweepLost(tx, s, true)
		if _, kept := craftDoc.Of(s).Tasks[key]; kept {
			t.Fatal("saved task of a broken controller remains")
		}
		got := droppedItems(tx)
		if len(got) != 1 || got[KeyOf(ore)] != 1 {
			t.Fatalf("dropped %v, want one ore", got)
		}
	})
}

// A reservation takes its items from an external storage, which holds no
// transaction of its own: At binds one.
func TestReservationFromExternal(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		stateAt(tx, p(0)).Unlimited = true
		put(tx, p(1), machine.KindNetExternal)
		put(tx, p(2), machine.KindNetAutocrafter)
		chest := p(1).Side(cube.FaceSouth)
		tx.SetBlock(chest, block.NewChest(), nil)
		inv := tx.Block(chest).(block.Chest).Inventory(tx, chest)
		_, _ = inv.AddItem(item.NewStack(block.Stone{}, 4))
		Invalidate(tx, p(0))
		n := At(tx, p(0))
		cs := craftOf(n)
		runController(s, stateAt(tx, p(0)), p(0), tx)
		cs.nextScan = 1 << 40
		key := KeyOf(item.NewStack(block.Stone{}, 1))
		r := storagenet.NewRepo()
		r.Add(storagenet.Pattern{ID: "p", Inputs: []storagenet.Ingredient{{Count: 2, Options: []storagenet.Key{key}}}, Outputs: []storagenet.Stack{{Key: "out", Count: 1}}})
		cs.repo = r
		cs.envFor = func(*world.Tx) storagenet.Env { return &fakeCraftEnv{crafts: 1} }
		var done bool
		var task *storagenet.Task
		if err := cs.Request("out", 1, true, func(_ *world.Tx, _ *storagenet.Plan, t *storagenet.Task, _ error) { done, task = true, t }); err != nil {
			t.Fatal(err)
		}
		for end := time.Now().Add(2 * time.Second); !done && time.Now().Before(end); time.Sleep(time.Millisecond) {
			At(tx, p(0)) // an entry point: the autocrafter's run starts with it
			cs.applyJobs(tx)
		}
		if task == nil {
			t.Fatal("no task: the reservation could not use the external chest")
		}
		left := 0
		for _, st := range inv.Slots() {
			left += st.Count()
		}
		if left != 2 {
			t.Fatalf("chest holds %d, want 2 after reserving 2", left)
		}
	})
}

// The autocrafter pushes into the block in front of it (south), not behind.
func TestPushGoesToFront(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, cs := craftNet(t, tx, s)
		_ = n
		cs.crafters = map[string][]cube.Pos{"x": {p(1)}}
		south, north := p(1).Side(cube.FaceSouth), p(1).Side(cube.FaceNorth)
		tx.SetBlock(south, block.NewChest(), nil)
		tx.SetBlock(north, block.NewChest(), nil)
		key := KeyOf(item.NewStack(block.Stone{}, 1))
		env := &craftEnv{cs: cs, tx: tx}
		if !env.Push("x", []storagenet.Stack{{Key: key, Count: 3}}) {
			t.Fatal("push refused")
		}
		count := func(pos cube.Pos) int {
			c := 0
			for _, st := range tx.Block(pos).(block.Chest).Inventory(tx, pos).Slots() {
				c += st.Count()
			}
			return c
		}
		if count(south) != 3 || count(north) != 0 {
			t.Fatalf("south %d north %d", count(south), count(north))
		}
	})
}
