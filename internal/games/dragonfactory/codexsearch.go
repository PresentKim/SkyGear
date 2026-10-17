package dragonfactory

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/textsearch"
	"github.com/presentkim/dragonfactory/internal/ui"
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
	ui.Input(p, i18n.T(lang, "codex.title")+" · "+i18n.T(lang, "codex.search"), i18n.T(lang, "codex.search.label"), query,
		func(p *player.Player, _ *world.Tx, value string) { s.showCodexSearch(p, strings.TrimSpace(value), 0) })
}

// showCodexSearch lists the items matching query; pressing one opens its page.
func (s *session) showCodexSearch(p *player.Player, query string, page int) {
	lang := i18n.Of(p)
	found := codexFind(query)
	pages := max(1, (len(found)+codexPage-1)/codexPage)
	page = max(0, min(page, pages-1))
	var slots []ui.ItemSlot
	for _, e := range found[page*codexPage : min(len(found), (page+1)*codexPage)] {
		slots = append(slots, ui.ItemSlot{
			Icon: iconOf(e.stack), Tip: "§f" + e.name + "\n§7" + i18n.T(lang, "codex.search.tip"),
			Run: func(p *player.Player, _ *world.Tx) { s.showCodexItem(p, e.stack) },
		})
	}
	var actions []ui.Option
	if page+1 < pages {
		actions = append(actions, s.opt(i18n.T(lang, "storage.next"), "", func(p *player.Player, _ *world.Tx) { s.showCodexSearch(p, query, page+1) }))
	}
	if page > 0 {
		actions = append(actions, s.opt(i18n.T(lang, "storage.prev"), "", func(p *player.Player, _ *world.Tx) { s.showCodexSearch(p, query, page-1) }))
	}
	actions = append(actions,
		s.opt(i18n.T(lang, "codex.search.again"), "", func(p *player.Player, _ *world.Tx) { s.askCodexSearch(p, query) }),
		s.opt(i18n.T(lang, "codex.back"), iconBack, func(p *player.Player, _ *world.Tx) { s.showCodex(p) }))
	body := fmt.Sprintf(i18n.T(lang, "codex.search.all"), len(found))
	if query != "" {
		body = fmt.Sprintf(i18n.T(lang, "codex.search.found"), query, len(found))
	}
	ui.ItemGrid(p, i18n.T(lang, "codex.title")+" · "+i18n.T(lang, "codex.search"), body, slots, actions...)
}

// codexLink is an item linked from a page with how many of it the recipe
// uses or makes (0 when it is not a recipe: the item a machine turns it into).
type codexLink struct {
	stack item.Stack
	count int
}

// codexLinks are the items an item is made from (from) and the items made
// from it (into), read from the same tables as the codex's text.
func codexLinks(it item.Stack) (from, into []codexLink) {
	k := itemKey(it.Item())
	seenFrom, seenInto := map[string]bool{k: true}, map[string]bool{k: true}
	add := func(list *[]codexLink, seen map[string]bool, w world.Item, count int) {
		if w == nil || seen[itemKey(w)] {
			return
		}
		seen[itemKey(w)] = true
		*list = append(*list, codexLink{item.NewStack(w, 1), count})
	}
	addStack := func(list *[]codexLink, seen map[string]bool, st item.Stack, count int) {
		if !st.Empty() {
			add(list, seen, st.Item(), count)
		}
	}
	for _, cr := range crafterRecipes() {
		if itemKey(cr.out.Item()) == k {
			for _, req := range cr.reqs {
				if st, ok := req.need.(item.Stack); ok {
					addStack(&from, seenFrom, st, req.count)
				}
			}
		}
		if cr.wants(it) {
			addStack(&into, seenInto, cr.out, cr.out.Count())
		}
	}
	for _, r := range alloyRecipes {
		if itemKey(r.Out) == k {
			add(&from, seenFrom, r.A, r.CountA)
			add(&from, seenFrom, r.B, r.CountB)
		}
		if itemKey(r.A) == k || itemKey(r.B) == k {
			add(&into, seenInto, r.Out, r.OutCount)
		}
	}
	for i := range labRecipes {
		if itemKey(labRecipes[i].Out) == k {
			for _, in := range labRecipes[i].In {
				add(&from, seenFrom, in.Item, in.Count)
			}
		}
		for _, in := range labRecipes[i].In {
			if itemKey(in.Item) == k {
				add(&into, seenInto, labRecipes[i].Out, labRecipes[i].OutCount)
			}
		}
	}
	for _, in := range append([]world.Item{block.Stone{}, block.Cobblestone{}, block.Gravel{}, block.Sand{}}, chunkItems()...) {
		if got, ok := crusherOutput(item.NewStack(in, 1)); ok && itemKey(got.Item()) == k {
			add(&from, seenFrom, in, 0)
		}
	}
	if got, ok := crusherOutput(it); ok {
		addStack(&into, seenInto, got, 0)
	}
	if got, ok := smeltOf(it); ok {
		addStack(&into, seenInto, got, 0)
	}
	for _, in := range codexSieveInputs() {
		for _, t := range meshTiers {
			for _, d := range sieveTables[t.tier][itemKey(in)] {
				if itemKey(d.Item.Item()) == k {
					add(&from, seenFrom, in, 0)
				}
			}
		}
	}
	for _, pk := range plantKinds {
		for _, y := range pk.yield {
			if itemKey(y.item) == k {
				add(&from, seenFrom, PlantSeed{Seed: pk.seed}, 0)
			}
		}
	}
	return from, into
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
			Icon: iconOf(st), Count: count, Tip: "§f" + itemName(st),
			Run: func(p *player.Player, _ *world.Tx) { s.showCodexItem(p, st) },
		})
	}
	return slots
}
