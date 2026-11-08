package skygear

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/assets"
	"github.com/presentkim/skygear/content/skygear/conduit"
)

// fillEntities counts the conduit display entities of tx.
func fillEntities(tx *world.Tx) (n int) {
	for e := range tx.Entities() {
		if _, ok := e.(interface{ MarkVariant() int32 }); ok {
			n++
		}
	}
	return n
}

// TestDebugFill: each mode fills the air of the cube (and only air), the fill
// is taken away on stop and after its time, and one fill runs at a time.
func TestDebugFill(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s) // feet at 0.5 64 2.5
	centre := cube.Pos{0, 64, 2}
	stone := centre.Add(cube.Pos{1, 0, 0})
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(stone, block.Stone{}, nil)
	})
	const cube1 = 27 // radius 1; the stone is not filled
	run := func(p *player.Player, tx *world.Tx, radius, seconds int, what string) *cmd.Output {
		t.Helper()
		var o cmd.Output
		debugFill{adminBase: adminBase{g: s.g}, Radius: radius, Seconds: seconds, What: fillEnum(what)}.Run(p, &o, tx)
		return &o
	}
	stop := func(p *player.Player, tx *world.Tx) *cmd.Output {
		t.Helper()
		var o cmd.Output
		debugFillStop{adminBase: adminBase{g: s.g}}.Run(p, &o, tx)
		return &o
	}
	// layers reports what the cube holds: blocks on layer 0 other than air and
	// stone, and on layer 1.
	layers := func(tx *world.Tx) (l0, l1 int) {
		for x := -1; x <= 1; x++ {
			for y := -1; y <= 1; y++ {
				for z := -1; z <= 1; z++ {
					pos := centre.Add(cube.Pos{x, y, z})
					if pos == stone {
						continue
					}
					if _, air := tx.BlockLayer(pos, 0).(block.Air); !air {
						l0++
					}
					if _, air := tx.BlockLayer(pos, 1).(block.Air); !air {
						l1++
					}
				}
			}
		}
		return l0, l1
	}
	old := fillUnit
	fillUnit = time.Millisecond
	t.Cleanup(func() { fillUnit = old })

	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if o := run(p, tx, 0, 10, "layers"); len(o.Errors()) != 1 {
			t.Errorf("radius 0: %v", o.Errors())
		}
		if o := run(p, tx, 1, 4, "layers"); len(o.Errors()) != 1 {
			t.Errorf("4 seconds: %v", o.Errors())
		}
		if o := stop(p, tx); len(o.Errors()) != 1 {
			t.Errorf("stop with no fill: %v", o.Errors())
		}

		// layers: probes on layers 0 and 1 in every air position.
		if o := run(p, tx, 1, 600, "layers"); len(o.Errors()) != 0 || len(o.Messages()) != 1 {
			t.Fatalf("layers: %v %v", o.Errors(), o.Messages())
		}
		if l0, l1 := layers(tx); l0 != cube1-1 || l1 != cube1-1 {
			t.Errorf("layers: %d on layer 0, %d on layer 1, want %d each", l0, l1, cube1-1)
		}
		if tx.Block(stone) != (block.Stone{}) {
			t.Errorf("layers: the stone is %#v", tx.Block(stone))
		}
		if o := run(p, tx, 1, 10, "entity"); len(o.Errors()) != 1 {
			t.Errorf("second fill: %v", o.Errors())
		}
		if o := stop(p, tx); len(o.Errors()) != 0 {
			t.Errorf("stop: %v", o.Errors())
		}
		if l0, l1 := layers(tx); l0 != 0 || l1 != 0 {
			t.Errorf("stop: %d on layer 0, %d on layer 1 left", l0, l1)
		}
		if tx.Block(stone) != (block.Stone{}) {
			t.Errorf("stop: the stone is %#v", tx.Block(stone))
		}

		// block: needs the state probe; off, it refuses and puts nothing.
		o := run(p, tx, 1, 600, "block")
		if _, on := stateProbeBlock(0, 1); on {
			if len(o.Errors()) != 0 {
				t.Errorf("block with the probe on: %v", o.Errors())
			}
		} else {
			if len(o.Errors()) != 1 {
				t.Errorf("block with the probe off: %v", o.Errors())
			}
			if l0, l1 := layers(tx); l0 != 0 || l1 != 0 {
				t.Errorf("block refused but %d on layer 0, %d on layer 1", l0, l1)
			}
			if o := stop(p, tx); len(o.Errors()) != 1 {
				t.Errorf("a refused block fill is running: %v", o.Errors())
			}
		}
		stop(p, tx)
		if l0, l1 := layers(tx); l0 != 0 || l1 != 0 {
			t.Errorf("block stop: %d on layer 0, %d on layer 1 left", l0, l1)
		}

		// entity: one display entity per position, no block.
		run(p, tx, 1, 600, "entity")
		if n := fillEntities(tx); n != cube1-1 {
			t.Errorf("entity: %d entities, want %d", n, cube1-1)
		}
		if l0, l1 := layers(tx); l0 != 0 || l1 != 0 {
			t.Errorf("entity: %d on layer 0, %d on layer 1", l0, l1)
		}
	})
	// The visuals check their bundle every 20 ticks: they stay.
	tick(s, 60)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if n := fillEntities(tx); n != cube1-1 {
			t.Errorf("after 60 ticks: %d entities, want %d", n, cube1-1)
		}
		stop(p, tx)
		if n := fillEntities(tx); n != 0 {
			t.Errorf("entity stop: %d entities left", n)
		}
	})

	// After its time a fill takes itself away, and what changed meanwhile stays.
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Teleport(mgl64.Vec3{0.5, 64, 2.5}) // it fell during the ticks
		if o := run(p, tx, 1, fillMinSeconds, "layers"); len(o.Errors()) > 0 {
			t.Fatalf("timed fill: %v", o.Errors())
		}
		tx.SetBlock(centre.Add(cube.Pos{0, 1, 0}), block.Dirt{}, nil) // replaced
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		done := false
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			l0, l1 := layers(tx)
			done = l0 == 1 && l1 == 0
		})
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the fill was not taken away in time")
		}
		time.Sleep(5 * time.Millisecond)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if tx.Block(centre.Add(cube.Pos{0, 1, 0})) != (block.Dirt{}) {
			t.Errorf("the dirt placed in between was removed")
		}
		if o := stop(p, tx); len(o.Errors()) != 1 {
			t.Errorf("stop after the end: %v", o.Errors())
		}
	})
}

