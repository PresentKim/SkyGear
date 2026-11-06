package netstore

import (
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/invclick"
	"github.com/presentkim/skygear/common/textsearch"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// The recipe book of the terminal window (after Bedrock's): the island's
// open crafting-table recipes of the open tab (the creative category of the
// result) or, on the search tab, of every category matching the book's
// search, those the network, the inventory and the grid can make first. A
// left click lays a recipe's ingredients into the grid from the network,
// then the inventory; shift crafts up to a stack into the inventory, and a
// right click with nothing held crafts once into it.

// bookSlots is how many recipes the book shows.
const bookSlots = 240

// bookTabs are the book's tabs in the vanilla order; the last is the search
// tab.
var bookTabs = []struct {
	cat       creative.Category
	key, icon string
}{
	{creative.ConstructionCategory(), "netstore.terminal.tab.construction", "textures/ui/icon_recipe_construction"},
	{creative.EquipmentCategory(), "netstore.terminal.tab.equipment", "textures/ui/icon_recipe_equipment"},
	{creative.ItemsCategory(), "netstore.terminal.tab.items", "textures/ui/icon_recipe_item"},
	{creative.NatureCategory(), "netstore.terminal.tab.nature", "textures/ui/icon_recipe_nature"},
	{key: "netstore.terminal.tab.search", icon: "textures/ui/magnifyingGlass"},
}

// searchTab is the index of the search tab in bookTabs.
var searchTab = len(bookTabs) - 1

var (
	// bookCanOnly are the players whose book hides the recipes they cannot
	// make; every other book shows all of them.
	bookCanOnly = core.NewPlayers[bool]()
	// bookTab is the tab each player's book shows (bookTabs; the search tab,
	// every recipe, until one is picked).
	bookTab = core.NewPlayers[int]()
	// bookQuery is the search of each player's search tab.
	bookQuery = core.NewPlayers[string]()
)

var (
	categoriesOnce sync.Once
	categories     map[string]creative.Category
)

// categoryOf is the creative category of it: that of its creative group, or
// the items category when it has none.
func categoryOf(it world.Item) creative.Category {
	categoriesOnce.Do(func() {
		groups := map[string]creative.Category{}
		for _, g := range creative.Groups() {
			if _, ok := groups[g.Name]; !ok {
				groups[g.Name] = g.Category
			}
		}
		categories = map[string]creative.Category{}
		for _, c := range creative.Items() {
			k := naming.ItemKey(c.Stack.Item())
			if _, ok := categories[k]; !ok {
				if cat, ok := groups[c.Group]; ok {
					categories[k] = cat
				}
			}
		}
	})
	if cat, ok := categories[naming.ItemKey(it)]; ok {
		return cat
	}
	return creative.ItemsCategory()
}

// stock counts the items the book may use, by item key and by name.
type stock struct {
	byKey  map[string]int
	byName map[string]int
}

func newStock(n *Network, areas ...invclick.Area) stock {
	s := stock{byKey: map[string]int{}, byName: map[string]int{}}
	add := func(st item.Stack, c int) {
		name, _ := st.Item().EncodeItem()
		s.byKey[naming.ItemKey(st.Item())] += c
		s.byName[name] += c
	}
	if n != nil && n.Online() {
		for _, e := range entriesOf(n) {
			add(e.Item, int(min(e.Count, 1<<20)))
		}
	}
	for _, a := range areas {
		for i := range a.Len() {
			if st := a.Item(i); !st.Empty() {
				add(st, st.Count())
			}
		}
	}
	return s
}

// has reports whether the stock holds count of the ingredient.
func (s stock) has(need recipe.Item, count int) bool {
	switch need := need.(type) {
	case item.Stack:
		if _, variants := need.Value("variants"); variants {
			name, _ := need.Item().EncodeItem()
			return s.byName[name] >= count
		}
		return s.byKey[naming.ItemKey(need.Item())] >= count
	case recipe.ItemTag:
		total := 0
		for name, c := range s.byName {
			if need.Contains(name) {
				total += c
			}
		}
		return total >= count
	}
	return false
}

func (s stock) makes(cr *machinerecipe.Crafter) bool {
	for _, req := range cr.Reqs {
		if !s.has(req.Need, req.Count) {
			return false
		}
	}
	return true
}

// bookEntry is a recipe of the book and whether it can be made now.
type bookEntry struct {
	cr  *machinerecipe.Crafter
	can bool
}

// openRecipe reports whether the island of s may craft r (s nil: any).
func openRecipe(s *core.Session, r recipe.Recipe) bool {
	if s == nil {
		return true
	}
	f := progress.IslandRecipeFilter(s)
	return f == nil || f(r)
}

// gridRecipe is the open recipe the grid makes.
func gridRecipe(s *core.Session, grid invclick.Area) (*machinerecipe.Crafter, bool) {
	cr, ok := machinerecipe.MatchGrid(gridCells(grid))
	if !ok || !openRecipe(s, cr.Recipe) {
		return nil, false
	}
	return cr, true
}

// bookOf lists the recipes the book shows to p: open ones of the open tab,
// or on the search tab those whose result matches the book's search, the
// makeable first; only those unless p shows all.
func bookOf(s *core.Session, p *player.Player, n *Network, grid invclick.Area) []bookEntry {
	lang := i18n.Of(p)
	inv := p.Inventory()
	st := newStock(n, invclick.Hotbar(inv), invclick.Main(inv), grid)
	all := !bookCanOnly.Of(s)[p.H()]
	tab := openTab(s, p)
	query := strings.TrimSpace(bookQuery.Of(s)[p.H()])
	var can, cannot []bookEntry
	for _, cr := range machinerecipe.Crafters() {
		if _, ok := gridPattern(cr); !ok || !openRecipe(s, cr.Recipe) {
			continue
		}
		if tab != searchTab && categoryOf(cr.Out.Item()) != bookTabs[tab].cat {
			continue
		}
		if tab == searchTab && query != "" && !textsearch.Match(query, naming.ItemName(lang, cr.Out)) {
			continue
		}
		if st.makes(cr) {
			can = append(can, bookEntry{cr, true})
		} else if all {
			cannot = append(cannot, bookEntry{cr, false})
		}
	}
	out := append(can, cannot...)
	if len(out) > bookSlots {
		out = out[:bookSlots]
	}
	return out
}

func openTab(s *core.Session, p *player.Player) int {
	if t, ok := bookTab.Of(s)[p.H()]; ok {
		return t
	}
	return searchTab
}

func bookPanel(s *core.Session, p *player.Player, book []bookEntry, again func(p *player.Player, tx *world.Tx)) *ui.InvBook {
	lang := i18n.Of(p)
	open := openTab(s, p)
	b := &ui.InvBook{}
	for i, t := range bookTabs {
		b.Tabs = append(b.Tabs, ui.InvTab{Label: i18n.T(lang, t.key), Icon: t.icon, On: i == open, End: i == searchTab, Run: func(p *player.Player, tx *world.Tx) {
			bookTab.Of(s)[p.H()] = i
			again(p, tx)
		}})
	}
	if open == searchTab {
		b.Search = &ui.InvSearch{Query: bookQuery.Of(s)[p.H()], Placeholder: i18n.T(lang, "storage.search.label"), Change: func(p *player.Player, tx *world.Tx, q string) {
			bookQuery.Of(s)[p.H()] = q
			again(p, tx)
		}}
	} else {
		b.Title = i18n.T(lang, bookTabs[open].key)
	}
	key, icon := "netstore.terminal.bookCan", "textures/ui/craft_toggle_on"
	if !bookCanOnly.Of(s)[p.H()] {
		key, icon = "netstore.terminal.bookAll", "textures/ui/craft_toggle_off"
	}
	b.Filter = &ui.Option{Label: i18n.T(lang, key), Icon: icon, Run: func(p *player.Player, tx *world.Tx) {
		bookCanOnly.Of(s)[p.H()] = !bookCanOnly.Of(s)[p.H()]
		again(p, tx)
	}}
	for _, e := range book {
		tip := decor.TipName(lang, e.cr.Out) + "\n" + strings.Join(e.cr.Tip(lang), "\n") + "\n§7" + i18n.T(lang, "netstore.terminal.bookTip")
		sl := ui.StackSlot(e.cr.Out, tip)
		sl.Dim = !e.can
		b.Slots = append(b.Slots, sl)
	}
	if len(book) == 0 {
		b.Note = i18n.T(lang, "netstore.terminal.bookNone")
	}
	return b
}

// gridPattern is what each grid cell of a recipe holds: a shaped recipe
// from the top left, a shapeless one cell by cell.
func gridPattern(cr *machinerecipe.Crafter) ([]recipe.Item, bool) {
	cells := make([]recipe.Item, gridSlots)
	switch r := cr.Recipe.(type) {
	case recipe.Shaped:
		w, h := r.Shape().Width(), r.Shape().Height()
		if w > 3 || h > 3 {
			return nil, false
		}
		in := r.Input()
		for y := range h {
			for x := range w {
				cells[y*3+x] = in[y*w+x]
			}
		}
	case recipe.Shapeless:
		i := 0
		for _, in := range r.Input() {
			if in == nil || in.Empty() {
				continue
			}
			for range max(1, in.Count()) {
				if i >= gridSlots {
					return nil, false
				}
				cells[i] = in
				i++
			}
		}
	default:
		return nil, false
	}
	return cells, true
}

// takeOne takes one item matching want out of the network, then out of the
// areas.
func takeOne(n *Network, want recipe.Item, from ...invclick.Area) item.Stack {
	req := machinerecipe.CrafterReq{Need: want}
	if n != nil && n.Online() {
		for _, e := range entriesOf(n) {
			if machinerecipe.MatchReq(e.Item, req) {
				if got := netExtract(n, KeyOf(e.Item), 1); !got.Empty() {
					return got
				}
			}
		}
	}
	for _, a := range from {
		for i := range a.Len() {
			if st := a.Item(i); machinerecipe.MatchReq(st, req) {
				a.Set(i, st.Grow(-1))
				return st.Grow(1 - st.Count())
			}
		}
	}
	return item.Stack{}
}

// layRecipe empties the grid (clearGrid) and lays one set of the recipe's
// ingredients into it; it reports whether every cell got its ingredient
// (if not, what it laid goes back the same way).
func layRecipe(n *Network, grid invclick.Area, cr *machinerecipe.Crafter, p *player.Player) bool {
	cells, ok := gridPattern(cr)
	if !ok {
		return false
	}
	clearGrid(n, grid, p)
	inv := p.Inventory()
	for i, want := range cells {
		if want == nil || want.Empty() {
			continue
		}
		got := takeOne(n, want, invclick.Hotbar(inv), invclick.Main(inv))
		if got.Empty() {
			clearGrid(n, grid, p)
			return false
		}
		grid.Set(i, got)
	}
	return true
}

// bookClick runs a click on recipe cr of the book: lay it (left), craft a
// stack into the inventory (shift) or craft once into it (right, nothing
// held). It returns the cursor stack.
func bookClick(s *core.Session, n *Network, grid invclick.Area, cr *machinerecipe.Crafter, cur item.Stack, right, shift bool, p *player.Player) item.Stack {
	if !shift && !right {
		layRecipe(n, grid, cr, p)
		return cur
	}
	if right && !cur.Empty() {
		return cur
	}
	made := 0
	for made < max(1, cr.Out.MaxCount()) {
		if c, ok := gridRecipe(s, grid); !ok || c != cr {
			if !layRecipe(n, grid, cr, p) {
				break
			}
		}
		out, ok := craftOnce(s, n, grid)
		if !ok {
			break
		}
		inv := p.Inventory()
		if rest := invclick.Move(out, invclick.Hotbar(inv), invclick.Main(inv)); !rest.Empty() {
			if n != nil && n.Online() {
				rest = rest.Grow(-int(netInsert(n, rest)))
			}
			if !rest.Empty() {
				p.Drop(rest)
			}
		}
		made += out.Count()
		if right {
			break
		}
	}
	return cur
}
