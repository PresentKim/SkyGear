// Package island is a save's island: the chunks it owns (where players may
// build), where its players start, buying chunks next to it, the void below
// it and the lost-and-found box that keeps what fell into the void.
package island

import (
	"fmt"
	"math"
	"slices"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/platform/account"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// IslandY is the Y level of the island surface.
const IslandY = 64

// Centre is the block (x and z) at the middle of the starting chunk,
// where the starting island is built.
const Centre = 8

// VoidReturnY is the height players are put at above the nearest block when
// they fall into the void.
const VoidReturnY = 300

// VoidY is the height below which a player counts as fallen into the void.
const VoidY = 0

func ChunkOf(pos cube.Pos) ChunkPos {
	return ChunkPos{int32(pos.X() >> 4), int32(pos.Z() >> 4)}
}

// ChunkCost is what the next chunk costs when n chunks are owned: items
// from the buyer's inventory. 1 chunk to start:
// 64 cobblestone for each of the next four, 64 more for each after. Draft.
func ChunkCost(n int) progress.Requirement {
	return progress.Requirement{Item: block.Cobblestone{}, Count: 64 * max(1, n-3)}
}

// CostText describes a cost, e.g. "조약돌 128개".
func CostText(lang string, r progress.Requirement) string {
	return fmt.Sprintf(i18n.T(lang, "island.chunk.cost"), naming.ItemName(lang, item.NewStack(r.Item, 1)), textfmt.Int(int64(r.Count)))
}

// Radius is how far (in chunks from the centre) the island may grow.
// Owned chunks outside the instance load radius get their own loaders
// (instance.KeepLoaded). Balancing draft.
const Radius = 12

// ChunkPos is a chunk coordinate.
type ChunkPos [2]int32

// State is what a save keeps of its island: its fields are part of the
// save's state document (core.Saved).
type State struct {
	// Chunks are the owned island chunks.
	Chunks []ChunkPos `json:"chunks"`
	// Home is where the island's players start (x, z); zero for a save made
	// when the island was around the origin.
	Home [2]float64 `json:"home,omitempty"`
	// Lost is the lost-and-found box, most recent first.
	Lost []*LostItem `json:"lost,omitempty"`
}

var saved = core.NewSaved(func() *State {
	// The starting island covers one chunk, with the 3×3×3 island in its
	// centre.
	return &State{Chunks: []ChunkPos{{0, 0}}, Home: [2]float64{Centre + 0.5, Centre + 0.5}}
})

// StateOf is the island of the save of s.
func StateOf(s *core.Session) *State { return saved.Of(s) }

// keep keeps an owned chunk of a save loaded (instance.KeepLoaded); nil in
// tests.
var keep = core.NewPart[func(tx *world.Tx, c ChunkPos)](nil)

// SetKeep sets how the save of s keeps its owned chunks loaded and ticking
// while it runs.
func SetKeep(s *core.Session, f func(tx *world.Tx, c ChunkPos)) { *keep.Of(s) = f }

// KeepLoaded keeps an owned chunk loaded and ticking while the save runs.
func KeepLoaded(s *core.Session, tx *world.Tx, c ChunkPos) {
	if f := *keep.Of(s); f != nil {
		f(tx, c)
	}
}

// Spawn is next to the tree of the starting island: Home is the island's
// centre, where the trunk grows, so the spawn is one block east of it. An old
// save without Home has its island around the origin.
func Spawn(s *core.Session) mgl64.Vec3 {
	home := [2]float64{1.5, 1.5}
	if h := StateOf(s).Home; h != ([2]float64{}) {
		home = h
	}
	return mgl64.Vec3{home[0] + 1, IslandY, home[1]}
}

func Owns(s *core.Session, c ChunkPos) bool {
	return slices.Contains(StateOf(s).Chunks, c)
}

// CanBuild reports if blocks at pos may be changed.
func CanBuild(s *core.Session, pos cube.Pos) bool { return Owns(s, ChunkOf(pos)) }

// Purchasable returns the chunks next to the island within
// islandRadius.
func Purchasable(s *core.Session) []ChunkPos {
	r := int32(Radius)
	var out []ChunkPos
	for _, c := range StateOf(s).Chunks {
		for _, d := range []ChunkPos{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := ChunkPos{c[0] + d[0], c[1] + d[1]}
			if n[0] < -r || n[0] > r || n[1] < -r || n[1] > r || Owns(s, n) || slices.Contains(out, n) {
				continue
			}
			out = append(out, n)
		}
	}
	return out
}

func Buy(s *core.Session, p *player.Player, tx *world.Tx, c ChunkPos) {
	if !slices.Contains(Purchasable(s), c) {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "island.chunk.notBuyable"))
		return
	}
	cost := ChunkCost(len(StateOf(s).Chunks))
	if !core.TakePlain(p.Inventory(), cost.Item, cost.Count) {
		ui.Notify(p, i18n.T(i18n.Of(p), "island.chunk.needs"), CostText(i18n.Of(p), cost))
		return
	}
	StateOf(s).Chunks = append(StateOf(s).Chunks, c)
	achievement.Count(p, "chunk_buy", 1)
	if len(StateOf(s).Chunks) >= 10 {
		account.Grant(p, achievement.Chunks10)
	}
	KeepLoaded(s, tx, c)
	s.MarkDirty()
	progress.Advance(s, tx, advancement.Chunk)
	if len(StateOf(s).Chunks) >= 8 {
		progress.Advance(s, tx, advancement.Chunks8)
	}
	if len(StateOf(s).Chunks) >= 16 {
		progress.Advance(s, tx, advancement.Chunks16)
	}
	s.Broadcast(tx, "island.chunk.bought", p.Name(), c[0], c[1], len(StateOf(s).Chunks))
}

// ReturnFromVoid puts p high above the block nearest to where it fell.
func ReturnFromVoid(s *core.Session, p *player.Player, tx *world.Tx) {
	achievement.Count(p, "void_return", 1)
	pos := p.Position()
	x, z, ok := NearestBlockColumn(s, tx, int(math.Floor(pos.X())), int(math.Floor(pos.Z())))
	if !ok {
		p.Teleport(Spawn(s))
		return
	}
	p.Teleport(mgl64.Vec3{float64(x) + 0.5, VoidReturnY, float64(z) + 0.5})
	p.SendTip(i18n.T(i18n.Of(p), "island.returnedFromVoid"))
}

// NearestBlockColumn searches outwards from (x, z) for a column with a block.
func NearestBlockColumn(s *core.Session, tx *world.Tx, x, z int) (int, int, bool) {
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
				if max(core.Abs(dx), core.Abs(dz)) != r || !has(x+dx, z+dz) {
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
