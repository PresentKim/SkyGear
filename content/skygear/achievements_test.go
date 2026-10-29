package skygear

import (
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/catalog"

	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/platform/account"
)

// TestFirstResearchGrantsAchievement: completing a research earns the
// research achievement for the player who paid it.
func TestFirstResearchGrantsAchievement(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	account.Init(s.g.db, slog.Default())
	t.Cleanup(func() { account.Init(nil, slog.Default()) })
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if _, err := s.g.db.TouchAccount(p.UUID().String(), "", p.Name()); err != nil {
			t.Fatal(err)
		}
		sticks, _ := progress.ByID("stick")
		_, _ = p.Inventory().AddItem(item.NewStack(catalog.OakPlanks, 16))
		progress.Do(s.Session, p, tx, sticks)
		if got := account.Owned(p.UUID().String()); !slices.Contains(got, achievement.FirstResearch) {
			t.Fatalf("achievements %v", got)
		}
	})
}

// TestAchievementsAreWritten: every registered achievement has its name,
// description and title in the catalog (a missing one shows its key).
func TestAchievementsAreWritten(t *testing.T) {
	for _, a := range account.All() {
		for _, s := range []string{a.Name(i18n.Default), a.Desc(i18n.Default), a.Title(i18n.Default)} {
			if s == "" || strings.HasPrefix(s, "achievement.") {
				t.Errorf("achievement %s: %q", a.ID, s)
			}
		}
	}
}
