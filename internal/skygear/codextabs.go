package skygear

import (
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/skygear/formicon"
	"github.com/presentkim/skygear/internal/skygear/naming"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
)

// The codex is one window with a tab strip (search, sieve, processing and
// gear, mobs and waves, plants) that stays visible on every page, also the
// deeper ones, so the player moves between them in one press.

type codexTab int

const (
	codexNoTab codexTab = iota - 1
	codexSearchTab
	codexSieveTab
	codexProcessTab
	codexMobsTab
	codexPlantsTab
	codexTabCount
)

var codexTabKeys = [codexTabCount]string{"codex.tab.search", "codex.tab.sieve", "codex.tab.process", "codex.tab.mobs", "codex.tab.plants"}

func codexTabIcon(t codexTab) string {
	switch t {
	case codexSearchTab:
		return formicon.Of(item.NewStack(item.Compass{}, 1))
	case codexSieveTab:
		return formicon.Of(machineItem(KindSieve))
	case codexProcessTab:
		return formicon.Of(machineItem(KindAlloy))
	case codexMobsTab:
		return formicon.Of(machineItem(KindGate))
	}
	return formicon.Of(item.NewStack(sgitem.Petal{Colour: "red"}, 1))
}

// codexTabs is the tab strip with cur selected (codexNoTab: none).
func (s *session) codexTabs(p *player.Player, cur codexTab) []ui.Option {
	lang := i18n.Of(p)
	var out []ui.Option
	for t := codexTab(0); t < codexTabCount; t++ {
		t := t
		o := ui.Tab(i18n.T(lang, codexTabKeys[t]), codexTabIcon(t), t == cur, func(p *player.Player, _ *world.Tx) { s.showCodexTab(p, t) })
		out = append(out, s.opt(o.Label, o.Icon, o.Run))
	}
	return out
}

// showCodexTab shows the first page of a tab.
func (s *session) showCodexTab(p *player.Player, t codexTab) {
	s.codexStart(p)
	switch t {
	case codexSieveTab:
		s.showCodexSieve(p)
	case codexProcessTab:
		s.showCodexProcess(p)
	case codexMobsTab:
		s.showCodexMobs(p)
	case codexPlantsTab:
		s.showCodexPlants(p)
	default:
		s.showCodexSearch(p, "")
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

func (s *session) codexNavOf(p *player.Player) *codexNav {
	if s.codexNav == nil {
		s.codexNav = map[*world.EntityHandle]*codexNav{}
	}
	n := s.codexNav[p.H()]
	if n == nil {
		n = &codexNav{}
		s.codexNav[p.H()] = n
	}
	return n
}

// codexAt records the page p is shown now.
func (s *session) codexAt(p *player.Player, show func(p *player.Player)) {
	s.codexNavOf(p).cur = show
}

// codexGo shows next and keeps the page p is on to go back to.
func (s *session) codexGo(p *player.Player, next func(p *player.Player)) {
	n := s.codexNavOf(p)
	if n.cur != nil {
		n.hist = append(n.hist, n.cur)
		if len(n.hist) > codexHistMax {
			n.hist = n.hist[1:]
		}
	}
	next(p)
}

// codexStart forgets where p came from: a tab or /도감 starts over.
func (s *session) codexStart(p *player.Player) {
	n := s.codexNavOf(p)
	n.cur, n.hist = nil, nil
}

// codexFrom shows an item's page with back as the page to go back to (the
// research screen).
func (s *session) codexFrom(p *player.Player, back func(p *player.Player), it item.Stack) {
	n := s.codexNavOf(p)
	n.cur, n.hist = nil, []func(*player.Player){back}
	s.showCodexItem(p, it)
}

// codexBack is the back button of a codex page: to the page p came from, or
// to the game's menu when p came from nowhere.
func (s *session) codexBack(p *player.Player) ui.Option {
	n := s.codexNavOf(p)
	if len(n.hist) == 0 {
		return s.backOption(i18n.Of(p))
	}
	b := ui.Back(i18n.T(i18n.Of(p), "codex.back"), func(p *player.Player, _ *world.Tx) {
		n := s.codexNavOf(p)
		if len(n.hist) == 0 {
			s.showMenu(p)
			return
		}
		prev := n.hist[len(n.hist)-1]
		n.hist, n.cur = n.hist[:len(n.hist)-1], nil
		prev(p)
	})
	return s.opt(b.Label, b.Icon, b.Run)
}

// codexRecentMax is how many viewed items the search tab keeps.
const codexRecentMax = 11

// codexSeen records that p looked at an item's page.
func (s *session) codexSeen(p *player.Player, st item.Stack) {
	if s.codexRecent == nil {
		s.codexRecent = map[*world.EntityHandle][]item.Stack{}
	}
	k := naming.ItemKey(st.Item())
	list := slices.DeleteFunc(slices.Clone(s.codexRecent[p.H()]), func(o item.Stack) bool { return naming.ItemKey(o.Item()) == k })
	list = append([]item.Stack{st}, list...)
	if len(list) > codexRecentMax {
		list = list[:codexRecentMax]
	}
	s.codexRecent[p.H()] = list
}
