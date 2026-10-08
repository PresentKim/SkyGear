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
		_, opts := s.shopMenu(p)
		if opts[0].Icon != "textures/items/delivery_box" {
			t.Errorf("delivery box icon = %q", opts[0].Icon)
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

func TestExpansionMode(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // at (0.5, 64, 2.5): chunk (0, 0)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, expandFloorY, 0}, block.Grass{}, nil)
		s.money[p.UUID().String()] = 5000
		s.showExpand(p)
		e, ok := s.expansionOf(p)
		if !ok {
			t.Fatal("not in expansion mode")
		}
		// (1, 0) is next to the island: yellow floor. The island's own air
		// positions get a green floor; its blocks are left alone.
		if b := e.floor[cube.Pos{20, expandFloorY, 5}]; b != world.Block(floorBuyable) {
			t.Fatalf("buyable floor = %v", b)
		}
		if _, ok := e.floor[cube.Pos{0, expandFloorY, 0}]; ok {
			t.Fatal("fake floor over a real island block")
		}
		if b := e.floor[cube.Pos{10, expandFloorY, 10}]; b != world.Block(floorOwned) {
			t.Fatalf("owned floor = %v", b)
		}
		// Walking into (1, 0) targets it.
		s.expandMove(p, mgl64.Vec3{20.5, 64, 5.5})
		if e.target != (ChunkPos{1, 0}) || e.floor[cube.Pos{20, expandFloorY, 5}] != world.Block(floorTarget) {
			t.Fatalf("target = %v, floor %v", e.target, e.floor[cube.Pos{20, expandFloorY, 5}])
		}
		// Building is off.
		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleBlockPlace(ctx, cube.Pos{1, 64, 1}, block.Dirt{})
		if !ctx.Cancelled() {
			t.Fatal("placed a block in expansion mode")
		}
		s.buyChunk(p, tx, ChunkPos{1, 0})
		s.drawFloor(p, e)
		if e.floor[cube.Pos{20, expandFloorY, 5}] != world.Block(floorOwned) {
			t.Fatal("bought chunk not shown as owned")
		}
		if b := e.floor[cube.Pos{36, expandFloorY, 5}]; b != world.Block(floorBuyable) {
			t.Fatalf("chunk (2, 0) not buyable after buying (1, 0): %v", b)
		}
		// Leaving while over the void puts the player back on the island.
		p.Teleport(mgl64.Vec3{40.5, 64, 5.5})
		s.stopExpand(p, tx)
		if _, ok := s.expansionOf(p); ok {
			t.Fatal("still in expansion mode")
		}
		if p.Position() != s.Spawn() {
			t.Fatalf("player left over the void at %v", p.Position())
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
