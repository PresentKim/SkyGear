package skygear

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/hud"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/screens"
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

func TestGiveItems(t *testing.T) {
	items := giveItems()
	for _, id := range []string{"drawer", "upgrade_capacity", "research_table", machine.KindMiner, machine.KindCrusher} {
		if f, ok := items[id]; !ok || f().Empty() {
			t.Errorf("give item %q missing", id)
		}
	}
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