// TestDebugFillCap: a cube of more than the cap in air fills the cap only.
func TestDebugFillCap(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	s.World.Do(func(tx *world.Tx) {
		pos, capped := fillPositions(tx, cube.Pos{0, 80, 0}, fillMaxRadius)
		if len(pos) != fillMaxPositions || !capped {
			t.Errorf("%d positions (capped %v), want %d capped", len(pos), capped, fillMaxPositions)
		}
		pos, capped = fillPositions(tx, cube.Pos{0, 80, 0}, 1)
		if len(pos) != 27 || capped {
			t.Errorf("%d positions (capped %v), want 27", len(pos), capped)
		}
	})
}

// TestDebugFillBlockProbe: with the state probe on (go test with
// SKYGEAR_STATE_PROBE set; the mode is read at init), the block mode puts
// probe blocks on layer 0, a layer-1 probe block on about half of the
// positions (three quarters with four kinds, where layer 0 may be the empty
// value under a same-faces pair), and the timed restore leaves both layers
// air. With four kinds it reports the cells drawn whole.
func TestDebugFillBlockProbe(t *testing.T) {
	if stateProbeMode == "" || stateProbeMode == "off" {
		t.Skip("SKYGEAR_STATE_PROBE is off")
	}
	four := stateProbeFourKinds(stateProbeMode)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	centre := cube.Pos{0, 64, 2}
	const r = 3
	count := func(tx *world.Tx) (l0, l1, air int) {
		for x := -r; x <= r; x++ {
			for y := -r; y <= r; y++ {
				for z := -r; z <= r; z++ {
					pos := centre.Add(cube.Pos{x, y, z})
					if sp, ok := tx.BlockLayer(pos, 0).(StateProbe); ok && sp.Layer == 0 && (sp.V > 0 || four) {
						l0++
					}
					if sp, ok := tx.BlockLayer(pos, 1).(StateProbe); ok && sp.Layer == 1 && sp.V > 0 {
						l1++
					}
					_, a0 := tx.BlockLayer(pos, 0).(block.Air)
					_, a1 := tx.BlockLayer(pos, 1).(block.Air)
					if a0 && a1 {
						air++
					}
				}
			}
		}
		return l0, l1, air
	}
	old := fillUnit
	fillUnit = time.Millisecond
	t.Cleanup(func() { fillUnit = old })

	total := 0
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		var o cmd.Output
		debugFill{adminBase: adminBase{g: s.g}, Radius: r, Seconds: fillMinSeconds, What: "block"}.Run(p, &o, tx)
		if len(o.Errors()) != 0 {
			t.Fatalf("fill: %v", o.Errors())
		}
		if want := map[bool]int{false: 1, true: 2}[four]; len(o.Messages()) != want {
			t.Errorf("fill messages %v, want %d", o.Messages(), want)
		}
		_, _, air := count(tx)
		l0, l1, _ := count(tx)
		total = (2*r + 1) * (2*r + 1) * (2*r + 1)
		if air != 0 {
			t.Errorf("%d positions still air", air)
		}
		if l0 == 0 || l0 > total {
			t.Errorf("%d probe blocks on layer 0 of %d", l0, total)
		}
		if hi := map[bool]int{false: l0 * 3 / 4, true: l0 * 7 / 8}[four]; l1 < l0/4 || l1 > hi {
			t.Errorf("%d probe blocks on layer 1 of %d, want about half (3/4 with four kinds)", l1, l0)
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		done := false
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			l0, l1, _ := count(tx)
			done = l0 == 0 && l1 == 0
		})
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the fill was not taken away in time")
		}
		time.Sleep(5 * time.Millisecond)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		if _, _, air := count(tx); air != total-1 && air != total {
			// the player's own position may hold the player, not a block
			t.Errorf("after restore %d of %d positions are air", air, total)
		}
	})
}

