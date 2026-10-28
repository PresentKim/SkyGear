package skygear

import (
	"github.com/presentkim/skygear/internal/account"
	"github.com/presentkim/skygear/internal/games/skygear/achievement"
	"github.com/presentkim/skygear/internal/games/skygear/advancement"
)

func init() {
	for _, h := range achievement.Haves {
		id := "sg_have_" + h.Short
		account.Register(account.Keyed(id, achievement.Group("collect")))
	}
	for _, id := range []string{achievement.AllResearch, achievement.AllAdvancements} {
		account.Register(account.Keyed(id, achievement.Group("progress")))
	}
}

// grantCollection gives the first-possession achievements to the players who
// hold the items (advancement.InventoryNames). It runs every second.
func grantCollection(held []advancement.PlayerItems) {
	for _, ph := range held {
		for _, h := range achievement.Haves {
			if ph.Names[h.Item] {
				account.Grant(ph.P, "sg_have_"+h.Short)
			}
		}
	}
}
