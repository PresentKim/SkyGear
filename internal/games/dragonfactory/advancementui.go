package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/advancement"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/formicon"
)

// The advancement screen: a tab per chapter with its tree laid out left to
// right by the server (advancement.LayoutOf). A parent in another chapter is
// an entrance that leads there; pressing a node opens its detail.

// showAdvancements opens the advancement screen on the tab of the advancement
// the quest panel follows.
func (s *session) showAdvancements(p *player.Player) {
	count(p, "adv_view", 1)
	var v advancement.View
	if a, ok := s.nextAdvancement(); ok {
		v.Tab = a.Chapter
	}
	s.showAdvancementView(p, v)
}

// advancementTree builds the form's tree of a view: every row of the tab's
// layout.
func (s *session) advancementTree(p *player.Player, v advancement.View) ui.Tree {
	lang := i18n.Of(p)
	l := advancement.LayoutOf(v.Tab)
	var t ui.Tree
	byPos := map[advancement.Pos]*advancement.Advancement{}
	gateAt := map[advancement.Pos]*advancement.Advancement{}
	for _, a := range l.Order {
		byPos[l.Pos[a.ID]] = a
	}
	for _, g := range l.Gates {
		gateAt[l.Pos[advancement.GateID(g.ID)]] = g
	}
	for y := range min(l.H, ui.TreeRows) {
		for x := range ui.TreeCols {
			ps := advancement.Pos{X: x, Y: y}
			c := ui.TreeCell{Strip: l.Strips[ps]}
			if a := byPos[ps]; a != nil {
				c.Frame = advancement.Frame(a)
				c.Icon, c.Tip = advancement.Icon(a), s.advancementTip(lang, a)
				if len(a.Parents) > 1 && (s.advanced(a.ID) || s.advancementOpen(a)) {
					c.Tip += "\n" + advancement.Parents(lang, a)
				}
				c.State = s.advTreeState(a)
				c.Run = func(p *player.Player, tx *world.Tx) {
					if tx.World() == s.world {
						s.showAdvancementNode(p, a)
					}
				}
			} else if g := gateAt[ps]; g != nil {
				c.Frame, c.State = ui.FrameTask, s.advTreeState(g)
				c.Icon, c.Tip = advancement.Icon(g), advancement.GateTip(lang, g)
				c.Run = func(p *player.Player, tx *world.Tx) {
					if tx.World() == s.world {
						s.showAdvancementView(p, advancement.View{Tab: g.Chapter})
					}
				}
			}
			t.Cells = append(t.Cells, c)
		}
	}
	for tab := advancement.Chapter(0); tab < advancement.ChapterCount; tab++ {
		o := ui.Tab(i18n.T(lang, "advancement.tab."+advancement.ChapterKeys[tab]), advancement.ChapterIcon(tab), tab == v.Tab, nil)
		t.Tabs = append(t.Tabs, s.opt(o.Label, o.Icon, func(p *player.Player, _ *world.Tx) {
			s.showAdvancementView(p, advancement.View{Tab: tab})
		}))
	}
	done := 0
	list := advancement.In(v.Tab)
	for _, a := range list {
		if s.advanced(a.ID) {
			done++
		}
	}
	t.Body = fmt.Sprintf(i18n.T(lang, "advancement.progress"), i18n.T(lang, "advancement.tab."+advancement.ChapterKeys[v.Tab]), done, len(list))
	t.Actions = []ui.Option{s.backOption()}
	return t
}

func (s *session) showAdvancementView(p *player.Player, v advancement.View) {
	if ui.OreUIFor(p) {
		s.showQuestBook(p, v.Tab, nil)
		return
	}
	t := s.advancementTree(p, v)
	ui.TreeMenu(p, i18n.T(i18n.Of(p), "advancement.title"), t)
}

