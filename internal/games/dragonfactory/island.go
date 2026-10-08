package dragonfactory

import (
	"fmt"
	"math"
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// VoidReturnY is the height players are put at above the nearest block when
// they fall into the void (docs/design/02-dragonfactory.md §9.2).
const VoidReturnY = 300

// VoidY is the height below which a player counts as fallen into the void.
const VoidY = 0

func chunkOf(pos cube.Pos) ChunkPos {
	return ChunkPos{int32(pos.X() >> 4), int32(pos.Z() >> 4)}
}

// owns reports if the island owns the chunk.
func (s *session) owns(c ChunkPos) bool { return slices.Contains(s.state.Chunks, c) }

// canBuild reports if blocks at pos may be changed.
func (s *session) canBuild(pos cube.Pos) bool { return s.owns(chunkOf(pos)) }

// chunkPrice is the price of the next chunk when n chunks are owned.
func chunkPrice(n int) int64 {
	// 4 chunks to start: 1,000G for the fifth, then ×1.5 each.
	p := 1000 * math.Pow(1.5, float64(n-4))
	return int64(math.Round(p/100) * 100)
}

// purchasableChunks returns the chunks next to the island that are within the
// instance's load radius (owned chunks must stay loaded and ticked).
func (s *session) purchasableChunks() []ChunkPos {
	r := int32(s.radius)
	var out []ChunkPos
	for _, c := range s.state.Chunks {
		for _, d := range []ChunkPos{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := ChunkPos{c[0] + d[0], c[1] + d[1]}
			if n[0] < -r || n[0] > r || n[1] < -r || n[1] > r || s.owns(n) || slices.Contains(out, n) {
				continue
			}
			out = append(out, n)
		}
	}
	return out
}

func direction(from mgl64.Vec3, c ChunkPos) string {
	dx := float64(c[0])*16 + 8 - from.X()
	dz := float64(c[1])*16 + 8 - from.Z()
	var ns, ew string
	if dz < -8 {
		ns = "북"
	} else if dz > 8 {
		ns = "남"
	}
	if dx < -8 {
		ew = "서"
	} else if dx > 8 {
		ew = "동"
	}
	if ns+ew == "" {
		return "현재 위치"
	}
	return ns + ew + "쪽"
}

func (s *session) buyChunk(p *player.Player, tx *world.Tx, c ChunkPos) {
	if !slices.Contains(s.purchasableChunks(), c) {
		p.Message("§c이 청크는 살 수 없습니다.")
		return
	}
	price := chunkPrice(len(s.state.Chunks))
	if !s.TakeMoney(p, price) {
		p.Message("§c돈이 부족합니다.")
		return
	}
	s.state.Chunks = append(s.state.Chunks, c)
	s.storeState()
	s.broadcast(tx, fmt.Sprintf("§a%s 님이 청크 (%d, %d)를 구매했습니다. (소유 %d개)", p.Name(), c[0], c[1], len(s.state.Chunks)))
}

// returnFromVoid puts p high above the block nearest to where it fell.
func (s *session) returnFromVoid(p *player.Player, tx *world.Tx) {
	pos := p.Position()
	x, z, ok := s.nearestBlockColumn(tx, int(math.Floor(pos.X())), int(math.Floor(pos.Z())))
	if !ok {
		p.Teleport(s.Spawn())
		return
	}
	p.Teleport(mgl64.Vec3{float64(x) + 0.5, VoidReturnY, float64(z) + 0.5})
	p.SendTip("§e공허에서 돌아왔습니다.")
}

// nearestBlockColumn searches outwards from (x, z) for a column with a block.
func (s *session) nearestBlockColumn(tx *world.Tx, x, z int) (int, int, bool) {
	has := func(x, z int) bool {
		y := tx.HighestBlock(x, z)
		_, air := tx.Block(cube.Pos{x, y, z}).(block.Air)
		return y > tx.Range().Min() && !air
	}
	const maxRadius = 48
	for r := 0; r <= maxRadius; r++ {
		best, found := 0, false
		var bx, bz int
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if max(abs(dx), abs(dz)) != r || !has(x+dx, z+dz) {
					continue
				}
				if d := dx*dx + dz*dz; !found || d < best {
					best, bx, bz, found = d, x+dx, z+dz, true
				}
			}
		}
		if found {
			return bx, bz, true
		}
	}
	return 0, 0, false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
