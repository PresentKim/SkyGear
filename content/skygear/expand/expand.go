// Package expand is the island expansion mode (섬 확장): the body is frozen
// and the camera looks down on the selected chunk of the map (expandmap.go),
// under the expansion screen (expandplane.go) that moves the selection and
// buys. A view-only game
// mode keeps the client from breaking or placing blocks.
package expand

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
)

const (
	// The camera looks down on the map with a narrow field of view from high
	// up, so the map looks nearly flat.
	expandMapFOV          = 30
	expandMapCameraHeight = 65
	// expandFloorY is the height of the map's glass (the island surface).
	ExpandFloorY = island.IslandY - 1
	// expandRestore is how long after a click on the map its chunk is sent
	// again (restoreMapNear).
	expandRestore = 50 * time.Millisecond
	// expandMapReach is how far around the cursor (in chunks) the map is
	// drawn.
	ExpandMapReach = 4
)

var (
	// floorOwned and floorOwnedAlt alternate between owned chunks.
	FloorOwned    = block.StainedGlass{Colour: item.ColourLime()}
	FloorOwnedAlt = block.StainedGlass{Colour: item.ColourGreen()}
	FloorBuyable  = block.StainedGlass{Colour: item.ColourYellow()}
)

// ViewOnlyMode is the game mode of expansion mode: no breaking, placing or
// interacting, no damage. It has no game mode ID, so a player state saved
// meanwhile records survival.
type ViewOnlyMode struct{}

func (ViewOnlyMode) AllowsEditing() bool { return false }

func (ViewOnlyMode) AllowsTakingDamage() bool { return false }

func (ViewOnlyMode) CreativeInventory() bool { return false }

func (ViewOnlyMode) HasCollision() bool { return true }

func (ViewOnlyMode) AllowsFlying() bool { return false }

func (ViewOnlyMode) AllowsInteraction() bool { return false }

func (ViewOnlyMode) Visible() bool { return true }

func (ViewOnlyMode) InstantPortalTravel() bool { return false }

type expansion struct {
	Target island.ChunkPos
	// mode is the game mode to restore.
	mode world.GameMode
	// body is the chunk the frozen body stands in.
	Body island.ChunkPos
	// started tells apart a later expansion from this one in delayed tasks.
	started time.Time
	// mapShown is the glass of each map chunk sent in the "map" view
	// (expandmap.go); home is where the body stood, to go back to.
	MapShown map[island.ChunkPos]world.Block
	Home     mgl64.Vec3
}

func Of(s *core.Session, p *player.Player) (*expansion, bool) {
	e, ok := expanding.Of(s)[p.H()]
	return e, ok
}

// Show starts expansion mode, or shows its screen again if already in
// it.
func Show(s *core.Session, p *player.Player) {
	if e, ok := Of(s, p); ok {
		showExpandPlane(s, p, e)
		return
	}
	at := island.ChunkOf(cube.PosFromVec3(p.Position()))
	e := &expansion{Target: at, Body: at, started: time.Now(), mode: p.GameMode(), Home: p.Position()}
	expanding.Of(s)[p.H()] = e
	p.SetImmobile()
	p.SetGameMode(ViewOnlyMode{})
	DrawMap(s, p, e, true)
	moveExpandCamera(s, p, e, 0)
	core.UpdateBoard(s, p)
	showExpandPlane(s, p, e)
}

// Stop leaves expansion mode: the camera, the real blocks and the
// player's movement come back.
func Stop(s *core.Session, p *player.Player, tx *world.Tx) {
	e, ok := Of(s, p)
	if !ok {
		return
	}
	delete(expanding.Of(s), p.H())
	DrawMap(s, p, e, false)
	sghook.ClearCamera(p)
	sghook.ClearCameraFOV(p)
	p.SetGameMode(e.mode)
	p.SetMobile()
	core.UpdateBoard(s, p)
}

// cursorPos is the centre of chunk c at the island surface.
func cursorPos(c island.ChunkPos) mgl64.Vec3 {
	return mgl64.Vec3{float64(c[0])*16 + 8, island.IslandY, float64(c[1])*16 + 8}
}

// moveCursor moves the cursor to chunk c if it is within the island range.
func moveCursor(s *core.Session, p *player.Player, e *expansion, c island.ChunkPos) {
	r := int32(island.Radius)
	switch {
	case c == e.Target:
		return
	case c[0] < -r || c[0] > r || c[1] < -r || c[1] > r:
		p.SendTip(i18n.T(i18n.Of(p), "expand.edge"))
		return
	}
	e.Target = c
	// The map first: the body moves over the cursor with it.
	DrawMap(s, p, e, true)
	moveExpandCamera(s, p, e, 0.2)
}

func moveExpandCamera(s *core.Session, p *player.Player, e *expansion, ease float64) {
	// Pitch 90 looks straight down; yaw 180 puts north at the top.
	sghook.SetCamera(p, Camera(s, e), 90, 180, ease)
	sghook.SetCameraFOV(p, expandMapFOV)
}

// Camera is where the camera looks down from: above the selected
// chunk on the map.
func Camera(s *core.Session, e *expansion) mgl64.Vec3 {
	return mapShift(cursorPos(e.Target)).Add(mgl64.Vec3{0, expandMapCameraHeight, 0})
}

// RestoreMapNear shows the map chunk at pos again after the client clicked
// it: the click made the server resend the real (empty) blocks there.
func RestoreMapNear(s *core.Session, p *player.Player, pos cube.Pos) {
	p.H().DoAfter(expandRestore, func(tx *world.Tx, ent world.Entity) {
		p := ent.(*player.Player)
		e, ok := Of(s, p)
		if !ok || tx.World() != s.World {
			return
		}
		c := island.ChunkOf(pos)
		c = island.ChunkPos{c[0] - MapOffset, c[1] - MapOffset}
		if glass, ok := e.MapShown[c]; ok {
			p.ViewFakeChunk(mapChunk(c), MapColumn(s, tx, c, glass))
		}
	})
}

// expanding are the players in island expansion mode.
var expanding = core.NewPlayers[*expansion]()