// TestFillLook: the 3-kind look of a position (fillLook3, the look of the
// probe off, perm and bones) is a function of the position, has one or two
// conduits of different kinds with a tier 1 to 3 and a set of arms, and
// varies over positions.
func TestFillLook(t *testing.T) {
	seen := map[fillShape]bool{}
	var twos, kinds [3]int
	for x := -8; x < 8; x++ {
		for y := 60; y < 64; y++ {
			for z := -8; z < 8; z++ {
				pos := cube.Pos{x, y, z}
				l := fillLook3(pos)
				if l != fillLook3(pos) {
					t.Fatalf("%v: the look changes between calls", pos)
				}
				if l.n < 1 || l.n > 2 {
					t.Fatalf("%v: %d conduits", pos, l.n)
				}
				for i, k := range l.k[:l.n] {
					if k.kind < 0 || k.kind > 2 || k.tier < 1 || k.tier > 3 || k.faces < 0 || k.faces > 63 {
						t.Fatalf("%v: %+v", pos, k)
					}
					if v := k.probeValue(); v < 1 || v >= stateProbeValues {
						t.Fatalf("%v: probe value %d", pos, v)
					}
					if i == 0 {
						kinds[k.kind]++
					}
				}
				if l.n == 2 {
					twos[0]++
					if l.k[0].kind == l.k[1].kind {
						t.Fatalf("%v: two conduits of kind %d", pos, l.k[0].kind)
					}
				}
				seen[l] = true
			}
		}
	}
	total := 16 * 4 * 16
	if twos[0] < total/3 || twos[0] > total*2/3 {
		t.Errorf("%d of %d looks have a second conduit", twos[0], total)
	}
	for k, n := range kinds {
		if n < total/6 {
			t.Errorf("kind %d is the first of only %d looks", k, n)
		}
	}
	if len(seen) < total/2 {
		t.Errorf("%d different looks over %d positions", len(seen), total)
	}
}

