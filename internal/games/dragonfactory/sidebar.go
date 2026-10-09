package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/scoreboard"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/perf"
)

// updateBoard refreshes p's sidebar (docs/design/02-dragonfactory.md §11).
func (s *session) updateBoard(p *player.Player) {
	total, day, _ := s.clock.Now()
	b := scoreboard.New(fmt.Sprintf("§l§6드래곤팩토리 · Lv.%d", s.state.Level))
	lines := []string{
		fmt.Sprintf("§f%d일차 §7· 플레이 %s", day, formatPlayTime(total)),
		s.missionLine(),
		fmt.Sprintf("§f돈: §6%s", formatMoney(s.Money(p))),
	}
	lines = append(lines, s.boardLines()...)
	if s.isOperator(p) {
		lines = append(lines, perf.Lines(s.world)...)
	}
	for i, l := range lines {
		b.Set(i, l)
	}
	p.SendScoreboard(b)
}

// formatPlayTime formats the total instance time in whole minutes.
func formatPlayTime(d time.Duration) string {
	m := int(d / time.Minute)
	if m < 60 {
		return fmt.Sprintf("%d분", m)
	}
	return fmt.Sprintf("%d시간 %d분", m/60, m%60)
}

// refreshPlayTime updates the sidebars when the shown play time minute
// changes. It runs with the autosave, so boards change at most twice a
// minute.
func (s *session) refreshPlayTime(tx *world.Tx) {
	total, _, _ := s.clock.Now()
	if m := int(total / time.Minute); m != s.boardMinute {
		s.boardMinute = m
		s.updateBoards(tx)
	}
}

// boardLines are extra sidebar lines contributed by game systems. The
// urgent mission has its own boss bar.
func (s *session) boardLines() []string { return nil }

// isOperator reports whether p sees the operator tools and the server load.
func (s *session) isOperator(p *player.Player) bool {
	return s.g.isOperator != nil && s.g.isOperator(p)
}

// updateLoadBoards refreshes the sidebars of operators, whose server load
// lines change every second.
func (s *session) updateLoadBoards(tx *world.Tx) {
	for e := range tx.Players() {
		if p := e.(*player.Player); s.isOperator(p) {
			s.updateBoard(p)
		}
	}
}

func (s *session) updateBoards(tx *world.Tx) {
	for e := range tx.Players() {
		s.updateBoard(e.(*player.Player))
	}
}
