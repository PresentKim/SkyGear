package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// Sieves work like the sieve of the Dorios' Basics add-on: a wooden frame
// on legs with a mesh, and the block being sifted lying on the mesh, sinking
// as it is sifted. Using a sieve with gravel, sand or dirt puts it in; using
// it again sifts one stage (every sieve in the 3×3 around it on the same
// level takes part), and the last stage drops the products on top. There is no
// window: the products are never stored, since a block gives several kinds.
//
// The hand sieve (KindSieve) takes one block at a time and sifts only by
// hand. The auto sieve (KindAutoSieve) also takes stacks from hoppers or the
// hand, sifts on its own, and puts its products into containers beside or
// below it, dropping them on top when none takes them.

func isSieve(kind string) bool {
	return kind == KindSieve || kind == KindAutoSieve || kind == KindMagicSieve
}

// sieveStages is the number of uses that sift one block by hand, and the
// heights the content shows (half a pixel each).
const sieveStages = 8

// sieveCooldown is the least time between counted uses (clients repeat the
// use packet while the button is held).
const sieveCooldown = 150 * time.Millisecond

// sieveLookState is the one block state of a sieve: the content shown on
// the mesh and its height, as lookValue(content, fill). One state with
// values in registration order that also sort alphabetically: with two
// states (content, fill) the client paired runtime IDs with permutations in
// another order, and the hand sieve showed a different block at every use.
const sieveLookState = "dragonfactory:look"

// lookIndex numbers a content and fill; lookValue is its state value.
func lookIndex(content, fill int) int { return content*(sieveStages+1) + fill }
func lookValue(content, fill int) string {
	return fmt.Sprintf("%02d", lookIndex(content, fill))
}

// sieveContents are the contents a sieve shows, with their vanilla terrain
// textures (index 0 is empty).
var sieveContents = []struct {
	name string
	item world.Item
}{{"none", nil}, {"gravel", block.Gravel{}}, {"sand", block.Sand{}}, {"dirt", block.Dirt{}}, {"dragonfactory_dust", Dust{}}}

func sieveContentOf(it item.Stack) int {
	if it.Empty() {
		return 0
	}
	for i, c := range sieveContents[1:] {
		if itemKey(c.item) == itemKey(it.Item()) {
			return i + 1
		}
	}
	return 0
}

// sieveMaterials are the materials of a sieve look; without a mesh fitted
// the mesh is a transparent texture, so the sieve looks empty.
func sieveMaterials(kind string, content int, meshed bool) map[string]customblock.Material {
	mesh := "dragonfactory_sieve_nomesh"
	if meshed {
		mesh = "dragonfactory_sieve_mesh"
	}
	m := map[string]customblock.Material{
		"*":    customblock.NewMaterial("dragonfactory_"+kind, customblock.AlphaTestRenderMethod()),
		"mesh": customblock.NewMaterial(mesh, customblock.AlphaTestRenderMethod()),
	}
	if content > 0 {
		m["content"] = customblock.NewMaterial(sieveContents[content].name, customblock.AlphaTestRenderMethod())
	}
	return m
}

func sieveProperties(kind string) customblock.Properties {
	return customblock.Properties{Geometry: "geometry.dragonfactory_" + kind + "_0", Textures: sieveMaterials(kind, 0, false)}
}

func sieveStates() map[string][]any {
	var values []any
	for c := range sieveContents {
		for fill := range sieveStages + 1 {
			values = append(values, lookValue(c, fill))
		}
	}
	return map[string][]any{sieveLookState: values}
}

func sievePermutations(kind string) []customblock.Permutation {
	// Look (0, 1): a mesh fitted and nothing on it.
	out := []customblock.Permutation{{
		Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", sieveLookState, lookValue(0, 1)),
		Properties: customblock.Properties{Geometry: "geometry.dragonfactory_" + kind + "_0", Textures: sieveMaterials(kind, 0, true)},
	}}
	for c := 1; c < len(sieveContents); c++ {
		for fill := 1; fill <= sieveStages; fill++ {
			out = append(out, customblock.Permutation{
				Condition: fmt.Sprintf("q.block_state('%s') == '%s'", sieveLookState, lookValue(c, fill)),
				Properties: customblock.Properties{
					Geometry: fmt.Sprintf("geometry.dragonfactory_%s_%d", kind, fill),
					Textures: sieveMaterials(kind, c, true),
				},
			})
		}
	}
	return out
}