// TestFillLookEncodings: the entity codes and the probe values of a look show
// the same kinds, tiers and arms (stateProbeParts decodes the value; the entity
// code is 1 + (tier-1)*64 + faces of conduit.Look).
func TestFillLookEncodings(t *testing.T) {
	for x := 0; x < 20; x++ {
		for z := 0; z < 20; z++ {
			l := fillLook3(cube.Pos{x, 70, z})
			variant, mark := l.visual()
			codes := [3]int32{variant % conduit.Codes, variant / conduit.Codes, mark}
			var want [3]int32
			for _, k := range l.k[:l.n] {
				kind, tier, faces := stateProbeParts(k.probeValue())
				if kind != k.kind || tier != k.tier || faces != k.faces {
					t.Fatalf("%+v: probe value decodes to %d %d %d", k, kind, tier, faces)
				}
				want[kind] = 1 + int32(tier-1)*64 + int32(faces)
			}
			if codes != want {
				t.Fatalf("%v: entity codes %v, want %v", l, codes, want)
			}
			// faces are in the order of conduit.Faces for both: bit i is Faces[i]
			if variant < 0 || variant >= conduit.Codes*conduit.Codes || mark < 0 || mark >= conduit.Codes {
				t.Fatalf("variant %d mark %d out of range", variant, mark)
			}
		}
	}
}

// fillEntityLooks reads the Variant and MarkVariant of every conduit display
// entity by block position.
func fillEntityLooks(tx *world.Tx) map[cube.Pos][2]int32 {
	out := map[cube.Pos][2]int32{}
	for e := range tx.Entities() {
		if v, ok := e.(interface {
			Variant() int32
			MarkVariant() int32
		}); ok {
			out[cube.PosFromVec3(e.Position())] = [2]int32{v.Variant(), v.MarkVariant()}
		}
	}
	return out
}

// TestDebugFillEntityMatchesBlock: with the state probe on, the entity mode
// and the block mode of a position show the same kinds, tiers and arms, and the
// pair mode puts the entity on even x and the block of the look of x-1 on odd
// x. With the probe off, block and pair refuse.
func TestDebugFillEntityMatchesBlock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	centre := cube.Pos{0, 64, 2}
	const r = 2
	old := fillUnit
	fillUnit = time.Millisecond
	t.Cleanup(func() { fillUnit = old })
	run := func(p *player.Player, tx *world.Tx, what string) *cmd.Output {
		t.Helper()
		var o cmd.Output
		debugFill{adminBase: adminBase{g: s.g}, Radius: r, Seconds: 600, What: fillEnum(what)}.Run(p, &o, tx)
		return &o
	}
	stop := func(p *player.Player, tx *world.Tx) {
		var o cmd.Output
		debugFillStop{adminBase: adminBase{g: s.g}}.Run(p, &o, tx)
	}
	_, probeOn := stateProbeBlock(0, 1)
	if !probeOn {
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			for _, what := range []string{"block", "pair"} {
				if o := run(p, tx, what); len(o.Errors()) != 1 {
					t.Errorf("%s with the probe off: %v", what, o.Errors())
				}
				if n := fillEntities(tx); n != 0 {
					t.Errorf("%s refused but %d entities", what, n)
				}
				var o cmd.Output
				debugFillStop{adminBase: adminBase{g: s.g}}.Run(p, &o, tx)
				if len(o.Errors()) != 1 {
					t.Errorf("%s refused but a fill runs", what)
				}
			}
		})
		return
	}
	// shapeOf is the look a position holds in blocks: the conduits of the probe
	// blocks on its two layers.
	shapeOf := func(tx *world.Tx, pos cube.Pos) (fillShape, bool) {
		var l fillShape
		probe := false
		for layer := uint8(0); layer < 2; layer++ {
			sp, ok := tx.BlockLayer(pos, layer).(StateProbe)
			if !ok || sp.Layer != int(layer) {
				continue
			}
			probe = true
			look, _ := stateProbeDecode(stateProbeMode, sp.Layer, sp.V)
			for _, k := range look.k[:look.n] {
				l.k[l.n] = k
				l.n++
			}
		}
		if stateProbeFourKinds(stateProbeMode) {
			l = sortedShape(l.k[:l.n])
		}
		return l, probe
	}
	// drawn is what of the look of a position its blocks draw (TestFillPlace
	// checks place).
	drawn := func(l fillShape) fillShape {
		if !stateProbeFourKinds(stateProbeMode) {
			return l
		}
		l0, l1, _ := l.place(stateProbeMode)
		return sortedShape(append(append([]fillKind{}, l0.k[:l0.n]...), l1.k[:l1.n]...))
	}
	positions := func() (out []cube.Pos) {
		for x := -r; x <= r; x++ {
			for y := -r; y <= r; y++ {
				for z := -r; z <= r; z++ {
					out = append(out, centre.Add(cube.Pos{x, y, z}))
				}
			}
		}
		return out
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.Teleport(mgl64.Vec3{0.5, 64, 2.5})
		// entity and block modes: the same look at the same position
		if o := run(p, tx, "entity"); len(o.Errors()) != 0 {
			t.Fatalf("entity: %v", o.Errors())
		}
		ents := fillEntityLooks(tx)
		stop(p, tx)
		if o := run(p, tx, "block"); len(o.Errors()) != 0 {
			t.Fatalf("block: %v", o.Errors())
		}
		checked := 0
		for _, pos := range positions() {
			l, ok := shapeOf(tx, pos)
			if !ok {
				continue
			}
			want := fillLook(pos)
			if l != drawn(want) {
				t.Fatalf("%v: block look %+v, want %+v", pos, l, drawn(want))
			}
			variant, mark := want.visual()
			if got, ok := ents[pos]; !ok || got != [2]int32{variant, mark} {
				t.Fatalf("%v: entity look %v (%v), want %d %d", pos, got, ok, variant, mark)
			}
			checked++
		}
		if checked < 50 {
			t.Errorf("only %d positions compared", checked)
		}
		stop(p, tx)

		// pair: even x the entity, odd x the block of the look of x-1
		if o := run(p, tx, "pair"); len(o.Errors()) != 0 {
			t.Fatalf("pair: %v", o.Errors())
		}
		ents = fillEntityLooks(tx)
		pairs := 0
		for _, pos := range positions() {
			l, isBlock := shapeOf(tx, pos)
			got, isEntity := ents[pos]
			if isBlock == isEntity {
				continue // not filled: it holds the player or a block
			}
			if pos[0]&1 == 0 {
				variant, mark := fillLook(pos).visual()
				if !isEntity || got != [2]int32{variant, mark} {
					t.Fatalf("%v: even x is not the entity of its look", pos)
				}
				continue
			}
			if !isBlock || l != drawn(fillLook(cube.Pos{pos[0] - 1, pos[1], pos[2]})) {
				t.Fatalf("%v: odd x holds %+v, want the look of x-1", pos, l)
			}
			if even, ok := ents[cube.Pos{pos[0] - 1, pos[1], pos[2]}]; ok {
				variant, mark := fillLook(cube.Pos{pos[0] - 1, pos[1], pos[2]}).visual()
				if even != [2]int32{variant, mark} {
					t.Fatalf("%v: neighbour entity %v differs", pos, even)
				}
				pairs++
			}
		}
		if pairs < 10 {
			t.Errorf("only %d neighbouring pairs compared", pairs)
		}
		stop(p, tx)
	})
}

