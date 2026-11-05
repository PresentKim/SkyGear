package machine

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/ui"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The machines that work on an area around them (the harvester, the item
// vacuum, the tree feller) take a range on their screen, up to what their
// tier gives, and outline that area in the world for a while.

// rangeShowTicks is how long the outline shows after it is turned on.
const rangeShowTicks = 60 * 20

const (
	// rangeParticle is the pack's dot the outline is drawn with
	// (particles/range_outline.json).
	rangeParticle = "skygear:range_outline"
	// rangeStep is how far apart the dots of an edge are, in blocks.
	rangeStep = 0.5
	// rangeSight is how far from the area's centre a player sees the outline.
	rangeSight = 48
)

// rangeKind is the kind of the machine st when it works on an area, and
// false otherwise.
func (st *State) rangeKind() (string, bool) {
	for kind := range tierRanges {
		if Kinds[kind].ID == st.Def.ID {
			return kind, true
		}
	}
	return "", false
}

// rangeHeight is how far below and above the machine its area reaches.
func rangeHeight(kind string) (down, up int) {
	switch kind {
	case KindVacuum:
		return vacuumDown, vacuumUp
	case KindFeller:
		return fellerDown, fellerUp
	}
	return 0, 0
}

// rangeBox is the area of the machine of kind st at pos, from its first
// block to past its last.
func (st *State) rangeBox(kind string, pos cube.Pos) (lo, hi mgl64.Vec3) {
	r := st.Reach(kind)
	down, up := rangeHeight(kind)
	lo = pos.Vec3().Sub(mgl64.Vec3{float64(r), float64(down), float64(r)})
	hi = pos.Vec3().Add(mgl64.Vec3{float64(r + 1), float64(up + 1), float64(r + 1)})
	return lo, hi
}

// rangeShown reports whether the outline of the machine's area is on.
func (st *State) rangeShown() bool { return st.rangeUntil > 0 && st.curTick <= st.rangeUntil }

// showRange draws the outline of the machine's area while it is turned on.
func (st *State) showRange(tx *world.Tx, pos cube.Pos, tick int64) {
	if !st.rangeShown() || tick%10 != 0 {
		return
	}
	kind, ok := st.rangeKind()
	if !ok {
		return
	}
	lo, hi := st.rangeBox(kind, pos)
	centre := lo.Add(hi).Mul(0.5)
	var near []*player.Player
	for e := range tx.Players() {
		if p := e.(*player.Player); p.Position().Sub(centre).Len() <= rangeSight {
			near = append(near, p)
		}
	}
	if len(near) == 0 {
		return
	}
	for _, at := range boxEdges(lo, hi) {
		pk := &packet.SpawnParticleEffect{EntityUniqueID: -1, Position: mgl32.Vec3{float32(at[0]), float32(at[1]), float32(at[2])}, ParticleName: rangeParticle}
		for _, p := range near {
			sghook.WritePacket(sghook.Session(p), pk)
		}
	}
}

// boxEdges are points rangeStep apart along the twelve edges of the box
// from lo to hi.
func boxEdges(lo, hi mgl64.Vec3) []mgl64.Vec3 {
	var out []mgl64.Vec3
	corners := [2]mgl64.Vec3{lo, hi}
	for axis := range 3 {
		a, b := (axis+1)%3, (axis+2)%3
		for _, ca := range corners {
			for _, cb := range corners {
				for v := lo[axis]; v <= hi[axis]; v += rangeStep {
					var p mgl64.Vec3
					p[axis], p[a], p[b] = v, ca[a], cb[b]
					out = append(out, p)
				}
			}
		}
	}
	return out
}

// rangeTools are the screen's buttons of a machine with a range: narrower,
// wider, and the outline on or off.
func (st *State) rangeTools(lang string, again func(p *player.Player, tx *world.Tx)) []ui.Option {
	kind, ok := st.rangeKind()
	if !ok {
		return nil
	}
	reach, most := st.Reach(kind), st.MaxReach(kind)
	set := func(r int) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			st.Range = max(1, min(r, most))
			if st.Range == most {
				st.Range = 0
			}
			again(p, tx)
		}
	}
	size := fmt.Sprintf(i18n.T(lang, "machine.range.size"), 2*reach+1, 2*reach+1, reach, most)
	viewKey := "machine.range.show"
	if st.rangeShown() {
		viewKey = "machine.range.hide"
	}
	return []ui.Option{
		{Label: i18n.T(lang, "machine.range.wider") + "\n" + size, Icon: "textures/ui/sg/arrow_right", Run: set(reach + 1)},
		{Label: i18n.T(lang, "machine.range.narrower") + "\n" + size, Icon: "textures/ui/sg/arrow_left", Run: set(reach - 1)},
		{Label: i18n.T(lang, viewKey), Icon: "textures/ui/sg/icon_cube", Run: func(p *player.Player, tx *world.Tx) {
			if st.rangeShown() {
				st.rangeUntil = 0
			} else {
				st.rangeUntil = st.curTick + rangeShowTicks
			}
			again(p, tx)
		}},
	}
}
