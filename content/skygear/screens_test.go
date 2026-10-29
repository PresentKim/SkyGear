package skygear

import (
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/hud"
	"github.com/presentkim/skygear/common/ui"
)

// press runs the option with the given label prefix, as a form submit does.
func press(t *testing.T, p *player.Player, tx *world.Tx, opts []ui.Option, prefix string) {
	t.Helper()
	for _, o := range opts {
		if strings.HasPrefix(o.Label, prefix) {
			o.Run(p, tx)
			return
		}
	}
	t.Fatalf("no option %q", prefix)
}

func TestMainMenuHasIcons(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		opts := s.mainMenu(p)
		for _, o := range opts {
			if o.Icon == "" && !strings.HasPrefix(o.Label, ui.HelpToken) {
				t.Errorf("option %q has no icon", o.Label)
			}
		}
	})
}

func TestLostMenu(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	island.AddLost(s.Session, item.NewStack(item.Diamond{}, 3))
	island.AddLost(s.Session, item.NewStack(item.Apple{}, 100))
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, slots, _ := s.lostMenu(i18n.Default)
		if len(slots) != 2 || slots[0].Count != "100" || !strings.Contains(slots[0].Tip, "100") {
			t.Fatalf("slots = %+v", slots)
		}
		slots[0].Run(p, tx)
		if !p.Inventory().ContainsItem(item.NewStack(item.Apple{}, 100)) {
			t.Fatal("apples were not taken")
		}
		if len(island.StateOf(s.Session).Lost) != 1 {
			t.Fatalf("lost = %d entries, want 1", len(island.StateOf(s.Session).Lost))
		}
	})
}

func TestExpansionMode(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // at (0.5, 64, 2.5): chunk (0, 0)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, _ = p.Inventory().AddItem(item.NewStack(block.Cobblestone{}, 64))
		s.showExpand(p)
		e, ok := s.expansionOf(p)
		if !ok {
			t.Fatal("not in expansion mode")
		}
		if b := e.mapShown[island.ChunkPos{1, 0}]; b != world.Block(floorBuyable) {
			t.Fatalf("buyable chunk = %v", b)
		}
		if b := e.mapShown[island.ChunkPos{0, 0}]; b != world.Block(floorOwned) {
			t.Fatalf("selected owned chunk = %v", b)
		}
		if _, ok := p.GameMode().(viewOnlyMode); !ok {
			t.Fatalf("game mode %T in expansion mode", p.GameMode())
		}
		// The body is frozen; the right key moves the cursor east to (1, 0).
		if !p.Immobile() {
			t.Fatal("player can still move")
		}
		move := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleMove(move, mgl64.Vec3{3, 64, 3}, cube.Rotation{})
		if !move.Cancelled() {
			t.Fatal("movement not cancelled in expansion mode")
		}
		s.expandStep(1, 0)(p, tx)
		if e.target != (island.ChunkPos{1, 0}) || e.mapShown[island.ChunkPos{1, 0}] != world.Block(floorBuyable) {
			t.Fatalf("target = %v, glass %v", e.target, e.mapShown[island.ChunkPos{1, 0}])
		}
		s.expandStep(0, -1)(p, tx) // up: north
		if e.target != (island.ChunkPos{1, -1}) {
			t.Fatalf("up moved to %v", e.target)
		}
		s.expandStep(0, 1)(p, tx)
		// Building is off.
		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleBlockPlace(ctx, cube.Pos{1, 64, 1}, block.Dirt{})
		if !ctx.Cancelled() {
			t.Fatal("placed a block in expansion mode")
		}
		island.Buy(s.Session, p, tx, island.ChunkPos{1, 0})
		s.drawMap(p, e, true)
		if b := e.mapShown[island.ChunkPos{1, 0}]; b != world.Block(floorOwnedAlt) {
			t.Fatalf("owned chunk (1, 0) floor = %v, want the other green of the checkerboard", b)
		}
		s.expandStep(-1, 0)(p, tx) // back west to (0, 0)
		if b := e.mapShown[island.ChunkPos{2, 0}]; b != world.Block(floorBuyable) {
			t.Fatalf("chunk (2, 0) not buyable after buying (1, 0): %v", b)
		}
		s.stopExpand(p, tx)
		if _, ok := s.expansionOf(p); ok {
			t.Fatal("still in expansion mode")
		}
		if _, ok := p.GameMode().(viewOnlyMode); ok {
			t.Fatal("game mode not restored")
		}
		if p.Immobile() {
			t.Fatal("player still frozen after leaving")
		}
	})
}

