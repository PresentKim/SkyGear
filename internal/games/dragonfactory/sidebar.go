package dragonfactory

import (
	"fmt"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/hud"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/perf"
)

// updateBoard refreshes p's HUD (docs/design/02-dragonfactory.md §16): day
// and play time at the top left (with the server load under it for an
// operator who turned it on) and the next thing to do as a quest panel: the
// first research that can be done now, with its cost. The texts travel as
// sidebar rows (internal/hud), only when they changed since the last time.
func (s *session) updateBoard(p *player.Player) {
	total, day, _ := s.clock.Now()
	lang := i18n.Of(p)
	var b hud.Board
	b.Add(hud.TopLeft, fmt.Sprintf(i18n.T(lang, "dragonfactory.sidebar.updateBoard.1"), day, formatPlayTime(lang, total)))
	if s.showsLoad(p) {
		b.Add(hud.TopLeft, perf.Line(s.world))
	}
	title, rows := s.questRows(lang, p.Inventory())
	b.Add(hud.QuestTitle, title)
	for _, r := range rows {
		b.Add(hud.Quest, r)
	}
	s.addGoalRows(lang, &b)
	if s.boards == nil {
		s.boards = map[*world.EntityHandle]string{}
	}
	if key := b.Key(); s.boards[p.H()] != key {
		s.boards[p.H()] = key
		b.Send(p)
	}
}

// nextResearch is the first research in tree order that can be done now.
func (s *session) nextResearch() (*Research, bool) {
	for i := range researchTree {
		if r := &researchTree[i]; s.researchOpen(r) {
			return r, true
		}
	}
	return nil, false
}

// questRows are the quest panel: the next advancement and its hint (with
// the research cost for a research advancement), then once every
// advancement is done the next research and its cost. The cost rows count
// what inv holds of each item, when inv is not nil.
func (s *session) questRows(lang string, inv *inventory.Inventory) (string, []string) {
	if a, ok := s.nextAdvancement(); ok {
		rows := strings.Split(a.HintText(lang), "\n")
		for i := range rows {
			rows[i] = "§f" + rows[i]
		}
		if g, ok := goalResearch(a); ok {
			if r, ok := s.researchStep(g); ok {
				if r != g || a.Research == "" {
					rows = append(rows, fmt.Sprintf(i18n.T(lang, "sidebar.quest.research"), r.Title(lang)))
				}
				rows = append(rows, s.researchCostRows(lang, r, inv)...)
			}
		}
		return i18n.T(lang, "dragonfactory.sidebar.questRows.1") + a.Title(lang), rows
	}
	r, ok := s.nextResearch()
	if !ok {
		return i18n.T(lang, "dragonfactory.sidebar.questRows.2"), nil
	}
	return fmt.Sprintf(i18n.T(lang, "dragonfactory.sidebar.questRows.3"), r.Title(lang)), s.researchCostRows(lang, r, inv)
}

// researchCostRows are what a research spends, one row per item: how many
// of it inv holds out of the number still needed (green once there are
// enough), or the number alone when inv is nil.
func (s *session) researchCostRows(lang string, r *Research, inv *inventory.Inventory) []string {
	var rows []string
	for _, c := range r.Cost {
		name, need := itemName(lang, item.NewStack(c.Item, 1)), s.researchNeed(r, c)
		if inv == nil {
			rows = append(rows, fmt.Sprintf("§f%s §e×%d", name, need))
			continue
		}
		have, colour := min(countPlain(inv, c.Item), need), "§e"
		if have >= need {
			colour = "§a"
		}
		rows = append(rows, fmt.Sprintf("§f%s %s%d§7/%d", name, colour, have, need))
	}
	return rows
}

// nextGoal is a one line summary of the next thing to do (menu).
func (s *session) nextGoal(lang string) string {
	if a, ok := s.nextAdvancement(); ok {
		return i18n.T(lang, "dragonfactory.sidebar.nextGoal.1") + a.Title(lang) + " — " + flatHint(lang, a)
	}
	if r, ok := s.nextResearch(); ok {
		return i18n.T(lang, "dragonfactory.sidebar.nextGoal.2") + r.Title(lang) + i18n.T(lang, "dragonfactory.sidebar.nextGoal.3")
	}
	return i18n.T(lang, "dragonfactory.sidebar.nextGoal.4")
}

// formatPlayTime formats the total instance time in whole minutes in lang.
func formatPlayTime(lang string, d time.Duration) string {
	m := int(d / time.Minute)
	if m < 60 {
		return fmt.Sprintf(i18n.T(lang, "dragonfactory.sidebar.formatPlayTime.1"), m)
	}
	return fmt.Sprintf(i18n.T(lang, "dragonfactory.sidebar.formatPlayTime.2"), m/60, m%60)
}

// isOperator reports whether p sees the operator tools.
func (s *session) isOperator(p *player.Player) bool {
	return s.g.isOperator != nil && s.g.isOperator(p)
}

// showsLoad reports whether p sees the server load line (an operator who
// turned it on with /dfadmin tps).
func (s *session) showsLoad(p *player.Player) bool {
	return s.isOperator(p) && perf.Shown(p.Name())
}

// updateBoards refreshes every player's HUD. It runs every second (the play
// time, the server load and the counts of the cost rows change) and after
// the events that change the quest panel.
func (s *session) updateBoards(tx *world.Tx) {
	for e := range tx.Players() {
		s.updateBoard(e.(*player.Player))
	}
}
