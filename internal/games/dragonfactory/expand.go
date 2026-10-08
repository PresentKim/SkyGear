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
// player's body stays where it is, frozen, and the camera looks straight
// down on a cursor: a fake entity only this player sees, standing on the
// target chunk. The movement keys move the cursor a chunk at a time (read
// from the client's input, Player.MoveInput). Owned and buyable chunks get a
// translucent glass floor only this player sees (ViewFakeBlock). Sneaking
// (or /menu) opens buy and exit. Building and block clicks are off.

const (
	// expandCameraHeight is how far above the island the camera looks down.
	expandCameraHeight = 30
	// expandFloorY is the height of the fake glass floor (the island surface).
	expandFloorY = IslandY - 1
	// expandStepEvery is how often a held movement key moves the cursor on
	// by a chunk; expandPoll is how often the keys are read.
	expandStepEvery = 350 * time.Millisecond
	expandPoll      = 50 * time.Millisecond
	// expandCursorEntity is the entity shown as the cursor.
	expandCursorEntity = "minecraft:allay"
	// expandKeySign turns the keys into map directions: forward is north
	// (-z, the top of the screen) and left is west (-x). Flip if the cursor
	// moves the wrong way on real clients.
	expandKeySign = -1
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
	// cursor is the runtime ID of the cursor entity.
	cursor uint64
	// lastStep is when the cursor last moved; a key held from before waits
	// expandStepEvery, a new key press moves at once.
	lastStep time.Time
	held     bool
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
	p.SetImmobile()
	s.drawFloor(p, e)
	e.cursor = p.ViewFakeEntity(expandCursorEntity, cursorPos(e.target), 180, 3)
	s.moveExpandCamera(p, e, 0)
	p.Message("§a섬 확장 모드: 위에서 내려다봅니다. §e이동 키§a로 청크를 고르고, §e웅크리면§a 구매·종료 메뉴가 열립니다.")
	s.expandTip(p, e)
	s.repeatExpandTip(p, e)
	s.pollExpand(p, e)
}

// stopExpand leaves expansion mode: the camera, the real blocks and the
// player's movement come back.
func (s *session) stopExpand(p *player.Player, tx *world.Tx) {
	e, ok := s.expansionOf(p)
	if !ok {
		return
	}
	delete(s.expanding, p.H())
	for pos := range e.floor {
		p.ViewFakeBlock(pos, tx.Block(pos))
	}
	p.RemoveFakeEntity(e.cursor)
	p.ClearCamera()
	p.SetMobile()
}

// cursorPos is where the cursor entity stands on chunk c.
func cursorPos(c ChunkPos) mgl64.Vec3 {
	return mgl64.Vec3{float64(c[0])*16 + 8, IslandY, float64(c[1])*16 + 8}
}

// pollExpand reads the movement keys while the mode lasts and moves the
// cursor.
func (s *session) pollExpand(p *player.Player, e *expansion) {
	p.H().DoAfter(expandPoll, func(tx *world.Tx, ent world.Entity) {
		if tx.World() != s.world {
			return
		}
		p := ent.(*player.Player)
		if cur, ok := s.expansionOf(p); !ok || cur.started != e.started {
			return
		}
		s.expandInput(p, e, p.MoveInput())
		s.pollExpand(p, e)
	})
}

// expandInput moves the cursor by the held keys (forward/back in Y,
// left/right in X).
func (s *session) expandInput(p *player.Player, e *expansion, keys mgl64.Vec2) {
	dx, dz := keyStep(keys.X()), keyStep(keys.Y())
	if dx == 0 && dz == 0 {
		e.held = false
		return
	}
	if e.held && time.Since(e.lastStep) < expandStepEvery {
		return
	}
	e.held, e.lastStep = true, time.Now()
	s.moveCursor(p, e, ChunkPos{e.target[0] + int32(expandKeySign*dx), e.target[1] + int32(expandKeySign*dz)})
}

// keyStep turns an analogue key value into -1, 0 or 1.
func keyStep(v float64) int {
	switch {
	case v > 0.5:
		return 1
	case v < -0.5:
		return -1
	}
	return 0
}

// moveCursor moves the cursor to chunk c if it is within the island range.
func (s *session) moveCursor(p *player.Player, e *expansion, c ChunkPos) {
	r := int32(islandRadius)
	if c[0] < -r || c[0] > r || c[1] < -r || c[1] > r || c == e.target {
		return
	}
	e.target = c
	p.MoveFakeEntity(e.cursor, cursorPos(c), 180)
	s.moveExpandCamera(p, e, 0.25)
	s.drawFloor(p, e)
	s.expandTip(p, e)
}

func (s *session) moveExpandCamera(p *player.Player, e *expansion, ease float64) {
	// Pitch 90 looks straight down; yaw 180 puts north at the top.
	p.SetCamera(cursorPos(e.target).Add(mgl64.Vec3{0, expandCameraHeight, 0}), 90, 180, ease)
}

// restoreFloorNear shows the fake floor again around pos after the client
// clicked it: the click made the server resend the real (air) blocks.
func (s *session) restoreFloorNear(p *player.Player, pos cube.Pos) {
	p.H().DoAfter(expandPoll, func(tx *world.Tx, ent world.Entity) {
		p := ent.(*player.Player)
		e, ok := s.expansionOf(p)
		if !ok || tx.World() != s.world {
			return
		}
		for x := -1; x <= 1; x++ {
			for z := -1; z <= 1; z++ {
				at := cube.Pos{pos[0] + x, expandFloorY, pos[2] + z}
				if b, ok := e.floor[at]; ok {
					p.ViewFakeBlock(at, b)
				}
			}
		}
	})
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
		p.SendTip(fmt.Sprintf("§8청크 (%d, %d) · 구매 불가\n§7이동 키로 소유한 청크와 맞닿은 노란 칸을 고르세요", c[0], c[1]))
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
		body += "\n§7이 청크는 살 수 없습니다. 이동 키로 소유한 청크와 맞닿은 노란 칸을 고르세요."
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
