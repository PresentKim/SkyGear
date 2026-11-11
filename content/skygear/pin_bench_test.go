package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// BenchmarkHandleMove moves one player by one block through the session's
// handler, back and forth, so every call is a real step.
func BenchmarkHandleMove(b *testing.B) {
	s := benchSession(b)
	h := spawnPlayer(s)
	b.ReportAllocs()
	withPlayer(b, s, h, func(p *player.Player, tx *world.Tx) {
		hd := s.Handler(p).(*handler)
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
