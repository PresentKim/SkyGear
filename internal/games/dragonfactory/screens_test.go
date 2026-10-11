package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
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
		_, opts := s.mainMenu(p)
		for _, o := range opts {
			if o.Icon == "" {
				t.Errorf("option %q has no icon", o.Label)
			}
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
		_, opts := s.lostMenu()
		if len(opts) != 3 || !strings.Contains(opts[0].Label, "100") {
			t.Fatalf("options = %+v", opts)
		}
		press(t, p, tx, opts, "apple")
		if !p.Inventory().ContainsItem(item.NewStack(item.Apple{}, 100)) {
			t.Fatal("apples were not taken")
		}
		if len(s.state.Lost) != 1 {
			t.Fatalf("lost = %d entries, want 1", len(s.state.Lost))
		}
	})
}

func TestShopMenuBuys(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.money[p.UUID().String()] = 10000
		_, opts := s.shopMenu(p, 2) // 설비
		if opts[0].Icon != iconOf(item.NewStack(DeliveryBox{}, 1)) || !strings.Contains(opts[0].Label, "재고 2") || strings.Contains(opts[0].Label, "/") {
			t.Errorf("first product %q icon %q", opts[0].Label, opts[0].Icon)
		}
		// The tabs and the back button come last, marked for the theme.
		tabs := 0
		for _, o := range opts {
			if strings.HasPrefix(o.Label, ui.TabToken) || strings.HasPrefix(o.Label, ui.TabOnToken) {
				tabs++
			}
		}
		if tabs != len(shopCategories) || !strings.HasPrefix(opts[len(opts)-1].Label, ui.BackToken) {
			t.Fatalf("%d tabs, last option %q", tabs, opts[len(opts)-1].Label)
		}
		press(t, p, tx, opts, "납품함")
		if !p.Inventory().ContainsItem(item.NewStack(DeliveryBox{}, 1)) {
			t.Fatal("delivery box not bought")
		}
		if s.Money(p) != 10000-shopItem("delivery_box").Price {
			t.Fatalf("money = %d", s.Money(p))
		}
	})
}

func TestShopCategoriesCoverAllProducts(t *testing.T) {
	tabs := map[string]bool{}
	for _, c := range shopCategories {
		tabs[c.Name] = true
	}
	for _, it := range ShopItems {
		if !tabs[shopGroup(it)] {
			t.Errorf("%s is in no tab (%q)", it.ID, shopGroup(it))
		}
	}
	if shopGroup(shopItem("crusher")) != "기계" || shopGroup(shopItem("core_t1")) != "코어" || shopGroup(shopItem("mat_cobblestone")) != "재료" || shopGroup(shopItem("hammer")) != "도구" {
		t.Fatal("a product is in the wrong tab")
	}
}

