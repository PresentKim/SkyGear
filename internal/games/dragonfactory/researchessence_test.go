package dragonfactory

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// essenceCostOf is how many essences of a kind a research costs.
func essenceCostOf(r *Research, kind string) int {
	n := 0
	for _, c := range r.Cost {
		if e, ok := c.Item.(ResearchEssence); ok && e.Kind == kind {
			n += c.Count
		}
	}
	return n
}

// closureOf are the researches that have to be done before the ones given
// (and those themselves).
func closureOf(ids ...string) map[string]bool {
	done := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		if done[id] {
			return
		}
		done[id] = true
		if r, ok := researchByID(id); ok {
			for _, n := range r.Needs {
				walk(n)
			}
		}
	}
	for _, id := range ids {
		walk(id)
	}
	return done
}

// TestEssenceCostsAreInTheTree: every research of the cost table exists, has
// the essences in its cost after its materials, and comes after the
// laboratory.
func TestEssenceCostsAreInTheTree(t *testing.T) {
	for id, costs := range researchEssenceCosts {
		r, ok := researchByID(id)
		if !ok {
			t.Errorf("no research %s", id)
			continue
		}
		for kind, n := range costs {
			if got := essenceCostOf(r, kind); got != n {
				t.Errorf("%s: %d essences of %s, want %d", id, got, kind, n)
			}
		}
		if !closureOf(id)["laboratory"] {
			t.Errorf("%s costs essences but comes before the laboratory", id)
		}
	}
	for _, r := range researchTree {
		for _, kind := range researchEssenceKinds {
			if essenceCostOf(&r, kind) > 0 && researchEssenceCosts[r.ID][kind] == 0 {
				t.Errorf("%s: essence of %s that is not in the table", r.ID, kind)
			}
		}
	}
}

// TestEssencesCanBeMade: what it takes to make each essence is never behind
// a research that costs that essence (or the lab itself would be locked).
func TestEssencesCanBeMade(t *testing.T) {
	makers := map[string][]string{
		EssenceEngineering: {"laboratory", "iron"},
		EssenceAlchemy:     {"laboratory", "alloy_smelter", "tin", "nickel", "copper"},
		EssenceFarming:     {"laboratory", "farming", "harvest"},
		EssenceHunting:     {"laboratory", "mob_simulator"},
		EssenceMagic:       {"laboratory", "dimension_gate"},
	}
	for _, kind := range researchEssenceKinds {
		ids, ok := makers[kind]
		if !ok {
			t.Fatalf("no makers listed for %s", kind)
		}
		for id := range closureOf(ids...) {
			if r, _ := researchByID(id); essenceCostOf(r, kind) > 0 {
				t.Errorf("%s costs the essence of %s but is needed to make it", id, kind)
			}
		}
	}
	// the laboratory itself costs none
	lab, _ := researchByID("laboratory")
	for _, kind := range researchEssenceKinds {
		if essenceCostOf(lab, kind) > 0 {
			t.Errorf("the laboratory costs the essence of %s", kind)
		}
	}
	if !slices.Contains(lab.Needs, "generator") || !slices.Contains(lab.Needs, "glasswork") {
		t.Errorf("the laboratory needs %v", lab.Needs)
	}
}

// TestEveryEssenceIsUsed: each kind of essence is asked by some research, and
// the last project asks for all of them.
func TestEveryEssenceIsUsed(t *testing.T) {
	last, _ := researchByID("project_dragon")
	for _, kind := range researchEssenceKinds {
		used := false
		for _, r := range researchTree {
			used = used || essenceCostOf(&r, kind) > 0
		}
		if !used {
			t.Errorf("no research costs the essence of %s", kind)
		}
		if essenceCostOf(last, kind) < 32 {
			t.Errorf("the dragon project asks %d of %s", essenceCostOf(last, kind), kind)
		}
	}
}

// TestResearchPaidWithEssences: a research that costs essences takes them
// with the materials, and says what is missing when they are not all there.
func TestResearchPaidWithEssences(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	crusher, _ := researchByID("crusher")
	s.state.Researched = map[string]bool{}
	for id := range closureOf("crusher") {
		if id != "crusher" {
			s.state.Researched[id] = true
		}
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		inv := p.Inventory()
		_, _ = inv.AddItem(item.NewStack(block.Cobblestone{}, 64))
		_, _ = inv.AddItem(item.NewStack(item.IronIngot{}, 8))
		_, _ = inv.AddItem(item.NewStack(ResearchEssence{EssenceEngineering}, 3))
		missing, done := s.doResearch(p, tx, crusher)
		if done || len(missing) != 1 || missing[0].Count != 1 || missing[0].Item != (ResearchEssence{EssenceEngineering}) {
			t.Fatalf("with 3 of 4 essences: done %v, missing %v", done, missing)
		}
		_, _ = inv.AddItem(item.NewStack(ResearchEssence{EssenceEngineering}, 1))
		if missing, done := s.doResearch(p, tx, crusher); !done || len(missing) != 0 {
			t.Fatalf("with every essence: done %v, missing %v", done, missing)
		}
		if got := countPlain(inv, ResearchEssence{EssenceEngineering}); got != 0 {
			t.Errorf("%d essences left", got)
		}
		if !s.advanced(advEssenceResearch) {
			t.Error("the first research paid with essences did not finish its advancement")
		}
	})
}
