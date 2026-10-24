package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/account"
)

// TestProjectsLeadToTheDragonEgg: the last project needs the three others,
// opens the egg recipe, and placing the egg finishes the game.
func TestProjectsLeadToTheDragonEgg(t *testing.T) {
	last, ok := researchByID("project_dragon")
	if !ok || len(last.Needs) != 3 {
		t.Fatalf("last project: %#v", last)
	}
	for _, id := range last.Needs {
		if r, ok := researchByID(id); !ok || len(r.Cost) < 4 {
			t.Errorf("project %s is missing or cheap", id)
		}
	}
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if s.recipeFilter()(recipeFor(block.DragonEgg{})) {
			t.Fatal("the egg recipe is open before the project")
		}
		s.completeResearch(tx, last)
		if !s.recipeFilter()(recipeFor(block.DragonEgg{})) {
			t.Fatal("the last project did not open the egg recipe")
		}
		s.completeDragon(p, tx, machinePos)
		for _, id := range []string{advDragon, "project_dragon", "project_botanist", "project_hunter", "project_industry"} {
			if !s.advanced(id) {
				t.Errorf("%s not done by placing the egg", id)
			}
		}
	})
}

// TestAchievementsOfAdvancements: every advancement that gives an account
// achievement exists, and so does the achievement.
func TestAchievementsOfAdvancements(t *testing.T) {
	for adv, ach := range advAchievements {
		if _, ok := advancementByID(adv); !ok {
			t.Errorf("no advancement %s", adv)
		}
		if a, ok := account.Get(ach); !ok || a.Title(i18n.Default) == "" || a.Name(i18n.Default) == "" {
			t.Errorf("achievement %s is not written", ach)
		}
	}
}
