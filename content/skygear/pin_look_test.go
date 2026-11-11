package skygear

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// pinStack is a stack as a pin line shows it: "-" when empty, else
// "<id>:<meta>x<count>".
func pinStack(st item.Stack) string {
	if st.Empty() {
		return "-"
	}
	name, meta := st.Item().EncodeItem()
	return fmt.Sprintf("%s:%dx%d", name, meta, st.Count())
}

// pinInfo is a panel as one pin line: every field of waila.Info.
func pinInfo(info waila.Info) string {
	icon := "-"
	if info.Item != nil {
		name, meta := info.Item.EncodeItem()
		icon = fmt.Sprintf("%s:%d", name, meta)
	}
	line := "-"
	if l := info.Line; l != nil {
		line = fmt.Sprintf("{%s name=%q text=%q gauge=%d show=%t}", pinStack(l.Item), l.Name, l.Text, l.Gauge, l.ShowGauge)
	}
	cont := "-"
	if c := info.Container; c != nil {
		var items []string
		for _, it := range c.Items {
			items = append(items, pinStack(it))
		}
		cont = fmt.Sprintf("{%s [%s] progress=%s}", c.Layout, strings.Join(items, " "), pinStack(c.Progress))
	}
	var tools []string
	for _, t := range info.Tools {
		tools = append(tools, string(t.Tool)+string(rune(t.Mark)))
	}
	return fmt.Sprintf("name=%q key=%q icon=%s lines=%q line=%s container=%s tools=%v marks=%+v source=%q",
		info.Name, info.NameKey, icon, info.Lines, line, cont, tools, info.Marks, info.Source)
}

