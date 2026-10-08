package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/chestui"
)

// click clicks a menu slot the way a session does and lets the button run.
func click(t *testing.T, s *session, h *world.EntityHandle, m *chestui.Menu, slot int) {
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
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
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) { m = s.lostMenu(p) })
	if got, _ := m.Inventory().Item(0); got.Count() != 64 {
		t.Fatalf("first entry icon = %v, want 64 apples", got)
	}
	click(t, s, h, m, 0)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
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
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.money[p.UUID().String()] = 10000
		m = s.shopMenu(p)
	})
	click(t, s, h, m, 0) // delivery box
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if !p.Inventory().ContainsItem(item.NewStack(DeliveryBox{}, 1)) {
			t.Fatal("delivery box not bought")
		}
		if s.Money(p) != 10000-shopItem("delivery_box").Price {
			t.Fatalf("money = %d", s.Money(p))
		}
	})
}

func TestExpandMap(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // at (0.5, 64, 2.5): chunk (0, 0)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		m := s.expandMenu(p, ChunkPos{0, 0})
		centre := (mapRows/2)*mapCols + mapCols/2
		if it, _ := m.Inventory().Item(centre); itemKey(it.Item()) != itemKey(item.Compass{}) {
			t.Fatalf("centre slot = %v, want the compass for the player's chunk", it)
		}
		// (1, 0) is east of the starting 2×2 island: buyable.
		if it, _ := m.Inventory().Item(centre + 1); !strings.Contains(it.CustomName(), "구매 가능") {
			t.Fatalf("east chunk = %q", it.CustomName())
		}
		// (-1, -1) is owned.
		if it, _ := m.Inventory().Item(centre - mapCols - 1); !strings.Contains(it.CustomName(), "소유") {
			t.Fatalf("north-west chunk = %q", it.CustomName())
		}
		// (3, 0) is not next to the island.
		if it, _ := m.Inventory().Item(centre + 3); strings.Contains(it.CustomName(), "구매") {
			t.Fatalf("far chunk = %q", it.CustomName())
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
