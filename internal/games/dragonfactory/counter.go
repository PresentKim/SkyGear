package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// The counter achievements of the game (docs/design/01-platform.md §5,
// docs/tech/achievements.md): what a player does is counted by key
// (account.Count), and counterCatalog (achcatalog.go) says which counts
// grant which achievement and title. The code that counts is next to what
// it counts; counterhooks.go, collect.go and machinecount.go have the
// shared parts.

// counterDef is one counter: its key, the group its achievements are
// listed in and the counts they are granted at.
type counterDef struct {
	key, group string
	needs      []int64
}

// counterID is the achievement a counter grants at a count.
func counterID(key string, need int64) string { return fmt.Sprintf("df_%s_%d", key, need) }

func init() {
	account.SetGroupOrder(achievementGroup("hand"), achievementGroup("farm"), achievementGroup("sift"),
		achievementGroup("factory"), achievementGroup("energy"), achievementGroup("gear"), achievementGroup("combat"),
		achievementGroup("progress"), achievementGroup("island"), achievementGroup("collect"), achievementGroup("life"), achievementGroup("account"))
	for _, c := range counterCatalog {
		var tiers []account.Tier
		for _, n := range c.needs {
			id := counterID(c.key, n)
			account.Register(account.Achievement{
				ID:    id,
				Name:  i18n.T(i18n.Default, "achievement."+id+".name"),
				Desc:  i18n.T(i18n.Default, "achievement."+id+".desc"),
				Title: i18n.T(i18n.Default, "achievement."+id+".title"),
				Group: achievementGroup(c.group),
			})
			tiers = append(tiers, account.Tier{Need: n, ID: id})
		}
		account.RegisterCounter(c.key, tiers...)
	}
}

// achievementGroup is the name of a group of achievements.
func achievementGroup(g string) string { return i18n.T(i18n.Default, "achgroup."+g) }

// count adds to a counter of p.
func count(p *player.Player, key string, n int64) {
	if p != nil {
		account.Count(p, key, n)
	}
}

// countAll adds to a counter of every player in the world, for what the
// island's machines and plants do on their own.
func countAll(tx *world.Tx, key string, n int64) {
	for e := range tx.Players() {
		account.Count(e.(*player.Player), key, n)
	}
}
