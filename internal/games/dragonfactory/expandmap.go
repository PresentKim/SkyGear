package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/dfhook"
)

// The expansion map is a flattened copy of the island sent as fake chunks
// (Player.ViewFakeChunk) mapOffset chunks away, with the floor glass on top.
// The client only draws chunks around the player, so the frozen body follows
// the cursor above the camera there and goes back when the view ends.

// mapBodyAbove is how far above the camera the body waits.
const mapBodyAbove = 4

// mapOffset is how far (in chunks, on x and z) the map is from the island:
// about 30000 blocks, past the islands of the world.
const mapOffset = 1875

// mapChunk is where island chunk c is shown on the map.
func mapChunk(c ChunkPos) world.ChunkPos {
	return world.ChunkPos{c[0] + mapOffset, c[1] + mapOffset}
}

// mapShift moves an island position onto the map.
func mapShift(v mgl64.Vec3) mgl64.Vec3 {
	return v.Add(mgl64.Vec3{mapOffset * 16, 0, mapOffset * 16})
}

// mapKinds returns the glass of each chunk drawn near the cursor: owned
// chunks in the two greens, buyable ones yellow.
func (s *session) mapKinds(e *expansion) map[ChunkPos]world.Block {
	want := map[ChunkPos]world.Block{}
	near := int32(expandMapReach)
	add := func(c ChunkPos, b world.Block) {
		if abs(c[0]-e.target[0]) <= near && abs(c[1]-e.target[1]) <= near {
			want[c] = b
		}
	}
	for _, c := range s.state.Chunks {
		b := world.Block(floorOwned)
		if (c[0]+c[1])%2 != 0 {
			b = floorOwnedAlt
		}
		add(c, b)
	}
	for _, c := range s.purchasableChunks() {
		add(c, floorBuyable)
	}
	return want
}

// drawMap brings the map up to date: chunks whose glass changed or that
// came near the cursor are sent. on is false to end the map view.
func (s *session) drawMap(p *player.Player, e *expansion, on bool) {
	if !on {
		if e.mapShown == nil {
			return
		}
		for c := range e.mapShown {
			p.ClearFakeChunk(mapChunk(c))
		}
		e.mapShown = nil
		p.Teleport(e.home)
		return
	}
	if e.mapShown == nil {
		e.mapShown = map[ChunkPos]world.Block{}
	}
	tx := p.Tx()
	// The body goes over the cursor first; the client takes chunks around
	// it from then on (ViewChunkCentre: at once, not at the next chunk
	// send).
	body := s.expandCamera(e).Add(mgl64.Vec3{0, mapBodyAbove, 0})
	p.Teleport(body)
	dfhook.ViewChunkCentre(p, body)
	for c, glass := range s.mapKinds(e) {
		if b, ok := e.mapShown[c]; ok && b == glass {
			continue
		}
		e.mapShown[c] = glass
		p.ViewFakeChunk(mapChunk(c), s.mapColumn(tx, c, glass))
	}
}

// mapColumn builds the map chunk of island chunk c: the top block of each
// column just under the island surface, glass on the surface. A top block
// light passes through (torch, crop, slab, glass, our own models) gets the
// next block below it underneath, so the map has no holes. Only owned
// chunks are read (others are empty void and may not be loaded).
func (s *session) mapColumn(tx *world.Tx, c ChunkPos, glass world.Block) *chunk.Chunk {
	reg := tx.World().BlockRegistry()
	col := chunk.New(reg, tx.Range())
	glassID := reg.BlockRuntimeID(glass)
	owned := s.owns(c)
	low := tx.Range().Min()
	for x := range 16 {
		for z := range 16 {
			col.SetBlock(uint8(x), int16(expandFloorY), uint8(z), 0, glassID)
			if !owned {
				continue
			}
			wx, wz := int(c[0])*16+x, int(c[1])*16+z
			y := tx.HighestBlock(wx, wz)
			top := tx.Block(cube.Pos{wx, y, wz})
			if _, air := top.(block.Air); air || y < low {
				continue
			}
			col.SetBlock(uint8(x), int16(expandFloorY-1), uint8(z), 0, reg.BlockRuntimeID(top))
			if !seeThrough(top) {
				continue
			}
			for y--; y >= low; y-- {
				if b := tx.Block(cube.Pos{wx, y, wz}); !isAir(b) {
					col.SetBlock(uint8(x), int16(expandFloorY-2), uint8(z), 0, reg.BlockRuntimeID(b))
					break
				}
			}
		}
	}
	return col
}

// seeThrough reports if what is below b shows from above: b is not a full
// opaque cube.
func seeThrough(b world.Block) bool {
	if _, custom := b.(world.CustomBlock); custom {
		return true
	}
	if d, ok := b.(block.LightDiffuser); ok && d.LightDiffusionLevel() < 15 {
		return true
	}
	_, solid := b.Model().(model.Solid)
	return !solid
}

func isAir(b world.Block) bool {
	_, air := b.(block.Air)
	return air
}
