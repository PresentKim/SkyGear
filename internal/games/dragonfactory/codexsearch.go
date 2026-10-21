package dragonfactory

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/textsearch"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The codex as a recipe viewer (docs/tech/codex-jei.md): an index of every
// item it can say something about, a search over the index, and links from an
// item to the items it is made from and the items it is made into, so that a
// player follows a chain of production by pressing items.

// codexEntry is one item of the index.
type codexEntry struct {
	stack item.Stack
	name  string
}

var (
	codexIndexOnce sync.Once
	codexIndexList []codexEntry
)

// codexIndex lists the items the codex knows something about (a source or a
// use), by name. It is built once, from the creative inventory: every item
// and block of the game is registered there, and so are the vanilla ones the
// game's tables use.
func codexIndex() []codexEntry {
	codexIndexOnce.Do(func() {
		var s *session // the lookups read static tables, not session state
		seen := map[string]bool{}
		for _, c := range creative.Items() {
			st := c.Stack
			if st.Empty() {
				continue
			}
			k := itemKey(st.Item())
			if seen[k] {
				continue
			}
			seen[k] = true
			st = st.Grow(1 - st.Count())
			if len(s.codexSources(i18n.Default, st)) == 0 && len(s.codexUses(i18n.Default, st)) == 0 {
				continue
			}
			codexIndexList = append(codexIndexList, codexEntry{stack: st, name: itemName(st)})
		}
		slices.SortStableFunc(codexIndexList, func(a, b codexEntry) int {
			return strings.Compare(plainName(a.name), plainName(b.name))
		})
	})
	return codexIndexList
}

// plainName is a name without colour codes.
func plainName(s string) string {
	var b strings.Builder
	skip := false
	for _, r := range s {
		switch {
		case skip:
			skip = false
		case r == '§':
			skip = true
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// codexFind returns the entries whose name matches query.
func codexFind(query string) []codexEntry {
	var out []codexEntry
	for _, e := range codexIndex() {
		if textsearch.Match(query, e.name) {
			out = append(out, e)
		}
	}
	return out
}

// codexPage is how many results a page of the search shows.
const codexPage = 44

// askCodexSearch asks for the words to search for.
func (s *session) askCodexSearch(p *player.Player, query string) {
	lang := i18n.Of(p)
	ui.Input(p, ui.SubTitle(i18n.T(lang, "codex.title"), i18n.T(lang, "codex.search")), i18n.T(lang, "codex.search.label"), query,
		func(p *player.Player, _ *world.Tx, value string) { s.showCodexSearch(p, strings.TrimSpace(value), 0) })
}

// showCodexSearch is the search tab: the items matching query (all when it is
// empty) a page at a time, and above the first page the items looked at last.
// Pressing one opens its page.
func (s *session) showCodexSearch(p *player.Player, query string, page int) {
	lang := i18n.Of(p)
	s.codexRemember(p, func(p *player.Player) { s.showCodexSearch(p, query, page) })
	found := codexFind(query)
	pages := max(1, (len(found)+codexPage-1)/codexPage)
	page = max(0, min(page, pages-1))
	var slots []ui.ItemSlot
	for _, e := range found[page*codexPage : min(len(found), (page+1)*codexPage)] {
		slots = append(slots, ui.ItemSlot{
			Icon: iconOf(e.stack), Tip: tipName(e.stack) + "\n§7" + i18n.T(lang, "codex.search.tip"),
			Run: func(p *player.Player, _ *world.Tx) { s.showCodexItem(p, e.stack) },
		})
	}
	actions := s.guard(ui.Pager(p, page, pages, func(p *player.Player, _ *world.Tx, to int) { s.showCodexSearch(p, query, to) }))
	actions = append(actions, s.opt(i18n.T(lang, "codex.search.again"), "", func(p *player.Player, _ *world.Tx) { s.askCodexSearch(p, query) }))
	if query != "" {
		actions = append(actions, s.opt(fmt.Sprintf(i18n.T(lang, "storage.search.clear"), query), "", func(p *player.Player, _ *world.Tx) { s.showCodexSearch(p, "", 0) }))
	}
	actions = append(actions, s.codexTabs(p, codexSearchTab)...)
	actions = append(actions, s.backOption())
	body := fmt.Sprintf(i18n.T(lang, "codex.search.all"), len(found))
	if query != "" {
		body = fmt.Sprintf(i18n.T(lang, "codex.search.found"), query, len(found))
	}
	title := i18n.T(lang, "codex.title")
	var sections []ui.GridSection
	if query == "" && page == 0 {
		if goals := s.goalSlots(); len(goals) > 0 {
			sections = append(sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.goals"), Slots: goals})
		}
		if recent := s.codexRecent[p.H()]; len(recent) > 0 {
			var links []codexLink
			for _, st := range recent {
				links = append(links, codexLink{stack: st})
			}
			sections = append(sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.recent"), Slots: s.codexLinkSlots(links)})
		}
	}
	if len(sections) > 0 {
		ui.ItemGridSections(p, title, body, append(sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.list"), Slots: slots}), actions...)
		return
	}
	ui.ItemGrid(p, title, body, slots, actions...)
}

// codexLink is an item linked from a page with how many of it the recipe
// uses or makes (0 when it is not a recipe: the item a machine turns it into).
type codexLink struct {
	stack item.Stack
	count int
}

// codexLinkSlots are slots that open the codex page of each linked item. A
// recipe count shows on the slot like a stack size.
func (s *session) codexLinkSlots(links []codexLink) []ui.ItemSlot {
	var slots []ui.ItemSlot
	for _, l := range links {
		st := l.stack
		count := ""
		if l.count > 1 {
			count = "§f" + ui.CountText(l.count)
		}
		slots = append(slots, ui.ItemSlot{
			Icon: iconOf(st), Count: count, Tip: tipName(st),
			Run: func(p *player.Player, _ *world.Tx) { s.showCodexItem(p, st) },
		})
	}
	return slots
}
