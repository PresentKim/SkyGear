package skygear

import (
	"time"

	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/sghook"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Island expansion mode (섬 확장): the body is frozen and the camera looks
// down on the selected chunk of the map (expandmap.go), under the expansion
// screen (expandplane.go) that moves the selection and buys. A view-only game
// mode keeps the client from breaking or placing blocks.

const (
	// The camera looks down on the map with a narrow field of view from high
	// up, so the map looks nearly flat.
	expandMapFOV          = 30
	expandMapCameraHeight = 65
	// expandFloorY is the height of the map's glass (the island surface).
	expandFloorY = IslandY - 1
	// expandRestore is how long after a click on the map its chunk is sent
	// again (restoreMapNear).
	expandRestore = 50 * time.Millisecond
	// expandMapReach is how far around the cursor (in chunks) the map is
	// drawn.
	expandMapReach = 4
)

var (
	// floorOwned and floorOwnedAlt alternate between owned chunks.
	floorOwned    = block.StainedGlass{Colour: item.ColourLime()}
	floorOwnedAlt = block.StainedGlass{Colour: item.ColourGreen()}
	floorBuyable  = block.StainedGlass{Colour: item.ColourYellow()}
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

type expansion struct {
	target ChunkPos
	// mode is the game mode to restore.
	mode world.GameMode
	// body is the chunk the frozen body stands in.
	body ChunkPos
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

// showExpand starts expansion mode, or shows its screen again if already in
// it.
func (s *session) showExpand(p *player.Player) {
	if e, ok := s.expansionOf(p); ok {
		s.showExpandPlane(p, e)
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
	s.updateBoard(p)
	s.showExpandPlane(p, e)
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
	sghook.ClearCamera(p)
	sghook.ClearCameraFOV(p)
	p.SetGameMode(e.mode)
	p.SetMobile()
	s.updateBoard(p)
}

// cursorPos is the centre of chunk c at the island surface.
func cursorPos(c ChunkPos) mgl64.Vec3 {
	return mgl64.Vec3{float64(c[0])*16 + 8, IslandY, float64(c[1])*16 + 8}
}

// moveCursor moves the cursor to chunk c if it is within the island range.
func (s *session) moveCursor(p *player.Player, e *expansion, c ChunkPos) {
	r := int32(islandRadius)
	switch {
	case c == e.target:
		return
	case c[0] < -r || c[0] > r || c[1] < -r || c[1] > r:
		p.SendTip(i18n.T(i18n.Of(p), "expand.edge"))
		return
	}
	e.target = c
	// The map first: the body moves over the cursor with it.
	s.drawMap(p, e, true)
	s.moveExpandCamera(p, e, 0.2)
}

func (s *session) moveExpandCamera(p *player.Player, e *expansion, ease float64) {
	// Pitch 90 looks straight down; yaw 180 puts north at the top.
	sghook.SetCamera(p, s.expandCamera(e), 90, 180, ease)
	sghook.SetCameraFOV(p, expandMapFOV)
}

// expandCamera is where the camera looks down from: above the selected
// chunk on the map.
func (s *session) expandCamera(e *expansion) mgl64.Vec3 {
	return mapShift(cursorPos(e.target)).Add(mgl64.Vec3{0, expandMapCameraHeight, 0})
}

// restoreMapNear shows the map chunk at pos again after the client clicked
// it: the click made the server resend the real (empty) blocks there.
func (s *session) restoreMapNear(p *player.Player, pos cube.Pos) {
	p.H().DoAfter(expandRestore, func(tx *world.Tx, ent world.Entity) {
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
