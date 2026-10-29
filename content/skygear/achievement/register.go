package achievement

import (
	_ "github.com/presentkim/skygear/content/skygear/locales" // the achievements' texts
	"github.com/presentkim/skygear/platform/account"
)

// The counter achievements of the game: what a player does is counted by key
// (Count), and Counters says which counts grant which achievement and title.
// The code that counts sits next to what it counts; counter.go has the
// shared parts.

func init() {
	account.SetGroupOrder(Group("hand"), Group("farm"), Group("sift"),
		Group("factory"), Group("energy"), Group("gear"), Group("combat"),
		Group("progress"), Group("island"), Group("collect"), Group("life"), Group("account"))
	for _, c := range Counters {
		var tiers []account.Tier
		for _, n := range c.Needs {
			id := CounterID(c.Key, n)
			account.Register(account.Keyed(id, Group(c.Group)))
			tiers = append(tiers, account.Tier{Need: n, ID: id})
		}
		account.RegisterCounter(c.Key, tiers...)
	}
	for _, h := range Haves {
		account.Register(account.Keyed("sg_have_"+h.Short, Group("collect")))
	}
	for _, id := range []string{AllResearch, AllAdvancements} {
		account.Register(account.Keyed(id, Group("progress")))
	}
	account.Register(
		account.Keyed(FirstMachine, Group("factory")),
		account.Keyed(Chunks10, Group("island")),
		account.Keyed(FirstResearch, Group("progress")),
	)
}
