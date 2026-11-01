package dimension

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
)

// What a run leaves lying about: the experience gems on the ground (drawn as
// experience orbs; the gear moves them, gear.go) and the damage numbers that
// rise over a hit monster for a moment. Both belong to their run: one whose
// run is over (or that was saved with its chunk) closes itself.

// runMark is the data of a gem or a damage number: its run, and for a
// number the ticks it has left.
type runMark struct {
	run  *Run
	left int
}

// mark is a gem or a damage number.
type mark struct {
	tx     *world.Tx
	Handle *world.EntityHandle
	Data   *world.EntityData
	gem    bool
}

func (m *mark) H() *world.EntityHandle  { return m.Handle }
func (m *mark) Position() mgl64.Vec3    { return m.Data.Pos }
func (m *mark) Rotation() cube.Rotation { return m.Data.Rot }
func (m *mark) md() *runMark            { return m.Data.Data.(*runMark) }
func (m *mark) NameTag() string         { return m.Data.Name }
func (m *mark) AlwaysShowNameTag() bool { return true }

// Experience sizes a gem's orb: the client draws an orb by the experience
// it holds, so a gem says it holds gemLook (its worth is gemXP, gear.go).
func (m *mark) Experience() int { return gemLook }

// Scale draws a gem bigger still, to be seen from the hunt's high camera.
func (m *mark) Scale() float64 {
	if m.gem {
		return gemScale
	}
	return 1
}

const (
	// gemLook is the experience whose orb a gem looks like (the client's
	// orb sizes step at 3, 7, 17, 37, 73, 149, 307...); gemScale how much
	// bigger it is drawn.
	gemLook  = 150
	gemScale = 1.8
)

// Tick lets a damage number rise and go, and closes a mark whose run is
// over.
func (m *mark) Tick(tx *world.Tx, _ int64) {
	d := m.md()
	s, ok := core.Of(tx)
	if !ok || d.run == nil || RunOf(s) != d.run {
		_ = m.Close()
		return
	}
	if m.gem {
		return
	}
	if d.left--; d.left <= 0 {
		_ = m.Close()
		return
	}
	m.moveTo(tx, m.Data.Pos.Add(mgl64.Vec3{0, popupRise, 0}))
}

// moveTo puts the mark at pos and shows its viewers.
func (m *mark) moveTo(tx *world.Tx, pos mgl64.Vec3) {
	m.Data.Pos = pos
	for _, v := range tx.Viewers(pos) {
		v.ViewEntityMovement(m, pos, m.Data.Rot, false)
	}
}

// Close takes the mark out of the world before closing its handle.
func (m *mark) Close() error {
	m.tx.RemoveEntity(m)
	return m.Handle.Close()
}

type markConfig struct {
	run  *Run
	left int
}

func (c markConfig) Apply(data *world.EntityData) {
	data.Data = &runMark{run: c.run, left: c.left}
}

// markType is what the gems' and the numbers' entity types share.
type markType struct{ gem bool }

type gemType struct{ markType }

type popupType struct{ markType }

// GemType and PopupType are the world.EntityTypes of the gems and the damage
// numbers.
var (
	GemType   = gemType{markType{gem: true}}
	PopupType = popupType{}
)

func (t markType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &runMark{}
	}
	return &mark{tx: tx, Handle: handle, Data: data, gem: t.gem}
}

func (gemType) EncodeEntity() string        { return "skygear:hunt_gem" }
func (gemType) NetworkEncodeEntity() string { return "minecraft:xp_orb" }

func (popupType) EncodeEntity() string        { return "skygear:hunt_popup" }
func (popupType) NetworkEncodeEntity() string { return "minecraft:falling_block" }

// FloatingText draws a damage number as its text alone (session/world.go).
func (popupType) FloatingText() {}

func (markType) BBox(world.Entity) cube.BBox                     { return cube.BBox{} }
func (markType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &runMark{} }
func (markType) EncodeNBT(*world.EntityData) map[string]any      { return nil }

const (
	// popupTicks is how long a damage number stays, popupRise how far it
	// rises each tick; at most maxPopups are up at once.
	popupTicks = 14
	popupRise  = 0.04
	maxPopups  = 40
)

// popup shows dmg rising over a monster at pos for a moment.
func (r *Run) popup(tx *world.Tx, pos mgl64.Vec3, dmg float64) {
	r.popups = slices.DeleteFunc(r.popups, func(h *world.EntityHandle) bool {
		_, ok := h.Entity(tx)
		return !ok
	})
	if len(r.popups) >= maxPopups {
		if e, ok := r.popups[0].Entity(tx); ok {
			_ = e.(*mark).Close()
		}
		r.popups = r.popups[1:]
	}
	h := world.EntitySpawnOpts{Position: pos, NameTag: damageText(dmg)}.New(PopupType, markConfig{run: r, left: popupTicks})
	tx.AddEntity(h)
	r.popups = append(r.popups, h)
}

// damageText is a damage number: whole, or with one decimal.
func damageText(dmg float64) string {
	t := strings.TrimSuffix(fmt.Sprintf("%.1f", dmg), ".0")
	return "§c" + t
}

// healthBar is a monster's health as a bar of healthBars marks, green,
// yellow or red by what is left.
func healthBar(health, maxHealth float64) string {
	f := max(0, min(1, health/maxHealth))
	n := int(f*healthBars + 0.999)
	col := "§a"
	switch {
	case f <= 0.25:
		col = "§c"
	case f <= 0.5:
		col = "§e"
	}
	return col + strings.Repeat("|", n) + "§8" + strings.Repeat("|", healthBars-n)
}

// healthBars is how many marks a health bar has.
const healthBars = 20

// closeMarks closes the run's gems and damage numbers.
func (r *Run) closeMarks(tx *world.Tx) {
	for _, h := range r.popups {
		if e, ok := h.Entity(tx); ok {
			_ = e.(*mark).Close()
		}
	}
	r.popups = nil
	if r.gear == nil {
		return
	}
	for _, g := range r.gear.gems {
		if e, ok := g.h.Entity(tx); ok {
			_ = e.(*mark).Close()
		}
	}
	r.gear.gems = nil
}
