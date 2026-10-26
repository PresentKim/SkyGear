package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/achievement"
)

// The counter achievements of the game: what a player does is counted by key
// (account.Count), and achievement.Counters says which counts grant which
// achievement and title. The code that counts sits next to what it counts;
// counterhooks.go, collect.go and machinecount.go have the shared parts.

func init() {
	account.SetGroupOrder(achievement.Group("hand"), achievement.Group("farm"), achievement.Group("sift"),
		achievement.Group("factory"), achievement.Group("energy"), achievement.Group("gear"), achievement.Group("combat"),
		achievement.Group("progress"), achievement.Group("island"), achievement.Group("collect"), achievement.Group("life"), achievement.Group("account"))
	for _, c := range achievement.Counters {
		var tiers []account.Tier
		for _, n := range c.Needs {
			id := achievement.CounterID(c.Key, n)
			account.Register(account.Keyed(id, achievement.Group(c.Group)))
			tiers = append(tiers, account.Tier{Need: n, ID: id})
		}
		account.RegisterCounter(c.Key, tiers...)
	}
}

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
