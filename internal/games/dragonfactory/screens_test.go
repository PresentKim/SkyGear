package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/ui"
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

// TestMainMenuLayout: eight cards, the codex first and the settings last (no
// menu item card: /메뉴발급 gives it), the help button and the lobby exit in
// the back button's place.
func TestMainMenuLayout(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		opts := s.mainMenu(p)
		if len(opts) != 10 {
			t.Fatalf("%d options, want 8 cards, the help button and the lobby exit", len(opts))
		}
		if !strings.Contains(opts[0].Label, "도감") {
			t.Errorf("first card %q is not the codex", opts[0].Label)
		}
		if !strings.Contains(opts[7].Label, "설정") {
			t.Errorf("last card %q is not the settings", opts[7].Label)
		}
		for _, o := range opts[:8] {
			if strings.HasPrefix(o.Label, "§0§") || strings.Contains(o.Label, "메뉴 아이템") || strings.Contains(o.Label, "연구") {
				t.Errorf("card %q", o.Label)
			}
		}
		if help := opts[8]; !strings.HasPrefix(help.Label, ui.HelpToken) {
			t.Errorf("option %q is not the help button", help.Label)
		}
		if last := opts[9]; !strings.HasPrefix(last.Label, ui.BackToken) || !strings.Contains(last.Label, "로비로") {
			t.Errorf("last option %q is not the lobby exit", last.Label)
		}
	})
}

