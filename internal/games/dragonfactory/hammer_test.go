package dragonfactory

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestHammerCrushesInPlace(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 1}
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(pos, block.Cobblestone{}, nil) })
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		hit := func() *item.UseContext {
			ctx := &item.UseContext{}
			if !(Hammer{}).UseOnBlock(pos, cube.FaceUp, [3]float64{}, tx, p, ctx) {
				t.Fatal("hammer not used")
			}
			s.hammering[p.H()].last = time.Time{} // skip the cooldown
			return ctx
		}
		for range hammerHits - 1 {
			if ctx := hit(); ctx.Damage != 0 {
				t.Fatal("damaged before crushing")
			}
		}
		if _, ok := tx.Block(pos).(block.Cobblestone); !ok {
			t.Fatalf("crushed early: %T", tx.Block(pos))
		}
		s.hammering[p.H()].last = time.Time{}
		ctx := &item.UseContext{}
		(Hammer{}).UseOnBlock(pos, cube.FaceUp, [3]float64{}, tx, p, ctx)
		if _, ok := tx.Block(pos).(block.Gravel); !ok || ctx.Damage != 1 {
			t.Fatalf("after %d hits: %T, damage %d", hammerHits, tx.Block(pos), ctx.Damage)
		}
		tx.SetBlock(pos, block.Dirt{}, nil)
		if (Hammer{}).UseOnBlock(pos, cube.FaceUp, [3]float64{}, tx, p, &item.UseContext{}) {
			t.Fatal("hammer used on dirt")
		}
	})
}
