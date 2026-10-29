package skygear

import (
	"fmt"
	"strings"

	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/formicon"
)

// The advancement screen: a quest book with a chapter per category and its
// tree laid out left to right by the server (advancement.LayoutOf). A parent
// in another chapter is an entrance that leads there; pressing a node opens
// its quest panel.

// showAdvancements opens the advancement screen on the chapter of the
// advancement the quest panel follows.
func (s *session) showAdvancements(p *player.Player) {
	achievement.Count(p, "adv_view", 1)
	tab := advancement.ChapterStart
	if a, ok := progress.NextAdvancement(s.Session); ok {
		tab = a.Chapter
	}
	s.showQuestBook(p, tab, nil)
}

// advancementTip is a node's tooltip: its name, coloured by its state, and
// its hint; a locked one also names the parents left.
func (s *session) advancementTip(lang string, a *advancement.Advancement) string {
	switch {
	case progress.Advanced(s.Session, a.ID):
		return "§a" + a.Title(lang) + "\n§7" + advancement.FlatHint(lang, a)
	case progress.AdvancementOpen(s.Session, a):
		return "§e" + a.Title(lang) + "\n§7" + advancement.FlatHint(lang, a)
	}
	return "§7" + a.Title(lang) + "\n§8" + advancement.FlatHint(lang, a) + "\n" + s.advancementBefore(lang, a)
}

// advancementBefore names the parents of an advancement not done yet.
func (s *session) advancementBefore(lang string, a *advancement.Advancement) string {
	var names []string
	for _, id := range a.Parents {
		if p, ok := advancement.ByID(id); ok && !progress.Advanced(s.Session, id) {
			names = append(names, p.Title(lang))
		}
	}
	return i18n.T(lang, "advancement.node.before") + strings.Join(names, ", ")
}

// advQuestState is the state of a node in the quest book.
func (s *session) advQuestState(a *advancement.Advancement) string {
	switch {
	case progress.Advanced(s.Session, a.ID):
		return ui.QuestDone
	case progress.AdvancementOpen(s.Session, a):
		return ui.QuestOpen
	}
	return ui.QuestLocked
}

// showQuestBook shows the advancement screen as a quest book: the categories
// as chapters, the tab's tree on a plane and, when open is set, its quest
// panel over it.
func (s *session) showQuestBook(p *player.Player, tab advancement.Chapter, open *advancement.Advancement) {
	lang := i18n.Of(p)
	b := ui.QuestBook{Key: "adv_" + advancement.ChapterKeys[tab]}
	for t := advancement.Chapter(0); t < advancement.ChapterCount; t++ {
		done, list := 0, advancement.In(t)
		for _, a := range list {
			if progress.Advanced(s.Session, a.ID) {
				done++
			}
		}
		b.Chapters = append(b.Chapters, ui.QuestChapter{
			Name: i18n.T(lang, "advancement.tab."+advancement.ChapterKeys[t]), Icon: advancement.ChapterIcon(t), Count: fmt.Sprintf("%d/%d", done, len(list)), On: t == tab,
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() == s.World {
					s.showQuestBook(p, t, nil)
				}
			},
		})
		if t == tab {
			b.Heading = fmt.Sprintf(i18n.T(lang, "advancement.progress"), i18n.T(lang, "advancement.tab."+advancement.ChapterKeys[t]), done, len(list))
		}
	}
	l := advancement.LayoutOf(tab)
	if next, ok := progress.NextAdvancement(s.Session); ok && next.Chapter == tab {
		b.Focus = next.ID
	} else if len(l.Order) > 0 {
		b.Focus = l.Order[0].ID
	}
	for _, a := range l.Order {
		pos := l.Pos[a.ID]
		q := ui.QuestNode{ID: a.ID, X: advancement.QuestX(pos.X), Y: float64(pos.Y), Shape: advancement.Shapes[advancement.Frame(a)], Tracked: progress.StateOf(s.Session).Tracked == a.ID}
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
			if tx.World() == s.World {
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
				if tx.World() == s.World {
					s.showQuestBook(p, g.Chapter, g)
				}
			},
		})
	}
	if open != nil {
		b.Panel = s.questPanel(p, tab, open)
	}
	ui.QuestBookMenu(p, i18n.T(lang, "advancement.title"), b, s.backOption(lang))
}

// questPanel is an advancement's quest panel: its state and hint, the item
// it asks for (pressing it opens the codex) and the track button.
func (s *session) questPanel(p *player.Player, tab advancement.Chapter, a *advancement.Advancement) *ui.QuestPanel {
	lang := i18n.Of(p)
	back := func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.World {
			s.showQuestBook(p, tab, nil)
		}
	}
	again := func(p *player.Player) { s.showQuestBook(p, tab, a) }
	pn := &ui.QuestPanel{Icon: advancement.Icon(a), Title: a.Title(lang), Close: back}
	switch {
	case progress.Advanced(s.Session, a.ID):
		pn.Status = "§2" + i18n.T(lang, "advancement.node.done")
		pn.Body = advancement.FlatHint(lang, a)
	case progress.AdvancementOpen(s.Session, a):
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
	if !progress.Advanced(s.Session, a.ID) {
		label := i18n.T(lang, "advancement.node.track")
		if progress.StateOf(s.Session).Tracked == a.ID {
			label = i18n.T(lang, "advancement.node.untrack")
		}
		pn.Actions = append(pn.Actions, s.opt(label, formicon.Mission, func(p *player.Player, tx *world.Tx) {
			if progress.StateOf(s.Session).Tracked != a.ID {
				achievement.Count(p, "adv_track", 1)
			}
			progress.Track(s.Session, tx, a)
			again(p)
		}))
	}
	return pn
}
