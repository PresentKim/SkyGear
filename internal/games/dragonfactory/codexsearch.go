package dragonfactory

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/item/recipe"
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
// use) that play a part in Dragon Factory (codexRelevant), by name. It is
// built once, from the creative inventory: every item and block of the game
// is registered there, and so are the vanilla ones the game's tables use.
func codexIndex() []codexEntry {
	codexIndexOnce.Do(func() {
		var s *session // the lookups read static tables, not session state
		relevant := codexRelevant()
		seen := map[string]bool{}
		for _, c := range creative.Items() {
			st := c.Stack
			if st.Empty() {
				continue
			}
			k := itemKey(st.Item())
			if seen[k] || !relevant[k] {
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

// codexRelevant are the items (itemKey) a player meets in Dragon Factory:
// the game's own items, what its tables take and give (research costs, the
// sieve, the machines, the simulator, the waves, the plants, tool
// materials), the results of the recipes research opens, and then everything
// those are made of (the open crafting recipes and the furnace, down to the
// raw materials) and what the furnace makes of them. Vanilla items the game
// never asks for (tipped arrows, lingering potions, ...) are left out.
func codexRelevant() map[string]bool {
	open := map[string]bool{}
	for _, n := range baseUnlocks {
		open[unlockName(n)] = true
	}
	for i := range researchTree {
		for _, n := range researchTree[i].Unlocks {
			open[unlockName(n)] = true
		}
	}
	rel := map[string]bool{}
	var queue []item.Stack
	add := func(w world.Item) {
		if w == nil {
			return
		}
		if k := itemKey(w); !rel[k] {
			rel[k] = true
			queue = append(queue, item.NewStack(w, 1))
		}
	}
	pick := func(it world.Item) {
		name, _ := it.EncodeItem()
		if strings.HasPrefix(name, "dragonfactory:") || open[recipeKey(it)] {
			add(it)
		}
	}
	for _, it := range world.Items() {
		pick(it)
	}
	for _, c := range creative.Items() {
		if !c.Stack.Empty() {
			pick(c.Stack.Item())
		}
	}
	for i := range researchTree {
		for _, c := range researchTree[i].Cost {
			add(c.Item)
		}
	}
	for _, in := range codexSieveInputs() {
		add(in)
		for _, t := range meshTiers {
			for _, d := range sieveTables[t.tier][itemKey(in)] {
				add(d.Item.Item())
			}
		}
		for _, mob := range mobKinds {
			for _, d := range magicThemes[mob].Adds[itemKey(in)] {
				add(d.Item.Item())
			}
		}
	}
	for _, in := range crushInputs() {
		add(in)
	}
	for _, m := range barrelMixes() {
		add(m.in)
		add(m.out)
	}
	for _, r := range alloyRecipes {
		add(r.A)
		add(r.B)
		add(r.Out)
	}
	for i := range labRecipes {
		for _, in := range labRecipes[i].In {
			add(in.Item)
		}
	}
	for _, mob := range mobKinds {
		for _, d := range simDrops[mob] {
			add(d.Item)
		}
	}
	for _, w := range waveLevels {
		for _, r := range w.Reward {
			add(r.Item())
		}
	}
	for _, pk := range plantKinds {
		for _, y := range pk.yield {
			add(y.item)
		}
	}
	for _, h := range headMaterials {
		add(h.item)
	}
	for _, h := range handleMaterials {
		add(h.item)
	}
	for _, m := range toolModifiers {
		add(m.item)
	}
	// down to the raw materials, and through the furnace
	for len(queue) > 0 {
		st := queue[0]
		queue = queue[1:]
		if open[recipeKey(st.Item())] {
			for _, cr := range crafterRecipes() {
				if itemKey(cr.out.Item()) != itemKey(st.Item()) {
					continue
				}
				for _, req := range cr.reqs {
					switch need := req.need.(type) {
					case item.Stack:
						add(need.Item())
					case recipe.ItemTag:
						if t := tagStack(need); !t.Empty() {
							add(t.Item())
						}
					}
				}
			}
		}
		// many things smelt into one (every log into charcoal): one of them
		// is enough, the one the game has, else the plainest
		if in, ok := codexOneOf(codexSmeltInputs(st), rel); ok {
			add(in.Item())
		}
		if out, ok := smeltOf(st); ok {
			add(out.Item())
		}
		if out, ok := crusherOutput(st); ok {
			add(out.Item())
		}
	}
	return rel
}

// codexOneOf picks the one of alternatives (things that each make the same
// item) a player has: none when one of them is already relevant (it is
// reachable that way), else the oak one or the one without a variant in its
// name (no deepslate, colour or other wood), as tagStack picks.
func codexOneOf(alts []item.Stack, rel map[string]bool) (item.Stack, bool) {
	if len(alts) == 0 {
		return item.Stack{}, false
	}
	for _, a := range alts {
		if rel[itemKey(a.Item())] {
			return item.Stack{}, false
		}
	}
	best, score := alts[0], -1
	for _, a := range alts {
		name, _ := a.Item().EncodeItem()
		name = strings.TrimPrefix(name, "minecraft:")
		sc := 0
		switch {
		case strings.HasPrefix(name, "oak_"):
			sc = 3
		case !strings.Contains(name, "_"):
			sc = 2
		case !strings.Contains(name, "deepslate") && !strings.Contains(name, "stripped"):
			sc = 1
		}
		if sc > score {
			best, score = a, sc
		}
	}
	return best, true
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
	if ui.OreUIFor(p) {
		s.showCodexSearchLive(p, query)
		return
	}
	s.codexAt(p, func(p *player.Player) { s.showCodexSearch(p, query, page) })
	found := codexFind(query)
	pages := max(1, (len(found)+codexPage-1)/codexPage)
	page = max(0, min(page, pages-1))
	var slots []ui.ItemSlot
	for _, e := range found[page*codexPage : min(len(found), (page+1)*codexPage)] {
		slots = append(slots, ui.ItemSlot{
			Icon: iconOf(e.stack), Tip: tipName(e.stack) + "\n§7" + i18n.T(lang, "codex.search.tip"),
			Run: func(p *player.Player, _ *world.Tx) {
				s.codexGo(p, func(p *player.Player) { s.showCodexItem(p, e.stack) })
			},
		})
	}
	actions := s.guard(ui.Pager(p, page, pages, func(p *player.Player, _ *world.Tx, to int) { s.showCodexSearch(p, query, to) }))
	actions = append(actions, s.opt(i18n.T(lang, "codex.search.again"), "", func(p *player.Player, _ *world.Tx) { s.askCodexSearch(p, query) }))
	if query != "" {
		actions = append(actions, s.opt(fmt.Sprintf(i18n.T(lang, "storage.search.clear"), query), "", func(p *player.Player, _ *world.Tx) { s.showCodexSearch(p, "", 0) }))
	}
	actions = append(actions, s.codexTabs(p, codexSearchTab)...)
	actions = append(actions, s.codexBack(p))
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

// showCodexSearchLive is the search tab as a DDUI screen: the search box in
// the window and the results, all of them in one scrolling grid, drawn again
// as the player types. Going back from an item returns to the last query.
func (s *session) showCodexSearchLive(p *player.Player, query string) {
	lang := i18n.Of(p)
	find := func(p *player.Player, query string) ui.SearchResult {
		query = strings.TrimSpace(query)
		s.codexAt(p, func(p *player.Player) { s.showCodexSearch(p, query, 0) })
		found := codexFind(query)
		var slots []ui.ItemSlot
		for _, e := range found {
			slots = append(slots, ui.ItemSlot{
				Icon: iconOf(e.stack), Tip: tipName(e.stack) + "\n§7" + i18n.T(lang, "codex.search.tip"),
				Run: func(p *player.Player, _ *world.Tx) {
					s.codexGo(p, func(p *player.Player) { s.showCodexItem(p, e.stack) })
				},
			})
		}
		r := ui.SearchResult{Info: fmt.Sprintf(i18n.T(lang, "codex.search.all"), len(found))}
		if query != "" {
			r.Info = fmt.Sprintf(i18n.T(lang, "codex.search.found"), query, len(found))
		} else {
			if goals := s.goalSlots(); len(goals) > 0 {
				r.Sections = append(r.Sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.goals"), Slots: goals})
			}
			if recent := s.codexRecent[p.H()]; len(recent) > 0 {
				var links []codexLink
				for _, st := range recent {
					links = append(links, codexLink{stack: st})
				}
				r.Sections = append(r.Sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.recent"), Slots: s.codexLinkSlots(links)})
			}
		}
		r.Sections = append(r.Sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.list"), Slots: slots})
		return r
	}
	actions := append(s.codexTabs(p, codexSearchTab), s.codexBack(p))
	ui.SearchGrid(p, i18n.T(lang, "codex.title"), i18n.T(lang, "codex.search.label"), query, find,
		func(p *player.Player, _ *world.Tx) { s.askCodexSearch(p, query) }, actions...)
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
			Run: func(p *player.Player, _ *world.Tx) { s.codexGo(p, func(p *player.Player) { s.showCodexItem(p, st) }) },
		})
	}
	return slots
}
