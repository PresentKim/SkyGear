package skygear

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/presentkim/skygear/content/skygear/codex"

	"github.com/presentkim/skygear/content/skygear/decor"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// Research: every crafting recipe but baseUnlocks is locked. A research,
// paid whole with items at the research table, opens the recipes of its
// results for everyone on the island (recipes.go). Researches form a tree;
// trees and costs are drafts.

// researchIcon is a research's form icon: its first result.
func researchIcon(r *progress.Research) string {
	if len(r.Unlocks) == 0 {
		if it, ok := world.ItemByName(naming.UnlockName(r.Icon), 0); ok {
			return formicon.Of(item.NewStack(it, 1))
		}
		return formicon.Mission
	}
	if it, ok := world.ItemByName(naming.UnlockName(r.Unlocks[0]), 0); ok {
		return formicon.Of(item.NewStack(it, 1))
	}
	return formicon.Mission
}

func (s *session) showResearch(p *player.Player) {
	achievement.Count(p, "research_open", 1)
	s.showResearchSplit(p, researchView{})
}

// researchView is what the research screen shows: the tab (0 to do, 1 done),
// the page of the grid and the chosen research ("" the first of the tab);
// help shows the explanation as the text of the form.
type researchView struct {
	tab, page int
	sel       string
}

// researchKind is how a research stands.
type researchKind int

const (
	researchTodo researchKind = iota
	researchPreview
	researchDone
)

// researchEntry is a research with how it stands.
type researchEntry struct {
	r    *progress.Research
	kind researchKind
}

// researchEntries are the researches of the tabs: the ones to do (those that
// can be done now first, then the grey ones a step from open) and the done
// ones. A research is hidden until everything before it is done, so the tree
// is found step by step. openN is how many are to do without the grey ones.
func (s *session) researchEntries() (todo, done []researchEntry, openN int) {
	var previews []researchEntry
	for i := range progress.Tree {
		r := &progress.Tree[i]
		switch {
		case progress.Researched(s.Session, r.ID):
			done = append(done, researchEntry{r, researchDone})
		case progress.Open(s.Session, r):
			openN++
			todo = append(todo, researchEntry{r, researchTodo})
		default:
			if _, ok := progress.Blocker(s.Session, r); ok {
				previews = append(previews, researchEntry{r, researchPreview})
			}
		}
	}
	for _, r := range progress.LocalOf(s.Session).Copies {
		if progress.Researched(s.Session, r.ID) {
			done = append(done, researchEntry{r, researchDone})
		} else {
			openN++
			todo = append(todo, researchEntry{r, researchTodo})
		}
	}
	return append(todo, previews...), done, openN
}

// researchSlot is the grid slot of a research. The tooltip of a research to
// do is its name in yellow, in green when p has all it spends and can do it
// now. p may be nil.
func (s *session) researchSlot(p *player.Player, e researchEntry, show func(p *player.Player, tx *world.Tx)) ui.ItemSlot {
	r := e.r
	switch e.kind {
	case researchDone:
		// The done tab holds only done researches: no mark.
		return ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(i18n.Of(p), "§7", r), Run: show}
	case researchTodo:
		colour := "§e"
		if p != nil && len(progress.Missing(s.Session, p, r)) == 0 {
			colour = "§a"
		}
		return ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(i18n.Of(p), colour, r), Run: show}
	}
	return ui.ItemSlot{Icon: researchIcon(r), Tip: researchTip(i18n.Of(p), "§8", r), Run: show}
}

