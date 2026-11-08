package skygear

import (
	"math/rand/v2"
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
// game uses it.

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
	return []string{"entity", "block", "layers"}
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

// fillProbeValues is how many values the state probe block has (0 is air).
const fillProbeValues = stateProbeValues

// fillStateProbe is stateProbeBlock(layer, v): the state probe block of a
// value, false when the probe is off.
func fillStateProbe(layer, v int) (world.Block, bool) {
	return stateProbeBlock(layer, v)
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
	mspt := tx.World().TickStats().MSPT
	st := &fillState{}
	positions, capped := fillPositions(tx, cube.PosFromVec3(p.Position()), c.Radius)
	probeOff := false
	for _, pos := range positions {
		switch what {
		case "entity":
			// A random look: every kind has a code 1 to Codes-1.
			energy, item, fluid := 1+rand.Int32N(conduit.Codes-1), 1+rand.Int32N(conduit.Codes-1), 1+rand.Int32N(conduit.Codes-1)
			st.ents = append(st.ents, conduit.NewDebugVisual(tx, pos, energy+conduit.Codes*item, fluid))
		case "block":
			cell := fillCell{pos: pos}
			if b, ok := fillStateProbe(0, 1+rand.IntN(fillProbeValues-1)); ok {
				cell.b0 = b
				if rand.IntN(2) == 0 {
					if b1, ok := fillStateProbe(1, 1+rand.IntN(fillProbeValues-1)); ok {
						cell.b1 = b1
					}
				}
			} else {
				probeOff = true
				cell.b0 = LayerProbe{}
			}
			st.put(tx, cell)
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
	if probeOff {
		o.Printf("%s", i18n.T(lang, "skygear.debug.fill.probeOff"))
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
