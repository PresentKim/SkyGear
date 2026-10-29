package codex

import (
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The codex is one window with a tab strip (search, sieve, processing and
// gear, mobs and waves, plants) that stays visible on every page, also the
// deeper ones, so the player moves between them in one press.

type Tab int

const (
	codexNoTab Tab = iota - 1
	codexSearchTab
	codexSieveTab
	codexProcessTab
	codexMobsTab
	codexPlantsTab
	CodexTabCount
)

var TabKeys = [CodexTabCount]string{"codex.tab.search", "codex.tab.sieve", "codex.tab.process", "codex.tab.mobs", "codex.tab.plants"}

func codexTabIcon(t Tab) string {
	switch t {
	case codexSearchTab:
		return formicon.Of(item.NewStack(item.Compass{}, 1))
	case codexSieveTab:
		return formicon.Of(machine.Item(machine.KindSieve))
	case codexProcessTab:
		return formicon.Of(machine.Item(machine.KindAlloy))
	case codexMobsTab:
		return formicon.Of(machine.Item(machine.KindGate))
	}
	return formicon.Of(item.NewStack(sgitem.Petal{Colour: "red"}, 1))
}

// codexTabs is the tab strip with cur selected (codexNoTab: none).
func codexTabs(s *core.Session, p *player.Player, cur Tab) []ui.Option {
	lang := i18n.Of(p)
	var out []ui.Option
	for t := Tab(0); t < CodexTabCount; t++ {
		t := t
		o := ui.Tab(i18n.T(lang, TabKeys[t]), codexTabIcon(t), t == cur, func(p *player.Player, _ *world.Tx) { showCodexTab(s, p, t) })
		out = append(out, s.Opt(o.Label, o.Icon, o.Run))
	}
	return out
}

// showCodexTab shows the first page of a tab.
func showCodexTab(s *core.Session, p *player.Player, t Tab) {
	Start(s, p)
	switch t {
	case codexSieveTab:
		showCodexSieve(s, p)
	case codexProcessTab:
		showCodexProcess(s, p)
	case codexMobsTab:
		showCodexMobs(s, p)
	case codexPlantsTab:
		showCodexPlants(s, p)
	default:
		showCodexSearch(s, p, "")
	}
}

// The codex keeps where each player came from like a browser's history: a
// page reached by pressing a slot (an item of a recipe, a research's
// material) can go back to the page it was reached from, and back again. A
// tab or /도감 starts over. The research screen goes into the codex with
// itself as the page to go back to (codexFrom).
type codexNav struct {
	cur  func(*player.Player)
	hist []func(*player.Player)
}

// codexHistMax is how many pages back a player can go.
const codexHistMax = 32

func codexNavOf(s *core.Session, p *player.Player) *codexNav {
	n := codexNavs.Of(s)[p.H()]
	if n == nil {
		n = &codexNav{}
		codexNavs.Of(s)[p.H()] = n
	}
	return n
}

// codexAt records the page p is shown now.
func codexAt(s *core.Session, p *player.Player, show func(p *player.Player)) {
	codexNavOf(s, p).cur = show
}

// codexGo shows next and keeps the page p is on to go back to.
func codexGo(s *core.Session, p *player.Player, next func(p *player.Player)) {
	n := codexNavOf(s, p)
	if n.cur != nil {
		n.hist = append(n.hist, n.cur)
		if len(n.hist) > codexHistMax {
			n.hist = n.hist[1:]
		}
	}
	next(p)
}

// Start forgets where p came from: a tab or /도감 starts over.
func Start(s *core.Session, p *player.Player) {
	n := codexNavOf(s, p)
	n.cur, n.hist = nil, nil
}

// From shows an item's page with back as the page to go back to (the
// research screen).
func From(s *core.Session, p *player.Player, back func(p *player.Player), it item.Stack) {
	n := codexNavOf(s, p)
	n.cur, n.hist = nil, []func(*player.Player){back}
	ShowItem(s, p, it)
}

// codexBack is the back button of a codex page: to the page p came from, or
// to the game's menu when p came from nowhere.
func codexBack(s *core.Session, p *player.Player) ui.Option {
	n := codexNavOf(s, p)
	if len(n.hist) == 0 {
		return s.BackOption(i18n.Of(p))
	}
	b := ui.Back(i18n.T(i18n.Of(p), "codex.back"), func(p *player.Player, _ *world.Tx) {
		n := codexNavOf(s, p)
		if len(n.hist) == 0 {
			core.ShowMenu(s, p)
			return
		}
		prev := n.hist[len(n.hist)-1]
		n.hist, n.cur = n.hist[:len(n.hist)-1], nil
		prev(p)
	})
	return s.Opt(b.Label, b.Icon, b.Run)
}

// codexRecentMax is how many viewed items the search tab keeps.
const codexRecentMax = 11

// codexSeen records that p looked at an item's page.
func codexSeen(s *core.Session, p *player.Player, st item.Stack) {
	k := naming.ItemKey(st.Item())
	list := slices.DeleteFunc(slices.Clone(codexRecent.Of(s)[p.H()]), func(o item.Stack) bool { return naming.ItemKey(o.Item()) == k })
	list = append([]item.Stack{st}, list...)
	if len(list) > codexRecentMax {
		list = list[:codexRecentMax]
	}
	codexRecent.Of(s)[p.H()] = list
}
