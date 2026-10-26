package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/achievement"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/advancement"
)

func init() {
	for _, h := range achievement.Haves {
		id := "df_have_" + h.Short
		account.Register(account.Keyed(id, achievement.Group("collect")))
	}
	for _, id := range []string{achievement.AllResearch, achievement.AllAdvancements} {
		account.Register(account.Keyed(id, achievement.Group("progress")))
	}
}

// grantCollection gives the first-possession achievements to the players who
// hold the items (inventoryNames). It runs every second.
func grantCollection(held []advancement.PlayerItems) {
	for _, ph := range held {
		for _, h := range achievement.Haves {
			if ph.Names[h.Item] {
				account.Grant(ph.P, "df_have_"+h.Short)
			}
		}
	}
}
