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
// down on the selected chunk. The movement keys move the selection a chunk
// at a time (read from the client's input, Player.MoveInput). Chunks get a
// translucent glass floor only this player sees (ViewFakeBlock): owned ones
// in a checkerboard of two greens so their borders show, buyable ones
// yellow, and the selected one in its own colour with its border drawn over
// any real blocks. Sneaking (or /menu) opens buy and exit. The player is in
// a view-only game mode meanwhile, so the client neither breaks nor places
// blocks (and does not predict it on the fake floor).

const (
	// expandCameraHeight is how far above the island the camera looks down.
	expandCameraHeight = 30
	// Over the map the camera has a narrow field of view from higher up, so
	// the map looks nearly flat (less perspective at the edges, 13th
	// playtest) at about the same size: height × tan(fov/2) stays that of
	// the default 60° at expandCameraHeight.
	expandMapFOV          = 30
	expandMapCameraHeight = 65
	// expandFloorY is the height of the fake glass floor (the island surface).
	expandFloorY = IslandY - 1
	// expandStepEvery is how often a held movement key moves the cursor on
	// by a chunk; expandPoll is how often the keys are read.
	expandStepEvery = 350 * time.Millisecond
	expandPoll      = 50 * time.Millisecond
	// expandReach is how far (in chunks) the cursor may go from the frozen
	// body: the client only has the chunks around the body. expandFloorReach
	// is how far around the cursor the glass floor is drawn.
	expandReach      = 6
	expandFloorReach = 4
	// expandKeySign turns the keys into map directions: forward is north
	// (-z, the top of the screen) and left is west (-x). Flip if the cursor
	// moves the wrong way on real clients.
	expandKeySign = -1
)

var (
	// floorOwned and floorOwnedAlt alternate between owned chunks.
	floorOwned    = block.StainedGlass{Colour: item.ColourLime()}
	floorOwnedAlt = block.StainedGlass{Colour: item.ColourGreen()}
	floorBuyable  = block.StainedGlass{Colour: item.ColourYellow()}
	// The selected chunk: buyable, owned, or neither.
	floorTarget      = block.StainedGlass{Colour: item.ColourOrange()}
	floorTargetOwned = block.StainedGlass{Colour: item.ColourLightBlue()}
	floorTargetNone  = block.StainedGlass{Colour: item.ColourRed()}
)

// viewOnlyMode is the game mode of expansion mode: no breaking, placing or
// interacting, no damage. It has no game mode ID, so a player state saved
// meanwhile records survival.
type viewOnlyMode struct{}

func (viewOnlyMode) AllowsEditing() bool       { return false }
func (viewOnlyMode) AllowsTakingDamage() bool  { return false }
func (viewOnlyMode) CreativeInventory() bool   { return false }
func (viewOnlyMode) HasCollision() bool        { return true }
func (viewOnlyMode) AllowsFlying() bool        { return false }
func (viewOnlyMode) AllowsInteraction() bool   { return false }
func (viewOnlyMode) Visible() bool             { return true }
func (viewOnlyMode) InstantPortalTravel() bool { return false }

// expansion is the expansion mode of one player.
type expansion struct {
	target ChunkPos
	// floor are the fake blocks shown, by position.
	floor map[cube.Pos]world.Block
	// mode is the game mode to restore.
	mode world.GameMode
	// body is the chunk the frozen body stands in.
	body ChunkPos
	// lastStep is when the cursor last moved; a key held from before waits
	// expandStepEvery, a new key press moves at once.
	lastStep time.Time
	held     bool
	// started tells apart a later expansion from this one in delayed tasks.
	started time.Time
	// mapShown is the glass of each map chunk sent in the "map" view
	// (expandmap.go); home is where the body stood, to go back to.
	mapShown map[ChunkPos]world.Block
	home     mgl64.Vec3
}

// Expansion views (/dfadmin expandview). The floor at the island surface
// hides under buildings (11th playtest); glass above the buildings, debug
// shape outlines and plate entities did not match the real chunks (12th),
// so the map is the default:
//   - map: the island's top blocks flattened under glass, far away
//     (expandmap.go)
//   - floor: stained glass in the air at the island surface (the first view)
const (
	viewMap   = "map"
	viewFloor = "floor"
)

// expandViews are the expansion views.
var expandViews = []string{viewMap, viewFloor}

// currentExpandView is the session's expansion view.
func (s *session) currentExpandView() string {
	if s.expandView == "" {
		return viewMap
	}
	return s.expandView
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
	at := chunkOf(cube.PosFromVec3(p.Position()))
	e := &expansion{target: at, body: at, floor: map[cube.Pos]world.Block{}, started: time.Now(), mode: p.GameMode(), home: p.Position()}
	s.expanding[p.H()] = e
	p.SetImmobile()
	p.SetGameMode(viewOnlyMode{})
	s.drawFloor(p, e)
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
	s.drawMap(p, e, false)
	p.ClearCamera()
	p.ClearCameraFOV()
	p.SetGameMode(e.mode)
	p.SetMobile()
}

