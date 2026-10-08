package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/scoreboard"
	"github.com/df-mc/dragonfly/server/world"
)

// updateBoard refreshes p's sidebar (docs/design/02-dragonfactory.md §11).
func (s *session) updateBoard(p *player.Player) {
	_, day, _ := s.clock.Now()
	b := scoreboard.New(fmt.Sprintf("§l§6드래곤팩토리 · Lv.%d", s.state.Level))
	lines := []string{
		fmt.Sprintf("§f%d일차", day),
		s.missionLine(),
		fmt.Sprintf("§f돈: §6%s", formatMoney(s.Money(p))),
	}
	lines = append(lines, s.boardLines()...)
	for i, l := range lines {
		b.Set(i, l)
	}
	p.SendScoreboard(b)
}

// boardLines are extra sidebar lines contributed by game systems.
func (s *session) boardLines() []string { return nil }

func (s *session) updateBoards(tx *world.Tx) {
	for e := range tx.Players() {
		s.updateBoard(e.(*player.Player))
	}
}
