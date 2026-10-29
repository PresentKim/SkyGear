package skygear

import (
	"github.com/presentkim/skygear/content/skygear/look"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/farm"

	"github.com/presentkim/skygear/content/skygear/storage"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// eyeAt returns a player position whose eyes are dist blocks north of the
// centre of target, looking south at it.
func eyeAt(p *player.Player, target cube.Pos, dist float64) (mgl64.Vec3, cube.Rotation) {
	c := target.Vec3Centre()
	return mgl64.Vec3{c[0], c[1] - p.EyeHeight(), c[2] - dist}, cube.Rotation{0, 0}
}

// TestLookedAt finds the block in view, also crops without a collision box,
// and nothing past the reach.
func TestLookedAt(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	target, crop := cube.Pos{4, island.IslandY + 3, 8}, cube.Pos{8, island.IslandY + 3, 8}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(target, block.Stone{}, nil)
		tx.SetBlock(crop.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		tx.SetBlock(crop, block.WheatSeeds{}, nil)
		pos, rot := eyeAt(p, target, 3)
		if at, b, ok := look.LookedAt(p, tx, pos, rot); !ok || at != target {
			t.Fatalf("looked at %v %T, want stone at %v", at, b, target)
		}
		// A young crop is low: aim near its bottom.
		pos, rot = eyeAt(p, crop, 2)
		pos = pos.Sub(mgl64.Vec3{0, 0.45, 0})
		if at, _, ok := look.LookedAt(p, tx, pos, rot); !ok || at != crop {
			t.Fatalf("looked at %v, want the crop at %v", at, crop)
		}
		// Over a young (low) crop the view goes on to the block behind it.
		behind := crop.Add(cube.Pos{0, 0, 1})
		tx.SetBlock(behind, block.Stone{}, nil)
		pos, rot = eyeAt(p, behind, 2)
		pos = pos.Add(mgl64.Vec3{0, 0.3, 0})
		if at, _, ok := look.LookedAt(p, tx, pos, rot); !ok || at != behind {
			t.Fatalf("over a young crop looked at %v, want the block behind at %v", at, behind)
		}
		pos, rot = eyeAt(p, target, look.Reach+2)
		if _, b, ok := look.LookedAt(p, tx, pos, rot); ok {
			t.Fatalf("found %T beyond the reach", b)
		}
	})
}

// TestLookedAtPlacedBarrel: a barrel placed by hand stays in view, over the
// fire that heats it.
func TestLookedAtPlacedBarrel(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	target := cube.Pos{2, island.IslandY + 3, 5}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		below := target.Side(cube.FaceDown)
		tx.SetBlock(below.Side(cube.FaceDown), block.Netherrack{}, nil)
		tx.SetBlock(below, block.Stone{}, nil)
		held, off := p.HeldItems()
		p.SetHeldItems(item.NewStack(farm.Barrel{Wood: farm.StoneWood}, 1), off)
		p.UseItemOnBlock(below, cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
		p.SetHeldItems(held, off)
		if _, ok := tx.Block(target).(farm.Barrel); !ok {
			t.Fatalf("placed %T, want a barrel", tx.Block(target))
		}
		tx.SetBlock(below, block.Fire{}, nil)
		pos, rot := eyeAt(p, target, 3)
		if at, b, ok := look.LookedAt(p, tx, pos, rot); !ok || at != target {
			t.Fatalf("looked at %v %T, want the barrel at %v", at, b, target)
		}
	})
}

