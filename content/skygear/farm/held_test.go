package farm_test

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
)

// held are the players the test's takeover holds.
var held = map[*world.EntityHandle]bool{}

func init() {
	core.RegisterTakeover("farmtest", func(_ *core.Session, p *player.Player) bool { return held[p.H()] })
}

// TestHeldPlayerGrowsNoSapling: sneaking next to a sapling grows it by
// chance, but not for a player a takeover holds (island expansion mode, a
// hunt): that sneak belongs to the mode.
func TestHeldPlayerGrowsNoSapling(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.PosFromVec3(p.Position()).Add(cube.Pos{1, 0, 0})
		tx.SetBlock(pos.Side(cube.FaceDown), block.Dirt{}, nil)
		tx.SetBlock(pos, block.Sapling{Type: block.OakSapling()}, nil)
		hd := s.Handler(p).(*core.PlayerHandler)
		sneak := func() { hd.HandleToggleSneak(player.NewEventContext(tx, p), true) }
		sapling := func() bool {
			_, ok := tx.Block(pos).(block.Sapling)
			return ok
		}

		held[p.H()] = true
		for range 200 {
			sneak()
		}
		if b, ok := tx.Block(pos).(block.Sapling); !ok || b.Aged {
			t.Fatalf("a held player's sneaks grew the sapling: %#v", tx.Block(pos))
		}
		delete(held, p.H())
		for range 200 {
			if sneak(); !sapling() {
				break
			}
		}
		if sapling() {
			t.Fatal("a free player's sneaks did not grow the sapling")
		}
	})
}
