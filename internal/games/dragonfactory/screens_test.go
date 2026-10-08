package dragonfactory

import (
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

func TestExpandMap(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // at (0.5, 64, 2.5): chunk (0, 0)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, opts := s.expandMenu(p, ChunkPos{0, 0})
		if len(opts) != mapRows*mapCols+6 || len(opts) > 54 {
			t.Fatalf("%d options do not fit the 9×6 grid", len(opts))
		}
		centre := (mapRows/2)*mapCols + mapCols/2
		if opts[centre].Icon != iconLobby {
			t.Fatalf("centre = %+v, want the compass for the player's chunk", opts[centre])
		}
		// (1, 0) is east of the starting 2×2 island: buyable.
		if opts[centre+1].Icon != "textures/blocks/glass_yellow" {
			t.Fatalf("east chunk = %+v", opts[centre+1])
		}
		// (-1, -1) is owned.
		if opts[centre-mapCols-1].Icon != "textures/blocks/glass_lime" {
			t.Fatalf("north-west chunk = %+v", opts[centre-mapCols-1])
		}
		// (3, 0) is not next to the island.
		if opts[centre+3].Icon != "textures/blocks/glass_gray" {
			t.Fatalf("far chunk = %+v", opts[centre+3])
		}
		seen := map[string]bool{}
		for _, o := range opts {
			if seen[o.Label] {
				t.Fatalf("label %q twice: form buttons are matched by label", o.Label)
			}
			seen[o.Label] = true
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