// sieveLook returns the content and fill a sieve shows for its state: no
// mesh (0, 0), a mesh and nothing on it (0, 1).
func sieveLook(st *machineState) (content, fill int) {
	if st.mesh == "" {
		return 0, 0
	}
	_, in, ok := st.firstInput()
	if !ok {
		return 0, 1
	}
	if content = sieveContentOf(in); content == 0 {
		return 0, 1
	}
	fill = sieveStages - st.progress*sieveStages/st.kind.interval
	return content, max(1, min(sieveStages, fill))
}

// showSieve sets the sieve block to show its content when it changed.
func showSieve(tx *world.Tx, pos cube.Pos, st *machineState) {
	m, ok := tx.Block(pos).(Machine)
	if !ok || m.st != st {
		return
	}
	content, fill := sieveLook(st)
	if m.content == content && m.fill == fill {
		return
	}
	m.content, m.fill = content, fill
	tx.SetBlock(pos, m, nil)
}

// sieveDrops rolls the products of sifting one in.
func (s *session) sieveDrops(in item.Stack, mesh string) []item.Stack {
	var out []item.Stack
	for _, d := range sieveTables[mesh][itemKey(in.Item())] {
		if s.rand.IntN(100) < d.Chance {
			out = append(out, d.Item)
		}
	}
	return out
}

// sieveHolds reports whether a held item is something a sieve sifts.
func sieveHolds(it item.Stack) bool {
	if it.Empty() || !plain(it) {
		return false
	}
	_, ok := sieveTables["string"][itemKey(it.Item())]
	return ok
}

// sieveReach is how far from the clicked sieve other sieves on the same
// level work along: the 3×3 around it.
const sieveReach = 1

// sieveAt is a sieve block of the game with its state.
type sieveAt struct {
	pos cube.Pos
	st  *machineState
}

// sievesAround are the sieves in the 3×3 on the level of pos, the clicked
// one first.
func sievesAround(tx *world.Tx, pos cube.Pos, clicked *machineState) []sieveAt {
	out := []sieveAt{{pos, clicked}}
	for x := -sieveReach; x <= sieveReach; x++ {
		for z := -sieveReach; z <= sieveReach; z++ {
			other := pos.Add(cube.Pos{x, 0, z})
			if m, ok := tx.Block(other).(Machine); ok && other != pos && isSieve(m.Kind) && m.st != nil {
				out = append(out, sieveAt{other, m.st})
			}
		}
	}
	return out
}

// useSieve handles using a sieve: fitting a mesh, putting in a block (the
// hand sieve takes one, the auto sieve the held stack) or sifting a stage.
// Putting in and sifting work on every sieve in the 3×3 around the clicked
// one at once; the mesh only on the clicked one.
func (s *session) useSieve(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState) {
	held, off := p.HeldItems()
	group := sievesAround(tx, pos, st)
	// Fitting and taking out a mesh work on the clicked sieve only.
	if _, ok := held.Item().(Mesh); ok {
		s.equipMesh(p, tx, pos, st, held, off)
		return
	}
	if held.Empty() && p.Sneaking() && s.takeMesh(p, tx, pos, st) {
		return
	}
	if st.mesh == "" && sieveHolds(held) {
		p.SendTip(i18n.T(i18n.Of(p), "dragonfactory.sieve.useSieve.1"))
		return
	}
	if sieveHolds(held) {
		put := 0
		for _, sv := range group {
			if held, off = p.HeldItems(); held.Empty() {
				break
			}
			put += s.putIntoSieve(p, tx, sv, held, off)
		}
		if put > 0 {
			return
		}
	}
	if st.kind.autoSieve {
		// it sifts on its own, a click only opens its window
		st.storage.Open(p, nil)
		return
	}
	swung := false
	for _, sv := range group {
		if sv.st.kind.autoSieve {
			continue
		}
		if _, in, ok := sv.st.firstInput(); ok && time.Since(sv.st.lastUse) >= sieveCooldown {
			if !swung {
				p.SwingArm()
				swung = true
			}
			if s.siftStage(tx, sv.pos, sv.st, in) {
				count(p, "sift_hand", 1)
			}
		}
	}
}

