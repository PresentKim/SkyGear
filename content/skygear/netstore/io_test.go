package netstore

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

var (
	stoneS = item.NewStack(block.Stone{}, 1)
	dirtS  = item.NewStack(block.Dirt{}, 1)
)

// ioNet builds a controller at p(0), a node of each kind in kinds at p(1..)
// with a chest in front (south, where the front texture is) of each, and a disk drive with a 1k disk after
// them. It returns the network and the node positions.
func ioNet(tx *world.Tx, kinds ...string) (*Network, []cube.Pos) {
	put(tx, p(0), machine.KindNetController)
	stateAt(tx, p(0)).Unlimited = true
	var ps []cube.Pos
	for i, k := range kinds {
		pos := p(i + 1)
		put(tx, pos, k)
		ps = append(ps, pos)
		if k != machine.KindNetExternal {
			tx.SetBlock(pos.Side(cube.FaceSouth), block.NewChest(), nil)
		}
	}
	d := put(tx, p(len(kinds)+1), machine.KindDiskDrive)
	_ = d.State.Inv().SetItem(0, item.NewStack(Disk{Tier: 0}, 1))
	Invalidate(tx, p(0))
	return At(tx, p(0)), ps
}

func chestOf(tx *world.Tx, node cube.Pos) block.Chest {
	return tx.Block(node.Side(cube.FaceSouth)).(block.Chest)
}

func fill(tx *world.Tx, node cube.Pos, st item.Stack) {
	c := chestOf(tx, node)
	_, _ = c.Inventory(tx, node.Side(cube.FaceSouth)).AddItem(st)
}

func held(tx *world.Tx, node cube.Pos, st item.Stack) int {
	n := 0
	for _, s := range chestOf(tx, node).Inventory(tx, node.Side(cube.FaceSouth)).Slots() {
		if s.Comparable(st) {
			n += s.Count()
		}
	}
	return n
}

// step is one controller tick of the scheduler and the runs of the nodes; it
// returns how many jobs the scheduler ran.
func step(s *core.Session, tx *world.Tx, n *Network, nodes ...cube.Pos) int {
	n.Tick++
	ran := n.Sched.Run(n.Tick)
	for _, pos := range nodes {
		nodeRun(s, nil, pos, tx)
	}
	return ran
}

func TestListChest(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		pos := cube.Pos{0, 64, 5}
		tx.SetBlock(pos, block.NewChest(), nil)
		inv := tx.Block(pos).(block.Chest).Inventory(tx, pos)
		_, _ = inv.AddItem(stoneS.Grow(2))
		_, _ = inv.AddItem(dirtS.Grow(63))
		_, _ = inv.AddItem(dirtS.Grow(63))
		got := itemio.List(tx, pos, cube.FaceUp)
		if len(got) != 2 || !got[0].Comparable(stoneS) || got[0].Count() != 3 || !got[1].Comparable(dirtS) || got[1].Count() != 128 {
			t.Fatalf("list %v", got)
		}
	})
}

func TestImporterMoves(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, ps := ioNet(tx, machine.KindNetImporter)
		fill(tx, ps[0], stoneS.Grow(4))
		for range 5 * ioRate {
			step(s, tx, n, ps...)
		}
		if got := n.Root.Count(KeyOf(stoneS)); got != 5 || held(tx, ps[0], stoneS) != 0 {
			t.Fatalf("network %d chest %d", got, held(tx, ps[0], stoneS))
		}
	})
}

func TestImporterMovesOneItemEveryNineTicks(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, ps := ioNet(tx, machine.KindNetImporter)
		fill(tx, ps[0], stoneS.Grow(63))
		ran := 0
		for range 10 * ioRate {
			ran += step(s, tx, n, ps...)
		}
		if ran != 10 || n.Root.Count(KeyOf(stoneS)) != 10 {
			t.Fatalf("%d runs moved %d in %d ticks, want 10 and 10", ran, n.Root.Count(KeyOf(stoneS)), 10*ioRate)
		}
	})
}