// TestLookInfo: the panel lines of the game's blocks.
func TestLookInfo(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
	st.Insert(item.NewStack(block.Cobblestone{}, 5))
	tick(s, machine.Kinds[machine.KindCrusher].Interval/2+1)
	drawerPos := cube.Pos{6, island.IslandY, 3}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		text := func(pos cube.Pos) string {
			info := look.Info(s.Session, p, tx, pos, tx.Block(pos))
			title, _ := waila.Format(info)
			return title
		}
		if got := text(machinePos); !strings.Contains(got, "tile.skygear:crusher.name") || strings.Contains(got, "스카이기어") {
			t.Errorf("crusher panel %q", got)
		}
		if info := look.Info(s.Session, p, tx, machinePos, tx.Block(machinePos)); info.Line == nil || info.Line.Name != "조약돌" || !info.Line.ShowGauge {
			t.Errorf("crusher line %+v, want the cobblestone with a gauge", info.Line)
		}
		tx.SetBlock(drawerPos, storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70, Locked: true}, nil)
		if info := look.Info(s.Session, p, tx, drawerPos, tx.Block(drawerPos)); info.Container == nil || len(info.Container.Items) != 1 ||
			info.Container.Items[0].Count() != 70 || info.Line != nil || info.Marks != (waila.Marks{}) {
			t.Errorf("drawer panel %+v, want the dirt in a slot with its count, as a chest shows it, and no marks", info)
		} else if _, ok := info.Item.(storage.Drawer); !ok {
			t.Errorf("drawer icon %v, want the drawer", info.Item)
		}
		p.StartSneaking()
		if info := look.Info(s.Session, p, tx, drawerPos, tx.Block(drawerPos)); len(info.Lines) < 2 || !strings.Contains(info.Lines[len(info.Lines)-1], "잠김") {
			t.Errorf("sneaking drawer lines %q, want the upgrades and the lock", info.Lines)
		}
		p.StopSneaking()
		chestPos := cube.Pos{11, island.IslandY, 3}
		tx.SetBlock(chestPos, block.NewChest(), nil)
		if c, ok := tx.Block(chestPos).(block.Chest); ok {
			_, _ = c.Inventory(tx, chestPos).AddItem(item.NewStack(block.Dirt{}, 20))
		}
		if title, _ := waila.Format(look.Info(s.Session, p, tx, chestPos, tx.Block(chestPos))); !strings.Contains(title, "__waila:inv.chest__") {
			t.Errorf("chest panel %q has no contents", title)
		}
		if info := look.Info(s.Session, p, tx, machinePos, tx.Block(machinePos)); info.Container != nil {
			t.Errorf("crusher panel has slots: %+v", info.Container)
		}
		stone := cube.Pos{1, island.IslandY + 4, 1}
		tx.SetBlock(stone, block.Stone{}, nil)
		info := look.Info(s.Session, p, tx, stone, block.Stone{})
		if info.Name != "돌" || info.Source != "" || len(info.Tools) == 0 || info.Item == nil {
			t.Errorf("stone panel %+v", info)
		}
		fire := stone.Side(cube.FaceUp)
		tx.SetBlock(fire, block.Fire{}, nil)
		if info := look.Info(s.Session, p, tx, fire, block.Fire{}); info.Item != (item.FlintAndSteel{}) || info.NameKey != "tile.fire.name" {
			t.Errorf("fire panel %+v, want flint and steel as the icon", info)
		}
	})
}

// TestFurnaceArrow: an unlit furnace shows an empty arrow.
func TestFurnaceArrow(t *testing.T) {
	it := look.FurnaceArrow(block.NewFurnace(cube.North))
	if a, ok := it.Item().(machine.ProgressArrow); !ok || a.Step != 0 {
		t.Errorf("unlit furnace arrow %v, want an empty progress arrow", it)
	}
	if !look.FurnaceArrow(block.Stone{}).Empty() {
		t.Errorf("stone has a furnace arrow")
	}
}

// TestCustomTool: our blocks show a fitting tool, marked when held.
func TestCustomTool(t *testing.T) {
	if got := look.CustomTool(storage.Drawer{}, item.Stack{}); len(got) != 1 || got[0].Tool != waila.AxeWood || got[0].Mark != waila.MarkNone {
		t.Errorf("drawer by hand: %v", got)
	}
	if got := look.CustomTool(machine.Machine{Kind: machine.KindCrusher}, item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1)); len(got) != 1 || got[0].Mark != waila.MarkRight {
		t.Errorf("machine with a pickaxe: %v", got)
	}
	if got := look.CustomTool(block.Stone{}, item.Stack{}); got != nil {
		t.Errorf("stone: %v", got)
	}
}

// TestBlockNameKeys: the client names vanilla blocks by the official keys,
// modern IDs by their legacy ones (the oak sapling, leaves, log and planks
// were shown by their identifier).
func TestBlockNameKeys(t *testing.T) {
	for b, want := range map[world.Block]string{
		block.Planks{Wood: block.OakWood()}:              "tile.planks.oak.name",
		block.Leaves{Type: block.OakLeaves()}:            "tile.leaves.oak.name",
		block.Sapling{Type: block.OakSapling()}:          "tile.sapling.oak.name",
		block.Log{Wood: block.OakWood()}:                 "tile.log.oak.name",
		block.Log{Wood: block.OakWood(), Stripped: true}: "tile.stripped_oak_log.name",
		block.CraftingTable{}:                            "tile.crafting_table.name",
		block.Grass{}:                                    "tile.grass.name",
		block.Water{Still: true, Depth: 8}:               "tile.water.name",
		block.Lava{Still: true, Depth: 8}:                "tile.lava.name",
		machine.Machine{Kind: machine.KindSieve}:         "tile.skygear:sieve.name",
	} {
		if got := naming.BlockNameKey(b); got != want {
			t.Errorf("%T: %s, want %s", b, got, want)
		}
	}
}

// TestLookDetailWhileSneaking: a sneaking viewer gets the cycle and energy
// figures under a machine's lines.
func TestLookDetailWhileSneaking(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	st := placeTestMachine(s, machinePos, machine.KindCrusher, "")
	st.Insert(item.NewStack(block.Cobblestone{}, 5))
	tick(s, machine.Kinds[machine.KindCrusher].Interval/2+1)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		plain := len(look.Info(s.Session, p, tx, machinePos, tx.Block(machinePos)).Lines)
		p.StartSneaking()
		defer p.StopSneaking()
		info := look.Info(s.Session, p, tx, machinePos, tx.Block(machinePos))
		if len(info.Lines) <= plain || !strings.Contains(strings.Join(info.Lines, "\n"), "주기") {
			t.Errorf("sneaking lines %q, plain %d", info.Lines, plain)
		}
	})
}
