package skygear

import (
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/conduit"
)

// /sgdebug fill <radius> <seconds> <what>: overwrites the air around the
// player for a while, to compare the client frame rate and the server tick
// cost of many conduit looks drawn as blocks and as entities. Nothing in the
// game uses it. The look of a position is a function of the position only
// (fillLook), drawn the same by the entity and block modes, so the two can be
// compared fairly; pair puts both side by side.

const (
	fillMaxRadius  = 32
	fillMinSeconds = 5
	fillMaxSeconds = 600
	// fillMaxPositions caps the positions of one fill.
	fillMaxPositions = 20000
)

// fillUnit is the length of a second of a fill (a test makes it shorter).
var fillUnit = time.Second

type fillEnum string

func (fillEnum) Type() string { return "sg_fill" }
func (fillEnum) Options(cmd.Source) []string {
	return []string{"entity", "block", "layers", "pair"}
}

type debugFill struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"fill"`
	Radius  int            `cmd:"radius"`
	Seconds int            `cmd:"seconds"`
	What    fillEnum       `cmd:"what"`
}

type debugFillStop struct {
	adminBase
	Sub  cmd.SubCommand `cmd:"fill"`
	Stop cmd.SubCommand `cmd:"stop"`
}

// fillCell is a position of a fill and the blocks put there (nil: none).
type fillCell struct {
	pos    cube.Pos
	b0, b1 world.Block
}

// fillState is the fill of a session: the blocks and the entities to take
// away again.
type fillState struct {
	cells []fillCell
	ents  []*world.EntityHandle
	task  *world.Task
}

// fills is the running fill (*fillState) of each session (*session). It is
// used inside the world's transactions only.
var fills sync.Map

// fillKind is one conduit of a look: kind 0 energy, 1 item, 2 fluid (the order
// of the state probe), tier 1 to 3 and faces the arms as a 6-bit set in the
// order of conduit.Faces, which is the order of both the entity's codes and
// the probe's value, so no conversion is needed.
type fillKind struct{ kind, tier, faces int }

// fillShape is the look of a position: one or two conduits of different kinds.
type fillShape struct {
	n int
	k [2]fillKind
}

// fillSeed is the fixed seed of fillLook.
const fillSeed = 0x5eed_5ca1_e000_0004

// fillMix is the splitmix64 finalizer.
func fillMix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ x>>30) * 0xbf58476d1ce4e5b9
	x = (x ^ x>>27) * 0x94d049bb133111eb
	return x ^ x>>31
}

// fillLook is the look of pos, the same on every run: from a hash of pos with a
// fixed seed, a first kind with a tier and faces, and half of the time a
// second kind of another kind.
func fillLook(pos cube.Pos) fillShape {
	h := fillMix(fillSeed ^ uint64(int64(pos[0])))
	h = fillMix(h ^ uint64(int64(pos[1])))
	h = fillMix(h ^ uint64(int64(pos[2])))
	first := fillKind{kind: int(h % 3), tier: 1 + int(h>>8%3), faces: int(h >> 16 % 64)}
	l := fillShape{n: 1, k: [2]fillKind{first}}
	if h>>24&1 == 0 {
		return l
	}
	h = fillMix(h)
	l.n = 2
	l.k[1] = fillKind{kind: (first.kind + 1 + int(h%2)) % 3, tier: 1 + int(h>>8%3), faces: int(h >> 16 % 64)}
	return l
}

// probeValue is the value of the state probe block of a conduit: 1 + kind*192 +
// (tier-1)*64 + faces.
func (k fillKind) probeValue() int {
	return 1 + k.kind*stateProbeTiers*stateProbeFaces + (k.tier-1)*stateProbeFaces + k.faces
}

// visual is the Variant and MarkVariant of the display entity of a look, with
// the codes of conduit.Look: energy + Codes*item, and fluid.
func (l fillShape) visual() (variant, mark int32) {
	var codes [3]int32
	for _, k := range l.k[:l.n] {
		codes[k.kind] = conduit.Code(k.tier, k.faces)
	}
	return codes[0] + conduit.Codes*codes[1], codes[2]
}

// cell is the blocks of a look at pos: the state probe of the first conduit on
// layer 0 and of the second, if any, on layer 1; false when the probe is off.
func (l fillShape) cell(pos cube.Pos) (fillCell, bool) {
	b0, ok := stateProbeBlock(0, l.k[0].probeValue())
	if !ok {
		return fillCell{}, false
	}
	c := fillCell{pos: pos, b0: b0}
	if l.n == 2 {
		c.b1, _ = stateProbeBlock(1, l.k[1].probeValue())
	}
	return c, true
}

func (c debugFill) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	lang := i18n.Of(p)
	if c.Radius < 1 || c.Radius > fillMaxRadius || c.Seconds < fillMinSeconds || c.Seconds > fillMaxSeconds {
		o.Errorf(i18n.T(lang, "skygear.debug.fill.range"), fillMaxRadius, fillMinSeconds, fillMaxSeconds)
		return
	}
	if _, busy := fills.Load(s); busy {
		o.Error(i18n.T(lang, "skygear.debug.fill.busy"))
		return
	}
	what := string(c.What)
	if _, on := stateProbeBlock(0, 1); !on && (what == "block" || what == "pair") {
		o.Errorf(i18n.T(lang, "skygear.debug.fill.noProbe"), what)
		return
	}
	mspt := tx.World().TickStats().MSPT
	st := &fillState{}
	positions, capped := fillPositions(tx, cube.PosFromVec3(p.Position()), c.Radius)
	for _, pos := range positions {
		switch what {
		case "entity":
			st.entity(tx, pos, fillLook(pos))
		case "block":
			cell, _ := fillLook(pos).cell(pos)
			st.put(tx, cell)
		case "pair":
			// x even: the entity; x odd: the block of the look of the position
			// before it, so the two neighbours show the same shape.
			if pos[0]&1 == 0 {
				st.entity(tx, pos, fillLook(pos))
			} else {
				cell, _ := fillLook(cube.Pos{pos[0] - 1, pos[1], pos[2]}).cell(pos)
				st.put(tx, cell)
			}
		default:
			st.put(tx, fillCell{pos: pos, b0: LayerProbe{Kind: 0}, b1: LayerProbe{Kind: 1}})
		}
	}
	fills.Store(s, st)
	st.task = tx.World().DoAfter(time.Duration(c.Seconds)*fillUnit, func(tx *world.Tx) {
		if cur, ok := fills.Load(s); ok && cur == st {
			st.restore(tx)
			fills.Delete(s)
		}
	})
	o.Printf(i18n.T(lang, "skygear.debug.fill.start"), len(positions), what, c.Seconds, c.Radius, msptMillis(mspt))
	if capped {
		o.Printf(i18n.T(lang, "skygear.debug.fill.capped"), fillMaxPositions)
	}
}

func (c debugFillStop) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	lang := i18n.Of(p)
	v, ok := fills.LoadAndDelete(s)
	if !ok {
		o.Error(i18n.T(lang, "skygear.debug.fill.none"))
		return
	}
	st := v.(*fillState)
	st.task.Cancel()
	cells, ents := st.restore(tx)
	o.Printf(i18n.T(lang, "skygear.debug.fill.stop"), cells, ents)
}

// msptMillis is a tick time in milliseconds.
func msptMillis(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// fillPositions are the positions of the cube of radius r around centre whose
// layer 0 is air, the rows nearest to centre's level first, at most
// fillMaxPositions (capped: more were left out).
func fillPositions(tx *world.Tx, centre cube.Pos, r int) (out []cube.Pos, capped bool) {
	rng := tx.Range()
	for i := 0; i <= 2*r; i++ {
		// 0, 1, -1, 2, -2 ...
		dy := (i + 1) / 2
		if i%2 == 0 {
			dy = -dy
		}
		y := centre[1] + dy
		if y < rng.Min() || y > rng.Max() {
			continue
		}
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				pos := cube.Pos{centre[0] + dx, y, centre[2] + dz}
				if _, air := tx.Block(pos).(block.Air); !air {
					continue
				}
				if len(out) == fillMaxPositions {
					return out, true
				}
				out = append(out, pos)
			}
		}
	}
	return out, false
}

// entity adds the display entity of a look at pos.
func (st *fillState) entity(tx *world.Tx, pos cube.Pos, l fillShape) {
	variant, mark := l.visual()
	st.ents = append(st.ents, conduit.NewDebugVisual(tx, pos, variant, mark))
}

// put sets the blocks of a cell and remembers them.
func (st *fillState) put(tx *world.Tx, c fillCell) {
	tx.SetBlock(c.pos, c.b0, nil)
	if c.b1 != nil {
		tx.SetBlockLayer(c.pos, c.b1, 1)
	}
	st.cells = append(st.cells, c)
}

// restore takes the fill away: a position still holding what was put there
// loses layer 1 first, then layer 0 (setting air on layer 0 moves layer 1
// down); the entities are closed. It returns how many of each.
func (st *fillState) restore(tx *world.Tx) (cells, ents int) {
	for _, c := range st.cells {
		if c.b1 != nil && sameBlock(tx.BlockLayer(c.pos, 1), c.b1) {
			tx.SetBlockLayer(c.pos, nil, 1)
		}
		if sameBlock(tx.BlockLayer(c.pos, 0), c.b0) {
			tx.SetBlock(c.pos, block.Air{}, nil)
			cells++
		}
	}
	for _, h := range st.ents {
		conduit.CloseDebugVisual(tx, h)
		ents++
	}
	st.cells, st.ents = nil, nil
	return cells, ents
}

func sameBlock(a, b world.Block) bool {
	return a != nil && b != nil && world.BlockHash(a) == world.BlockHash(b)
}
