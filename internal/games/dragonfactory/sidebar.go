package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/hud"
	"github.com/presentkim/dragonfactory/internal/perf"
)

// updateBoard refreshes p's HUD (docs/design/02-dragonfactory.md §11): day
// and play time at the top left (with the server load under it for an
// operator who turned it on), money at the top right, the mission as a
// quest panel, and the level as the experience level with the mission's
// progress on the experience bar. The texts travel as sidebar rows
// (internal/hud).
func (s *session) updateBoard(p *player.Player) {
	total, day, _ := s.clock.Now()
	var b hud.Board
	b.Add(hud.TopLeft, fmt.Sprintf("§f%d일차 §7· 플레이 %s", day, formatPlayTime(total)))
	if s.showsLoad(p) {
		b.Add(hud.TopLeft, perf.Line(s.world))
	}
	b.Add(hud.TopRight, fmt.Sprintf("§6%s", formatMoney(s.Money(p))))
	title, rows := s.questRows()
	b.Add(hud.QuestTitle, title)
	for _, r := range rows {
		b.Add(hud.Quest, r)
	}
	b.Send(p)
	p.SetExperienceLevel(s.state.Level)
	p.SetExperienceProgress(s.missionProgress())
}

// questRows are the quest panel of the current mission: its title and a
// row per item to deliver.
func (s *session) questRows() (string, []string) {
	m, ok := s.mission()
	if !ok {
		return "§a모든 미션 완료", nil
	}
	var rows []string
	for _, r := range m.Requirements {
		name := itemName(item.NewStack(r.Item, 1))
		if got := min(s.state.Delivered[itemKey(r.Item)], r.Count); got < r.Count {
			rows = append(rows, fmt.Sprintf("§f%s §e%d§7/%d", name, got, r.Count))
		} else {
			rows = append(rows, fmt.Sprintf("§a%s %d/%d", name, r.Count, r.Count))
		}
	}
	return fmt.Sprintf("§e미션 §f%s", m.Name), rows
}

// missionProgress is the share of the current mission's items delivered,
// 0 to 1 (1 when every mission is done).
func (s *session) missionProgress() float64 {
	m, ok := s.mission()
	if !ok {
		return 1
	}
	var got, need int
	for _, r := range m.Requirements {
		got += min(s.state.Delivered[itemKey(r.Item)], r.Count)
		need += r.Count
	}
	if need == 0 {
		return 1
	}
	return float64(got) / float64(need)
}

// formatPlayTime formats the total instance time in whole minutes.
func formatPlayTime(d time.Duration) string {
	m := int(d / time.Minute)
	if m < 60 {
		return fmt.Sprintf("%d분", m)
	}
	return fmt.Sprintf("%d시간 %d분", m/60, m%60)
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
