package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/waila"
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
	target, crop := cube.Pos{4, IslandY + 3, 8}, cube.Pos{8, IslandY + 3, 8}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(target, block.Stone{}, nil)
		tx.SetBlock(crop.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		tx.SetBlock(crop, block.WheatSeeds{}, nil)
		pos, rot := eyeAt(p, target, 3)
		if at, b, ok := lookedAt(p, tx, pos, rot); !ok || at != target {
			t.Fatalf("looked at %v %T, want stone at %v", at, b, target)
		}
		// A young crop is low: aim near its bottom.
		pos, rot = eyeAt(p, crop, 2)
		pos = pos.Sub(mgl64.Vec3{0, 0.45, 0})
		if at, _, ok := lookedAt(p, tx, pos, rot); !ok || at != crop {
			t.Fatalf("looked at %v, want the crop at %v", at, crop)
		}
		// Over a young (low) crop the view goes on to the block behind it.
		behind := crop.Add(cube.Pos{0, 0, 1})
		tx.SetBlock(behind, block.Stone{}, nil)
		pos, rot = eyeAt(p, behind, 2)
		pos = pos.Add(mgl64.Vec3{0, 0.3, 0})
		if at, _, ok := lookedAt(p, tx, pos, rot); !ok || at != behind {
			t.Fatalf("over a young crop looked at %v, want the block behind at %v", at, behind)
		}
		pos, rot = eyeAt(p, target, lookReach+2)
		if _, b, ok := lookedAt(p, tx, pos, rot); ok {
			t.Fatalf("found %T beyond the reach", b)
		}
	})
}

// TestLookInfo: the panel lines of the game's blocks.
func TestLookInfo(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	st := placeTestMachine(s, machinePos, KindCrusher, "")
	st.insert(item.NewStack(block.Cobblestone{}, 5))
	tick(s, machineKinds[KindCrusher].interval/2+1)
	drawerPos, corePos := cube.Pos{6, IslandY, 3}, cube.Pos{9, IslandY + 2, 9}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		text := func(pos cube.Pos) string {
			info := s.lookInfo(p, tx, pos, tx.Block(pos))
			title, _ := waila.Format(info)
			return title
		}
		if got := text(machinePos); !strings.Contains(got, "tile.dragonfactory:crusher.name") || strings.Contains(got, "드래곤팩토리") {
			t.Errorf("crusher panel %q", got)
		}
		if info := s.lookInfo(p, tx, machinePos, tx.Block(machinePos)); info.Line == nil || info.Line.Name != "조약돌" || !info.Line.ShowGauge {
			t.Errorf("crusher line %+v, want the cobblestone with a gauge", info.Line)
		}
		tx.SetBlock(drawerPos, Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70, Locked: true}, nil)
		if info := s.lookInfo(p, tx, drawerPos, tx.Block(drawerPos)); info.Line == nil || info.Line.Name != "흙" || info.Line.Text != "§f×70" || info.Marks != (waila.Marks{}) || info.Container != nil {
			t.Errorf("drawer panel %+v, want a dirt line with its count and no marks", info)
		} else if _, ok := info.Item.(Drawer); !ok {
			t.Errorf("drawer icon %v, want the drawer", info.Item)
		}
		p.StartSneaking()
		if info := s.lookInfo(p, tx, drawerPos, tx.Block(drawerPos)); len(info.Lines) < 2 || !strings.Contains(info.Lines[len(info.Lines)-1], "잠김") {
			t.Errorf("sneaking drawer lines %q, want the upgrades and the lock", info.Lines)
		}
		p.StopSneaking()
		tx.SetBlock(corePos, Core{Type: "t1"}, nil)
		if got := text(corePos); !strings.Contains(got, "더 이상 쓰이지 않음") {
			t.Errorf("old core panel %q", got)
		}
		chestPos := cube.Pos{11, IslandY, 3}
		tx.SetBlock(chestPos, block.NewChest(), nil)
		if c, ok := tx.Block(chestPos).(block.Chest); ok {
			_, _ = c.Inventory(tx, chestPos).AddItem(item.NewStack(block.Dirt{}, 20))
		}
		if title, _ := waila.Format(s.lookInfo(p, tx, chestPos, tx.Block(chestPos))); !strings.Contains(title, "__waila:inv.chest__") {
			t.Errorf("chest panel %q has no contents", title)
		}
		if info := s.lookInfo(p, tx, machinePos, tx.Block(machinePos)); info.Container != nil {
			t.Errorf("crusher panel has slots: %+v", info.Container)
		}
		stone := cube.Pos{1, IslandY + 4, 1}
		tx.SetBlock(stone, block.Stone{}, nil)
		info := s.lookInfo(p, tx, stone, block.Stone{})
		if info.Name != "돌" || info.Source != "" || len(info.Tools) == 0 || info.Item == nil {
			t.Errorf("stone panel %+v", info)
		}
	})
}

// TestFurnaceArrow: an unlit furnace shows an empty arrow.
func TestFurnaceArrow(t *testing.T) {
	it := furnaceArrow(block.NewFurnace(cube.North))
	if a, ok := it.Item().(ProgressArrow); !ok || a.Step != 0 {
		t.Errorf("unlit furnace arrow %v, want an empty progress arrow", it)
	}
	if !furnaceArrow(block.Stone{}).Empty() {
		t.Errorf("stone has a furnace arrow")
	}
}

// TestCustomTool: our blocks show a fitting tool, marked when held.
func TestCustomTool(t *testing.T) {
	if got := customTool(Drawer{}, item.Stack{}); len(got) != 1 || got[0].Tool != waila.AxeWood || got[0].Mark != waila.MarkNone {
		t.Errorf("drawer by hand: %v", got)
	}
	if got := customTool(Core{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1)); len(got) != 1 || got[0].Mark != waila.MarkRight {
		t.Errorf("core with a pickaxe: %v", got)
	}
	if got := customTool(block.Stone{}, item.Stack{}); got != nil {
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
		Machine{Kind: KindSieve}:                         "tile.dragonfactory:sieve.name",
	} {
		if got := blockNameKey(b); got != want {
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
	st := placeTestMachine(s, machinePos, KindCrusher, "")
	st.insert(item.NewStack(block.Cobblestone{}, 5))
	tick(s, machineKinds[KindCrusher].interval/2+1)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		plain := len(s.lookInfo(p, tx, machinePos, tx.Block(machinePos)).Lines)
		p.StartSneaking()
		defer p.StopSneaking()
		info := s.lookInfo(p, tx, machinePos, tx.Block(machinePos))
		if len(info.Lines) <= plain || !strings.Contains(strings.Join(info.Lines, "\n"), "주기") {
			t.Errorf("sneaking lines %q, plain %d", info.Lines, plain)
		}
	})
}