// showAdvancementNode shows one advancement with its buttons (track, codex,
// back to its chapter); a locked one also names the parents left.
func (s *session) showAdvancementNode(p *player.Player, a *advancement.Advancement) {
	lang := i18n.Of(p)
	title, body := a.Title(lang), ""
	switch {
	case s.advanced(a.ID):
		body = i18n.T(lang, "advancement.node.done") + advancement.FlatHint(lang, a)
	case s.advancementOpen(a):
		body = i18n.T(lang, "advancement.node.progress") + advancement.FlatHint(lang, a)
	default:
		body = i18n.T(lang, "advancement.node.progress") + advancement.FlatHint(lang, a) + "\n\n" + s.advancementBefore(lang, a)
	}
	var opts []ui.Option
	if !s.advanced(a.ID) {
		label := i18n.T(lang, "advancement.node.track")
		if s.state.Tracked == a.ID {
			label = i18n.T(lang, "advancement.node.untrack")
		}
		opts = append(opts, s.opt(label, formicon.Mission, func(p *player.Player, tx *world.Tx) {
			if s.state.Tracked != a.ID {
				count(p, "adv_track", 1)
			}
			s.track(tx, a)
			s.showAdvancementNode(p, a)
		}))
	}
	if it, ok := advancement.Stack(a); ok {
		opts = append(opts, s.opt(i18n.T(lang, "advancement.node.codex"), formicon.Of(it), func(p *player.Player, _ *world.Tx) {
			s.codexFrom(p, func(p *player.Player) { s.showAdvancementNode(p, a) }, it)
		}))
	}
	back := ui.Back(i18n.T(lang, "codex.back"), func(p *player.Player, _ *world.Tx) { s.showAdvancementView(p, advancement.View{Tab: a.Chapter}) })
	opts = append(opts, s.opt(back.Label, back.Icon, back.Run))
	ui.Menu(p, i18n.T(lang, "advancement.node.title")+title, body, opts...)
}

// advancementTip is a node's tooltip: its name, coloured by its state, and
// its hint; a locked one also names the parents left.
func (s *session) advancementTip(lang string, a *advancement.Advancement) string {
	switch {
	case s.advanced(a.ID):
		return "§a" + a.Title(lang) + "\n§7" + advancement.FlatHint(lang, a)
	case s.advancementOpen(a):
		return "§e" + a.Title(lang) + "\n§7" + advancement.FlatHint(lang, a)
	}
	return "§7" + a.Title(lang) + "\n§8" + advancement.FlatHint(lang, a) + "\n" + s.advancementBefore(lang, a)
}

// advancementBefore names the parents of an advancement not done yet.
func (s *session) advancementBefore(lang string, a *advancement.Advancement) string {
	var names []string
	for _, id := range a.Parents {
		if p, ok := advancement.ByID(id); ok && !s.advanced(id) {
			names = append(names, p.Title(lang))
		}
	}
	return i18n.T(lang, "advancement.node.before") + strings.Join(names, ", ")
}

// advTreeState is the state letter of a node in the tree form.
func (s *session) advTreeState(a *advancement.Advancement) byte {
	switch {
	case s.advanced(a.ID):
		return ui.StateDone
	case s.advancementOpen(a):
		return ui.StateOpen
	}
	return ui.StateLocked
}

// advQuestState is the state of a node in the quest book.
func (s *session) advQuestState(a *advancement.Advancement) string {
	switch {
	case s.advanced(a.ID):
		return ui.QuestDone
	case s.advancementOpen(a):
		return ui.QuestOpen
	}
	return ui.QuestLocked
}