func TestExpansionMode(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // at (0.5, 64, 2.5): chunk (0, 0)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{1, expandFloorY, 1}, block.Grass{}, nil)
		tx.SetBlock(cube.Pos{0, expandFloorY, 0}, block.Grass{}, nil)
		s.money[p.UUID().String()] = 5000
		s.expandView = viewFloor
		s.showExpand(p)
		e, ok := s.expansionOf(p)
		if !ok {
			t.Fatal("not in expansion mode")
		}
		// (1, 0) is next to the island: yellow floor. The selected chunk
		// (0, 0) is owned: light blue on its air positions and over its whole
		// border; its inner blocks are left alone.
		if b := e.floor[cube.Pos{20, expandFloorY, 5}]; b != world.Block(floorBuyable) {
			t.Fatalf("buyable floor = %v", b)
		}
		if _, ok := e.floor[cube.Pos{1, expandFloorY, 1}]; ok {
			t.Fatal("fake floor over a real island block inside the chunk")
		}
		if b := e.floor[cube.Pos{0, expandFloorY, 0}]; b != world.Block(floorTargetOwned) {
			t.Fatalf("selected chunk border = %v", b)
		}
		if b := e.floor[cube.Pos{10, expandFloorY, 10}]; b != world.Block(floorTargetOwned) {
			t.Fatalf("selected owned floor = %v", b)
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
		if e.target != (ChunkPos{1, 0}) || e.floor[cube.Pos{20, expandFloorY, 5}] != world.Block(floorTarget) {
			t.Fatalf("target = %v, floor %v", e.target, e.floor[cube.Pos{20, expandFloorY, 5}])
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
		s.drawFloor(p, e)
		if e.floor[cube.Pos{20, expandFloorY, 5}] != world.Block(floorTargetOwned) {
			t.Fatal("bought selected chunk not shown as owned")
		}
		s.expandInput(p, e, mgl64.Vec2{})
		s.expandInput(p, e, mgl64.Vec2{1, 0}) // back west to (0, 0)
		if b := e.floor[cube.Pos{20, expandFloorY, 5}]; b != world.Block(floorOwnedAlt) {
			t.Fatalf("owned chunk (1, 0) floor = %v, want the other green of the checkerboard", b)
		}
		if b := e.floor[cube.Pos{-10, expandFloorY, -10}]; b != world.Block(floorOwned) {
			t.Fatalf("owned chunk (-1, -1) floor = %v, want the first green", b)
		}
		if b := e.floor[cube.Pos{36, expandFloorY, 5}]; b != world.Block(floorBuyable) {
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
	for _, id := range []string{"drawer", "upgrade_capacity", "delivery_box", KindMiner, KindCrusher} {
		if f, ok := items[id]; !ok || f().Empty() {
			t.Errorf("give item %q missing", id)
		}
	}
}

func TestSellAllQuoteSellsNothing(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_ = p.Inventory().SetItem(0, item.NewStack(item.Flint{}, 10))
		_ = p.Inventory().SetItem(1, item.NewStack(item.Flint{}, 5))
		_ = p.Inventory().SetItem(2, item.NewStack(item.Coal{}, 2))
		before := s.Money(p)
		lines, total := s.quoteSale(p)
		if len(lines) != 2 || total != 15*4+2*8 {
			t.Fatalf("quote %v total %d", lines, total)
		}
		if s.Money(p) != before || !p.Inventory().ContainsItem(item.NewStack(item.Flint{}, 15)) {
			t.Fatal("quoting sold something")
		}
		s.sellAll(p, func(item.Stack) bool { return true })
		if s.Money(p)-before != total {
			t.Fatalf("sold for %d, quoted %d", s.Money(p)-before, total)
		}
	})
}

func TestExpansionCursorStaysNearBody(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.expandView = viewFloor // the map lets the cursor go farther
		s.showExpand(p)
		e, _ := s.expansionOf(p)
		for range expandReach + 3 {
			s.expandInput(p, e, mgl64.Vec2{})
			s.expandInput(p, e, mgl64.Vec2{-1, 0}) // east
		}
		if e.target[0] != e.body[0]+expandReach {
			t.Fatalf("cursor at %v, body %v", e.target, e.body)
		}
		for pos := range e.floor {
			if c := chunkOf(pos); abs32(c[0]-e.target[0]) > expandFloorReach {
				t.Fatalf("floor drawn at chunk %v, far from the cursor %v", c, e.target)
			}
		}
		s.stopExpand(p, tx)
	})
}

// TestShopTabsHaveIcons: every shop tab has an icon. Item icons made while
// package variables were initialised were empty (runtime IDs come later).
func TestShopTabsHaveIcons(t *testing.T) {
	for _, c := range shopCategories {
		if c.icon() == "" {
			t.Errorf("shop tab %s has no icon", c.Name)
		}
	}
}

// TestExpansionMap: the map view sends owned chunks far away with their top
// blocks flattened under glass, lets the cursor leave the body's reach and
// takes the map down when expansion ends.
func TestExpansionMap(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{3, expandFloorY + 7, 3}, block.Stone{}, nil)
		s.showExpand(p) // the map is the default view
		e, _ := s.expansionOf(p)
		if e.mapShown[ChunkPos{0, 0}] != world.Block(floorTargetOwned) || e.mapShown[ChunkPos{1, 0}] != world.Block(floorBuyable) || len(e.floor) != 0 {
			t.Fatalf("map chunks %v, %d fake blocks", e.mapShown, len(e.floor))
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
		for range expandReach + 2 {
			s.expandInput(p, e, mgl64.Vec2{})
			s.expandInput(p, e, mgl64.Vec2{-1, 0})
		}
		if e.target[0] <= e.body[0]+expandReach {
			t.Fatalf("cursor stopped at %v on the map", e.target)
		}
		s.stopExpand(p, tx)
		if e.mapShown != nil || p.Position() != home {
			t.Fatalf("after leaving: map %v, body at %v, want %v", e.mapShown, p.Position(), home)
		}
	})
}
