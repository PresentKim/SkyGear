package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// BenchmarkHandleMove moves one player by one block through the session's
// handler, back and forth, so every call is a real step.
func BenchmarkHandleMove(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		hd := s.Handler(p).(*core.PlayerHandler)
		ctx := player.NewEventContext(tx, p)
		a, c := mgl64.Vec3{0.5, 64, 2.5}, mgl64.Vec3{1.5, 64, 2.5}
		b.ResetTimer()
		for i := range b.N {
			to := c
			if i%2 == 1 {
				to = a
			}
			hd.HandleMove(ctx, to, p.Rotation())
		}
	})
}

// BenchmarkSecond runs the once-a-second session step with one player.
func BenchmarkSecond(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		b.ResetTimer()
		for range b.N {
			s.Second(tx)
		}
	})
}

// BenchmarkUseOnBlock clicks a block of the island with a stack of stone
// through the session's handler: every step of the chain is reached and none
// stops (the "plain" scenario of TestPinUse).
func BenchmarkUseOnBlock(b *testing.B) {
	s := benchSession(b)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	h := spawnPlayer(s)
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Stone{}, 3), item.Stack{})
		hd := s.Handler(p)
		ctx := player.NewEventContext(tx, p)
		at := cube.Pos{4, island.IslandY + 2, 4}
		click := mgl64.Vec3{0.5, 0.5, 0.5}
		b.ResetTimer()
		for range b.N {
			hd.HandleItemUseOnBlock(ctx, at, cube.FaceUp, click)
		}
	})
}

// BenchmarkUse uses a stack of stone in the air through the session's handler
// (the "plain" scenario of TestPinUse).
func BenchmarkUse(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Stone{}, 3), item.Stack{})
		hd := s.Handler(p)
		ctx := player.NewEventContext(tx, p)
		b.ResetTimer()
		for range b.N {
			hd.HandleItemUse(ctx)
		}
	})
}

// BenchmarkLookMachine updates the WAILA panel of a player who looks at a
// working crusher: the trace to the block, the panel's lines and the title.
func BenchmarkLookMachine(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	target := cube.Pos{4, island.IslandY + 3, 8}
	m := machine.New(machine.KindCrusher)
	m.State.Insert(item.NewStack(block.Cobblestone{}, 5))
	m.State.Progress = m.State.Def.Interval / 2
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(target, m, nil)
		pos, rot := eyeAt(p, target, 3)
		if at, _, ok := look.LookedAt(p, tx, pos, rot); !ok || at != target {
			b.Fatalf("the player looks at %v, want the crusher at %v", at, target)
		}
		b.ResetTimer()
		for range b.N {
			look.Update(s.Session, p, tx, pos, rot, true)
		}
	})
}

// BenchmarkLookInfoMachine builds the panel of a working crusher (look.Info).
func BenchmarkLookInfoMachine(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	target := cube.Pos{4, island.IslandY + 3, 8}
	m := machine.New(machine.KindCrusher)
	m.State.Insert(item.NewStack(block.Cobblestone{}, 5))
	m.State.Progress = m.State.Def.Interval / 2
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		b.ResetTimer()
		for range b.N {
			look.Info(s.Session, p, tx, target, m)
		}
	})
}

// BenchmarkLookInfoStone builds the panel of a block the game adds nothing to.
func BenchmarkLookInfoStone(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	target := cube.Pos{4, island.IslandY + 3, 8}
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		b.ResetTimer()
		for range b.N {
			look.Info(s.Session, p, tx, target, block.Stone{})
		}
	})
}
