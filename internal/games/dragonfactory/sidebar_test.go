package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// TestHUDQuestFollowsAdvancements: the quest panel shows the next
// advancement and its hint, with the research cost for a research
// advancement, then the next research once every advancement is done.
func TestHUDQuestFollowsAdvancements(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if title, rows := s.questRows(); !strings.Contains(title, "주먹이 곧 도끼") || len(rows) != 2 {
			t.Fatalf("quest %q %q", title, rows)
		}
		s.advance(tx, "research_table")
		title, rows := s.questRows()
		if !strings.Contains(title, "유레카") || len(rows) != 3 || !strings.Contains(rows[2], "×4") {
			t.Fatalf("quest %q %q", title, rows)
		}
		sticks, _ := researchByID("stick")
		_, _ = p.Inventory().AddItem(item.NewStack(oakPlanks, 16))
		s.doResearch(p, tx, sticks)
		s.checkAdvancements(tx)
		if title, _ := s.questRows(); !strings.Contains(title, "돌멩이") {
			t.Fatalf("quest after sticks %q", title)
		}
		for _, a := range advancementTree {
			s.state.Advancements[a.ID] = true
		}
		if title, _ := s.questRows(); !strings.Contains(title, "돌 다지기") {
			t.Fatalf("quest after every advancement %q", title)
		}
	})
}

// TestQuestShowsResearchStep: an advancement that waits on a research
// behind others shows the research to do now with its cost.
func TestQuestShowsResearchStep(t *testing.T) {
	s := testSession(t)
	a, _ := advancementByID("alloy_smelter")
	g, ok := goalResearch(a)
	if !ok || g.ID != "alloy_smelter" {
		t.Fatalf("goal %v %v", g, ok)
	}
	step, ok := s.researchStep(g)
	if !ok || step.ID == g.ID || !s.researchOpen(step) {
		t.Fatalf("step %v %v", step, ok)
	}
	s.state.Advancements = map[string]bool{}
	for _, x := range advancementTree {
		if x.ID != a.ID {
			s.state.Advancements[x.ID] = true
		}
	}
	_, rows := s.questRows()
	if len(rows) < 4 || !strings.Contains(rows[2], step.Title(i18n.Default)) {
		t.Fatalf("rows %q", rows)
	}
}
