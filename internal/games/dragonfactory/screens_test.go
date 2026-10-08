package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/chestui"
)

// click clicks a menu slot the way a session does and lets the button run.
func click(s *session, h *world.EntityHandle, m *chestui.Menu, slot int) {
	withPlayer(s, h, func(p *player.Player, tx *world.Tx) {
		ctx := event.C(inventory.Holder(p))
		m.Inventory().Handler().HandleTake(ctx, slot, item.Stack{})
	})
	tick(s, 1)
}

func TestLostMenu(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	s.addLost(item.NewStack(item.Diamond{}, 3))
	s.addLost(item.NewStack(item.Apple{}, 100))

	var m *chestui.Menu
	withPlayer(s, h, func(p *player.Player, tx *world.Tx) { m = s.lostMenu(p) })
	if got, _ := m.Inventory().Item(0); got.Count() != 64 {
		t.Fatalf("first entry icon = %v, want 64 apples", got)
	}
	click(s, h, m, 0)
	withPlayer(s, h, func(p *player.Player, tx *world.Tx) {
		if !p.Inventory().ContainsItem(item.NewStack(item.Apple{}, 100)) {
			t.Fatal("apples were not taken")
		}
	})
	if len(s.state.Lost) != 1 {
		t.Fatalf("lost = %d entries, want 1", len(s.state.Lost))
	}
	if got, _ := m.Inventory().Item(0); got.Count() != 3 {
		t.Fatalf("menu not refreshed: slot 0 = %v", got)
	}
}

func TestShopMenuBuys(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	var m *chestui.Menu
	withPlayer(s, h, func(p *player.Player, tx *world.Tx) {
		s.money[p.UUID().String()] = 10000
		m = s.shopMenu(p)
	})
	click(s, h, m, 0) // delivery box
	withPlayer(s, h, func(p *player.Player, tx *world.Tx) {
		if !p.Inventory().ContainsItem(item.NewStack(DeliveryBox{}, 1)) {
			t.Fatal("delivery box not bought")
		}
		if s.Money(p) != 10000-shopItem("delivery_box").Price {
			t.Fatalf("money = %d", s.Money(p))
		}
	})
}
