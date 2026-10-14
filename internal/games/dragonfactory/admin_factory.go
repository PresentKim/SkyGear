package dragonfactory

import (
	"container/heap"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// adminFactory builds a large test factory on the island: count modules in
// a grid centred on the operator, each a machine line (cobblestone, auto miner, crusher,
// hopper into a drawer, sell box), an auto sieve line fed by a chest of
// gravel, and a 9×9 farm with an auto harvester. It is a load test for
// letting players build freely (docs/tech/performance.md); watch the load
// lines on the sidebar.
type adminFactory struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"factory"`
	Count int            `cmd:"count"`
}

// Module size in blocks (x, z) with a one block gap.
const (
	factoryModuleX = 19
	factoryModuleZ = 11
	maxFactory     = 100
)

func (c adminFactory) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	n := max(1, min(c.Count, maxFactory))
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	rows := (n + cols - 1) / cols
	feet := cube.PosFromVec3(p.Position())
	// Centred on the operator: blocks tick only within the tick range (the
	// load radius in chunks) around a viewer, here the operator (the island's
	// own loaders may tick more).
	origin := feet.Add(cube.Pos{-cols * factoryModuleX / 2, -1, -rows * factoryModuleZ / 2})
	reach := s.tickReach
	var built factoryCount
	outside := 0
	for i := range n {
		at := origin.Add(cube.Pos{(i % cols) * factoryModuleX, 0, (i / cols) * factoryModuleZ})
		far := at.Add(cube.Pos{factoryModuleX, 0, factoryModuleZ})
		if max(abs(at[0]-feet[0]), abs(far[0]-feet[0]), abs(at[2]-feet[2]), abs(far[2]-feet[2])) > reach {
			outside++
		}
		s.buildFactoryModule(tx, at, instance.AccountID(p), p.Name(), &built)
	}
	o.Printf(i18n.T(i18n.Default, "dragonfactory.admin_factory.Run.1"), n, built.machines, built.drawers, built.hoppers, built.crops, built.blocks, outside)
}

// factoryCount counts what a factory build placed.
type factoryCount struct {
	machines, drawers, hoppers, crops, blocks int
}

// buildFactoryModule builds one module whose platform's north-west corner
// is o (the platform is at o's height, everything else above it).
func (s *session) buildFactoryModule(tx *world.Tx, o cube.Pos, owner, ownerName string, n *factoryCount) {
	set := func(dx, dy, dz int, b world.Block) cube.Pos {
		pos := o.Add(cube.Pos{dx, dy, dz})
		tx.SetBlock(pos, b, nil)
		n.blocks++
		return pos
	}
	machine := func(dx, dy, dz int, kind string) {
		m := newMachine(kind, owner, ownerName)
		// Test factories run without energy.
		m.st.unlimited = true
		if isSieve(kind) {
			m.st.mesh = "string"
		}
		set(dx, dy, dz, m)
		n.machines++
	}
	hopper := func(dx, dy, dz int, facing cube.Face) {
		h := block.NewHopper()
		h.Facing = facing
		set(dx, dy, dz, h)
		n.hoppers++
	}
	for x := range factoryModuleX - 1 {
		for z := range factoryModuleZ - 1 {
			set(x, 0, z, block.Stone{})
		}
	}

	// Machine line (docs/tech/machines.md): the auto miner mines the
	// cobblestone under it once and hands it to the crusher (into a hopper
	// and a drawer below).
	set(1, 1, 2, block.Cobblestone{})
	set(1, 2, 2, block.Cobblestone{})
	machine(1, 4, 2, KindMiner)
	machine(2, 4, 2, KindCrusher)
	hopper(2, 3, 2, cube.FaceDown)
	drawerPos := set(2, 2, 2, Drawer{Facing: cube.South})
	s.setDrawer(tx, drawerPos, Drawer{Facing: cube.South})
	n.drawers++

	// Sieve line: a chest of gravel, a hopper into the auto sieve, chests
	// beside and below it for the products.
	gravel := set(1, 3, 6, block.NewChest())
	if c, ok := tx.Block(gravel).(block.Chest); ok {
		inv := c.Inventory(tx, gravel)
		for slot := range inv.Size() {
			_ = inv.SetItem(slot, item.NewStack(block.Gravel{}, 64))
		}
	}
	hopper(1, 2, 6, cube.FaceEast)
	machine(2, 2, 6, KindAutoSieve)
	set(3, 2, 6, block.NewChest())
	set(2, 1, 6, block.NewChest())

	// Farm: 9×9 farmland around the auto harvester, water in the corners,
	// crops at random stages, a chest above the harvester.
	const fx, fz = 6, 0
	kinds := []string{"wheat", "carrot", "potato", "beetroot"}
	now, _, _ := s.clock.Now()
	for x := range 9 {
		for z := range 9 {
			corner := (x == 0 || x == 8) && (z == 0 || z == 8)
			switch {
			case x == 4 && z == 4:
				continue
			case corner:
				set(fx+x, 1, fz+z, block.Water{Still: true, Depth: 8})
				continue
			}
			set(fx+x, 1, fz+z, block.Farmland{Hydration: 7})
			kind := kinds[(x+z)%len(kinds)]
			pos := set(fx+x, 2, fz+z, cropBlock(kind))
			if s.state.Crops == nil {
				s.state.Crops = map[string]*CropRecord{}
			}
			grow := s.growTime(kind)
			r := &CropRecord{Kind: kind, Planted: now - time.Duration(s.rand.Int64N(int64(grow)+1))}
			s.state.Crops[cropKey(pos)] = r
			// Due at once: the first update brings the block to its stage.
			heap.Push(&s.cropQueue, cropDue{at: now, pos: pos, rec: r})
			n.crops++
		}
	}
	machine(fx+4, 2, fz+4, KindHarvester)
	set(fx+4, 3, fz+4, block.NewChest())
}

// cropBlock is a young crop block of kind.
func cropBlock(kind string) world.Block {
	switch kind {
	case "carrot":
		return block.Carrot{}
	case "potato":
		return block.Potato{}
	case "beetroot":
		return block.BeetrootSeeds{}
	default:
		return block.WheatSeeds{}
	}
}

// adminDense builds count compact machine cells in a grid centred on the
// operator: each cell is 2×1 blocks, cobblestone, an
// auto miner above its output, a crusher beside the miner over a hopper
// into a drawer. Many machines in little room, for load tests
// in game (docs/tech/performance.md).
type adminDense struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"dense"`
	Count int            `cmd:"count"`
}

