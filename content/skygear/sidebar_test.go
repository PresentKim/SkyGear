package skygear

import (
	"github.com/presentkim/skygear/content/skygear/screens"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/catalog"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/hud"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/platform/account"
)

func TestCostRowsCountTheInventory(t *testing.T) {
	s := testSession(t)
	progress.StateOf(s.Session).Researched = map[string]bool{}
	r, _ := progress.ByID("chest")
	inv := inventory.New(36, nil)
	_, _ = inv.AddItem(item.NewStack(catalog.OakPlanks, 10))
	rows := screens.ResearchCostRows(s.Session, i18n.Default, r, inv)
	if len(rows) != 1 || !strings.Contains(rows[0], "10§7/32") {
		t.Fatalf("rows %q", rows)
	}
	_, _ = inv.AddItem(item.NewStack(catalog.OakPlanks, 40))
	if rows := screens.ResearchCostRows(s.Session, i18n.Default, r, inv); !strings.Contains(rows[0], "§a32§7/32") {
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
		if title, rows := screens.QuestRows(s.Session, i18n.Default, nil); !strings.Contains(title, "주먹이 곧 도끼") || len(rows) != 2 {
			t.Fatalf("quest %q %q", title, rows)
		}
		rt, _ := advancement.ByID("research_table")
		for _, a := range advancement.WithAncestors(rt) {
			progress.Advance(s.Session, tx, a.ID)
		}
		title, rows := screens.QuestRows(s.Session, i18n.Default, nil)
		if !strings.Contains(title, "유레카") || len(rows) != 3 || !strings.Contains(rows[2], "×4") {
			t.Fatalf("quest %q %q", title, rows)
		}
		sticks, _ := progress.ByID("stick")
		_, _ = p.Inventory().AddItem(item.NewStack(catalog.OakPlanks, 16))
		progress.Do(s.Session, p, tx, sticks)
		progress.CheckHeld(s.Session, tx, advancement.InventoryNames(tx))
		if title, _ := screens.QuestRows(s.Session, i18n.Default, nil); !strings.Contains(title, "돌멩이") {
			t.Fatalf("quest after sticks %q", title)
		}
		for _, a := range advancement.Tree {
			progress.StateOf(s.Session).Advancements[a.ID] = true
		}
		if title, _ := screens.QuestRows(s.Session, i18n.Default, nil); !strings.Contains(title, "돌 다지기") {
			t.Fatalf("quest after every advancement %q", title)
		}
	})
}

// TestQuestShowsResearchStep: an advancement that waits on a research
// behind others shows the research to do now with its cost.
func TestQuestShowsResearchStep(t *testing.T) {
	s := testSession(t)
	a, _ := advancement.ByID("alloy_smelter")
	g, ok := progress.GoalResearch(a)
	if !ok || g.ID != "alloy_smelter" {
		t.Fatalf("goal %v %v", g, ok)
	}
	step, ok := progress.Step(s.Session, g)
	if !ok || step.ID == g.ID || !progress.Open(s.Session, step) {
		t.Fatalf("step %v %v", step, ok)
	}
	progress.StateOf(s.Session).Advancements = map[string]bool{}
	for _, x := range advancement.Tree {
		if x.ID != a.ID {
			progress.StateOf(s.Session).Advancements[x.ID] = true
		}
	}
	_, rows := screens.QuestRows(s.Session, i18n.Default, nil)
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
		screens.UpdateBoard(s.Session, p)
		if !strings.Contains(screens.Boards.Of(s.Session)[p.H()], hud.QuestTitle) {
			t.Fatalf("no quest panel by default: %q", screens.Boards.Of(s.Session)[p.H()])
		}
		if on, _ := account.Toggle(p, advancement.SettingQuestPanel); on {
			t.Fatal("the quest panel setting did not turn off")
		}
		defer func() { _, _ = account.Toggle(p, advancement.SettingQuestPanel) }()
		screens.UpdateBoard(s.Session, p)
		if b := screens.Boards.Of(s.Session)[p.H()]; strings.Contains(b, hud.QuestTitle) || strings.Contains(b, hud.Quest) {
			t.Fatalf("the quest panel shows while off: %q", b)
		}
	})
}
