package skygear

import (
	"fmt"
	"github.com/presentkim/skygear/content/skygear/wave"
	"slices"
	"strings"
	"sync"

	"github.com/presentkim/skygear/content/skygear/farm"

	"github.com/presentkim/skygear/content/skygear/textfmt"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/decor"

	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/textsearch"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// The codex as a recipe viewer: an index of every
// item it can say something about, a search over the index, and links from an
// item to the items it is made from and the items it is made into, so that a
// player follows a chain of production by pressing items.

// codexEntry is one item of the index.
type codexEntry struct {
	stack item.Stack
	name  string
}

var (
	codexIndexOnce   sync.Once
	codexIndexStacks []item.Stack
	codexIndexByKey  = map[string]item.Stack{}
	codexIndexMu     sync.Mutex
	codexIndexByLang = map[string][]codexEntry{}
)

// codexIndex lists the items the codex knows something about (a source or a
// use) that play a part in Sky Gear (codexRelevant), by their name in
// lang. The items are found once, from the creative inventory: every item
// and block of the game is registered there, and so are the vanilla ones the
// game's tables use; each language's list is sorted once.
func codexIndex(lang string) []codexEntry {
	codexIndexOnce.Do(func() {
		var s *session // the lookups read static tables, not session state
		relevant := codexRelevant()
		seen := map[string]bool{}
		for _, c := range creative.Items() {
			st := c.Stack
			if st.Empty() {
				continue
			}
			k := naming.ItemKey(st.Item())
			if seen[k] || !relevant[k] {
				continue
			}
			seen[k] = true
			st = st.Grow(1 - st.Count())
			if len(s.codexSources(i18n.Default, st)) == 0 && len(s.codexUses(i18n.Default, st)) == 0 &&
				len(s.codexWorkFor(i18n.Default, st, true)) == 0 && len(s.codexWorkFor(i18n.Default, st, false)) == 0 {
				continue
			}
			codexIndexStacks = append(codexIndexStacks, st)
			codexIndexByKey[k] = st
		}
	})
	codexIndexMu.Lock()
	defer codexIndexMu.Unlock()
	if list, ok := codexIndexByLang[lang]; ok {
		return list
	}
	list := make([]codexEntry, len(codexIndexStacks))
	for i, st := range codexIndexStacks {
		list[i] = codexEntry{stack: st, name: naming.ItemName(lang, st)}
	}
	slices.SortStableFunc(list, func(a, b codexEntry) int {
		return strings.Compare(textfmt.PlainName(a.name), textfmt.PlainName(b.name))
	})
	codexIndexByLang[lang] = list
	return list
}

// codexRelevant are the items (naming.ItemKey) a player meets in Dragon
// Factory: the game's own items, what its tables take and give, the results
// of the recipes research opens, and everything those are made of down to
// the raw materials. Vanilla items the game never asks for are left out.
func codexRelevant() map[string]bool {
	open := map[string]bool{}
	for _, n := range progress.BaseUnlocks {
		open[naming.UnlockName(n)] = true
	}
	for i := range progress.Tree {
		for _, n := range progress.Tree[i].Unlocks {
			open[naming.UnlockName(n)] = true
		}
	}
	rel := map[string]bool{}
	var queue []item.Stack
	add := func(w world.Item) {
		if w == nil {
			return
		}
		if k := naming.ItemKey(w); !rel[k] {
			rel[k] = true
			queue = append(queue, item.NewStack(w, 1))
		}
	}
	pick := func(it world.Item) {
		name, _ := it.EncodeItem()
		if strings.HasPrefix(name, "skygear:") || open[progress.RecipeKey(it)] {
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
	for i := range progress.Tree {
		for _, c := range progress.Tree[i].Cost {
			add(c.Item)
		}
	}
	for _, in := range codexSieveInputs() {
		add(in)
		for _, t := range sgitem.MeshTiers {
			for _, d := range sievetable.Tables[t.Tier][naming.ItemKey(in)] {
				add(d.Item.Item())
			}
		}
		for _, mob := range sgitem.MobKinds {
			for _, d := range sievetable.MagicThemes[mob].Adds[naming.ItemKey(in)] {
				add(d.Item.Item())
			}
		}
	}
	for _, in := range crushInputs() {
		add(in)
	}
	for _, m := range farm.BarrelMixes() {
		add(m.In)
		add(m.Out)
	}
	for _, r := range machinerecipe.Alloys {
		add(r.A)
		add(r.B)
		add(r.Out)
	}
	for i := range machinerecipe.Labs {
		for _, in := range machinerecipe.Labs[i].In {
			add(in.Item)
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, d := range sgitem.SimDrops[mob] {
			add(d.Item)
		}
	}
	for _, w := range wave.Levels {
		for _, r := range w.Reward {
			add(r.Item())
		}
	}
	for _, pk := range farm.PlantKinds {
		for _, y := range pk.Yield {
			add(y.Item)
		}
	}
	for _, h := range tinker.HeadMaterials {
		add(h.Item)
	}
	for _, h := range tinker.HandleMaterials {
		add(h.Item)
	}
	for _, m := range tinker.ToolModifiers {
		add(m.Item)
	}
	// down to the raw materials, and through the furnace
	for len(queue) > 0 {
		st := queue[0]
		queue = queue[1:]
		if open[progress.RecipeKey(st.Item())] {
			k := naming.ItemKey(st.Item())
			for _, cr := range machinerecipe.Crafters() {
				if cr.OutKey != k {
					continue
				}
				for _, req := range cr.Reqs {
					switch need := req.Need.(type) {
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
		if out, ok := machine.SmeltOf(st); ok {
			add(out.Item())
		}
		if out, ok := machine.CrusherOutput(st); ok {
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
		if rel[naming.ItemKey(a.Item())] {
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

// codexFind returns the entries whose name matches query.
func codexFind(lang, query string) []codexEntry {
	var out []codexEntry
	for _, e := range codexIndex(lang) {
		if textsearch.Match(query, e.name) {
			out = append(out, e)
		}
	}
	return out
}

// showCodexSearch is the search tab: the search box in the window and the
// items matching query (all when it is empty), all of them in one scrolling
// grid, drawn again as the player types; with an empty query the favourites
// and the items looked at last come first. Pressing one opens its page, and
// going back from it returns to the last query.
func (s *session) showCodexSearch(p *player.Player, query string) {
	lang := i18n.Of(p)
	find := func(p *player.Player, query string) ui.SearchResult {
		query = strings.TrimSpace(query)
		s.codexAt(p, func(p *player.Player) { s.showCodexSearch(p, query) })
		found := codexFind(lang, query)
		var slots []ui.ItemSlot
		for _, e := range found {
			slots = append(slots, ui.ItemSlot{
				Icon: formicon.Of(e.stack), Tip: decor.TipName(lang, e.stack) + "\n§7" + i18n.T(lang, "codex.search.tip"),
				Run: func(p *player.Player, _ *world.Tx) {
					s.codexGo(p, func(p *player.Player) { s.showCodexItem(p, e.stack) })
				},
			})
		}
		r := ui.SearchResult{Info: fmt.Sprintf(i18n.T(lang, "codex.search.all"), len(found))}
		if query != "" {
			r.Info = fmt.Sprintf(i18n.T(lang, "codex.search.found"), query, len(found))
		} else {
			if favs := s.favouriteSlots(p, lang); len(favs) > 0 {
				r.Sections = append(r.Sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.favourites"), Slots: favs})
			}
			if recent := s.codexRecent[p.H()]; len(recent) > 0 {
				var links []codexLink
				for _, st := range recent {
					links = append(links, codexLink{stack: st})
				}
				r.Sections = append(r.Sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.recent"), Slots: s.codexLinkSlots(lang, links)})
			}
		}
		r.Sections = append(r.Sections, ui.GridSection{Heading: i18n.T(lang, "codex.search.list"), Slots: slots})
		return r
	}
	actions := append(s.codexTabs(p, codexSearchTab), s.codexBack(p))
	ui.SearchGrid(p, i18n.T(lang, "codex.title"), i18n.T(lang, "codex.search.label"), query, find, actions...)
}

// codexLink is an item linked from a page with how many of it the recipe
// uses or makes (0 when it is not a recipe: the item a machine turns it into).
type codexLink struct {
	stack item.Stack
	count int
}

// codexLinkSlots are slots that open the codex page of each linked item. A
// recipe count shows on the slot like a stack size.
func (s *session) codexLinkSlots(lang string, links []codexLink) []ui.ItemSlot {
	var slots []ui.ItemSlot
	for _, l := range links {
		st := l.stack
		count := ""
		if l.count > 1 {
			count = "§f" + ui.CountText(l.count)
		}
		slots = append(slots, ui.ItemSlot{
			Icon: formicon.Of(st), Count: count, Tip: decor.TipName(lang, st),
			Run: func(p *player.Player, _ *world.Tx) { s.codexGo(p, func(p *player.Player) { s.showCodexItem(p, st) }) },
		})
	}
	return slots
}
