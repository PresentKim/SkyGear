package netstore

import (
	"log/slog"
	"math/rand/v2"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// netEnv runs f in a synchronous world with a session.
func netEnv(t *testing.T, f func(s *core.Session, tx *world.Tx)) {
	t.Helper()
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	s := core.New(core.Config{World: w, Log: slog.Default()})
	core.Register(s)
	t.Cleanup(func() { core.Unregister(s) })
	w.Do(func(tx *world.Tx) { f(s, tx) })
}

func put(tx *world.Tx, pos cube.Pos, kind string) machine.Machine {
	m := machine.New(kind, "", "")
	tx.SetBlock(pos, m, nil)
	Invalidate(tx, pos)
	return tx.Block(pos).(machine.Machine)
}

func clr(tx *world.Tx, pos cube.Pos) {
	tx.SetBlock(pos, block.Air{}, nil)
	Invalidate(tx, pos)
}

func p(x int) cube.Pos { return cube.Pos{x, 64, 0} }

func powered(tx *world.Tx, pos cube.Pos, energy int) *machine.State {
	st := stateAt(tx, pos)
	st.Energy = energy
	return st
}

func TestAtBasics(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		if At(tx, p(0)) != nil {
			t.Fatal("empty position has a network")
		}
		put(tx, p(0), machine.KindNetController)
		put(tx, p(1), machine.KindNetCable)
		d := put(tx, p(2), machine.KindDiskDrive)
		stone := item.NewStack(block.Stone{}, 1)
		_ = d.State.Inv().SetItem(0, item.NewStack(Disk{Tier: 0}, 1))
		// attach the disk, then fill it through the Root
		n := At(tx, p(1))
		if n.Controller != p(0) || n.Controllers != 1 || n.Size() != 3 {
			t.Fatalf("network %+v", n)
		}
		n.syncDrives(tx)
		if got := n.Root.Insert(KeyOf(stone), 5, false); got != 5 {
			t.Fatalf("inserted %d", got)
		}
		n.flush()
		Invalidate(tx, p(1))
		n2 := At(tx, p(2))
		if n2 == n || n2.Root.Count(KeyOf(stone)) != 5 {
			t.Fatalf("rebuilt count %d", n2.Root.Count(KeyOf(stone)))
		}
		if At(tx, p(0)) != n2 {
			t.Fatal("members do not share the network")
		}
		if got := n2.Nodes(machine.KindDiskDrive); len(got) != 1 || got[0] != p(2) {
			t.Fatalf("drives %v", got)
		}
		if id := DiskID(mustItem(d.State.Inv().Item(0))); id == "" {
			t.Fatal("disk got no id")
		}
	})
}

func mustItem(st item.Stack, _ error) item.Stack { return st }

func TestTwoControllersOffline(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		put(tx, p(1), machine.KindNetCable)
		put(tx, p(2), machine.KindNetController)
		powered(tx, p(0), 1000)
		n := At(tx, p(1))
		runController(s, stateAt(tx, p(0)), p(0), tx)
		if n.Online() {
			t.Fatal("two controllers online")
		}
	})
}

func TestBreakCableSplits(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		put(tx, p(1), machine.KindNetCable)
		put(tx, p(2), machine.KindNetCable)
		put(tx, p(3), machine.KindDiskDrive)
		whole := At(tx, p(3))
		if whole.Controllers != 1 || len(whole.Nodes(machine.KindDiskDrive)) != 1 {
			t.Fatal("whole network wrong")
		}
		clr(tx, p(2))
		if At(tx, p(2)) != nil {
			t.Fatal("broken cable still in a network")
		}
		left, right := At(tx, p(0)), At(tx, p(3))
		if left == right || left.Controllers != 1 || right.Controllers != 0 {
			t.Fatalf("split wrong: %+v / %+v", left, right)
		}
		if len(left.Nodes(machine.KindDiskDrive)) != 0 || left.Size() != 2 || right.Size() != 1 {
			t.Fatal("members wrong after split")
		}
		if right.Online() {
			t.Fatal("part without controller online")
		}
		put(tx, p(2), machine.KindNetCable)
		if joined := At(tx, p(3)); joined.Controllers != 1 || joined.Size() != 4 {
			t.Fatal("did not rejoin")
		}
	})
}

func TestNoEnergyOffline(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		put(tx, p(1), machine.KindDiskDrive)
		put(tx, p(2), machine.KindNetAutocrafter)
		put(tx, p(3), machine.KindNetCable)
		st := powered(tx, p(0), 0)
		n := At(tx, p(0))
		runController(s, st, p(0), tx)
		if n.Online() || !st.Stalled {
			t.Fatal("online without energy")
		}
		// 8 + drive 2 + autocrafter 4, the cable is free
		if n.Use() != 14 {
			t.Fatalf("use %d", n.Use())
		}
		st.Energy = 13
		runController(s, st, p(0), tx)
		if n.Online() || st.Energy != 13 {
			t.Fatal("paid with too little energy")
		}
		st.Energy = 20
		runController(s, st, p(0), tx)
		if !n.Online() || st.Energy != 6 || st.Stalled {
			t.Fatalf("online %v energy %d", n.Online(), st.Energy)
		}
		st.Energy = 0
		runController(s, st, p(0), tx)
		if n.Online() {
			t.Fatal("still online with an empty buffer")
		}
	})
}

