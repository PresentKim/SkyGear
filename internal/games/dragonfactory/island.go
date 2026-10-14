package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"math"
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/ui"
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

// chunkCost is what the next chunk costs when n chunks are owned: items
// from the buyer's inventory (13th playtest: no money). 4 chunks to start:
// 64 cobblestone for the fifth, 64 more for each after. Draft.
func chunkCost(n int) Requirement {
	return Requirement{block.Cobblestone{}, 64 * max(1, n-3)}
}

// costText describes a cost, e.g. "조약돌 128개".
func costText(r Requirement) string {
	return fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.island.costText.1"), itemName(item.NewStack(r.Item, 1)), formatInt(int64(r.Count)))
}

// countPlain counts the plain items of it in an inventory.
func countPlain(inv *inventory.Inventory, it world.Item) int {
	n, key := 0, itemKey(it)
	for _, st := range inv.Slots() {
		if !st.Empty() && plain(st) && itemKey(st.Item()) == key {
			n += st.Count()
		}
	}
	return n
}

// takePlain takes count plain items of it from an inventory if it holds
// that many.
func takePlain(inv *inventory.Inventory, it world.Item, count int) bool {
	if countPlain(inv, it) < count {
		return false
	}
	key := itemKey(it)
	for slot, st := range inv.Slots() {
		if count == 0 {
			break
		}
		if st.Empty() || !plain(st) || itemKey(st.Item()) != key {
			continue
		}
		n := min(count, st.Count())
		_ = inv.SetItem(slot, st.Grow(-n))
		count -= n
	}
	return true
}

// islandRadius is how far (in chunks from the centre) the island may grow.
// Owned chunks outside the instance load radius get their own loaders
// (instance.KeepLoaded). Balancing draft.
const islandRadius = 12

// purchasableChunks returns the chunks next to the island within
// islandRadius.
func (s *session) purchasableChunks() []ChunkPos {
	r := int32(islandRadius)
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
		ns = i18n.T(i18n.Default, "dragonfactory.island.direction.1")
	} else if dz > 8 {
		ns = i18n.T(i18n.Default, "dragonfactory.island.direction.2")
	}
	if dx < -8 {
		ew = i18n.T(i18n.Default, "dragonfactory.island.direction.3")
	} else if dx > 8 {
		ew = i18n.T(i18n.Default, "dragonfactory.island.direction.4")
	}
	if ns+ew == "" {
		return i18n.T(i18n.Default, "dragonfactory.island.direction.5")
	}
	return ns + ew + i18n.T(i18n.Default, "dragonfactory.island.direction.6")
}

func (s *session) buyChunk(p *player.Player, tx *world.Tx, c ChunkPos) {
	if !slices.Contains(s.purchasableChunks(), c) {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "dragonfactory.island.buyChunk.1"))
		return
	}
	cost := chunkCost(len(s.state.Chunks))
	if !takePlain(p.Inventory(), cost.Item, cost.Count) {
		ui.Notify(p, i18n.T(i18n.Of(p), "dragonfactory.island.buyChunk.2"), costText(cost))
		return
	}
	s.state.Chunks = append(s.state.Chunks, c)
	if len(s.state.Chunks) >= 10 {
		account.Grant(p, achChunks)
	}
	s.keepLoaded(tx, c)
	s.storeState()
	s.advance(tx, advChunk)
	if len(s.state.Chunks) >= 8 {
		s.advance(tx, advChunks8)
	}
	if len(s.state.Chunks) >= 16 {
		s.advance(tx, advChunks16)
	}
	s.broadcast(tx, fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.island.buyChunk.3"), p.Name(), c[0], c[1], len(s.state.Chunks)))
}

// keepLoaded keeps an owned chunk loaded and ticking while the instance is
// active.
func (s *session) keepLoaded(tx *world.Tx, c ChunkPos) {
	if s.keep != nil {
		s.keep(tx, c)
	}
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
	p.SendTip(i18n.T(i18n.Of(p), "dragonfactory.island.returnFromVoid.1"))
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