// sortedShape is the look of the conduits given, in the order of their kinds.
func sortedShape(ks []fillKind) fillShape {
	var l fillShape
	for kind := range stateProbeKinds4 {
		for _, k := range ks {
			if k.kind == kind {
				l.k[l.n] = k
				l.n++
			}
		}
	}
	return l
}

// TestFillLook4: the 4-kind look of a position is a function of the position
// with 1 to 4 conduits (about a quarter each) of different kinds in the order
// of their kinds, every kind shows, some looks have two kinds with the same
// faces, and the entity draws all but the network conduit.
func TestFillLook4(t *testing.T) {
	var ns [5]int
	var kinds [4]int
	same, total := 0, 0
	for x := -10; x < 10; x++ {
		for y := 60; y < 65; y++ {
			for z := -10; z < 10; z++ {
				pos := cube.Pos{x, y, z}
				l := fillLook4(pos)
				total++
				if l != fillLook4(pos) {
					t.Fatalf("%v: the look changes", pos)
				}
				if l.n < 1 || l.n > 4 {
					t.Fatalf("%v: %d conduits", pos, l.n)
				}
				ns[l.n]++
				hasSame := false
				for i, k := range l.k[:l.n] {
					if k.tier < 1 || k.tier > 3 || k.faces < 0 || k.faces > 63 {
						t.Fatalf("%v: %+v", pos, l)
					}
					if i > 0 && l.k[i-1].kind >= k.kind {
						t.Fatalf("%v: kinds out of order %+v", pos, l)
					}
					for _, o := range l.k[:i] {
						if o.faces == k.faces {
							hasSame = true
						}
					}
					kinds[k.kind]++
				}
				if hasSame {
					same++
				}
				variant, mark := l.visual()
				var want fillShape
				for _, k := range l.k[:l.n] {
					if k.kind < 3 {
						want.k[want.n] = k
						want.n++
					}
				}
				if wv, wm := want.visual(); wv != variant || wm != mark || l.entityLost() != l.n-want.n {
					t.Fatalf("%v: entity look of %+v", pos, l)
				}
			}
		}
	}
	for n := 1; n <= 4; n++ {
		if ns[n] < total/6 || ns[n] > total/3 {
			t.Errorf("%d of %d looks have %d conduits", ns[n], total, n)
		}
	}
	for k, c := range kinds {
		if c < total/3 {
			t.Errorf("kind %d in only %d of %d looks", k, c, total)
		}
	}
	if same < total/6 {
		t.Errorf("%d of %d looks have two kinds with the same faces", same, total)
	}
}