// showResearchSplit shows the research screen (ui.SplitMenu): on the left
// the tabs, the grid of researches and the page buttons, on the right the
// research chosen. Choosing a research, a tab or a page shows the screen
// again.
func (s *session) showResearchSplit(p *player.Player, v researchView) {
	lang := i18n.Of(p)
	todo, done, _ := s.researchEntries()
	list := todo
	if v.tab == 1 {
		list = done
	}
	idx := -1
	for i, e := range list {
		if e.r.ID == v.sel {
			idx = i
		}
	}
	pages := max(1, (len(list)+ui.SplitCells-1)/ui.SplitCells)
	switch {
	case idx >= 0:
		v.page = idx / ui.SplitCells
	case len(list) > 0:
		v.page = max(0, min(v.page, pages-1))
		idx = v.page * ui.SplitCells
		v.sel = list[idx].r.ID
	default:
		v.sel = ""
	}
	pick := func(tab, page int, sel string) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) {
			s.showResearchSplit(p, researchView{tab: tab, page: page, sel: sel})
		}
	}
	var slots []ui.ItemSlot
	for i := v.page * ui.SplitCells; i < min(len(list), (v.page+1)*ui.SplitCells); i++ {
		e := list[i]
		sl := s.researchSlot(p, e, pick(v.tab, v.page, e.r.ID))
		if e.r.ID == v.sel {
			sl.Selected, sl.Run = true, nil
		}
		slots = append(slots, sl)
	}
	tab := func(key string, icon string, t int) ui.Option {
		o := ui.Tab(i18n.T(lang, key), icon, v.tab == t, pick(t, 0, ""))
		return s.Opt(o.Label, o.Icon, o.Run)
	}
	sp := ui.Split{
		Tabs: []ui.Option{
			tab("researchtab.todo", formicon.Mission, 0),
			tab("researchtab.done", formicon.Done, 1),
		},
		Slots: slots,
		Page:  v.page, Pages: pages,
		Turn: func(p *player.Player, tx *world.Tx, to int) {
			if tx.World() == s.World {
				s.showResearchSplit(p, researchView{tab: v.tab, page: to})
			}
		},
	}
	tip := i18n.T(lang, "researchlist.tip")
	if idx >= 0 {
		again := func(p *player.Player) { s.showResearchSplit(p, v) }
		finished := func(p *player.Player) { s.showResearchSplit(p, researchView{tab: v.tab, page: v.page}) }
		goTo := func(p *player.Player, r *progress.Research) { s.showResearchAt(p, r) }
		d, rows, opts := s.researchNodeWith(p, list[idx].r, again, finished, goTo)
		sp.Detail, sp.Rows, sp.Actions = d, rows, opts
	}
	// the help is the button's tooltip; pressing it keeps the screen
	sp.Actions = append(sp.Actions, s.Opt(ui.Help(tip, nil).Label, "", func(p *player.Player, _ *world.Tx) { s.showResearchSplit(p, v) }), s.BackOption(lang))
	ui.SplitMenu(p, i18n.T(lang, "researchlist.title"), sp)
}

// showResearchAt shows the research screen with a research chosen, on the tab
// it belongs to.
func (s *session) showResearchAt(p *player.Player, r *progress.Research) {
	v := researchView{sel: r.ID}
	if progress.Researched(s.Session, r.ID) {
		v.tab = 1
	}
	s.showResearchSplit(p, v)
}

// researchSlots builds the two sections of the research list: the slots of the
// researches to do and of the done ones, with how many are to do (without the
// grey ones) and done. p may be nil.
func (s *session) researchSlots(p *player.Player) (todo, done []ui.ItemSlot, todoN, doneN int) {
	todoE, doneE, openN := s.researchEntries()
	for _, e := range todoE {
		r := e.r
		todo = append(todo, s.researchSlot(p, e, func(p *player.Player, _ *world.Tx) { s.showResearchAt(p, r) }))
	}
	for _, e := range doneE {
		r := e.r
		done = append(done, s.researchSlot(p, e, func(p *player.Player, _ *world.Tx) { s.showResearchAt(p, r) }))
	}
	return todo, done, openN, len(doneE)
}

// researchTip is the hover text of a research in the list: its name in
// colour and the short description.
func researchTip(lang, colour string, r *progress.Research) string {
	tip := colour + r.Title(lang)
	if short := progress.Text(lang, r.ID).Short; short != "" {
		tip += "\n§7" + short
	}
	return tip
}

// researchNode builds the screen of a research with a way back to the
// research screen. p may be nil (the owned counts are then left out).
func (s *session) researchNode(p *player.Player, r *progress.Research) (ui.Detail, []ui.SlotRow, []ui.Option) {
	d, rows, opts := s.researchNodeWith(p, r,
		func(p *player.Player) { s.showResearchAt(p, r) },
		func(p *player.Player) { s.showResearch(p) },
		func(p *player.Player, n *progress.Research) { s.showResearchAt(p, n) })
	back := ui.Back(i18n.T(i18n.Of(p), "researchnode.back"), func(p *player.Player, _ *world.Tx) { s.showResearch(p) })
	return d, rows, append(opts, s.Opt(back.Label, back.Icon, back.Run))
}