func TestUpgradesSpeedAndStack(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		st := machine.New(machine.KindNetImporter, "", "").State
		for i, want := range []int64{9, 7, 5, 3, 1, 1} {
			if got := ioRateOf(st); got != want {
				t.Fatalf("%d speed upgrades: every %d ticks, want %d", i, got, want)
			}
			if i < 4 && !upgradeFits(st, item.NewStack(Upgrade{}, 1)) {
				t.Fatalf("speed upgrade %d did not fit", i+1)
			}
			if i < len(st.Def.Input) {
				_ = st.Inv().SetItem(st.Def.Input[i], item.NewStack(Upgrade{}, 1))
			}
		}
		if upgradeFits(st, item.NewStack(Upgrade{}, 1)) {
			t.Fatal("a fifth speed upgrade fits")
		}
		_ = st.Inv().SetItem(st.Def.Input[3], item.NewStack(Upgrade{Stack: true}, 1))
		if ioAmountOf(st) != 64 || ioRateOf(st) != 3 {
			t.Fatalf("stack upgrade: %d every %d ticks, want 64 every 3", ioAmountOf(st), ioRateOf(st))
		}
		if upgradeFits(st, item.NewStack(Upgrade{Stack: true}, 1)) {
			t.Fatal("a second stack upgrade fits")
		}
	})
}

func TestStackUpgradeImportsAStack(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, ps := ioNet(tx, machine.KindNetImporter)
		_ = stateAt(tx, ps[0]).Inv().SetItem(0, item.NewStack(Upgrade{Stack: true}, 1))
		fill(tx, ps[0], stoneS.Grow(63))
		for range ioRate {
			step(s, tx, n, ps...)
		}
		if got := n.Root.Count(KeyOf(stoneS)); got != 64 {
			t.Fatalf("imported %d in one operation, want 64", got)
		}
	})
}

func TestExporterMovesOneFilteredItemEveryNineTicks(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		Config(s, p(1)).Filter = []string{string(KeyOf(stoneS))}
		n, ps := ioNet(tx, machine.KindNetExporter)
		n.Root.Insert(KeyOf(stoneS), 5, false)
		n.Root.Insert(KeyOf(dirtS), 5, false)
		for range 3 * ioRate {
			step(s, tx, n, ps...)
		}
		if got := held(tx, ps[0], stoneS); got != 3 {
			t.Fatalf("exported %d stone in 3 runs, want 3", got)
		}
		if held(tx, ps[0], dirtS) != 0 {
			t.Fatal("exported dirt")
		}
	})
}

func TestNoTransferIntoAnExternalStoragesBlock(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, chest := extNet(s, tx)
		rt := rtOf(n)
		node := chest.Side(cube.FaceSouth)
		for _, kind := range []string{machine.KindNetImporter, machine.KindNetExporter} {
			m := machine.New(kind, "", "")
			m.State.Facing = cube.FaceNorth
			tx.SetBlock(node, m, nil)
			Config(s, node).Filter = []string{string(KeyOf(stoneS))}
			bind(n, tx)
			if importWork(tx, n, rt, node) || exportWork(tx, n, rt, node) {
				t.Errorf("%s moved items in and out of the mirrored chest", kind)
			}
		}
	})
}

func TestExporterEmptyFilter(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, ps := ioNet(tx, machine.KindNetExporter)
		n.Root.Insert(KeyOf(stoneS), 5, false)
		for range 20 {
			step(s, tx, n, ps...)
		}
		if held(tx, ps[0], stoneS) != 0 || n.Root.Count(KeyOf(stoneS)) != 5 {
			t.Fatal("empty filter exported")
		}
	})
}

