package skygear

import (
	"github.com/presentkim/skygear/content/skygear/screens"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/expand"

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
		opts := screens.MainMenu(s.Session, p)
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
		_, slots, _ := screens.LostMenu(s.Session, i18n.Default)
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
		expand.Show(s.Session, p)
		e, ok := expand.Of(s.Session, p)
		if !ok {
			t.Fatal("not in expansion mode")
		}
		if b := e.MapShown[island.ChunkPos{1, 0}]; b != world.Block(expand.FloorBuyable) {
			t.Fatalf("buyable chunk = %v", b)
		}
		if b := e.MapShown[island.ChunkPos{0, 0}]; b != world.Block(expand.FloorOwned) {
			t.Fatalf("selected owned chunk = %v", b)
		}
		if _, ok := p.GameMode().(expand.ViewOnlyMode); !ok {
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
		expand.Step(s.Session, 1, 0)(p, tx)
		if e.Target != (island.ChunkPos{1, 0}) || e.MapShown[island.ChunkPos{1, 0}] != world.Block(expand.FloorBuyable) {
			t.Fatalf("target = %v, glass %v", e.Target, e.MapShown[island.ChunkPos{1, 0}])
		}
		expand.Step(s.Session, 0, -1)(p, tx) // up: north
		if e.Target != (island.ChunkPos{1, -1}) {
			t.Fatalf("up moved to %v", e.Target)
		}
		expand.Step(s.Session, 0, 1)(p, tx)
		// Building is off.
		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleBlockPlace(ctx, cube.Pos{1, 64, 1}, block.Dirt{})
		if !ctx.Cancelled() {
			t.Fatal("placed a block in expansion mode")
		}
		island.Buy(s.Session, p, tx, island.ChunkPos{1, 0})
		expand.DrawMap(s.Session, p, e, true)
		if b := e.MapShown[island.ChunkPos{1, 0}]; b != world.Block(expand.FloorOwnedAlt) {
			t.Fatalf("owned chunk (1, 0) floor = %v, want the other green of the checkerboard", b)
		}
		expand.Step(s.Session, -1, 0)(p, tx) // back west to (0, 0)
		if b := e.MapShown[island.ChunkPos{2, 0}]; b != world.Block(expand.FloorBuyable) {
			t.Fatalf("chunk (2, 0) not buyable after buying (1, 0): %v", b)
		}
		expand.Stop(s.Session, p, tx)
		if _, ok := expand.Of(s.Session, p); ok {
			t.Fatal("still in expansion mode")
		}
		if _, ok := p.GameMode().(expand.ViewOnlyMode); ok {
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
		tx.SetBlock(cube.Pos{3, expand.ExpandFloorY + 7, 3}, block.Stone{}, nil)
		expand.Show(s.Session, p)
		e, _ := expand.Of(s.Session, p)
		if e.MapShown[island.ChunkPos{0, 0}] != world.Block(expand.FloorOwned) || e.MapShown[island.ChunkPos{1, 0}] != world.Block(expand.FloorBuyable) {
			t.Fatalf("map chunks %v", e.MapShown)
		}
		col := expand.MapColumn(s.Session, tx, island.ChunkPos{0, 0}, expand.FloorOwned)
		reg := tx.World().BlockRegistry()
		at := func(x, y, z int) world.Block {
			b, _ := reg.BlockByRuntimeID(col.Block(uint8(x), int16(y), uint8(z), 0))
			return b
		}
		// A full block alone just under the surface, the glass on it.
		if at(3, expand.ExpandFloorY-1, 3) != (block.Stone{}) || !expand.IsAir(at(3, expand.ExpandFloorY-2, 3)) {
			t.Fatalf("map column = %v over %v, want stone alone", at(3, expand.ExpandFloorY-1, 3), at(3, expand.ExpandFloorY-2, 3))
		}
		if at(3, expand.ExpandFloorY, 3) != world.Block(expand.FloorOwned) {
			t.Fatalf("map glass = %v", at(3, expand.ExpandFloorY, 3))
		}
		// A torch shows the block it stands on under it.
		tx.SetBlock(cube.Pos{5, expand.ExpandFloorY + 3, 5}, block.Planks{}, nil)
		tx.SetBlock(cube.Pos{5, expand.ExpandFloorY + 4, 5}, block.Torch{Facing: cube.FaceDown}, nil)
		col = expand.MapColumn(s.Session, tx, island.ChunkPos{0, 0}, expand.FloorOwned)
		if _, ok := at(5, expand.ExpandFloorY-1, 5).(block.Torch); !ok || at(5, expand.ExpandFloorY-2, 5) != (block.Planks{}) {
			t.Fatalf("torch column = %v over %v", at(5, expand.ExpandFloorY-1, 5), at(5, expand.ExpandFloorY-2, 5))
		}
		home := e.Home
		cam := expand.Camera(s.Session, e)
		if cam[0] < expand.MapOffset*16 || p.Position() != cam.Add(mgl64.Vec3{0, expand.MapBodyAbove, 0}) {
			t.Fatalf("camera %v, body %v: not over the map", cam, p.Position())
		}
		for range expand.ExpandMapReach + 4 {
			expand.Step(s.Session, 1, 0)(p, tx)
		}
		if e.Target[0] < e.Body[0]+expand.ExpandMapReach+4 {
			t.Fatalf("cursor stopped at %v on the map", e.Target)
		}
		if want := expand.Camera(s.Session, e).Add(mgl64.Vec3{0, expand.MapBodyAbove, 0}); p.Position() != want {
			t.Fatalf("body at %v, want over the cursor %v", p.Position(), want)
		}
		expand.Stop(s.Session, p, tx)
		if e.MapShown != nil || p.Position() != home {
			t.Fatalf("after leaving: map %v, body at %v, want %v", e.MapShown, p.Position(), home)
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
		expand.Show(s.Session, p)
		e, ok := expand.Of(s.Session, p)
		if !ok {
			t.Fatal("not in expansion mode")
		}
		if strings.Contains(screens.Boards.Of(s.Session)[p.H()], hud.QuestTitle) {
			t.Error("the quest panel shows in expansion mode")
		}
		if got := expand.ChunkLabel(s.Session, i18n.Default, island.ChunkPos{1, 0}, island.Purchasable(s.Session)); !strings.Contains(got, "구매 가능") {
			t.Errorf("label of the chunk beside the island %q", got)
		}
		if got := expand.ChunkLabel(s.Session, i18n.Default, island.ChunkPos{5, 5}, island.Purchasable(s.Session)); !strings.Contains(got, "연결된 땅 없음") {
			t.Errorf("label of a chunk away from the island %q", got)
		}
		from := e.Target
		expand.Step(s.Session, 1, 0)(p, tx)
		if want := (island.ChunkPos{from[0] + 1, from[1]}); e.Target != want {
			t.Errorf("the right key moved the selection to %v, want %v", e.Target, want)
		}
		expand.Stop(s.Session, p, tx)
		if !strings.Contains(screens.Boards.Of(s.Session)[p.H()], hud.QuestTitle) {
			t.Error("the quest panel did not come back")
		}
	})
}
