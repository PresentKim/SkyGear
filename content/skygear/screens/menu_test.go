package screens_test

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/screens"
)

func TestMainMenuHasIcons(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		opts := screens.MainMenu(s.Session, p)
		for _, o := range opts {
			if o.Icon == "" && !strings.HasPrefix(o.Label, ui.HelpToken) {
				t.Errorf("option %q has no icon", o.Label)
			}
		}
	})
}

func TestLostMenu(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	island.AddLost(s.Session, item.NewStack(item.Diamond{}, 3))
	island.AddLost(s.Session, item.NewStack(item.Apple{}, 100))
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
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