// TestPinLook pins the WAILA panel (look.Info) of every kind of block the
// game adds lines to: each machine kind idle, to a standing and to a sneaking
// viewer, the machine states that change the lines, the drawer, the composter,
// the conduit bundle, the silkworm leaves, the barrels, the hoppers, the magic
// plant, the crops and the vanilla containers. The golden was recorded from
// the type switch of look.Info; it changes only when a panel is changed on
// purpose.
func TestPinLook(t *testing.T) {
	// The panel names blocks by the pack's names, as the running server does.
	saved := naming.PackNames
	t.Cleanup(func() { naming.PackNames = saved })
	if err := naming.LoadNames(os.DirFS(packDir)); err != nil {
		t.Fatal(err)
	}
	s := testSession(t)
	progress.LocalOf(s.Session).RecipesFree = true
	loadAround(s)
	h := spawnPlayer(s)
	lines := map[string]string{}
	y := island.IslandY + 6
	next := 0
	// free is a new position above the island, two blocks from the last.
	free := func() cube.Pos {
		next++
		return cube.Pos{2 + (next%6)*2, y + (next/36)*3, 2 + (next/6%6)*2}
	}
	fs := farm.StateOf(s.Session)
	if fs.Barrels == nil {
		fs.Barrels = map[string]*farm.BarrelRecord{}
	}
	if fs.Plants == nil {
		fs.Plants = map[string]*farm.PlantRecord{}
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		rec := func(name string, pos cube.Pos, b world.Block) {
			t.Helper()
			if _, dup := lines[name]; dup {
				t.Fatalf("pin scenario %q recorded twice", name)
			}
			lines[name] = pinInfo(look.Info(s.Session, p, tx, pos, b))
		}
		// both records name plain and name/sneak.
		both := func(name string, pos cube.Pos, b world.Block) {
			t.Helper()
			rec(name, pos, b)
			p.StartSneaking()
			rec(name+"/sneak", pos, b)
			p.StopSneaking()
		}
		// placed sets b at a free position and returns the position.
		placed := func(b world.Block) cube.Pos {
			pos := free()
			tx.SetBlock(pos, b, nil)
			return pos
		}

		// Every machine kind, idle.
		for _, k := range machine.Order {
			pos := free()
			both("machine/"+k, pos, machine.New(k))
		}

		// The machine states that change the lines.
		state := func(kind string) (machine.Machine, *machine.State) {
			m := machine.New(kind)
			return m, m.State
		}
		m, st := state(machine.KindCrusher)
		st.Insert(item.NewStack(block.Cobblestone{}, 5))
		st.Progress = st.Def.Interval / 2
		both("state/crusher/working", free(), m)
		m, st = state(machine.KindCrusher)
		_ = st.Inv().SetItem(st.Def.Output[0], item.NewStack(block.Gravel{}, 12))
		rec("state/crusher/output", free(), m)
		m, st = state(machine.KindCrusher)
		st.Progress = st.Def.Interval
		rec("state/crusher/blocked", free(), m)
		m, st = state(machine.KindCrusher)
		st.Boost = 400
		rec("state/crusher/boost", free(), m)
		m, st = state(machine.KindCrusher)
		st.Stalled = true
		rec("state/crusher/stalled", free(), m)
		m, st = state(machine.KindCrusher)
		st.Unlimited = true
		rec("state/crusher/unlimited", free(), m)
		m, st = state(machine.KindCrafter)
		st.SetRecipe(machine.CrafterCandidates(s.Session, machine.Item(machine.KindCrusher))[0])
		both("state/crafter/missing", free(), m)
		m, st = state(machine.KindLab)
		r := machinerecipe.Labs[0]
		st.SetLab(machinerecipe.LabByKind(r.Out.Kind))
		rec("state/lab/missing", free(), m)
		fillLab(st, r, 2)
		rec("state/lab/filled", free(), m)
		m, st = state(machine.KindGenerator)
		st.Energy, st.Burn = 300, 40
		rec("state/generator/burning", free(), m)
		m, st = state(machine.KindEnergyCell)
		st.Energy = 1234
		rec("state/energy_cell/charged", free(), m)
		m, st = state(machine.KindTank)
		st.Fluid, st.FluidAmt = fluidnet.Water, 3000
		rec("state/tank/water", free(), m)
		m, st = state(machine.KindCrucible)
		st.Insert(item.NewStack(block.Cobblestone{}, 3))
		st.Fluid, st.FluidAmt = fluidnet.Lava, 500
		rec("state/crucible/melting", free(), m)
		m, st = state(machine.KindLavaGenerator)
		st.Fluid, st.FluidAmt, st.Energy = fluidnet.Lava, 2000, 77
		rec("state/lava_generator/fuelled", free(), m)
		m, st = state(machine.KindMagicSieve)
		st.Insert(item.NewStack(block.Gravel{}, 4))
		st.SetMob("zombie")
		st.Charges = 3
		rec("state/magic_sieve/charged", free(), m)
		m, st = state(machine.KindMiner)
		st.Progress = st.Def.Interval / 2
		rec("state/miner/working", free(), m)
		m, st = state(machine.KindHarvester)
		st.Progress = st.Def.Interval / 2
		rec("state/harvester/working", free(), m)

		// A drawer controller counts the drawers of its set.
		ctrl := free()
		tx.SetBlock(ctrl, machine.New(machine.KindDrawerController), nil)
		tx.SetBlock(ctrl.Side(cube.FaceEast), storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70}, nil)
		next++ // the drawer took the next column
		rec("state/drawer_controller/set", ctrl, tx.Block(ctrl))

		// The drawer.
		both("drawer/empty", free(), storage.Drawer{Facing: cube.South})
		both("drawer/dirt", free(), storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70})
		both("drawer/upgraded", free(), storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 5000, Capacity: 2, Void: true, Locked: true})

		// The composter.
		pos := placed(block.Composter{})
		rec("composter/empty", pos, tx.Block(pos))
		pos = placed(block.Composter{})
		farm.SetComposterCount(s.Session, pos, 5)
		rec("composter/plants", pos, tx.Block(pos))
		pos = placed(block.Composter{Level: farm.ComposterLevels})
		rec("composter/level", pos, tx.Block(pos))

		// The conduit bundle.
		chest := free()
		next++ // the bundle stands east of the chest
		pos = chestWithBundle(tx, chest, 0, 2, 3)
		rec("bundle/item", pos, tx.Block(pos))
		pos = placed(conduit.Bundle{Energy: 1, Fluid: 2, Net: 1})
		both("bundle/all", pos, tx.Block(pos))

		// The silkworm leaves.
		rec("leaves/none", free(), farm.SilkwormLeaves{})
		rec("leaves/full", free(), farm.SilkwormLeaves{Cocoon: farm.CocoonMax})

		// The barrels.
		pos = placed(farm.Barrel{Wood: "oak"})
		rec("barrel/empty", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: 3}
		rec("barrel/plants", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true}
		rec("barrel/water", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true, Clay: 2}
		rec("barrel/clay", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: "oak"})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{Count: farm.MatterCapacity, Water: true, Witch: true}
		rec("barrel/witch", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: farm.StoneWood})
		rec("barrel/stone", pos, tx.Block(pos))
		pos = placed(farm.Barrel{Wood: farm.StoneWood})
		fs.Barrels[farm.CropKey(pos)] = &farm.BarrelRecord{LavaMB: 400}
		rec("barrel/stoneLava", pos, tx.Block(pos))

		// The hoppers.
		for _, id := range hopper.HopperOrder {
			rec("hopper/"+id+"/empty", free(), hopper.NewFlowHopper(id, cube.FaceDown))
		}
		fh := hopper.NewFlowHopper("wood_hopper", cube.FaceDown)
		_, _ = fh.Inv.AddItem(item.NewStack(block.Dirt{}, 7))
		rec("hopper/wood_hopper/dirt", free(), fh)

		// The magic plant: a bud with its time, a bud nobody recorded, a bloom.
		pos = placed(farm.MagicPlant{Kind: "ore_iron"})
		fs.Plants[farm.CropKey(pos)] = &farm.PlantRecord{Kind: "ore_iron", ReadyAt: 90_000_000_000}
		rec("plant/bud", pos, tx.Block(pos))
		rec("plant/unrecorded", free(), farm.MagicPlant{Kind: "ore_iron"})
		rec("plant/bloom", free(), farm.MagicPlant{Kind: "ore_iron", Bloom: true})
		rec("plant/unknown", free(), farm.MagicPlant{Kind: "no_such_plant"})

		// The crops: one planted (its time), one grown, one nobody recorded.
		for _, c := range []struct {
			name string
			b    world.Block
		}{
			{"wheat", block.WheatSeeds{}}, {"carrot", block.Carrot{}}, {"potato", block.Potato{}},
			{"beetroot", block.BeetrootSeeds{}}, {"melon", block.MelonSeeds{}}, {"pumpkin", block.PumpkinSeeds{}},
		} {
			pos = free()
			tx.SetBlock(pos.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
			tx.SetBlock(pos, c.b, nil)
			farm.PlantCrop(s.Session, pos, c.name)
			rec("crop/"+c.name+"/young", pos, tx.Block(pos))
			tx.SetBlock(pos, farm.WithGrowth(c.b, farm.MaxCropStage), nil)
			rec("crop/"+c.name+"/grown", pos, tx.Block(pos))
		}
		rec("crop/wheat/unrecorded", free(), block.WheatSeeds{})

		// Blocks the game adds no lines to, and the vanilla containers.
		rec("vanilla/stone", free(), block.Stone{})
		rec("vanilla/fire", free(), block.Fire{})
		pos = placed(block.NewChest())
		_, _ = tx.Block(pos).(block.Chest).Inventory(tx, pos).AddItem(item.NewStack(block.Dirt{}, 20))
		rec("vanilla/chest", pos, tx.Block(pos))
		pos = placed(block.NewFurnace(cube.North))
		rec("vanilla/furnace", pos, tx.Block(pos))
		pos = placed(block.NewBlastFurnace(cube.North))
		rec("vanilla/blast_furnace", pos, tx.Block(pos))
		pos = placed(block.NewSmoker(cube.North))
		rec("vanilla/smoker", pos, tx.Block(pos))
		pos = placed(block.NewHopper())
		rec("vanilla/hopper", pos, tx.Block(pos))
		pos = placed(block.NewBrewingStand())
		rec("vanilla/brewing_stand", pos, tx.Block(pos))
		pos = placed(block.NewBarrel())
		rec("vanilla/barrel", pos, tx.Block(pos))
	})
	pinGolden(t, "testdata/pin/look.golden", lines)
}
