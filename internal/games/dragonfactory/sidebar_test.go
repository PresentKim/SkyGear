package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/hud"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/account"
)

func TestCostRowsCountTheInventory(t *testing.T) {
	s := testSession(t)
	s.state.Researched = map[string]bool{}
	r, _ := researchByID("chest")
	inv := inventory.New(36, nil)
	_, _ = inv.AddItem(item.NewStack(oakPlanks, 10))
	rows := s.researchCostRows(i18n.Default, r, inv)
	if len(rows) != 1 || !strings.Contains(rows[0], "10§7/32") {
		t.Fatalf("rows %q", rows)
	}
	_, _ = inv.AddItem(item.NewStack(oakPlanks, 40))
	if rows := s.researchCostRows(i18n.Default, r, inv); !strings.Contains(rows[0], "§a32§7/32") {
		t.Fatalf("enough planks %q", rows)
	}
}

// TestHUDQuestFollowsAdvancements: the quest panel shows the next
// advancement and its hint, with the research cost for a research
// advancement, then the next research once every advancement is done.
func TestHUDQuestFollowsAdvancements(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if title, rows := s.questRows(i18n.Default, nil); !strings.Contains(title, "주먹이 곧 도끼") || len(rows) != 2 {
			t.Fatalf("quest %q %q", title, rows)
		}
		rt, _ := advancementByID("research_table")
		for _, a := range advancementWithAncestors(rt) {
			s.advance(tx, a.ID)
		}
		title, rows := s.questRows(i18n.Default, nil)
		if !strings.Contains(title, "유레카") || len(rows) != 3 || !strings.Contains(rows[2], "×4") {
			t.Fatalf("quest %q %q", title, rows)
		}
		sticks, _ := researchByID("stick")
		_, _ = p.Inventory().AddItem(item.NewStack(oakPlanks, 16))
		s.doResearch(p, tx, sticks)
		s.checkAdvancementsHeld(tx, inventoryNames(tx))
		if title, _ := s.questRows(i18n.Default, nil); !strings.Contains(title, "돌멩이") {
			t.Fatalf("quest after sticks %q", title)
		}
		for _, a := range advancementTree {
			s.state.Advancements[a.ID] = true
		}
		if title, _ := s.questRows(i18n.Default, nil); !strings.Contains(title, "돌 다지기") {
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
	_, rows := s.questRows(i18n.Default, nil)
	hint := len(strings.Split(a.HintText(i18n.Default), "\n"))
	if len(rows) < hint+2 || !strings.Contains(rows[hint], step.Title(i18n.Default)) {
		t.Fatalf("rows %q", rows)
	}
}

// TestQuestPanelSetting: a player who turns the quest panel off sees no
// quest rows on the HUD.
func TestQuestPanelSetting(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.updateBoard(p)
		if !strings.Contains(s.boards[p.H()], hud.QuestTitle) {
			t.Fatalf("no quest panel by default: %q", s.boards[p.H()])
		}
		if on, _ := account.Toggle(p, settingQuestPanel); on {
			t.Fatal("the quest panel setting did not turn off")
		}
		defer func() { _, _ = account.Toggle(p, settingQuestPanel) }()
		s.updateBoard(p)
		if b := s.boards[p.H()]; strings.Contains(b, hud.QuestTitle) || strings.Contains(b, hud.Quest) {
			t.Fatalf("the quest panel shows while off: %q", b)
		}
	})
}