func TestNodeCap(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		for i := 1; i < NodeCap; i++ {
			tx.SetBlock(p(i), machine.New(machine.KindNetCable, "", ""), nil)
		}
		st := powered(tx, p(0), 100000)
		n := At(tx, p(0))
		runController(s, st, p(0), tx)
		if n.Size() != NodeCap || !n.Online() {
			t.Fatalf("512 nodes: size %d online %v", n.Size(), n.Online())
		}
		put(tx, p(NodeCap), machine.KindNetCable)
		n = At(tx, p(0))
		runController(s, st, p(0), tx)
		if n.Size() != NodeCap+1 || n.Online() {
			t.Fatalf("513 nodes: size %d online %v", n.Size(), n.Online())
		}
	})
}

func TestCacheMatchesFresh(t *testing.T) {
	kinds := []string{machine.KindNetController, machine.KindNetCable, machine.KindDiskDrive, machine.KindNetTerminal}
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		rng := rand.New(rand.NewPCG(7, 9))
		var all []cube.Pos
		for x := range 5 {
			for z := range 3 {
				all = append(all, cube.Pos{x, 64, z})
			}
		}
		for step := range 50 {
			pos := all[rng.IntN(len(all))]
			if nodeAt(tx, pos) != "" && rng.IntN(2) == 0 {
				clr(tx, pos)
			} else {
				put(tx, pos, kinds[rng.IntN(len(kinds))])
			}
			for _, q := range all {
				got := At(tx, q) // fills the cache
				if nodeAt(tx, q) == "" {
					if got != nil {
						t.Fatalf("step %d: network at empty %v", step, q)
					}
					continue
				}
				fresh := flood(tx, q)
				if len(got.members) != len(fresh) {
					t.Fatalf("step %d: %v cached %d fresh %d", step, q, len(got.members), len(fresh))
				}
				for m, k := range fresh {
					if got.members[m] != k || At(tx, m) != got {
						t.Fatalf("step %d: %v differs at %v", step, q, m)
					}
				}
			}
		}
	})
}

func TestVerifyDropsStaleNetwork(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		put(tx, p(1), machine.KindNetCable)
		st := powered(tx, p(0), 1000)
		n := At(tx, p(0))
		tx.SetBlock(p(1), block.Air{}, nil) // changed without Invalidate
		n.Tick = checkEvery - 1
		runController(s, st, p(0), tx)
		if m := At(tx, p(0)); m == n || m.Size() != 1 {
			t.Fatal("stale network kept")
		}
	})
}

func TestConfig(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		c := Config(s, p(3))
		c.Priority = 5
		if Config(s, p(3)).Priority != 5 {
			t.Fatal("config not kept")
		}
		Forget(s, p(3))
		if Config(s, p(3)).Priority != 0 {
			t.Fatal("config not forgotten")
		}
	})
}

// A network dropped while a screen still holds it must not move items or
// start and cancel tasks: its successor owns them.
func TestStaleNetworkIsInert(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		put(tx, p(1), machine.KindNetTerminal)
		powered(tx, p(0), 1000)
		old := At(tx, p(1))
		if !old.Online() {
			t.Fatal("fresh network offline")
		}
		cs := craftOf(old)
		if cs == nil {
			t.Fatal("no craft state")
		}
		Invalidate(tx, p(1))
		if old.Online() {
			t.Fatal("dropped network still online")
		}
		stone := item.NewStack(block.Stone{}, 4)
		if netInsert(old, stone) != 0 {
			t.Fatal("dropped network took items")
		}
		if cur := At(tx, p(1)); !cur.Online() || cur == old {
			t.Fatal("successor not online")
		}
		if cs.Cancel(tx, "t1") || cs.Request(KeyOf(stone), 1, true, nil) != ErrStale {
			t.Fatal("dropped network's tasks usable")
		}
	})
}

// TestNetConduitCarriesTheNetwork: a conduit bundle with a network conduit
// joins the network as a cable does, also beside other conduits; taking the
// network conduit out of it splits the network again.
func TestNetConduitCarriesTheNetwork(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		tx.SetBlock(p(1), conduit.Bundle{Net: 1, Energy: 1}, nil)
		tx.SetBlock(p(2), conduit.Bundle{Net: 1}, nil)
		put(tx, p(3), machine.KindDiskDrive)
		n := At(tx, p(3))
		if n == nil || n.Controllers != 1 || n.Size() != 4 || len(n.Nodes(KindConduit)) != 2 {
			t.Fatalf("network over the conduits %+v", n)
		}
		if n.Use() != controllerUse+nodeUse(machine.KindDiskDrive) {
			t.Errorf("use %d: the conduits should take no energy", n.Use())
		}
		// the neighbour update that tells it of p(2) comes on the next tick
		tx.Block(p(1)).(conduit.Bundle).Refresh(p(1), tx)
		// the controller's list names the conduits instead of failing on a kind
		// that is no machine
		if _, title := kindLook("ko", KindConduit); title != "망 도관" {
			t.Errorf("conduit nodes listed as %q", title)
		}
		if b := tx.Block(p(1)).(conduit.Bundle); b.NConn&(1<<4) == 0 || b.NConn&(1<<5) == 0 {
			t.Errorf("bundle connects %06b, want west and east", b.NConn)
		}
		tx.SetBlock(p(1), conduit.Bundle{Energy: 1}, nil)
		if n := At(tx, p(3)); n == nil || n.Controllers != 0 {
			t.Fatalf("the drive still reaches the controller: %+v", n)
		}
		if At(tx, p(1)) != nil {
			t.Error("a bundle without a network conduit is a node")
		}
	})
}
