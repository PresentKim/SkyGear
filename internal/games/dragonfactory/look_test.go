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
		pos, rot = eyeAt(p, crop, 2)
		if at, _, ok := lookedAt(p, tx, pos, rot); !ok || at != crop {
			t.Fatalf("looked at %v, want the crop at %v", at, crop)
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
		if got := text(machinePos); !strings.Contains(got, "분쇄기") || !strings.Contains(got, "진행") || !strings.Contains(got, "드래곤팩토리") {
			t.Errorf("crusher panel %q", got)
		}
		tx.SetBlock(drawerPos, Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 70, Locked: true}, nil)
		if got := text(drawerPos); !strings.Contains(got, "흙") || !strings.Contains(got, "70") || !strings.Contains(got, "잠김") {
			t.Errorf("drawer panel %q", got)
		}
		placeCore(tx, corePos, "t1", block.Cobblestone{})
		if got := text(corePos); !strings.Contains(got, "만드는 것") || !strings.Contains(got, "조약돌") {
			t.Errorf("core panel %q", got)
		}
		tx.SetBlock(corePos.Side(cube.FaceDown), block.Iron{}, nil)
		if got := text(corePos); !strings.Contains(got, "3티어 이상") {
			t.Errorf("core on iron panel %q", got)
		}
		stone := cube.Pos{1, IslandY + 4, 1}
		tx.SetBlock(stone, block.Stone{}, nil)
		info := s.lookInfo(p, tx, stone, block.Stone{})
		if info.Name != "돌" || info.Source != "Minecraft" || len(info.Tools) == 0 || info.Item == nil {
			t.Errorf("stone panel %+v", info)
		}
	})
}
