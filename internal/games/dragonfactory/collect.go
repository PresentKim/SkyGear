package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/account"
)

// haveDef is an item whose first possession is an achievement.
type haveDef struct{ item, short string }

// achievements without a counter: the whole research tree and the whole
// advancement tree done.
const (
	achAllResearch = "df_all_research"
	achAllAdv      = "df_all_advancements"
)

func init() {
	for _, h := range haveCatalog {
		id := "df_have_" + h.short
		account.Register(account.Keyed(id, achievementGroup("collect")))
	}
	for _, id := range []string{achAllResearch, achAllAdv} {
		account.Register(account.Keyed(id, achievementGroup("progress")))
	}
}

// grantCollection gives the first-possession achievements to the players who
// hold the items (inventoryNames). It runs every second.
func grantCollection(held []playerItems) {
	for _, ph := range held {
		for _, h := range haveCatalog {
			if ph.names[h.item] {
				account.Grant(ph.p, "df_have_"+h.short)
			}
		}
	}
}
