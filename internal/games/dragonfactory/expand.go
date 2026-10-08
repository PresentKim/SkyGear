package dragonfactory

import (
	"fmt"
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Island expansion mode (섬 확장, docs/design/02-dragonfactory.md §8): the
// camera looks straight down from above the player and follows it, and the
// chunk under the player is the target. Owned and buyable chunks get a
// translucent glass floor only this player sees (ViewFakeBlock), so the
// areas are told apart and the player can walk onto chunks it has not bought
// yet. Sneaking (or /menu) opens buy and exit. Building is off meanwhile.

const (
	// expandCameraHeight is how far above the player the camera looks down.
	expandCameraHeight = 22
	// expandFloorY is the height of the fake glass floor (the island surface).
	expandFloorY = IslandY - 1
)

var (
	floorOwned   = block.StainedGlass{Colour: item.ColourLime()}
	floorBuyable = block.StainedGlass{Colour: item.ColourYellow()}
	floorTarget  = block.StainedGlass{Colour: item.ColourOrange()}
)

// expansion is the expansion mode of one player.
type expansion struct {
	target ChunkPos
	// floor are the fake blocks shown, by position.
	floor map[cube.Pos]world.Block
	// started tells apart a later expansion from this one in delayed tasks.
	started time.Time
}

func (s *session) expansionOf(p *player.Player) (*expansion, bool) {
	e, ok := s.expanding[p.H()]
	return e, ok
}

// showExpand starts expansion mode, or opens its actions if already in it.
func (s *session) showExpand(p *player.Player) {
	if _, ok := s.expansionOf(p); ok {
		s.showExpandActions(p)
		return
	}
	if s.expanding == nil {
		s.expanding = map[*world.EntityHandle]*expansion{}
	}
	e := &expansion{target: chunkOf(cube.PosFromVec3(p.Position())), floor: map[cube.Pos]world.Block{}, started: time.Now()}
	s.expanding[p.H()] = e
	s.drawFloor(p, e)
	s.moveExpandCamera(p, p.Position(), 0)
	p.Message("§a섬 확장 모드: 위에서 내려다봅니다. 걸어서 청크를 고르고, §e웅크리면§a 구매·종료 메뉴가 열립니다.")
	s.expandTip(p, e)
	s.repeatExpandTip(p, e)
}

// stopExpand leaves expansion mode: the camera and the real blocks come
// back, and a player standing on the fake floor is put back on the island.
func (s *session) stopExpand(p *player.Player, tx *world.Tx) {
	e, ok := s.expansionOf(p)
	if !ok {
		return
	}
	delete(s.expanding, p.H())
	for pos := range e.floor {
		p.ViewFakeBlock(pos, tx.Block(pos))
	}
	p.ClearCamera()
	feet := cube.PosFromVec3(p.Position())
	if _, air := tx.Block(feet.Side(cube.FaceDown)).(block.Air); air && !s.owns(chunkOf(feet)) {
		p.Teleport(s.Spawn())
	}
}

// expandMove follows a player in expansion mode: camera, target chunk and
// its colour.
func (s *session) expandMove(p *player.Player, newPos mgl64.Vec3) {
	e, ok := s.expansionOf(p)
	if !ok {
		return
	}
	s.moveExpandCamera(p, newPos, 0.15)
	if c := chunkOf(cube.PosFromVec3(newPos)); c != e.target {
		e.target = c
		s.drawFloor(p, e)
		s.expandTip(p, e)
	}
}

func (s *session) moveExpandCamera(p *player.Player, at mgl64.Vec3, ease float64) {
	// Pitch 90 looks straight down; yaw 180 puts north at the top.
	p.SetCamera(at.Add(mgl64.Vec3{0, expandCameraHeight, 0}), 90, 180, ease)
}

// floorWant returns the fake floor block for each position of the owned and
// buyable chunks that is air in the world.
func (s *session) floorWant(tx *world.Tx, e *expansion) map[cube.Pos]world.Block {
	want := map[cube.Pos]world.Block{}
	add := func(c ChunkPos, b world.Block) {
		for x := range 16 {
			for z := range 16 {
				pos := cube.Pos{int(c[0])*16 + x, expandFloorY, int(c[1])*16 + z}
				if _, air := tx.Block(pos).(block.Air); air {
					want[pos] = b
				}
			}
		}
	}
	for _, c := range s.state.Chunks {
		add(c, floorOwned)
	}
	for _, c := range s.purchasableChunks() {
		b := world.Block(floorBuyable)
		if c == e.target {
			b = floorTarget
		}
		add(c, b)
	}
	return want
}

// drawFloor brings the player's fake floor up to date, sending only changes.
func (s *session) drawFloor(p *player.Player, e *expansion) {
	tx := p.Tx()
	want := s.floorWant(tx, e)
	for pos := range e.floor {
		if _, ok := want[pos]; !ok {
			p.ViewFakeBlock(pos, tx.Block(pos))
		}
	}
	for pos, b := range want {
		if e.floor[pos] != b {
			p.ViewFakeBlock(pos, b)
		}
	}
	e.floor = want
}

// expandTip shows the target chunk and its price above the hotbar.
func (s *session) expandTip(p *player.Player, e *expansion) {
	c := e.target
	switch {
	case s.owns(c):
		p.SendTip(fmt.Sprintf("§a청크 (%d, %d) · 소유\n§7웅크리기: 메뉴", c[0], c[1]))
	case slices.Contains(s.purchasableChunks(), c):
		p.SendTip(fmt.Sprintf("§e청크 (%d, %d) · 구매 가능 §6%s\n§7웅크리기: 구매 · 소지금 %s", c[0], c[1], formatMoney(chunkPrice(len(s.state.Chunks))), formatMoney(s.Money(p))))
	default:
		p.SendTip(fmt.Sprintf("§8청크 (%d, %d) · 구매 불가\n§7소유한 청크와 맞닿은 노란 칸만 살 수 있습니다", c[0], c[1]))
	}
}

// repeatExpandTip keeps the tip on screen while the mode lasts.
func (s *session) repeatExpandTip(p *player.Player, e *expansion) {
	p.H().DoAfter(time.Second, func(tx *world.Tx, ent world.Entity) {
		if tx.World() != s.world {
			return
		}
		p := ent.(*player.Player)
		if cur, ok := s.expansionOf(p); !ok || cur.started != e.started {
			return
		}
		s.expandTip(p, e)
		s.repeatExpandTip(p, e)
	})
}

// showExpandActions offers buying the target chunk and leaving the mode.
func (s *session) showExpandActions(p *player.Player) {
	e, ok := s.expansionOf(p)
	if !ok {
		return
	}
	c := e.target
	price := chunkPrice(len(s.state.Chunks))
	var opts []ui.Option
	body := fmt.Sprintf("청크 (%d, %d)\n소유 %d청크 · 소지금 §6%s", c[0], c[1], len(s.state.Chunks), formatMoney(s.Money(p)))
	if slices.Contains(s.purchasableChunks(), c) {
		body += fmt.Sprintf("\n§f이 청크 가격: §6%s", formatMoney(price))
		opts = append(opts, s.opt(fmt.Sprintf("§a구매 (%s)", formatMoney(price)), iconIsland, func(p *player.Player, tx *world.Tx) {
			s.buyChunk(p, tx, c)
			if e, ok := s.expansionOf(p); ok {
				s.drawFloor(p, e)
				s.expandTip(p, e)
			}
		}))
	} else {
		body += "\n§7이 청크는 살 수 없습니다. 소유한 청크와 맞닿은 노란 칸으로 가세요."
	}
	opts = append(opts,
		s.opt("§f계속 둘러보기", iconLobby, func(*player.Player, *world.Tx) {}),
		s.opt("§c확장 끝내기", iconBack, func(p *player.Player, tx *world.Tx) {
			s.stopExpand(p, tx)
			p.Message("§7섬 확장 모드를 끝냈습니다.")
		}),
	)
	ui.Menu(p, "섬 확장", body, opts...)
}