// cursorPos is the centre of chunk c at the island surface.
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
	r, reach := int32(islandRadius), int32(expandReach)
	switch {
	case c == e.target:
		return
	case c[0] < -r || c[0] > r || c[1] < -r || c[1] > r:
		p.SendTip("§7섬은 여기까지 넓힐 수 있습니다.")
		return
	case s.currentExpandView() != viewMap && (abs32(c[0]-e.body[0]) > reach || abs32(c[1]-e.body[1]) > reach):
		p.SendTip("§7더 멀리 가려면 확장을 끝내고 그쪽으로 걸어간 뒤 다시 여세요.")
		return
	}
	e.target = c
	// The drawing first: the camera follows its height.
	s.drawFloor(p, e)
	s.moveExpandCamera(p, e, 0.2)
	s.expandTip(p, e)
}

func (s *session) moveExpandCamera(p *player.Player, e *expansion, ease float64) {
	// Pitch 90 looks straight down; yaw 180 puts north at the top.
	p.SetCamera(s.expandCamera(e), 90, 180, ease)
	if s.currentExpandView() == viewMap {
		p.SetCameraFOV(expandMapFOV)
	} else {
		p.ClearCameraFOV()
	}
}

// expandCamera is where the camera looks down from: above the selected
// chunk, on the island or on the map.
func (s *session) expandCamera(e *expansion) mgl64.Vec3 {
	if s.currentExpandView() == viewMap {
		return mapShift(cursorPos(e.target)).Add(mgl64.Vec3{0, expandMapCameraHeight, 0})
	}
	return cursorPos(e.target).Add(mgl64.Vec3{0, expandCameraHeight, 0})
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

// floorWant returns the fake floor block for each position: the air
// positions of owned and buyable chunks, and the selected chunk (its air
// positions and its whole border).
func (s *session) floorWant(tx *world.Tx, e *expansion) map[cube.Pos]world.Block {
	want := map[cube.Pos]world.Block{}
	y := expandFloorY
	near := int32(expandFloorReach)
	add := func(c ChunkPos, b world.Block, border bool) {
		// Only around the cursor: farther chunks may not be loaded or sent.
		if abs32(c[0]-e.target[0]) > near || abs32(c[1]-e.target[1]) > near {
			return
		}
		for x := range 16 {
			for z := range 16 {
				pos := cube.Pos{int(c[0])*16 + x, y, int(c[1])*16 + z}
				edge := x == 0 || z == 0 || x == 15 || z == 15
				if _, air := tx.Block(pos).(block.Air); air || (border && edge) {
					want[pos] = b
				}
			}
		}
	}
	for _, c := range s.state.Chunks {
		b := world.Block(floorOwned)
		if (c[0]+c[1])%2 != 0 {
			b = floorOwnedAlt
		}
		add(c, b, false)
	}
	buyable := s.purchasableChunks()
	for _, c := range buyable {
		add(c, floorBuyable, false)
	}
	target := world.Block(floorTargetNone)
	switch {
	case s.owns(e.target):
		target = floorTargetOwned
	case slices.Contains(buyable, e.target):
		target = floorTarget
	}
	add(e.target, target, true)
	return want
}

// drawFloor brings the player's view of the chunks up to date in the
// session's expansion view, sending only changes.
func (s *session) drawFloor(p *player.Player, e *expansion) {
	tx := p.Tx()
	view := s.currentExpandView()
	want := map[cube.Pos]world.Block{}
	if view == viewFloor {
		want = s.floorWant(tx, e)
	}
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
	s.drawMap(p, e, view == viewMap)
}

// expandTip shows the target chunk and its price above the hotbar.
func (s *session) expandTip(p *player.Player, e *expansion) {
	c := e.target
	switch {
	case s.owns(c):
		p.SendTip(fmt.Sprintf("§b청크 (%d, %d) · 소유\n§7웅크리기: 메뉴", c[0], c[1]))
	case slices.Contains(s.purchasableChunks(), c):
		cost := chunkCost(len(s.state.Chunks))
		p.SendTip(fmt.Sprintf("§e청크 (%d, %d) · 구매 가능 §6%s\n§7웅크리기: 구매 · 가진 것 %s개", c[0], c[1], costText(cost), formatInt(int64(countPlain(p.Inventory(), cost.Item)))))
	default:
		p.SendTip(fmt.Sprintf("§c청크 (%d, %d) · 구매 불가\n§7이동 키로 소유한 청크와 맞닿은 노란 칸을 고르세요", c[0], c[1]))
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
	cost := chunkCost(len(s.state.Chunks))
	var opts []ui.Option
	body := fmt.Sprintf("청크 (%d, %d)\n소유 %d청크", c[0], c[1], len(s.state.Chunks))
	if slices.Contains(s.purchasableChunks(), c) {
		body += fmt.Sprintf("\n§f이 청크 비용: §6%s §7(가진 것 %s개)", costText(cost), formatInt(int64(countPlain(p.Inventory(), cost.Item))))
		opts = append(opts, s.opt(fmt.Sprintf("§a구매 (%s)", costText(cost)), iconIsland, func(p *player.Player, tx *world.Tx) {
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

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