const maxDense = 4000

func (c adminDense) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	n := max(1, min(c.Count, maxDense))
	// Cells are 2 wide and 1 deep: twice as many rows as columns is square.
	cols := max(1, int(math.Ceil(math.Sqrt(float64(n)/2))))
	rows := (n + cols - 1) / cols
	feet := cube.PosFromVec3(p.Position())
	origin := feet.Add(cube.Pos{-cols, -1, -rows / 2})
	owner, name := instance.AccountID(p), p.Name()
	for i := range n {
		at := origin.Add(cube.Pos{(i % cols) * 2, 0, i / cols})
		set := func(dx, dy int, b world.Block) cube.Pos {
			pos := at.Add(cube.Pos{dx, dy, 0})
			tx.SetBlock(pos, b, nil)
			return pos
		}
		set(0, 0, block.Stone{})
		set(1, 0, block.Stone{})
		set(0, 1, block.Cobblestone{})
		set(0, 2, block.Cobblestone{})
		miner, crusher := newMachine(KindMiner, owner, name), newMachine(KindCrusher, owner, name)
		miner.st.unlimited, crusher.st.unlimited = true, true
		set(0, 4, miner)
		set(1, 4, crusher)
		h := block.NewHopper()
		h.Facing = cube.FaceDown
		set(1, 3, h)
		d := Drawer{Facing: cube.South}
		s.setDrawer(tx, set(1, 2, d), d)
	}
	o.Printf(i18n.T(i18n.Default, "dragonfactory.admin_factory.Run.2"), n, cols*2, rows, n*2, n, n)
}