// showQuestBook shows the advancement screen as a quest book (DDUI): the
// categories as chapters, the tab's tree on a plane (the same places as the
// tree form) and, when open is set, its quest panel over it.
func (s *session) showQuestBook(p *player.Player, tab advancement.Chapter, open *advancement.Advancement) {
	lang := i18n.Of(p)
	b := ui.QuestBook{Key: "adv_" + advancement.ChapterKeys[tab]}
	for t := advancement.Chapter(0); t < advancement.ChapterCount; t++ {
		done, list := 0, advancement.In(t)
		for _, a := range list {
			if s.advanced(a.ID) {
				done++
			}
		}
		b.Chapters = append(b.Chapters, ui.QuestChapter{
			Name: i18n.T(lang, "advancement.tab."+advancement.ChapterKeys[t]), Icon: advancement.ChapterIcon(t), Count: fmt.Sprintf("%d/%d", done, len(list)), On: t == tab,
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.world {
					s.showQuestBook(p, t, nil)
				}
			},
		})
		if t == tab {
			b.Heading = fmt.Sprintf(i18n.T(lang, "advancement.progress"), i18n.T(lang, "advancement.tab."+advancement.ChapterKeys[t]), done, len(list))
		}
	}
	l := advancement.LayoutOf(tab)
	if next, ok := s.nextAdvancement(); ok && next.Chapter == tab {
		b.Focus = next.ID
	} else if len(l.Order) > 0 {
		b.Focus = l.Order[0].ID
	}
	for _, a := range l.Order {
		pos := l.Pos[a.ID]
		q := ui.QuestNode{ID: a.ID, X: advancement.QuestX(pos.X), Y: float64(pos.Y), Shape: advancement.Shapes[advancement.Frame(a)], Tracked: s.state.Tracked == a.ID}
		q.Icon, q.Tip, q.State = advancement.Icon(a), s.advancementTip(lang, a), s.advQuestState(a)
		for _, id := range a.Parents {
			par, ok := advancement.ByID(id)
			if !ok {
				continue
			}
			from := id
			if par.Chapter != tab {
				from = advancement.GateID(id)
			}
			// A line past the next column goes through its vias, which take
			// the state of the whole line.
			viaState := ui.QuestOpen
			switch pst := s.advQuestState(par); {
			case pst == ui.QuestDone && q.State == ui.QuestDone:
				viaState = ui.QuestDone
			case q.State == ui.QuestLocked:
				viaState = ui.QuestLocked
			}
			for _, v := range l.Vias[advancement.ViaKey(from, a.ID)] {
				vp := l.Pos[v]
				b.Nodes = append(b.Nodes, ui.QuestNode{ID: v, X: advancement.QuestX(vp.X), Y: float64(vp.Y), Shape: ui.ShapeVia, State: viaState, Deps: []string{from}})
				from = v
			}
			q.Deps = append(q.Deps, from)
		}
		q.Run = func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				s.showQuestBook(p, tab, a)
			}
		}
		b.Nodes = append(b.Nodes, q)
	}
	for _, g := range l.Gates {
		pos := l.Pos[advancement.GateID(g.ID)]
		b.Nodes = append(b.Nodes, ui.QuestNode{
			ID: advancement.GateID(g.ID), X: advancement.QuestX(pos.X), Y: float64(pos.Y), Shape: ui.ShapeGate,
			Icon: advancement.Icon(g), Tip: advancement.GateTip(lang, g), State: s.advQuestState(g),
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.world {
					s.showQuestBook(p, g.Chapter, g)
				}
			},
		})
	}
	if open != nil {
		b.Panel = s.questPanel(p, tab, open)
	}
	ui.QuestBookMenu(p, i18n.T(lang, "advancement.title"), b, s.backOption())
}

// questPanel is an advancement's quest panel: its state and hint, the item
// it asks for (pressing it opens the codex) and the track and codex buttons,
// like showAdvancementNode.
func (s *session) questPanel(p *player.Player, tab advancement.Chapter, a *advancement.Advancement) *ui.QuestPanel {
	lang := i18n.Of(p)
	back := func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.world {
			s.showQuestBook(p, tab, nil)
		}
	}
	again := func(p *player.Player) { s.showQuestBook(p, tab, a) }
	pn := &ui.QuestPanel{Icon: advancement.Icon(a), Title: a.Title(lang), Close: back}
	switch {
	case s.advanced(a.ID):
		pn.Status = "§2" + i18n.T(lang, "advancement.node.done")
		pn.Body = advancement.FlatHint(lang, a)
	case s.advancementOpen(a):
		pn.Status = "§1" + i18n.T(lang, "advancement.node.progress")
		pn.Body = advancement.FlatHint(lang, a)
	default:
		pn.Status = s.advancementBefore(lang, a)
		pn.Body = advancement.FlatHint(lang, a)
	}
	if it, ok := advancement.Stack(a); ok {
		pn.Rows = []ui.SlotRow{{Slots: []ui.ItemSlot{{
			Icon: formicon.Of(it), Tip: tipName(lang, it),
			Run: func(p *player.Player, _ *world.Tx) { s.codexFrom(p, again, it) },
		}}}}
	}
	if !s.advanced(a.ID) {
		label := i18n.T(lang, "advancement.node.track")
		if s.state.Tracked == a.ID {
			label = i18n.T(lang, "advancement.node.untrack")
		}
		pn.Actions = append(pn.Actions, s.opt(label, formicon.Mission, func(p *player.Player, tx *world.Tx) {
			if s.state.Tracked != a.ID {
				count(p, "adv_track", 1)
			}
			s.track(tx, a)
			again(p)
		}))
	}
	return pn
}
