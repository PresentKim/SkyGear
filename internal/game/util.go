package game

import (
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/dragonfactory/internal/store"
)

func cubePos(p *player.Player) cube.Pos { return cube.PosFromVec3(p.Position()) }

// itemName returns a readable name for an item stack.
func itemName(s item.Stack) string {
	if s.CustomName() != "" {
		return s.CustomName()
	}
	name, _ := s.Item().EncodeItem()
	return strings.TrimPrefix(name, "minecraft:")
}

// addMoney gives amount to p and refreshes its sidebar.
func (g *Game) addMoney(p *player.Player, amount int64) {
	g.store.Do(func(tx store.Tx) {
		tx.Profile(p.UUID(), p.Name()).Money += amount
	})
	g.UpdateBoard(p)
}

// takeMoney removes amount from p if it has enough.
func (g *Game) takeMoney(p *player.Player, amount int64) bool {
	ok := false
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		if prof.Money >= amount {
			prof.Money -= amount
			ok = true
		}
	})
	if ok {
		g.UpdateBoard(p)
	}
	return ok
}
