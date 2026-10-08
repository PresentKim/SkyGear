package dragonfactory

import (
	"fmt"
	"math"
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/chestui"
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

// Chunk map (섬 확장): a 9×5 grid of chunks with north up, centred on the
// player's chunk, plus a control row. Clicking a chunk that can be bought
// asks to buy it.
const (
	mapCols = 9
	mapRows = 5
)

func (s *session) showExpand(p *player.Player) {
	s.showChunkBorders(p)
	s.expandMenu(p, chunkOf(cube.PosFromVec3(p.Position()))).Open(p, nil)
}

// expandMenu returns the chunk map centred on centre.
func (s *session) expandMenu(p *player.Player, centre ChunkPos) *chestui.Menu {
	m := chestui.NewMenu(mapCols*(mapRows+1), "섬 확장 · 위가 북쪽")
	var fill func(p *player.Player)
	fill = func(p *player.Player) {
		m.Clear()
		here := chunkOf(cube.PosFromVec3(p.Position()))
		buyable := s.purchasableChunks()
		price := chunkPrice(len(s.state.Chunks))
		for row := range mapRows {
			for col := range mapCols {
				c := ChunkPos{centre[0] + int32(col-mapCols/2), centre[1] + int32(row-mapRows/2)}
				m.Set(row*mapCols+col, s.chunkButton(p, c, here, slices.Contains(buyable, c), price))
			}
		}
		last := mapRows * mapCols
		pan := func(slot int, name string, d ChunkPos) {
			m.Set(last+slot, s.button(chestui.Lore(item.NewStack(item.Arrow{}, 1), name), func(p *player.Player, _ *world.Tx) {
				centre = ChunkPos{centre[0] + d[0], centre[1] + d[1]}
				fill(p)
			}))
		}
		m.Set(last, s.backButton())
		pan(2, "§f← 서쪽으로", ChunkPos{-1, 0})
		pan(3, "§f↑ 북쪽으로", ChunkPos{0, -1})
		pan(4, "§f↓ 남쪽으로", ChunkPos{0, 1})
		pan(5, "§f→ 동쪽으로", ChunkPos{1, 0})
		m.Set(last+6, s.button(chestui.Lore(item.NewStack(item.Compass{}, 1), "§f내 위치로"), func(p *player.Player, _ *world.Tx) {
			centre = chunkOf(cube.PosFromVec3(p.Position()))
			fill(p)
		}))
		info := []string{
			fmt.Sprintf("§7소유 청크: §f%d개", len(s.state.Chunks)),
			"§7다음 청크 가격: §6" + formatMoney(price),
			"§7소지금: §6" + formatMoney(s.Money(p)),
			"§a■ 소유  §e■ 구매 가능  §8■ 구매 불가",
			"§7섬과 맞닿은 청크만 살 수 있습니다.",
			"§7월드에도 30초 동안 경계가 표시됩니다.",
		}
		m.Set(last+8, chestui.Button{Item: chestui.Lore(item.NewStack(block.Grass{}, 1), "§a섬 정보", info...)})
	}
	fill(p)
	return m
}

// chunkButton is one chunk of the map.
func (s *session) chunkButton(p *player.Player, c, here ChunkPos, buyable bool, price int64) chestui.Button {
	lines := []string{fmt.Sprintf("§7블록 X %d~%d, Z %d~%d", c[0]*16, c[0]*16+15, c[1]*16, c[1]*16+15)}
	if c == here {
		lines = append(lines, "§b지금 서 있는 청크")
	} else {
		lines = append(lines, "§7"+direction(p.Position(), c))
	}
	name := fmt.Sprintf("청크 (%d, %d)", c[0], c[1])
	var icon item.Stack
	switch {
	case s.owns(c):
		icon, name = item.NewStack(block.StainedGlassPane{Colour: item.ColourLime()}, 1), "§a"+name+" · 소유"
	case buyable:
		icon, name = item.NewStack(block.StainedGlassPane{Colour: item.ColourYellow()}, 1), "§e"+name+" · 구매 가능"
		lines = append(lines, "§6"+formatMoney(price), "§e클릭해서 구매")
	default:
		icon, name = item.NewStack(block.StainedGlassPane{Colour: item.ColourGrey()}, 1), "§8"+name
	}
	if c == here {
		// The player's chunk stands out whatever its state.
		icon = item.NewStack(item.Compass{}, 1)
	}
	b := chestui.Button{Item: chestui.Lore(icon, name, lines...)}
	if buyable {
		b = s.button(b.Item, s.then(func(p *player.Player, _ *world.Tx) {
			ui.Confirm(p, "섬 확장", fmt.Sprintf("청크 (%d, %d)를 %s에 구매할까요?\n%s\n소지금: %s", c[0], c[1], formatMoney(price), direction(p.Position(), c), formatMoney(s.Money(p))), "§a구매", func(p *player.Player, tx *world.Tx) {
				if tx.World() != s.world {
					return
				}
				s.buyChunk(p, tx, c)
				s.showChunkBorders(p)
				s.expandMenu(p, c).Open(p, nil)
			})
		}))
	}
	return b
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