func TestLostMenu(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	s.addLost(item.NewStack(item.Diamond{}, 3))
	s.addLost(item.NewStack(item.Apple{}, 100))
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, slots, _ := s.lostMenu(i18n.Default)
		if len(slots) != 2 || slots[0].Count != "100" || !strings.Contains(slots[0].Tip, "100") {
			t.Fatalf("slots = %+v", slots)
		}
		slots[0].Run(p, tx)
		if !p.Inventory().ContainsItem(item.NewStack(item.Apple{}, 100)) {
			t.Fatal("apples were not taken")
		}
		if len(s.state.Lost) != 1 {
			t.Fatalf("lost = %d entries, want 1", len(s.state.Lost))
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
		// (1, 0) is next to the island: yellow glass. The selected chunk
		// (0, 0) is owned: light blue.
		if b := e.mapShown[ChunkPos{1, 0}]; b != world.Block(floorBuyable) {
			t.Fatalf("buyable chunk = %v", b)
		}
		if b := e.mapShown[ChunkPos{0, 0}]; b != world.Block(floorTargetOwned) {
			t.Fatalf("selected owned chunk = %v", b)
		}
		if _, ok := p.GameMode().(viewOnlyMode); !ok {
			t.Fatalf("game mode %T in expansion mode", p.GameMode())
		}
		// The body is frozen; the right key (X -1) moves the cursor east to
		// (1, 0).
		if !p.Immobile() {
			t.Fatal("player can still move")
		}
		move := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleMove(move, mgl64.Vec3{3, 64, 3}, cube.Rotation{})
		if !move.Cancelled() {
			t.Fatal("movement not cancelled in expansion mode")
		}
		s.expandInput(p, e, mgl64.Vec2{-1, 0})
		if e.target != (ChunkPos{1, 0}) || e.mapShown[ChunkPos{1, 0}] != world.Block(floorTarget) {
			t.Fatalf("target = %v, glass %v", e.target, e.mapShown[ChunkPos{1, 0}])
		}
		// A key held on does not move again at once; a new press does.
		s.expandInput(p, e, mgl64.Vec2{-1, 0})
		if e.target != (ChunkPos{1, 0}) {
			t.Fatalf("held key moved at once to %v", e.target)
		}
		s.expandInput(p, e, mgl64.Vec2{})
		s.expandInput(p, e, mgl64.Vec2{0, 1}) // forward: north
		if e.target != (ChunkPos{1, -1}) {
			t.Fatalf("forward moved to %v", e.target)
		}
		s.expandInput(p, e, mgl64.Vec2{})
		s.expandInput(p, e, mgl64.Vec2{0, -1})
		// Building is off.
		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleBlockPlace(ctx, cube.Pos{1, 64, 1}, block.Dirt{})
		if !ctx.Cancelled() {
			t.Fatal("placed a block in expansion mode")
		}
		s.buyChunk(p, tx, ChunkPos{1, 0})
		s.drawMap(p, e, true)
		if e.mapShown[ChunkPos{1, 0}] != world.Block(floorTargetOwned) {
			t.Fatal("bought selected chunk not shown as owned")
		}
		s.expandInput(p, e, mgl64.Vec2{})
		s.expandInput(p, e, mgl64.Vec2{1, 0}) // back west to (0, 0)
		if b := e.mapShown[ChunkPos{1, 0}]; b != world.Block(floorOwnedAlt) {
			t.Fatalf("owned chunk (1, 0) floor = %v, want the other green of the checkerboard", b)
		}
		if b := e.mapShown[ChunkPos{2, 0}]; b != world.Block(floorBuyable) {
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
	for _, id := range []string{"drawer", "upgrade_capacity", "research_table", KindMiner, KindCrusher} {
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
		if e.mapShown[ChunkPos{0, 0}] != world.Block(floorTargetOwned) || e.mapShown[ChunkPos{1, 0}] != world.Block(floorBuyable) {
			t.Fatalf("map chunks %v", e.mapShown)
		}
		col := s.mapColumn(tx, ChunkPos{0, 0}, floorTargetOwned)
		reg := tx.World().BlockRegistry()
		at := func(x, y, z int) world.Block {
			b, _ := reg.BlockByRuntimeID(col.Block(uint8(x), int16(y), uint8(z), 0))
			return b
		}
		// A full block alone just under the surface, the glass on it.
		if at(3, expandFloorY-1, 3) != (block.Stone{}) || !isAir(at(3, expandFloorY-2, 3)) {
			t.Fatalf("map column = %v over %v, want stone alone", at(3, expandFloorY-1, 3), at(3, expandFloorY-2, 3))
		}
		if at(3, expandFloorY, 3) != world.Block(floorTargetOwned) {
			t.Fatalf("map glass = %v", at(3, expandFloorY, 3))
		}
		// A torch shows the block it stands on under it.
		tx.SetBlock(cube.Pos{5, expandFloorY + 3, 5}, block.Planks{}, nil)
		tx.SetBlock(cube.Pos{5, expandFloorY + 4, 5}, block.Torch{Facing: cube.FaceDown}, nil)
		col = s.mapColumn(tx, ChunkPos{0, 0}, floorTargetOwned)
		if _, ok := at(5, expandFloorY-1, 5).(block.Torch); !ok || at(5, expandFloorY-2, 5) != (block.Planks{}) {
			t.Fatalf("torch column = %v over %v", at(5, expandFloorY-1, 5), at(5, expandFloorY-2, 5))
		}
		home := e.home
		cam := s.expandCamera(e)
		if cam[0] < mapOffset*16 || p.Position() != cam.Add(mgl64.Vec3{0, mapBodyAbove, 0}) {
			t.Fatalf("camera %v, body %v: not over the map", cam, p.Position())
		}
		for range expandMapReach + 4 {
			s.expandInput(p, e, mgl64.Vec2{})
			s.expandInput(p, e, mgl64.Vec2{-1, 0})
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

// TestHelpTopicsAreWritten: every help topic has a title and a text, and the
// help names only things the game has.
func TestHelpTopicsAreWritten(t *testing.T) {
	for _, h := range helpTopics {
		title, text := i18n.T(i18n.Default, "help."+h.id+".title"), i18n.T(i18n.Default, "help."+h.id+".text")
		if title == "" || text == "" || strings.HasPrefix(title, "help.") {
			t.Errorf("help %s: title %q text %q", h.id, title, text)
		}
		for _, gone := range []string{"코어", "상점", "수표", "납품"} {
			if strings.Contains(text, gone) {
				t.Errorf("help %s mentions %q", h.id, gone)
			}
		}
	}
}
