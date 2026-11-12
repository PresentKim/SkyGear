package netstore

import (
	"fmt"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// TestEndToEnd runs a whole network: a controller with energy, a cable, a
// drive with a disk, a terminal, an importer and an exporter each facing a
// chest, and an autocrafter with a crafting pattern.
func TestEndToEnd(t *testing.T) {
	id := planksID(t)
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		const (
			cable, drive, term, imp, exp, auto = 1, 2, 3, 4, 5, 6
		)
		put(tx, p(0), machine.KindNetController)
		stateAt(tx, p(0)).Unlimited = true
		put(tx, p(cable), machine.KindNetCable)
		d := put(tx, p(drive), machine.KindDiskDrive)
		_ = d.State.Inv().SetItem(0, item.NewStack(Disk{Tier: 1}, 1))
		put(tx, p(term), machine.KindNetTerminal)
		put(tx, p(imp), machine.KindNetImporter)
		put(tx, p(exp), machine.KindNetExporter)
		a := put(tx, p(auto), machine.KindNetAutocrafter)
		_ = a.State.Inv().SetItem(0, EncodeCrafting(blank(), id))
		for _, x := range []int{imp, exp} {
			tx.SetBlock(p(x).Side(cube.FaceSouth), block.NewChest(), nil)
		}
		Config(s, p(exp)).Filter = []string{string(KeyOf(stoneS))}
		Config(s, p(exp)).Allow = true
		Invalidate(tx, p(0))

		n := At(tx, p(term))
		if n == nil || !n.Online() && n.Controllers != 1 {
			t.Fatal("no network")
		}
		cs := craftOf(n)
		tick := func() {
			runController(s, stateAt(tx, p(0)), p(0), tx)
			nodeRun(s, nil, p(imp), tx)
			nodeRun(s, nil, p(exp), tx)
			cs.tick(tx)
		}
		tick()
		if !n.Online() {
			t.Fatal("network offline")
		}

		// importer: a chest of stone and dirt goes into the disk
		fill(tx, p(imp), stoneS.Grow(19)) // 20 stone
		fill(tx, p(imp), dirtS.Grow(4))   // 5 dirt
		for range 50 * ioRate {
			tick()
		}
		if held(tx, p(imp), stoneS) != 0 || held(tx, p(imp), dirtS) != 0 {
			t.Fatal("importer left items in its chest")
		}
		// exporter: only the filtered kind leaves, into its own chest
		if got := held(tx, p(exp), stoneS); got != 20 {
			t.Fatalf("exported stone %d, want 20", got)
		}
		if held(tx, p(exp), dirtS) != 0 || n.Root.Count(KeyOf(dirtS)) != 5 {
			t.Fatalf("dirt: chest %d, network %d", held(tx, p(exp), dirtS), n.Root.Count(KeyOf(dirtS)))
		}

		// terminal: take the dirt, put it back
		out := netExtract(n, KeyOf(dirtS), 64)
		if out.Count() != 5 || n.Root.Count(KeyOf(dirtS)) != 0 {
			t.Fatalf("took %v, left %d", out, n.Root.Count(KeyOf(dirtS)))
		}
		if got := machine.Kinds[machine.KindNetTerminal].Hooks().Insert(tx, p(term), out); got != 5 {
			t.Fatalf("deposit %d", got)
		}

		// autocrafter: a request for planks completes and lands in Root
		log := item.NewStack(block.Log{Wood: block.OakWood()}, 1)
		planks := item.NewStack(block.Planks{Wood: block.OakWood()}, 1)
		if netInsert(n, log.Grow(1)) != 2 {
			t.Fatal("log deposit")
		}
		cs.nextScan = 0
		tick()
		var done bool
		if err := cs.Request(KeyOf(planks), 4, true, func(_ *world.Tx, plan *storagenet.Plan, _ *storagenet.Task, err error) {
			done = true
			if err != nil || !plan.OK() {
				t.Error("plan", err, plan)
			}
		}); err != nil {
			t.Fatal(err)
		}
		for end := time.Now().Add(2 * time.Second); !done && time.Now().Before(end); time.Sleep(time.Millisecond) {
			tick()
		}
		if !done {
			t.Fatal("calculation never applied")
		}
		for i := 0; i < 60 && len(cs.Tasks()) > 0; i++ {
			tick()
		}
		if len(cs.Tasks()) != 0 || n.Root.Count(KeyOf(planks)) != 4 || n.Root.Count(KeyOf(log)) != 1 {
			t.Fatalf("tasks %d, planks %d, logs %d", len(cs.Tasks()), n.Root.Count(KeyOf(planks)), n.Root.Count(KeyOf(log)))
		}
	})
}

// BenchmarkRootInsert is an insert into a Root of 64 disks holding 5000
// kinds.
func BenchmarkRootInsert(b *testing.B) {
	r := storagenet.NewRoot()
	for i := range 64 {
		r.Add(fmt.Sprint("d", i), storagenet.NewDisk(1<<40), i%5, nil)
	}
	keys := make([]storagenet.Key, 5000)
	for i := range keys {
		keys[i] = storagenet.Key(fmt.Sprint("k", i))
		r.Insert(keys[i], 10, false)
	}
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		r.Insert(keys[i%len(keys)], 1, false)
	}
}
