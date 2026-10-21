package dragonfactory

import (
	"slices"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The codex is one window with a tab strip: search (the whole index), the
// sieve tables, processing and gear, mobs and waves, plants. The tabs stay
// visible on every page of the codex, also on the pages that go deeper (a
// mesh's drops, an item), so the player moves between them in one press
// (docs/tech/ui-style.md).

// codexTab is a tab of the codex.
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

// codexTabKeys are the label keys of the tabs.
var codexTabKeys = [codexTabCount]string{"codex.tab.search", "codex.tab.sieve", "codex.tab.process", "codex.tab.mobs", "codex.tab.plants"}

// codexTabIcon is the icon of a tab.
func codexTabIcon(t codexTab) string {
	switch t {
	case codexSearchTab:
		return iconOf(item.NewStack(item.Compass{}, 1))
	case codexSieveTab:
		return iconOf(machineItem(KindSieve))
	case codexProcessTab:
		return iconOf(machineItem(KindAlloy))
	case codexMobsTab:
		return iconOf(machineItem(KindGate))
	}
	return iconOf(item.NewStack(Petal{Colour: "red"}, 1))
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
	switch t {
	case codexSieveTab:
		s.showCodexSieve(p)
	case codexProcessTab:
		s.showCodexProcess(p, 0)
	case codexMobsTab:
		s.showCodexMobs(p, 0)
	case codexPlantsTab:
		s.showCodexPlants(p, 0)
	default:
		s.showCodexSearch(p, "", 0)
	}
}

// codexRemember records the page to come back to from the pages that go
// deeper (an item), and returns it.
func (s *session) codexRemember(p *player.Player, show func(p *player.Player)) {
	if s.codexLast == nil {
		s.codexLast = map[*world.EntityHandle]func(*player.Player){}
	}
	s.codexLast[p.H()] = show
}

// codexBack is the back button of a page that goes deeper: it returns to the
// list page the player came from, or to the codex's first page.
func (s *session) codexBack(p *player.Player) ui.Option {
	b := ui.Back(i18n.T(i18n.Of(p), "codex.title"), func(p *player.Player, _ *world.Tx) {
		if f := s.codexLast[p.H()]; f != nil {
			f(p)
			return
		}
		s.showCodex(p)
	})
	return s.opt(b.Label, b.Icon, b.Run)
}

// codexBackTo is a back button that shows another page.
func (s *session) codexBackTo(p *player.Player, show func(p *player.Player)) ui.Option {
	b := ui.Back(i18n.T(i18n.Of(p), "codex.title"), func(p *player.Player, _ *world.Tx) { show(p) })
	return s.opt(b.Label, b.Icon, b.Run)
}

// codexRecentMax is how many viewed items the search tab keeps.
const codexRecentMax = 11

// codexSeen records that p looked at an item's page.
func (s *session) codexSeen(p *player.Player, st item.Stack) {
	if s.codexRecent == nil {
		s.codexRecent = map[*world.EntityHandle][]item.Stack{}
	}
	k := itemKey(st.Item())
	list := slices.DeleteFunc(slices.Clone(s.codexRecent[p.H()]), func(o item.Stack) bool { return itemKey(o.Item()) == k })
	list = append([]item.Stack{st}, list...)
	if len(list) > codexRecentMax {
		list = list[:codexRecentMax]
	}
	s.codexRecent[p.H()] = list
}
