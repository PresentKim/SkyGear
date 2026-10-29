package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/platform/account"
)

// TestProjectsLeadToTheDragonEgg: the last project needs the three others,
// opens the egg recipe, and placing the egg finishes the game.
func TestProjectsLeadToTheDragonEgg(t *testing.T) {
	last, ok := progress.ByID("project_dragon")
	if !ok || len(last.Needs) != 3 {
		t.Fatalf("last project: %#v", last)
	}
	for _, id := range last.Needs {
		if r, ok := progress.ByID(id); !ok || len(r.Cost) < 4 {
			t.Errorf("project %s is missing or cheap", id)
		}
	}
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if progress.RecipeFilter(s.Session)(recipeFor(block.DragonEgg{})) {
			t.Fatal("the egg recipe is open before the project")
		}
		progress.Complete(s.Session, tx, last)
		if !progress.RecipeFilter(s.Session)(recipeFor(block.DragonEgg{})) {
			t.Fatal("the last project did not open the egg recipe")
		}
		s.completeDragon(p, tx, machinePos)
		if !progress.Advanced(s.Session, advancement.Dragon) {
			t.Errorf("%s not done by placing the egg", advancement.Dragon)
		}
	})
}

// TestAchievementsOfAdvancements: every advancement that gives an account
// achievement exists, and so does the achievement.
func TestAchievementsOfAdvancements(t *testing.T) {
	for adv, ach := range progress.AdvAchievements {
		if _, ok := advancement.ByID(adv); !ok {
			t.Errorf("no advancement %s", adv)
		}
		if a, ok := account.Get(ach); !ok || a.Title(i18n.Default) == "" || a.Name(i18n.Default) == "" {
			t.Errorf("achievement %s is not written", ach)
		}
	}
}
