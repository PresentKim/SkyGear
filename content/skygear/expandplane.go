package skygear

import (
	"fmt"
	"slices"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/presentkim/skygear/content/skygear/core"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// The island expansion screen (ui.PlaneMenu): each chunk near the selected one is a button over the map (pressing selects it
// and the camera follows), with the price and buy/exit in a bottom panel.
// The arrow keys and WASD move the selection a chunk, Enter and the space bar
// buy it.

// expandPlaneReach is how many chunks around the selected one have a
// button: more than the camera shows, so a wide screen has them all.
const expandPlaneReach = 3

// showExpandPlane shows p the expansion screen for e.
func (s *session) showExpandPlane(p *player.Player, e *expansion) {
	lang := i18n.Of(p)
	t := e.target
	cx, cz := float64(t[0])*16+8, float64(t[1])*16+8
	buyable := island.Purchasable(s.Session)
	cost := island.ChunkCost(len(island.StateOf(s.Session).Chunks))
	r := int32(island.Radius)
	var spots []ui.PlaneSpot
	for dz := int32(-expandPlaneReach); dz <= expandPlaneReach; dz++ {
		for dx := int32(-expandPlaneReach); dx <= expandPlaneReach; dx++ {
			c := island.ChunkPos{t[0] + dx, t[1] + dz}
			if c[0] < -r || c[0] > r || c[1] < -r || c[1] > r {
				continue
			}
			sp := ui.PlaneSpot{X: float64(c[0])*16 - cx, Z: float64(c[1])*16 - cz, W: 16, H: 16, On: c == t, Tip: s.chunkLabel(lang, c, buyable)}
			if c != t {
				sp.Run = func(p *player.Player, tx *world.Tx) {
					if e, ok := s.expansionOf(p); ok && tx.World() == s.World {
						s.moveCursor(p, e, c)
						s.showExpandPlane(p, e)
					}
				}
			}
			spots = append(spots, sp)
		}
	}
	corner := fmt.Sprintf(i18n.T(lang, "expand.plane.owned"), len(island.StateOf(s.Session).Chunks)) + "\n" + fmt.Sprintf(i18n.T(lang, "expand.plane.cost"), island.CostText(lang, cost), textfmt.Int(int64(core.CountPlain(p.Inventory(), cost.Item))))
	var actions []ui.Option
	keys := map[ui.Key]func(p *player.Player, tx *world.Tx){
		ui.KeyUp: s.expandStep(0, -1), ui.KeyDown: s.expandStep(0, 1), ui.KeyLeft: s.expandStep(-1, 0), ui.KeyRight: s.expandStep(1, 0),
	}
	if slices.Contains(buyable, t) {
		buy := func(p *player.Player, tx *world.Tx) {
			island.Buy(s.Session, p, tx, t)
			if e, ok := s.expansionOf(p); ok {
				s.drawMap(p, e, true)
				s.showExpandPlane(p, e)
			}
		}
		actions = append(actions, s.Opt(i18n.T(lang, "expand.plane.buy"), formicon.Island, buy))
		keys[ui.KeyOK] = buy
	}
	leave := func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.World {
			s.stopExpand(p, tx)
		}
	}
	actions = append(actions, s.Opt(i18n.T(lang, "expand.plane.leave"), formicon.Back, leave))
	ui.PlaneMenu(p, ui.Plane{FOV: expandMapFOV, Distance: expandMapCameraHeight, Spots: spots, Info: s.chunkLabel(lang, t, buyable), Actions: actions, Corner: corner, Close: leave, Keys: keys})
}

// expandStep moves the selection of the expansion screen by dx, dz chunks
// (north at the top of the screen) and shows the screen again, also at the
// edge of the island, so the screen stays open.
func (s *session) expandStep(dx, dz int32) func(p *player.Player, tx *world.Tx) {
	return func(p *player.Player, tx *world.Tx) {
		if e, ok := s.expansionOf(p); ok && tx.World() == s.World {
			s.moveCursor(p, e, island.ChunkPos{e.target[0] + dx, e.target[1] + dz})
			s.showExpandPlane(p, e)
		}
	}
}

// chunkLabel is whether chunk c is owned, can be bought (buyable) or has no
// owned chunk beside it.
func (s *session) chunkLabel(lang string, c island.ChunkPos, buyable []island.ChunkPos) string {
	switch {
	case island.Owns(s.Session, c):
		return i18n.T(lang, "expand.plane.ownedChunk")
	case slices.Contains(buyable, c):
		return i18n.T(lang, "expand.plane.buyable")
	}
	return i18n.T(lang, "expand.plane.none")
}
