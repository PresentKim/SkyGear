package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestAdvancementTreeIsValid: ids are unique, parents come before their
// children, items exist and researches are in the research tree.
func TestAdvancementTreeIsValid(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range advancementTree {
		if seen[a.ID] {
			t.Errorf("%s twice", a.ID)
		}
		if a.Parent != "" && !seen[a.Parent] {
			t.Errorf("%s: parent %s not before it", a.ID, a.Parent)
		}
		seen[a.ID] = true
		if a.Hint == "" {
			t.Errorf("%s: no hint", a.ID)
		}
		for _, n := range []string{a.Item, a.Icon} {
			if _, ok := world.ItemByName(unlockName(n), 0); n != "" && !ok {
				t.Errorf("%s: no item %s", a.ID, n)
			}
		}
		if _, ok := researchByID(a.Research); a.Research != "" && !ok {
			t.Errorf("%s: no research %s", a.ID, a.Research)
		}
		if a.Item == "" && a.Research == "" {
			switch a.ID {
			case advGrowTree, advSift, advMachine, advChunk:
			default:
				t.Errorf("%s: nothing completes it", a.ID)
			}
		}
	}
}

// TestAdvancementsFromItems: having an item completes its advancement and
// the ones before it; events complete theirs.
func TestAdvancementsFromItems(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if a, _ := s.nextAdvancement(); a.ID != "wood" {
			t.Fatalf("first advancement %s", a.ID)
		}
		_, _ = p.Inventory().AddItem(item.NewStack(block.CraftingTable{}, 1))
		s.checkAdvancements(tx)
		for _, id := range []string{"wood", "planks", "crafting_table"} {
			if !s.advanced(id) {
				t.Errorf("%s not done", id)
			}
		}
		if a, _ := s.nextAdvancement(); a.ID != "research_table" {
			t.Errorf("next advancement %s", a.ID)
		}
		s.advance(tx, advSift)
		if !s.advanced(advSift) || !s.advanced("sieve") || !s.advanced("string") || !s.advanced("first_research") {
			t.Errorf("sift did not complete its ancestors: %v", s.state.Advancements)
		}
		if s.advanced(advGrowTree) {
			t.Error("grow tree done without growing")
		}
	})
}

// TestAdvancementMenuHidesLocked: done ones are green, open ones show
// their hint and the rest are hidden.
func TestAdvancementMenuHidesLocked(t *testing.T) {
	s := testSession(t)
	s.state.Advancements = map[string]bool{"wood": true}
	_, opts := s.advancementMenu()
	if len(opts) != len(advancementTree)+1 {
		t.Fatalf("%d options", len(opts))
	}
	want := map[string]string{"wood": "§a✔ 나무 캐기", "planks": "§e판자\n§7원목으로 참나무 판자를 만드세요", "crafting_table": "§8???"}
	for i, a := range advancementTree {
		if w, ok := want[a.ID]; ok && opts[i].Label != w {
			t.Errorf("%s: %q", a.ID, opts[i].Label)
		}
	}
}
