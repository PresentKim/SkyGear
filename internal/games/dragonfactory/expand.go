package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/presentkim/dragonfactory/common/i18n"
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Island expansion mode (섬 확장, docs/design/02-dragonfactory.md §13): the
// player's body stays where it is, frozen, and the camera looks straight
// down on the selected chunk. The movement keys move the selection a chunk
// at a time (read from the client's input, Player.MoveInput). The chunks
// are shown on a map far away (expandmap.go) under stained glass: owned ones
// in a checkerboard of two greens so their borders show, buyable ones
// yellow, and the selected one in its own colour. Sneaking (or /menu) opens
// buy and exit. The player is in a view-only game mode meanwhile, so the
// client neither breaks nor places blocks.

const (
	// The camera looks down on the map with a narrow field of view from high
	// up, so the map looks nearly flat (less perspective at the edges, 13th
	// playtest).
	expandMapFOV          = 30
	expandMapCameraHeight = 65
	// expandFloorY is the height of the map's glass (the island surface).
	expandFloorY = IslandY - 1
	// expandStepEvery is how often a held movement key moves the cursor on
	// by a chunk; expandPoll is how often the keys are read.
	expandStepEvery = 350 * time.Millisecond
	expandPoll      = 50 * time.Millisecond
	// expandMapReach is how far around the cursor (in chunks) the map is
	// drawn.
	expandMapReach = 4
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
	e := &expansion{target: at, body: at, started: time.Now(), mode: p.GameMode(), home: p.Position()}
	s.expanding[p.H()] = e
	p.SetImmobile()
	p.SetGameMode(viewOnlyMode{})
	s.drawMap(p, e, true)
	s.moveExpandCamera(p, e, 0)
	p.Message(i18n.T(i18n.Of(p), "dragonfactory.expand.showExpand.1"))
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
	s.drawMap(p, e, false)
	dfhook.ClearCamera(p)
	dfhook.ClearCameraFOV(p)
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
		s.expandInput(p, e, dfhook.MoveInput(p))
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
	switch {
	case c == e.target:
		return
	case c[0] < -r || c[0] > r || c[1] < -r || c[1] > r:
		p.SendTip(i18n.T(i18n.Of(p), "dragonfactory.expand.moveCursor.1"))
		return
	}
	e.target = c
	// The map first: the body moves over the cursor with it.
	s.drawMap(p, e, true)
	s.moveExpandCamera(p, e, 0.2)
	s.expandTip(p, e)
}

func (s *session) moveExpandCamera(p *player.Player, e *expansion, ease float64) {
	// Pitch 90 looks straight down; yaw 180 puts north at the top.
	dfhook.SetCamera(p, s.expandCamera(e), 90, 180, ease)
	dfhook.SetCameraFOV(p, expandMapFOV)
}

// expandCamera is where the camera looks down from: above the selected
// chunk on the map.
func (s *session) expandCamera(e *expansion) mgl64.Vec3 {
	return mapShift(cursorPos(e.target)).Add(mgl64.Vec3{0, expandMapCameraHeight, 0})
}

// restoreMapNear shows the map chunk at pos again after the client clicked
// it: the click made the server resend the real (empty) blocks there.
func (s *session) restoreMapNear(p *player.Player, pos cube.Pos) {
	p.H().DoAfter(expandPoll, func(tx *world.Tx, ent world.Entity) {
		p := ent.(*player.Player)
		e, ok := s.expansionOf(p)
		if !ok || tx.World() != s.world {
			return
		}
		c := chunkOf(pos)
		c = ChunkPos{c[0] - mapOffset, c[1] - mapOffset}
		if glass, ok := e.mapShown[c]; ok {
			p.ViewFakeChunk(mapChunk(c), s.mapColumn(tx, c, glass))
		}
	})
}

// expandTip shows the target chunk and its price above the hotbar.
func (s *session) expandTip(p *player.Player, e *expansion) {
	c := e.target
	switch {
	case s.owns(c):
		p.SendTip(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.expand.expandTip.1"), c[0], c[1]))
	case slices.Contains(s.purchasableChunks(), c):
		cost := chunkCost(len(s.state.Chunks))
		p.SendTip(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.expand.expandTip.2"), c[0], c[1], costText(cost), formatInt(int64(countPlain(p.Inventory(), cost.Item)))))
	default:
		p.SendTip(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.expand.expandTip.3"), c[0], c[1]))
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
	body := fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.1"), c[0], c[1], len(s.state.Chunks))
	if slices.Contains(s.purchasableChunks(), c) {
		body += fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.2"), costText(cost), formatInt(int64(countPlain(p.Inventory(), cost.Item))))
		opts = append(opts, s.opt(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.3"), costText(cost)), iconIsland, func(p *player.Player, tx *world.Tx) {
			s.buyChunk(p, tx, c)
			if e, ok := s.expansionOf(p); ok {
				s.drawMap(p, e, true)
				s.expandTip(p, e)
			}
		}))
	} else {
		body += i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.4")
	}
	opts = append(opts,
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.5"), iconLobby, func(*player.Player, *world.Tx) {}),
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.6"), iconBack, func(p *player.Player, tx *world.Tx) {
			s.stopExpand(p, tx)
			p.Message(i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.7"))
		}),
	)
	ui.Menu(p, i18n.T(i18n.Of(p), "dragonfactory.expand.showExpandActions.8"), body, opts...)
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
