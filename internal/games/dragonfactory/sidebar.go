package dragonfactory

import (
	"fmt"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/hud"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/perf"
)

// updateBoard refreshes p's HUD (docs/design/02-dragonfactory.md §16): day
// and play time at the top left (with the server load under it for an
// operator who turned it on) and the next thing to do as a quest panel: the
// first research that can be done now, with its cost. The texts travel as
// sidebar rows (internal/hud).
func (s *session) updateBoard(p *player.Player) {
	total, day, _ := s.clock.Now()
	var b hud.Board
	b.Add(hud.TopLeft, fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.sidebar.updateBoard.1"), day, formatPlayTime(total)))
	if s.showsLoad(p) {
		b.Add(hud.TopLeft, perf.Line(s.world))
	}
	title, rows := s.questRows()
	b.Add(hud.QuestTitle, title)
	for _, r := range rows {
		b.Add(hud.Quest, r)
	}
	s.addGoalRows(&b)
	b.Send(p)
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
// advancement is done the next research and its cost.
func (s *session) questRows() (string, []string) {
	if a, ok := s.nextAdvancement(); ok {
		rows := strings.Split(a.Hint, "\n")
		for i := range rows {
			rows[i] = "§f" + rows[i]
		}
		if g, ok := goalResearch(a); ok {
			if r, ok := s.researchStep(g); ok {
				if r != g || a.Research == "" {
					rows = append(rows, fmt.Sprintf(i18n.T(i18n.Default, "sidebar.quest.research"), r.Title(i18n.Default)))
				}
				rows = append(rows, s.researchCostRows(r)...)
			}
		}
		return i18n.T(i18n.Default, "dragonfactory.sidebar.questRows.1") + a.Name, rows
	}
	r, ok := s.nextResearch()
	if !ok {
		return i18n.T(i18n.Default, "dragonfactory.sidebar.questRows.2"), nil
	}
	return fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.sidebar.questRows.3"), r.Title(i18n.Default)), s.researchCostRows(r)
}

// researchCostRows are what a research spends, one row per item.
func (s *session) researchCostRows(r *Research) []string {
	var rows []string
	for _, c := range r.Cost {
		rows = append(rows, fmt.Sprintf("§f%s §e×%d", itemName(item.NewStack(c.Item, 1)), s.researchNeed(r, c)))
	}
	return rows
}

// nextGoal is a one line summary of the next thing to do (menu).
func (s *session) nextGoal() string {
	if a, ok := s.nextAdvancement(); ok {
		return i18n.T(i18n.Default, "dragonfactory.sidebar.nextGoal.1") + a.Name + " — " + flatHint(a)
	}
	if r, ok := s.nextResearch(); ok {
		return i18n.T(i18n.Default, "dragonfactory.sidebar.nextGoal.2") + r.Title(i18n.Default) + i18n.T(i18n.Default, "dragonfactory.sidebar.nextGoal.3")
	}
	return i18n.T(i18n.Default, "dragonfactory.sidebar.nextGoal.4")
}

// formatPlayTime formats the total instance time in whole minutes.
func formatPlayTime(d time.Duration) string {
	m := int(d / time.Minute)
	if m < 60 {
		return fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.sidebar.formatPlayTime.1"), m)
	}
	return fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.sidebar.formatPlayTime.2"), m/60, m%60)
}

// refreshPlayTime updates the HUDs when the shown play time minute
// changes. It runs with the autosave, so they change at most twice a
// minute.
func (s *session) refreshPlayTime(tx *world.Tx) {
	total, _, _ := s.clock.Now()
	if m := int(total / time.Minute); m != s.boardMinute {
		s.boardMinute = m
		s.updateBoards(tx)
	}
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

// updateLoadBoards refreshes the HUDs that show the server load, which
// changes every second.
func (s *session) updateLoadBoards(tx *world.Tx) {
	for e := range tx.Players() {
		if p := e.(*player.Player); s.showsLoad(p) {
			s.updateBoard(p)
		}
	}
}

func (s *session) updateBoards(tx *world.Tx) {
	for e := range tx.Players() {
		s.updateBoard(e.(*player.Player))
	}
}
