package expand

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/island"
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
func showExpandPlane(s *core.Session, p *player.Player, e *expansion) {
	lang := i18n.Of(p)
	t := e.Target
	cx, cz := float64(t[0])*16+8, float64(t[1])*16+8
	buyable := island.Purchasable(s)
	cost := island.ChunkCost(len(island.StateOf(s).Chunks))
	r := int32(island.Radius)
	var spots []ui.PlaneSpot
	for dz := int32(-expandPlaneReach); dz <= expandPlaneReach; dz++ {
		for dx := int32(-expandPlaneReach); dx <= expandPlaneReach; dx++ {
			c := island.ChunkPos{t[0] + dx, t[1] + dz}
			if c[0] < -r || c[0] > r || c[1] < -r || c[1] > r {
				continue
			}
			sp := ui.PlaneSpot{X: float64(c[0])*16 - cx, Z: float64(c[1])*16 - cz, W: 16, H: 16, On: c == t, Tip: ChunkLabel(s, lang, c, buyable)}
			if c != t {
				sp.Run = func(p *player.Player, tx *world.Tx) {
					if e, ok := Of(s, p); ok && tx.World() == s.World {
						moveCursor(s, p, e, c)
						showExpandPlane(s, p, e)
					}
				}
			}
			spots = append(spots, sp)
		}
	}
	corner := fmt.Sprintf(i18n.T(lang, "expand.plane.owned"), len(island.StateOf(s).Chunks)) + "\n" + fmt.Sprintf(i18n.T(lang, "expand.plane.cost"), island.CostText(lang, cost), textfmt.Int(int64(core.CountPlain(p.Inventory(), cost.Item))))
	var actions []ui.Option
	keys := map[ui.Key]func(p *player.Player, tx *world.Tx){
		ui.KeyUp: Step(s, 0, -1), ui.KeyDown: Step(s, 0, 1), ui.KeyLeft: Step(s, -1, 0), ui.KeyRight: Step(s, 1, 0),
	}
	if slices.Contains(buyable, t) {
		buy := func(p *player.Player, tx *world.Tx) {
			island.Buy(s, p, tx, t)
			if e, ok := Of(s, p); ok {
				DrawMap(s, p, e, true)
				showExpandPlane(s, p, e)
			}
		}
		actions = append(actions, s.Opt(i18n.T(lang, "expand.plane.buy"), formicon.Island, buy))
		keys[ui.KeyOK] = buy
	}
	leave := func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.World {
			Stop(s, p, tx)
		}
	}
	actions = append(actions, s.Opt(i18n.T(lang, "expand.plane.leave"), formicon.Back, leave))
	ui.PlaneMenu(p, ui.Plane{FOV: expandMapFOV, Distance: expandMapCameraHeight, Spots: spots, Info: ChunkLabel(s, lang, t, buyable), Actions: actions, Corner: corner, Close: leave, Keys: keys})
}

// Step moves the selection of the expansion screen by dx, dz chunks
// (north at the top of the screen) and shows the screen again, also at the
// edge of the island, so the screen stays open.
func Step(s *core.Session, dx, dz int32) func(p *player.Player, tx *world.Tx) {
	return func(p *player.Player, tx *world.Tx) {
		if e, ok := Of(s, p); ok && tx.World() == s.World {
			moveCursor(s, p, e, island.ChunkPos{e.Target[0] + dx, e.Target[1] + dz})
			showExpandPlane(s, p, e)
		}
	}
}

// ChunkLabel is whether chunk c is owned, can be bought (buyable) or has no
// owned chunk beside it.
func ChunkLabel(s *core.Session, lang string, c island.ChunkPos, buyable []island.ChunkPos) string {
	switch {
	case island.Owns(s, c):
		return i18n.T(lang, "expand.plane.ownedChunk")
	case slices.Contains(buyable, c):
		return i18n.T(lang, "expand.plane.buyable")
	}
	return i18n.T(lang, "expand.plane.none")
}