func TestGiveItems(t *testing.T) {
	items := giveItems()
	for _, id := range []string{"drawer", "upgrade_capacity", "research_table", machine.KindMiner, machine.KindCrusher} {
		if f, ok := items[id]; !ok || f().Empty() {
			t.Errorf("give item %q missing", id)
		}
	}
}

// TestExpansionMap: the map sends owned chunks far away with their top
// blocks flattened under glass, the body follows the cursor over the map,
// and the map comes down when expansion ends.
func TestExpansionMap(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{3, expandFloorY + 7, 3}, block.Stone{}, nil)
		s.showExpand(p)
		e, _ := s.expansionOf(p)
		if e.mapShown[island.ChunkPos{0, 0}] != world.Block(floorOwned) || e.mapShown[island.ChunkPos{1, 0}] != world.Block(floorBuyable) {
			t.Fatalf("map chunks %v", e.mapShown)
		}
		col := s.mapColumn(tx, island.ChunkPos{0, 0}, floorOwned)
		reg := tx.World().BlockRegistry()
		at := func(x, y, z int) world.Block {
			b, _ := reg.BlockByRuntimeID(col.Block(uint8(x), int16(y), uint8(z), 0))
			return b
		}
		// A full block alone just under the surface, the glass on it.
		if at(3, expandFloorY-1, 3) != (block.Stone{}) || !isAir(at(3, expandFloorY-2, 3)) {
			t.Fatalf("map column = %v over %v, want stone alone", at(3, expandFloorY-1, 3), at(3, expandFloorY-2, 3))
		}
		if at(3, expandFloorY, 3) != world.Block(floorOwned) {
			t.Fatalf("map glass = %v", at(3, expandFloorY, 3))
		}
		// A torch shows the block it stands on under it.
		tx.SetBlock(cube.Pos{5, expandFloorY + 3, 5}, block.Planks{}, nil)
		tx.SetBlock(cube.Pos{5, expandFloorY + 4, 5}, block.Torch{Facing: cube.FaceDown}, nil)
		col = s.mapColumn(tx, island.ChunkPos{0, 0}, floorOwned)
		if _, ok := at(5, expandFloorY-1, 5).(block.Torch); !ok || at(5, expandFloorY-2, 5) != (block.Planks{}) {
			t.Fatalf("torch column = %v over %v", at(5, expandFloorY-1, 5), at(5, expandFloorY-2, 5))
		}
		home := e.home
		cam := s.expandCamera(e)
		if cam[0] < mapOffset*16 || p.Position() != cam.Add(mgl64.Vec3{0, mapBodyAbove, 0}) {
			t.Fatalf("camera %v, body %v: not over the map", cam, p.Position())
		}
		for range expandMapReach + 4 {
			s.expandStep(1, 0)(p, tx)
		}
		if e.target[0] < e.body[0]+expandMapReach+4 {
			t.Fatalf("cursor stopped at %v on the map", e.target)
		}
		if want := s.expandCamera(e).Add(mgl64.Vec3{0, mapBodyAbove, 0}); p.Position() != want {
			t.Fatalf("body at %v, want over the cursor %v", p.Position(), want)
		}
		s.stopExpand(p, tx)
		if e.mapShown != nil || p.Position() != home {
			t.Fatalf("after leaving: map %v, body at %v, want %v", e.mapShown, p.Position(), home)
		}
	})
}

// TestExpansionPlane: the expansion screen shows with no quest panel
// meanwhile, and its arrow keys move the selection; leaving the mode brings
// the quest panel back.
func TestExpansionPlane(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.showExpand(p)
		e, ok := s.expansionOf(p)
		if !ok {
			t.Fatal("not in expansion mode")
		}
		if strings.Contains(s.boards[p.H()], hud.QuestTitle) {
			t.Error("the quest panel shows in expansion mode")
		}
		if got := s.chunkLabel(i18n.Default, island.ChunkPos{1, 0}, island.Purchasable(s.Session)); !strings.Contains(got, "구매 가능") {
			t.Errorf("label of the chunk beside the island %q", got)
		}
		if got := s.chunkLabel(i18n.Default, island.ChunkPos{5, 5}, island.Purchasable(s.Session)); !strings.Contains(got, "연결된 땅 없음") {
			t.Errorf("label of a chunk away from the island %q", got)
		}
		from := e.target
		s.expandStep(1, 0)(p, tx)
		if want := (island.ChunkPos{from[0] + 1, from[1]}); e.target != want {
			t.Errorf("the right key moved the selection to %v, want %v", e.target, want)
		}
		s.stopExpand(p, tx)
		if !strings.Contains(s.boards[p.H()], hud.QuestTitle) {
			t.Error("the quest panel did not come back")
		}
	})
}
