package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func shopItem(id string) ShopItem {
	for _, it := range ShopItems {
		if it.ID == id {
			return it
		}
	}
	panic(id)
}

func TestShopStockAndRestock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.money[p.UUID().String()] = 10000
		box := shopItem("delivery_box")
		if !s.buy(p, tx, box) || !s.buy(p, tx, box) {
			t.Fatal("could not buy the stock")
		}
		if s.buy(p, tx, box) {
			t.Fatal("bought beyond the stock")
		}
		if s.Money(p) != 10000-2*box.Price {
			t.Fatalf("money = %d", s.Money(p))
		}
		if s.buy(p, tx, shopItem("hopper")) {
			t.Fatal("bought a product above the save level")
		}
		s.clock.Advance(s.clock.DayLength())
		if s.stock(box).Count != box.MaxStock {
			t.Fatal("stock was not refilled after a day")
		}

		s.state.Level = 2
		once := shopItem("pioneer_pickaxe")
		if !s.buy(p, tx, once) || s.buy(p, tx, once) {
			t.Fatal("one-time product must sell exactly once")
		}
		s.clock.Advance(10 * s.clock.DayLength())
		if s.stock(once).Count != 0 {
			t.Fatal("one-time product was refilled")
		}
	})
}

func TestBuybackBudgetAndLimits(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if n := s.sell(p, item.NewStack(block.Cobblestone{}, 64)); n != 64 {
			t.Fatalf("sold %d cobblestone", n)
		}
		for range 10 {
			s.sell(p, item.NewStack(block.Cobblestone{}, 64))
		}
		if got := s.buyback().Sold[itemKey(block.Cobblestone{})]; got != 512 {
			t.Fatalf("cobblestone sold %d, want the daily limit 512", got)
		}
		if s.sell(p, item.NewStack(item.Apple{}, 1)) != 0 {
			t.Fatal("the shop bought an item it does not trade")
		}
		// Exhaust the budget with charcoal and glass.
		for range 50 {
			s.sell(p, item.NewStack(block.Glass{}, 64))
			s.sell(p, item.NewStack(item.Charcoal{}, 64))
			s.sell(p, item.NewStack(block.Log{Wood: block.OakWood()}, 64))
			s.sell(p, item.NewStack(block.Stone{}, 64))
		}
		if s.buyback().Budget < 0 || s.Money(p) > buybackBudget(1) {
			t.Fatalf("budget %d, earned %d > daily budget", s.buyback().Budget, s.Money(p))
		}
		s.clock.Advance(s.clock.DayLength())
		if s.buyback().Budget != buybackBudget(1) || s.buyback().Sold[itemKey(block.Cobblestone{})] != 0 {
			t.Fatal("buyback was not restored on a new day")
		}
	})
}

func TestShopItemsAreValid(t *testing.T) {
	seen := map[string]bool{}
	for _, it := range ShopItems {
		if seen[it.ID] || it.Stack().Empty() || it.Price < 0 || (it.Price == 0 && it.ID != "core_t1_free") || it.MaxStock <= 0 {
			t.Errorf("invalid shop item %+v", it.ID)
		}
		seen[it.ID] = true
		if it.Restock == 0 && it.MaxStock != 1 {
			t.Errorf("one-time product %s must have stock 1", it.ID)
		}
	}
}

// TestLostStartCoreCanBeReplaced: one tier 1 core a day is free, so a
// save without money that lost its start core can go on.
func TestLostStartCoreCanBeReplaced(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.money[p.UUID().String()] = 0
		if !s.buy(p, tx, shopItem("core_t1_free")) {
			t.Fatal("could not take the free tier 1 core")
		}
		if !p.Inventory().ContainsItem(coreItem("t1")) {
			t.Fatal("no tier 1 core in the inventory")
		}
	})
}

// TestSellCountAndQuote: selling a chosen number takes only that many, and
// the sale quote has a line per item with its price.
func TestSellCountAndQuote(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, _ = p.Inventory().AddItem(item.NewStack(block.Cobblestone{}, 40))
		_, _ = p.Inventory().AddItem(item.NewStack(block.Dirt{}, 5))
		lines, total := s.quoteSale(p)
		if len(lines) != 2 || lines[0].Count != 40 || lines[0].Price != 1 || lines[0].Icon == "" || total != 40+5 {
			t.Fatalf("quote %+v total %d", lines, total)
		}
		s.sellCount(p, itemKey(block.Cobblestone{}), 25)
		if !p.Inventory().ContainsItem(item.NewStack(block.Cobblestone{}, 15)) || p.Inventory().ContainsItem(item.NewStack(block.Cobblestone{}, 16)) {
			t.Fatal("did not sell exactly 25")
		}
	})
}
