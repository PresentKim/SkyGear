package skygear

import (
	"slices"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/assets"
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

// fillKind is one conduit of a look: kind 0 energy, 1 item, 2 fluid, 3 network
// (4-kind probe modes only; the order of the state probe), tier 1 to 3 and
// faces the arms as a 6-bit set in the order of conduit.Faces, which is the
// order of both the entity's codes and the probe's value, so no conversion is
// needed.
type fillKind struct{ kind, tier, faces int }

// fillShape is the look of a position: one to four conduits of different
// kinds (at most two but in the 4-kind probe modes).
type fillShape struct {
	n int
	k [4]fillKind
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

// fillHash is the hash of pos with the fixed seed.
func fillHash(pos cube.Pos) uint64 {
	h := fillMix(fillSeed ^ uint64(int64(pos[0])))
	h = fillMix(h ^ uint64(int64(pos[1])))
	return fillMix(h ^ uint64(int64(pos[2])))
}

// fillLook is the look of pos, the same on every run: fillLook4 in the 4-kind
// probe modes, fillLook3 otherwise.
func fillLook(pos cube.Pos) fillShape {
	if stateProbeFourKinds(stateProbeMode) {
		return fillLook4(pos)
	}
	return fillLook3(pos)
}

// fillLook3 is the look of pos with three kinds: from a hash of pos with a
// fixed seed, a first kind with a tier and faces, and half of the time a
// second kind of another kind.
func fillLook3(pos cube.Pos) fillShape {
	h := fillHash(pos)
	first := fillKind{kind: int(h % 3), tier: 1 + int(h>>8%3), faces: int(h >> 16 % 64)}
	l := fillShape{n: 1, k: [4]fillKind{first}}
	if h>>24&1 == 0 {
		return l
	}
	h = fillMix(h)
	l.n = 2
	l.k[1] = fillKind{kind: (first.kind + 1 + int(h%2)) % 3, tier: 1 + int(h>>8%3), faces: int(h >> 16 % 64)}
	return l
}

// fillLook4 is the look of pos with four kinds: 1 to 4 kinds (a quarter of the
// positions each), each with a tier and faces, in the order of their kinds.
// In a third of the looks of two kinds or more two of them have the same
// faces, so same4's same-faces pairs show.
func fillLook4(pos cube.Pos) fillShape {
	h := fillHash(pos)
	n := 1 + int(h%4)
	order := [4]int{0, 1, 2, 3}
	// a shuffle of the kinds by the hash: take the first n
	for i, r := 3, h>>2; i > 0; i, r = i-1, r/uint64(i+1) {
		j := int(r % uint64(i+1))
		order[i], order[j] = order[j], order[i]
	}
	var kinds [4]bool
	for _, k := range order[:n] {
		kinds[k] = true
	}
	l := fillShape{}
	for k := range kinds {
		if !kinds[k] {
			continue
		}
		hk := fillMix(h ^ uint64(k+1))
		l.k[l.n] = fillKind{kind: k, tier: 1 + int(hk%3), faces: int(hk >> 8 % 64)}
		l.n++
	}
	if l.n >= 2 && h>>16%3 == 0 {
		hk := fillMix(h ^ 0x11)
		i := int(hk % uint64(l.n))
		j := (i + 1 + int(hk>>8%uint64(l.n-1))) % l.n
		l.k[j].faces = l.k[i].faces
	}
	return l
}

// probeValue is the value of the state probe block of a conduit drawn alone:
// 1 + kind*192 + (tier-1)*64 + faces.
func (k fillKind) probeValue() int {
	return 1 + k.kind*stateProbeTiers*stateProbeFaces + (k.tier-1)*stateProbeFaces + k.faces
}

// visual is the Variant and MarkVariant of the display entity of a look, with
// the codes of conduit.Look: energy + Codes*item, and fluid. The entity has no
// network conduit: it is left out (lost counts it).
func (l fillShape) visual() (variant, mark int32) {
	var codes [3]int32
	for _, k := range l.k[:l.n] {
		if k.kind < 3 {
			codes[k.kind] = conduit.Code(k.tier, k.faces)
		}
	}
	return codes[0] + conduit.Codes*codes[1], codes[2]
}

// entityLost is the number of conduits of the look the entity cannot draw.
func (l fillShape) entityLost() (lost int) {
	for _, k := range l.k[:l.n] {
		if k.kind >= 3 {
			lost++
		}
	}
	return lost
}

// probeLookOf is the look of the conduits given (at most two).
func probeLookOf(ks ...fillKind) probeLook {
	l := probeLook{n: len(ks)}
	copy(l.k[:], ks)
	return l
}

// place splits a look over the two probe blocks of the mode: the looks of
// layer 0 and of layer 1 (n 0 when none) and how many conduits are left out.
//
//   - perm, bones: the first conduit on layer 0, the second on layer 1.
//   - single4: the same; a third and fourth are left out.
//   - same4: layer 1 takes two kinds with the same faces when the look has
//     them, layer 0 one of the others (empty, value 0, when there is none),
//     the rest are left out; with no such pair as single4.
//   - all4, all4split: layer 0 the first two, layer 1 the others: all are drawn.
//   - pair2 (and pair2ab, pair2bits): the looks are those of its two blocks,
//     not of the storage layers: l0 the kinds energy and item (block 0), l1
//     fluid and network (block 1); all are drawn (cell puts the non-empty
//     blocks on layer 0, then 1).
func (l fillShape) place(mode string) (l0, l1 probeLook, lost int) {
	ks := l.k[:l.n]
	switch mode {
	case assets.StateProbePair2, assets.StateProbePair2AB, assets.StateProbePair2Bits:
		for _, k := range ks {
			if k.kind < 2 {
				l0.k[l0.n] = k
				l0.n++
			} else {
				l1.k[l1.n] = k
				l1.n++
			}
		}
		return l0, l1, 0
	case assets.StateProbeAll4, assets.StateProbeAll4Split:
		if len(ks) <= 2 {
			return probeLookOf(ks...), probeLook{}, 0
		}
		return probeLookOf(ks[:2]...), probeLookOf(ks[2:]...), 0
	case assets.StateProbeSame4:
		for i := range ks {
			for j := i + 1; j < len(ks); j++ {
				if ks[i].faces != ks[j].faces {
					continue
				}
				var rest []fillKind
				for x, k := range ks {
					if x != i && x != j {
						rest = append(rest, k)
					}
				}
				if len(rest) == 0 {
					return probeLook{}, probeLookOf(ks[i], ks[j]), 0
				}
				return probeLookOf(rest[0]), probeLookOf(ks[i], ks[j]), len(rest) - 1
			}
		}
	}
	l0 = probeLookOf(ks[0])
	if len(ks) >= 2 {
		l1 = probeLookOf(ks[1])
	}
	return l0, l1, max(len(ks)-2, 0)
}

// cell is the blocks of a look at pos in the probe's mode (place): layer 0
// always (value 0 when it draws nothing), layer 1 when it draws something;
// lost is how many conduits are left out, false when the probe is off.
func (l fillShape) cell(pos cube.Pos) (c fillCell, lost int, ok bool) {
	if stateProbePair2(stateProbeMode) {
		return l.cellPair2(pos)
	}
	l0, l1, lost := l.place(stateProbeMode)
	v0, ok0 := stateProbeEncode(stateProbeMode, 0, l0)
	b0, ok := stateProbeBlock(0, v0)
	if !ok || !ok0 {
		return fillCell{}, 0, false
	}
	c = fillCell{pos: pos, b0: b0}
	if l1.n > 0 {
		v1, _ := stateProbeEncode(stateProbeMode, 1, l1)
		c.b1, _ = stateProbeBlock(1, v1)
	}
	return c, lost, true
}

// cellPair2 is cell in mode pair2: the pair blocks that are non-empty fill
// layer 0 first, then layer 1 (block 0 before block 1; block 1 alone goes on
// layer 0, since layer 1 draws over a visible layer 0 only). lost counts the
// conduits of the look that the blocks put, decoded again, do not draw.
func (l fillShape) cellPair2(pos cube.Pos) (c fillCell, lost int, ok bool) {
	l0, l1, _ := l.place(stateProbeMode)
	c = fillCell{pos: pos}
	var drawn []fillKind
	for blk, pl := range []probeLook{l0, l1} {
		if pl.n == 0 {
			continue
		}
		v, ok := stateProbeEncode(stateProbeMode, blk, pl)
		b, ok2 := stateProbeBlock(blk, v)
		if !ok || !ok2 {
			continue
		}
		if c.b0 == nil {
			c.b0 = b
		} else {
			c.b1 = b
		}
		back, _ := stateProbeDecode(stateProbeMode, blk, v)
		drawn = append(drawn, back.k[:back.n]...)
	}
	if c.b0 == nil {
		return fillCell{}, 0, false
	}
	for _, k := range l.k[:l.n] {
		if !slices.Contains(drawn, k) {
			lost++
		}
	}
	return c, lost, true
}

// fillTally counts the cells of a fill drawn whole and with conduits left out,
// as blocks and as entities.
type fillTally struct{ blocks, blocksLost, ents, entsLost int }

func (t *fillTally) block(lost int) {
	t.blocks++
	if lost > 0 {
		t.blocksLost++
	}
}

func (t *fillTally) entity(l fillShape) {
	t.ents++
	if l.entityLost() > 0 {
		t.entsLost++
	}
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
	var tally fillTally
	for _, pos := range positions {
		switch what {
		case "entity":
			l := fillLook(pos)
			st.entity(tx, pos, l)
			tally.entity(l)
		case "block":
			cell, lost, _ := fillLook(pos).cell(pos)
			st.put(tx, cell)
			tally.block(lost)
		case "pair":
			// x even: the entity; x odd: the block of the look of the position
			// before it, so the two neighbours show the same shape.
			if pos[0]&1 == 0 {
				l := fillLook(pos)
				st.entity(tx, pos, l)
				tally.entity(l)
			} else {
				cell, lost, _ := fillLook(cube.Pos{pos[0] - 1, pos[1], pos[2]}).cell(pos)
				st.put(tx, cell)
				tally.block(lost)
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
	if stateProbeFourKinds(stateProbeMode) {
		if tally.blocks > 0 {
			o.Printf(i18n.T(lang, "skygear.debug.fill.blocks4"), stateProbeMode, tally.blocks-tally.blocksLost, tally.blocks, tally.blocksLost)
		}
		if tally.ents > 0 {
			o.Printf(i18n.T(lang, "skygear.debug.fill.entity3"), tally.ents-tally.entsLost, tally.ents, tally.entsLost)
		}
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