// TestFillPlace: in every mode, the looks place puts on the two layers encode
// to values of their blocks and draw part of the look, the rest counted as
// lost: perm and bones (3-kind looks) draw both conduits; single4 the first
// two; same4 a same-faces pair on layer 1 when the look has one; all4
// everything, up to two kinds a layer.
func TestFillPlace(t *testing.T) {
	for _, mode := range probeModes {
		var cells, whole, pairs int
		for x := -12; x < 12; x++ {
			for z := -12; z < 12; z++ {
				look := fillLook3(cube.Pos{x, 64, z})
				if stateProbeFourKinds(mode) {
					look = fillLook4(cube.Pos{x, 64, z})
				}
				l0, l1, lost := look.place(mode)
				v0, ok0 := stateProbeEncode(mode, 0, l0)
				v1, ok1 := stateProbeEncode(mode, 1, l1)
				if !ok0 || !ok1 {
					t.Fatalf("%s: %+v places %+v and %+v, not values (%d %v, %d %v)", mode, look, l0, l1, v0, ok0, v1, ok1)
				}
				got := append(append([]fillKind{}, l0.k[:l0.n]...), l1.k[:l1.n]...)
				for _, k := range got {
					found := false
					for _, w := range look.k[:look.n] {
						found = found || w == k
					}
					if !found {
						t.Fatalf("%s: %+v draws %+v", mode, look, k)
					}
				}
				if len(got)+lost != look.n {
					t.Fatalf("%s: %+v draws %d, lost %d", mode, look, len(got), lost)
				}
				cells++
				if lost == 0 {
					whole++
				}
				hasSame := false
				for i := range look.n {
					for j := i + 1; j < look.n; j++ {
						hasSame = hasSame || look.k[i].faces == look.k[j].faces
					}
				}
				switch mode {
				case assets.StateProbeAll4:
					if lost != 0 || l0.n == 0 || (look.n > 2) != (l1.n > 0) {
						t.Fatalf("all4: %+v placed %+v %+v", look, l0, l1)
					}
				case assets.StateProbeSame4:
					if hasSame != (l1.n == 2) {
						t.Fatalf("same4: %+v placed %+v on layer 1", look, l1)
					}
					if l1.n == 2 {
						pairs++
					}
				case assets.StateProbeSingle4, assets.StateProbePerm, assets.StateProbeBones:
					if l0.n != 1 || l0.k[0] != look.k[0] || (look.n > 1) != (l1.n == 1) || (l1.n == 1 && l1.k[0] != look.k[1]) {
						t.Fatalf("%s: %+v placed %+v %+v", mode, look, l0, l1)
					}
				}
			}
		}
		switch mode {
		case assets.StateProbeAll4, assets.StateProbePerm, assets.StateProbeBones:
			if whole != cells {
				t.Errorf("%s: %d of %d cells whole", mode, whole, cells)
			}
		default:
			if whole == cells || whole < cells/3 {
				t.Errorf("%s: %d of %d cells whole", mode, whole, cells)
			}
		}
		if mode == assets.StateProbeSame4 && pairs < cells/8 {
			t.Errorf("same4: %d of %d cells with a pair", pairs, cells)
		}
	}
}