// researchNodeWith builds the detail of a research: the card, the text, the
// materials and the recipes it opens, and the buttons to do it and to go to
// what it needs. redo shows the screen again, finished shows it after the
// research was done, goTo shows another research. p may be nil.
func (s *session) researchNodeWith(p *player.Player, r *progress.Research, redo, finished func(p *player.Player), goTo func(p *player.Player, r *progress.Research)) (ui.Detail, []ui.SlotRow, []ui.Option) {
	txt := progress.Text(i18n.Of(p), r.ID)
	done := progress.Researched(s.Session, r.ID)
	var b strings.Builder
	if txt.About != "" {
		fmt.Fprintf(&b, "§f%s\n", txt.About)
	}
	d := ui.Detail{Icon: researchIcon(r), Name: "§6" + r.Title(i18n.Of(p))}
	var opts []ui.Option
	switch {
	case done:
		d.Status = i18n.T(i18n.Of(p), "researchnode.complete")
	case progress.Open(s.Session, r):
		opts = append(opts, s.Opt(i18n.T(i18n.Of(p), "researchnode.start"), formicon.Mission, func(p *player.Player, tx *world.Tx) {
			if r.IsCopy() {
				progress.LocalOf(s.Session).CopiesDone[r.ID] = true
				finished(p)
				return
			}
			missing, ok := progress.Do(s.Session, p, tx, r)
			switch {
			case ok:
				ui.Flash(p, "%s", i18n.T(i18n.Of(p), "researchnode.finished"))
				finished(p)
				return
			case len(missing) > 0:
				var names []string
				for _, m := range missing {
					names = append(names, island.CostText(i18n.Of(p), m))
				}
				ui.Notify(p, i18n.T(i18n.Of(p), "researchnode.missing"), strings.Join(names, ", "))
			}
			redo(p)
		}))
	}
	// What is still to do before it: a button that goes to that research.
	for _, n := range r.Needs {
		if need, ok := progress.ByID(n); ok && !done && !progress.Researched(s.Session, n) {
			opts = append(opts, s.Opt(fmt.Sprintf(i18n.T(i18n.Of(p), "researchneeds.button"), need.Title(i18n.Of(p))), researchIcon(need), func(p *player.Player, _ *world.Tx) { goTo(p, need) }))
		}
	}
	d.Body = b.String()
	// A material or recipe slot opens the item's codex page, whose back
	// button comes back here; an item the codex does not know cannot be
	// pressed.
	toCodex := func(st item.Stack, tip string) ui.ItemSlot {
		if _, ok := codex.ItemByKey(naming.ItemKey(st.Item())); !ok {
			return ui.ItemSlot{Icon: formicon.Of(st), Tip: tip + "\n§8" + i18n.T(i18n.Of(p), "codex.fromResearch.none")}
		}
		return ui.ItemSlot{Icon: formicon.Of(st), Tip: tip + "\n§8" + i18n.T(i18n.Of(p), "codex.fromResearch.go"), Run: func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.World {
				codex.From(s.Session, p, redo, st)
			}
		}}
	}
	// The materials: the count shown is what the research spends (green
	// once p has enough, or the research is done); the tooltip says what
	// p has.
	var cost []ui.ItemSlot
	for _, c := range r.Cost {
		need := progress.Need(s.Session, r, c)
		st := item.NewStack(c.Item, 1)
		mark, tip := "§f", fmt.Sprintf(i18n.T(i18n.Of(p), "researchnode.cost.need"), naming.UIItemName(i18n.Of(p), st), need)
		switch {
		case done:
			mark, tip = "§a", fmt.Sprintf(i18n.T(i18n.Of(p), "researchnode.cost.uses"), naming.UIItemName(i18n.Of(p), st), c.Count)
			need = c.Count
		case p != nil:
			have := core.CountPlain(p.Inventory(), c.Item)
			if have >= need {
				mark = "§a"
			}
			tip += fmt.Sprintf(i18n.T(i18n.Of(p), "researchnode.cost.have"), mark, have)
		}
		sl := toCodex(st, tip)
		sl.Count = mark + strconv.Itoa(need)
		cost = append(cost, sl)
	}
	// The recipes it opens.
	var opens []ui.ItemSlot
	for _, n := range r.Unlocks {
		it, ok := world.ItemByName(naming.UnlockName(n), 0)
		if !ok {
			continue
		}
		st := item.NewStack(it, 1)
		tip := decor.TipName(i18n.Of(p), st) + i18n.T(i18n.Of(p), "researchnode.unlocks")
		c := progress.RewardCount(r, n)
		if c > 0 {
			tip += "\n" + fmt.Sprintf(i18n.T(i18n.Of(p), "researchreward.tip"), c)
		}
		sl := toCodex(st, tip)
		if c > 0 {
			sl.Count = "§e+" + strconv.Itoa(c)
		}
		opens = append(opens, sl)
	}
	rows := []ui.SlotRow{{Heading: i18n.T(i18n.Of(p), "researchnode.materials"), Slots: cost}, {Heading: i18n.T(i18n.Of(p), "researchnode.rewards"), Slots: opens}}
	return d, rows, opts
}

func init() {
	progress.ShowScreen = func(c *core.Session, p *player.Player) {
		if s, ok := activeGame.sessionOf(c.World); ok {
			s.showResearch(p)
		}
	}
}
