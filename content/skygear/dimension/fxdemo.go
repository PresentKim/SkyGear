package dimension

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The showcase (/sgdebug fx): a weapon's look played round a player for a
// few seconds, at grade 1 and level 1, hitting nothing, seen through the
// hunt camera as in a hunt. An entity no one sees draws it each tick and
// closes when it is done.

// fxDemoTicks is how long a showcase plays.
const fxDemoTicks = 100

// FxDemos are the looks a showcase plays: the weapons, a monster's death and
// the monsters' tricks.
var FxDemos = []string{sgitem.RelicBlade, sgitem.RelicCrossbow, sgitem.RelicFlameRing, sgitem.RelicLightning, sgitem.RelicBoomerang, "poof", "blast", "blink", "mend"}

// ShowFx plays the look of kind round p. It reports whether kind is one of
// FxDemos.
func ShowFx(tx *world.Tx, p *player.Player, kind string) bool {
	known := false
	for _, k := range FxDemos {
		known = known || k == kind
	}
	if known {
		tx.AddEntity(world.EntitySpawnOpts{Position: p.Position()}.New(FxDemoType, fxDemoConfig{kind: kind, viewer: p.H()}))
	}
	return known
}

type fxDemoData struct {
	kind   string
	viewer *world.EntityHandle
	age    int
	dir    mgl64.Vec3
}

type fxDemo struct {
	tx     *world.Tx
	Handle *world.EntityHandle
	Data   *world.EntityData
}

func (d *fxDemo) H() *world.EntityHandle  { return d.Handle }
func (d *fxDemo) Position() mgl64.Vec3    { return d.Data.Pos }
func (d *fxDemo) Rotation() cube.Rotation { return d.Data.Rot }
func (d *fxDemo) Invisible() bool         { return true }
func (d *fxDemo) md() *fxDemoData         { return d.Data.Data.(*fxDemoData) }

// Close takes the showcase out of the world before closing its handle.
func (d *fxDemo) Close() error {
	d.tx.RemoveEntity(d)
	return d.Handle.Close()
}

// Tick draws one tick of the look round the viewer, and closes the showcase
// when it is done or the viewer is gone.
func (d *fxDemo) Tick(tx *world.Tx, _ int64) {
	md := d.md()
	e, ok := md.viewer.Entity(tx)
	if !ok {
		_ = d.Close()
		return
	}
	p := e.(*player.Player)
	hunting := false
	if s, ok := core.Of(tx); ok {
		hunting = RunOf(s) != nil
	}
	if md.age >= fxDemoTicks {
		if !hunting {
			plainCamera(p)
		}
		_ = d.Close()
		return
	}
	feet := p.Position()
	if md.age == 0 {
		if !hunting {
			huntCamera(p)
		}
		look := p.Rotation().Vec3()
		md.dir = mgl64.Vec3{look[0], 0, look[2]}
		if md.dir.Len() < 0.01 {
			md.dir = mgl64.Vec3{0, 0, 1}
		}
		md.dir = md.dir.Normalize()
	}
	targets := []mgl64.Vec3{
		feet.Add(md.dir.Mul(6)),
		feet.Add(rotateY(md.dir, 30).Mul(7)),
		feet.Add(rotateY(md.dir, -30).Mul(7)),
	}
	t := md.age
	md.age++
	switch md.kind {
	case sgitem.RelicBlade:
		angle := float64(t) * bladeTurn
		for i := range 3 {
			a := angle + float64(i)*2*math.Pi/3
			drawBlade(tx, feet.Add(mgl64.Vec3{0, 0.9, 0}), mgl64.Vec3{math.Cos(a), 0, math.Sin(a)}, bladeInner, bladeOuter)
		}
	case sgitem.RelicCrossbow:
		if t%20 == 0 {
			for _, at := range targets {
				drawBolt(tx, feet.Add(mgl64.Vec3{0, 1.2, 0}), at.Add(mgl64.Vec3{0, 0.6, 0}))
			}
		}
	case sgitem.RelicFlameRing:
		if t%10 == 0 {
			drawFlameRing(tx, feet, 3)
		}
	case sgitem.RelicLightning:
		if t%40 == 0 {
			for _, at := range targets {
				drawLightning(tx, at)
			}
		}
	case sgitem.RelicBoomerang:
		const flight = 30
		out := t%(flight+10) + 1
		if out <= flight {
			along := 9 * (1 - math.Abs(float64(out)-flight/2)/(flight/2))
			drawDisc(tx, feet.Add(mgl64.Vec3{0, 1, 0}).Add(md.dir.Mul(along)))
		}
	case "poof":
		if t%30 == 0 {
			for _, at := range targets {
				drawPoof(tx, at)
			}
		}
	case "blast":
		if t%50 == 0 {
			drawBlast(tx, targets[0])
		}
	case "blink":
		if t%25 == 0 {
			from, to := targets[(t/25)%3], targets[(t/25+1)%3]
			drawBlink(tx, from)
			drawBlink(tx, to)
		}
	case "mend":
		if t%40 == 0 {
			for _, at := range targets {
				drawMend(tx, at)
			}
		}
	}
}

type fxDemoConfig struct {
	kind   string
	viewer *world.EntityHandle
}

func (c fxDemoConfig) Apply(data *world.EntityData) {
	data.Data = &fxDemoData{kind: c.kind, viewer: c.viewer}
}

// FxDemoType is the world.EntityType of a showcase. The client draws it as
// the gate entity, unseen.
var FxDemoType fxDemoType

type fxDemoType struct{}

func (fxDemoType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &fxDemoData{}
	}
	return &fxDemo{tx: tx, Handle: handle, Data: data}
}

func (fxDemoType) EncodeEntity() string        { return "skygear:fx_demo" }
func (fxDemoType) NetworkEncodeEntity() string { return "skygear:gate_portal" }

// Transient: never saved with the world.
func (fxDemoType) Transient() bool { return true }

func (fxDemoType) BBox(world.Entity) cube.BBox                     { return cube.BBox{} }
func (fxDemoType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &fxDemoData{} }
func (fxDemoType) EncodeNBT(*world.EntityData) map[string]any      { return nil }
