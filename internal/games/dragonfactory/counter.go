package dragonfactory

import (
	"fmt"
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// The counter achievements of the game (docs/design/01-platform.md §5): what
// a player does is counted by key (account.Count), and counterCatalog says
// which counts grant which achievement and title. The catalog and its
// texts are generated from the catalog of titles; the hooks below and in
// the game's code do the counting.

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
		account.Register(account.Achievement{
			ID:    id,
			Name:  i18n.T(i18n.Default, "achievement."+id+".name"),
			Desc:  i18n.T(i18n.Default, "achievement."+id+".desc"),
			Title: i18n.T(i18n.Default, "achievement."+id+".title"),
			Group: achievementGroup("collect"),
		})
	}
	for _, id := range []string{achAllResearch, achAllAdv} {
		account.Register(account.Achievement{
			ID:    id,
			Name:  i18n.T(i18n.Default, "achievement."+id+".name"),
			Desc:  i18n.T(i18n.Default, "achievement."+id+".desc"),
			Title: i18n.T(i18n.Default, "achievement."+id+".title"),
			Group: achievementGroup("progress"),
		})
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

// countBreak counts a block p breaks: all blocks, logs, leaves, and what a
// hammer crushes.
func countBreak(p *player.Player, b world.Block, hammer bool) {
	count(p, "block_break", 1)
	switch b.(type) {
	case block.Log:
		count(p, "log_break", 1)
	case block.Leaves:
		count(p, "leaves_break", 1)
	}
	if !hammer {
		return
	}
	switch b.(type) {
	case block.Stone:
		count(p, "hammer_stone", 1)
	case block.Cobblestone:
		count(p, "hammer_cobble", 1)
	case block.Gravel:
		count(p, "hammer_gravel", 1)
	case block.Sand:
		count(p, "hammer_sand", 1)
	}
}

// countWalk counts the blocks p walks to newPos: the horizontal distance of
// a move, kept as a fraction until it adds up to a block. A jump of more
// than a few blocks is a teleport, not walking.
func (s *session) countWalk(p *player.Player, newPos mgl64.Vec3) {
	from := p.Position()
	d := math.Hypot(newPos[0]-from[0], newPos[2]-from[2])
	if d == 0 || d > 4 {
		return
	}
	if s.walked == nil {
		s.walked = map[*world.EntityHandle]float64{}
	}
	acc := s.walked[p.H()] + d
	if whole := math.Floor(acc); whole >= 1 {
		count(p, "walk", int64(whole))
		acc -= whole
	}
	s.walked[p.H()] = acc
}

// cycleOnReset marks the machines whose finished work cycle shows as their
// progress going back to zero from a full interval; the others count their
// cycles where they finish them (machineState.done).
var cycleOnReset = map[string]bool{
	KindCrusher: true, KindFurnace: true, KindAlloy: true, KindWasher: true, KindCrucible: true,
	KindCrafter: true, KindSimulator: true, KindAutoComposter: true,
}

// countWork passes the work a machine did to the players' counters: its
// finished cycles at once, the RE it generated once a second.
func (st *machineState) countWork(tx *world.Tx, kind string, tick int64) {
	if st.done > 0 {
		countAll(tx, "cycle_total", int64(st.done))
		countAll(tx, "cycle_"+kind, int64(st.done))
		st.done = 0
	}
	if st.made > 0 && tick%20 == 0 {
		countAll(tx, "energy_made", int64(st.made))
		st.made = 0
	}
}