func TestFilterDeny(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		Config(s, p(1)).Filter = []string{string(KeyOf(stoneS))} // deny list
		n, ps := ioNet(tx, machine.KindNetImporter)
		fill(tx, ps[0], stoneS.Grow(4))
		fill(tx, ps[0], dirtS.Grow(4))
		for range 100 {
			step(s, tx, n, ps...)
		}
		if n.Root.Count(KeyOf(dirtS)) != 5 || n.Root.Count(KeyOf(stoneS)) != 0 || held(tx, ps[0], stoneS) != 5 {
			t.Fatalf("dirt %d stone %d left %d", n.Root.Count(KeyOf(dirtS)), n.Root.Count(KeyOf(stoneS)), held(tx, ps[0], stoneS))
		}
	})
}

func TestExternalChest(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		pos := p(1)
		tx.SetBlock(pos.Side(cube.FaceSouth), block.NewChest(), nil)
		inv := tx.Block(pos.Side(cube.FaceSouth)).(block.Chest).Inventory(tx, pos.Side(cube.FaceSouth))
		_, _ = inv.AddItem(stoneS.Grow(9))
		n, ps := ioNet(tx, machine.KindNetExternal)
		_ = ps
		if got := n.Root.Count(KeyOf(stoneS)); got != 10 {
			t.Fatalf("mirrored %d", got)
		}
		defer Touch(n, tx)()
		if got := n.Root.Extract(KeyOf(stoneS), 4, false); got != 4 {
			t.Fatalf("extracted %d", got)
		}
		// a change from outside shows after the poll
		_, _ = inv.AddItem(stoneS.Grow(2))
		for range 40 {
			step(s, tx, n, ps...)
		}
		if got := n.Root.Count(KeyOf(stoneS)); got != 9 {
			t.Fatalf("after poll %d", got)
		}
	})
}

func TestExternalDrawerController(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		put(tx, p(0), machine.KindNetController)
		ctl := cube.Pos{1, 64, 1} // in front of the external storage at p(1)
		tx.SetBlock(ctl, machine.New(machine.KindDrawerController, "", ""), nil)
		tx.SetBlock(ctl.Side(cube.FaceSouth), storage.Drawer{Item: stoneS, Count: 100}, nil)
		tx.SetBlock(ctl.Side(cube.FaceWest), storage.Drawer{Item: stoneS, Count: 50}, nil)
		n, ps := ioNet(tx, machine.KindNetExternal)
		if got := n.Root.Count(KeyOf(stoneS)); got != 150 {
			t.Fatalf("mirrored %d, want 150 (once)", got)
		}
		defer Touch(n, tx)()
		if got := n.Root.Extract(KeyOf(stoneS), 30, false); got != 30 {
			t.Fatalf("extracted %d", got)
		}
		list, _ := storage.Contents(tx, ctl)
		if len(list) != 1 || list[0].Count != 120 {
			t.Fatalf("drawers hold %v", list)
		}
		for range 40 {
			step(s, tx, n, ps...)
		}
		if got := n.Root.Count(KeyOf(stoneS)); got != 120 {
			t.Fatalf("after poll %d", got)
		}
	})
}

func TestImporterTakesFromItsFacing(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, ps := ioNet(tx, machine.KindNetImporter)
		stateAt(tx, ps[0]).Facing = cube.FaceNorth
		north := ps[0].Side(cube.FaceNorth)
		tx.SetBlock(north, block.NewChest(), nil)
		_, _ = tx.Block(north).(block.Chest).Inventory(tx, north).AddItem(dirtS.Grow(8))
		fill(tx, ps[0], stoneS.Grow(8))
		for range 80 {
			step(s, tx, n, ps...)
		}
		if n.Root.Count(KeyOf(dirtS)) != 9 || n.Root.Count(KeyOf(stoneS)) != 0 {
			t.Fatalf("dirt %d, stone %d", n.Root.Count(KeyOf(dirtS)), n.Root.Count(KeyOf(stoneS)))
		}
	})
}

