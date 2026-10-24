package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
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
// hold the items. It runs every second.
func grantCollection(tx *world.Tx) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		have := map[string]bool{}
		for _, st := range p.Inventory().Items() {
			name, _ := st.Item().EncodeItem()
			have[name] = true
		}
		for _, h := range haveCatalog {
			if have[h.item] {
				account.Grant(p, "df_have_"+h.short)
			}
		}
	}
}
