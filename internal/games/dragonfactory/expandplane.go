package dragonfactory

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/formicon"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/textfmt"
)

// Island expansion as a DDUI screen (ui.PlaneMenu) for Ore UI clients: each
// chunk near the selected one is a button over the map (pressing selects it
// and the camera follows), with the price and buy/exit in a bottom panel.
// The arrow keys and WASD move the selection a chunk, Enter and the space bar
// buy it.
// JSON UI clients use the movement keys and the sneak menu (expand.go).

// expandPlaneReach is how many chunks around the selected one have a
// button: more than the camera shows, so a wide screen has them all.
const expandPlaneReach = 3

// showExpandPlane shows p the expansion screen for e.
func (s *session) showExpandPlane(p *player.Player, e *expansion) {
	lang := i18n.Of(p)
	t := e.target
	cx, cz := float64(t[0])*16+8, float64(t[1])*16+8
	buyable := s.purchasableChunks()
	cost := chunkCost(len(s.state.Chunks))
	r := int32(islandRadius)
	var spots []ui.PlaneSpot
	for dz := int32(-expandPlaneReach); dz <= expandPlaneReach; dz++ {
		for dx := int32(-expandPlaneReach); dx <= expandPlaneReach; dx++ {
			c := ChunkPos{t[0] + dx, t[1] + dz}
			if c[0] < -r || c[0] > r || c[1] < -r || c[1] > r {
				continue
			}
			sp := ui.PlaneSpot{X: float64(c[0])*16 - cx, Z: float64(c[1])*16 - cz, W: 16, H: 16, On: c == t, Tip: s.chunkLabel(lang, c, buyable)}
			if c != t {
				sp.Run = func(p *player.Player, tx *world.Tx) {
					if e, ok := s.expansionOf(p); ok && tx.World() == s.world {
						s.moveCursor(p, e, c)
						s.showExpandPlane(p, e)
					}
				}
			}
			spots = append(spots, sp)
		}
	}
	info := s.chunkLabel(lang, t, buyable) + "\n" + fmt.Sprintf(i18n.T(lang, "expand.plane.owned"), len(s.state.Chunks))
	var actions []ui.Option
	keys := map[ui.Key]func(p *player.Player, tx *world.Tx){
		ui.KeyUp: s.expandStep(0, -1), ui.KeyDown: s.expandStep(0, 1), ui.KeyLeft: s.expandStep(-1, 0), ui.KeyRight: s.expandStep(1, 0),
	}
	if slices.Contains(buyable, t) {
		info += "\n" + fmt.Sprintf(i18n.T(lang, "expand.plane.cost"), costText(lang, cost), textfmt.Int(int64(countPlain(p.Inventory(), cost.Item))))
		buy := func(p *player.Player, tx *world.Tx) {
			s.buyChunk(p, tx, t)
			if e, ok := s.expansionOf(p); ok {
				s.drawMap(p, e, true)
				s.showExpandPlane(p, e)
			}
		}
		actions = append(actions, s.opt(fmt.Sprintf(i18n.T(lang, "expand.menu.buy"), costText(lang, cost)), formicon.Island, buy))
		keys[ui.KeyOK] = buy
	}
	leave := func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.world {
			s.stopExpand(p, tx)
		}
	}
	actions = append(actions, s.opt(i18n.T(lang, "expand.menu.stop"), formicon.Back, leave))
	ui.PlaneMenu(p, ui.Plane{FOV: expandMapFOV, Distance: expandMapCameraHeight, Spots: spots, Info: info, Actions: actions, Close: leave, Keys: keys})
}

// expandStep moves the selection of the expansion screen by dx, dz chunks
// (north at the top of the screen) and shows the screen again, also at the
// edge of the island, so the screen stays open.
func (s *session) expandStep(dx, dz int32) func(p *player.Player, tx *world.Tx) {
	return func(p *player.Player, tx *world.Tx) {
		if e, ok := s.expansionOf(p); ok && tx.World() == s.world {
			s.moveCursor(p, e, ChunkPos{e.target[0] + dx, e.target[1] + dz})
			s.showExpandPlane(p, e)
		}
	}
}

// chunkLabel is chunk c's line: where it is and whether it is owned, can be
// bought (buyable) or not.
func (s *session) chunkLabel(lang string, c ChunkPos, buyable []ChunkPos) string {
	key := "expand.plane.none"
	switch {
	case s.owns(c):
		key = "expand.plane.ownedChunk"
	case slices.Contains(buyable, c):
		key = "expand.plane.buyable"
	}
	return fmt.Sprintf(i18n.T(lang, key), c[0], c[1])
}