func TestCableArmsFollowNeighbours(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		update := func() {
			tx.Block(p(1)).(machine.Machine).NeighbourUpdateTick(p(1), p(2), tx)
		}
		put(tx, p(1), machine.KindNetCable)
		put(tx, p(0), machine.KindNetController)
		imp := machine.New(machine.KindNetImporter, "", "")
		imp.State.Facing = cube.FaceWest
		tx.SetBlock(p(2), imp, nil)
		update()
		if got := machine.NetMask(tx.Block(p(1)).(machine.Machine)); got != 1<<cube.FaceWest {
			t.Fatalf("cable shape %06b, want an arm west only", got)
		}
		imp.State.Facing = cube.FaceUp
		update()
		if got := machine.NetMask(tx.Block(p(1)).(machine.Machine)); got != 1<<cube.FaceWest|1<<cube.FaceEast {
			t.Fatalf("cable shape %06b, want arms west and east", got)
		}
	})
}

// An attachment placed without a clicked face faces south: an importer takes
// from the block south of it and ignores the one north of it.
func TestFrontIsSouth(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, ps := ioNet(tx, machine.KindNetImporter)
		north := ps[0].Side(cube.FaceNorth)
		tx.SetBlock(north, block.NewChest(), nil)
		_, _ = tx.Block(north).(block.Chest).Inventory(tx, north).AddItem(dirtS.Grow(8))
		fill(tx, ps[0], stoneS.Grow(8))
		for range 80 {
			step(s, tx, n, ps...)
		}
		if n.Root.Count(KeyOf(stoneS)) != 9 || held(tx, ps[0], stoneS) != 0 {
			t.Fatalf("south chest: net %d, left %d", n.Root.Count(KeyOf(stoneS)), held(tx, ps[0], stoneS))
		}
		if n.Root.Count(KeyOf(dirtS)) != 0 {
			t.Fatalf("imported %d from the back", n.Root.Count(KeyOf(dirtS)))
		}
	})
}

// extNet is a controller, an external storage and a drive-less network whose
// external storage mirrors a chest holding 9 stone. The controller has run
// once, which powers the network and ends its binding of the transaction.
func extNet(s *core.Session, tx *world.Tx) (*Network, cube.Pos) {
	put(tx, p(0), machine.KindNetController)
	stateAt(tx, p(0)).Unlimited = true
	ext := put(tx, p(1), machine.KindNetExternal)
	_ = ext
	chest := p(1).Side(cube.FaceSouth)
	tx.SetBlock(chest, block.NewChest(), nil)
	_, _ = tx.Block(chest).(block.Chest).Inventory(tx, chest).AddItem(stoneS.Grow(8))
	Invalidate(tx, p(0))
	n := At(tx, p(0))
	runController(s, stateAt(tx, p(0)), p(0), tx)
	return n, chest
}

func TestExternalOutsideNodeRuns(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n, chest := extNet(s, tx)
		if rtOf(n).tx != nil {
			t.Fatal("controller run left its transaction bound")
		}
		if !n.Online() || n.Root.Count(KeyOf(stoneS)) != 9 {
			t.Fatalf("online %v, mirrored %d", n.Online(), n.Root.Count(KeyOf(stoneS)))
		}
		// the controller's item I/O reaches the chest through At alone
		got := machine.Kinds[machine.KindNetController].Extract(tx, p(0), stoneS, 4)
		if got.Count() != 4 {
			t.Fatalf("controller extract %d", got.Count())
		}
		// so does the terminal's take
		if got := netExtract(At(tx, p(0)), KeyOf(stoneS), 3); got.Count() != 3 {
			t.Fatalf("terminal take %d", got.Count())
		}
		left := 0
		for _, st := range tx.Block(chest).(block.Chest).Inventory(tx, chest).Slots() {
			left += st.Count()
		}
		if left != 2 {
			t.Fatalf("chest holds %d, want 2", left)
		}
	})
}

// A transaction that ended is never used by an external storage.
func TestExternalStaleTx(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer func() { _ = w.Close() }()
	var old *world.Tx
	w.Do(func(tx *world.Tx) { old = tx })
	rt := &ioRT{tx: old}
	if rt.cur() != nil || rt.tx != nil {
		t.Fatal("stale transaction still in use")
	}
}