// putIntoSieve puts the held item into one sieve (the hand sieve takes one
// block while it is empty, the auto sieve the held stack) and returns how
// many it put.
func (s *session) putIntoSieve(p *player.Player, tx *world.Tx, sv sieveAt, held, off item.Stack) int {
	st := sv.st
	_, _, filled := st.firstInput()
	if !st.accepts(held) || (filled && !st.kind.autoSieve) {
		return 0
	}
	n := 1
	if st.kind.autoSieve {
		n = held.Count()
	}
	put := 0
	for put < n && st.insert(held) {
		put++
	}
	if put > 0 {
		p.SetHeldItems(held.Grow(-put), off)
		tx.PlaySound(sv.pos.Vec3Centre(), sound.BlockPlace{Block: held.Item().(world.Block)})
		showSieve(tx, sv.pos, st)
	}
	return put
}

// siftStage sifts one stage of in by hand. The last stage drops the
// products on top of the sieve.
func (s *session) siftStage(tx *world.Tx, pos cube.Pos, st *machineState, in item.Stack) (finished bool) {
	st.lastUse = time.Now()
	if b, ok := in.Item().(world.Block); ok {
		tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: b})
	}
	st.progress += (st.kind.interval + sieveStages - 1) / sieveStages
	if st.progress >= st.kind.interval {
		s.finishSifting(tx, pos, st, false)
		finished = true
	}
	showSieve(tx, pos, st)
	return finished
}

// finishSifting uses up one input block and gives its products: dropped on
// top of a hand sieve, or into the output slots of an auto sieve (toOutput),
// what does not fit waits in pending.
func (s *session) finishSifting(tx *world.Tx, pos cube.Pos, st *machineState, toOutput bool) {
	st.progress = 0
	slot, in, ok := st.firstInput()
	if !ok {
		return
	}
	_ = st.inv().SetItem(slot, in.Grow(-1))
	s.advance(tx, advSift)
	countAll(tx, "sift_mesh_"+st.mesh, 1)
	drops := s.siftDrops(st, in)
	for _, it := range drops {
		if !toOutput {
			s.recordMade(it)
			s.dropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 0.6, 0}), it)
		} else if st.roomFor([]item.Stack{it}) {
			st.emit(it)
		} else {
			st.pending = append(st.pending, it)
		}
	}
}

// outputHasRoom reports whether an output slot is empty or not yet full.
func outputHasRoom(st *machineState) bool {
	for _, slot := range st.kind.output {
		if it, _ := st.inv().Item(slot); it.Empty() || it.Count() < it.MaxCount() {
			return true
		}
	}
	return false
}

// runAutoSieve sifts the input of an auto sieve on its own.
func (s *session) runAutoSieve(st *machineState, pos cube.Pos, tx *world.Tx) {
	// products that did not fit go first; while any is left the sieve waits
	for len(st.pending) > 0 {
		if !st.roomFor(st.pending[:1]) {
			return
		}
		st.emit(st.pending[0])
		st.pending = st.pending[1:]
	}
	if _, _, ok := st.firstInput(); !ok {
		st.progress = 0
		return
	}
	if !outputHasRoom(st) {
		st.stuck = stuckOutput
		return
	}
	s.chargeSieve(st)
	if !st.tickWork() {
		return
	}
	if st.progress >= st.kind.interval {
		s.finishSifting(tx, pos, st, true)
		st.done++
	}
}
