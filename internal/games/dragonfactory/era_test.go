package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestBuyLifeCore(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		var product ShopItem
		for _, it := range ShopItems {
			if it.ID == "life_core" {
				product = it
			}
		}
		s.money[p.UUID().String()] = 40000
		if s.buy(p, tx, product) {
			t.Fatal("bought the life core before Lv.8")
		}
		s.state.Level = 8
		if !s.buy(p, tx, product) {
			t.Fatal("could not buy the life core at Lv.8")
		}
		rec := s.state.Cores["life"]
		if rec == nil || rec.Placed || rec.Generation != 1 {
			t.Fatalf("life core record %+v", rec)
		}
		if !p.Inventory().ContainsItemFunc(1, func(st item.Stack) bool {
			typ, gen, ok := coreItemGeneration(st)
			return ok && typ == "life" && gen == 1
		}) {
			t.Fatal("no life core item")
		}
		if s.Money(p) != 40000-product.Price {
			t.Fatalf("money %d", s.Money(p))
		}
		// It is a one-time product; a save that has the core is refunded.
		s.stock(product).Count = 1
		if s.buy(p, tx, product) || s.Money(p) != 40000-product.Price {
			t.Fatalf("bought a second life core, money %d", s.Money(p))
		}
	})
}
