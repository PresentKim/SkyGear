package instance

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	// placeTries is how often a player is put back on its spot while the
	// chunks around it load; placeEvery is the gap between the tries. An
	// instance world loads its chunks in the background when it activates,
	// so the ground under a player entering a save is air for a moment.
	placeTries = 50
	placeEvery = 100 * time.Millisecond
	// placeReleaseTimeout frees the player even if the chunks never load or
	// the client never answers the latency check, so it cannot stay frozen.
	placeReleaseTimeout = placeTries*placeEvery + time.Second
)

// PlaceAt puts p on the nearest safe ground at want and holds it still until
// the chunks around that spot are loaded and the client has them, then lets it
// move again. fallback is used when no ground is found near want.
//
// A player added to a world whose chunks the server has not loaded yet is
// placed over air, and both the server and the client drop it: the server
// teleports it again every placeEvery (which also clears the fall distance),
// and the client only starts its own physics once it has the chunk data
// (AfterClientSync). It must be called in p's transaction.
func PlaceAt(p *player.Player, want, fallback mgl64.Vec3) {
	p.SetImmobile()
	h, w := p.H(), p.Tx().World()
	placeAt(h, w, want, fallback, 0)
	// Never leave the player frozen: if the ground or the client's answer
	// never arrives, put it on the ground and let it move.
	go func() {
		time.Sleep(placeReleaseTimeout)
		h.Do(func(tx *world.Tx, e world.Entity) {
			if tx.World() != w {
				return
			}
			p := e.(*player.Player)
			p.Teleport(LandingSpot(tx, p.Position(), fallback))
			p.SetMobile()
		})
	}()
}

// placeAt puts the player at want again after placeEvery until there is
// ground below the spot; then it waits for the client and lets it move.
func placeAt(h *world.EntityHandle, w *world.World, want, fallback mgl64.Vec3, n int) {
	h.DoAfter(placeEvery, func(tx *world.Tx, e world.Entity) {
		if tx.World() != w {
			return // it changed worlds meanwhile: the move there places it
		}
		p := e.(*player.Player)
		spot := LandingSpot(tx, want, fallback)
		p.Teleport(spot)
		if n < placeTries && !grounded(tx, cube.PosFromVec3(spot)) {
			placeAt(h, w, want, fallback, n+1)
			return
		}
		// The client has everything sent so far, the position and chunks
		// included: put it on the ground once more, then let it move.
		p.AfterClientSync(func(p *player.Player, tx *world.Tx) {
			if tx.World() != w {
				return
			}
			p.Teleport(LandingSpot(tx, p.Position(), fallback))
			p.SetMobile()
		})
	})
}
