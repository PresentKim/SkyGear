package dragonfactory

import (
	"fmt"
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
// it again sifts one stage (every sieve within two blocks on the same level
// sifts along), and the last stage drops the products on top. There is no
// window: the products are never stored, since a block gives several kinds.
//
// The hand sieve (KindSieve) takes one block at a time and sifts only by
// hand. The auto sieve (KindAutoSieve) also takes stacks from hoppers or the
// hand, sifts on its own, and puts its products into containers beside or
// below it, dropping them on top when none takes them.

func isSieve(kind string) bool { return kind == KindSieve || kind == KindAutoSieve }

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

func sieveMaterials(kind string, content int) map[string]customblock.Material {
	m := map[string]customblock.Material{
		"*":    customblock.NewMaterial("dragonfactory_"+kind, customblock.AlphaTestRenderMethod()),
		"mesh": customblock.NewMaterial("dragonfactory_sieve_mesh", customblock.AlphaTestRenderMethod()),
	}
	if content > 0 {
		m["content"] = customblock.NewMaterial(sieveContents[content].name, customblock.AlphaTestRenderMethod())
	}
	return m
}

func sieveProperties(kind string) customblock.Properties {
	return customblock.Properties{Geometry: "geometry.dragonfactory_" + kind + "_0", Textures: sieveMaterials(kind, 0)}
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
	var out []customblock.Permutation
	for c := 1; c < len(sieveContents); c++ {
		for fill := 1; fill <= sieveStages; fill++ {
			out = append(out, customblock.Permutation{
				Condition: fmt.Sprintf("q.block_state('%s') == '%s'", sieveLookState, lookValue(c, fill)),
				Properties: customblock.Properties{
					Geometry: fmt.Sprintf("geometry.dragonfactory_%s_%d", kind, fill),
					Textures: sieveMaterials(kind, c),
				},
			})
		}
	}
	return out
}

// sieveLook returns the content and fill a sieve shows for its state.
func sieveLook(st *machineState) (content, fill int) {
	_, in, ok := st.firstInput()
	if !ok {
		return 0, 0
	}
	if content = sieveContentOf(in); content == 0 {
		return 0, 0
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
func (s *session) sieveDrops(in item.Stack) []item.Stack {
	var out []item.Stack
	for _, d := range sieveRecipes[itemKey(in.Item())] {
		if s.rand.IntN(100) < d.Chance {
			out = append(out, d.Item)
		}
	}
	return out
}

// useSieve handles using a sieve: putting in a block (the hand sieve takes
// one, the auto sieve the held stack) or sifting a stage.
func (s *session) useSieve(p *player.Player, tx *world.Tx, pos cube.Pos, st *machineState) {
	held, off := p.HeldItems()
	_, in, filled := st.firstInput()
	if st.accepts(held) && (!filled || st.kind == machineKinds[KindAutoSieve]) {
		n := 1
		if st.kind == machineKinds[KindAutoSieve] {
			n = held.Count()
		}
		put := 0
		for put < n && st.insert(held) {
			put++
		}
		if put > 0 {
			p.SetHeldItems(held.Grow(-put), off)
			tx.PlaySound(pos.Vec3Centre(), sound.BlockPlace{Block: held.Item().(world.Block)})
			showSieve(tx, pos, st)
			return
		}
	}
	if !filled || time.Since(st.lastUse) < sieveCooldown {
		return
	}
	p.SwingArm()
	s.siftStage(tx, pos, st, in)
	for x := -2; x <= 2; x++ {
		for z := -2; z <= 2; z++ {
			other := pos.Add(cube.Pos{x, 0, z})
			if m, ok := tx.Block(other).(Machine); ok && other != pos && isSieve(m.Kind) && m.st != nil {
				if _, in, ok := m.st.firstInput(); ok && time.Since(m.st.lastUse) >= sieveCooldown {
					s.siftStage(tx, other, m.st, in)
				}
			}
		}
	}
}

// siftStage sifts one stage of in by hand. The last stage drops the
// products on top of the sieve.
func (s *session) siftStage(tx *world.Tx, pos cube.Pos, st *machineState, in item.Stack) {
	st.lastUse = time.Now()
	if b, ok := in.Item().(world.Block); ok {
		tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: b})
	}
	st.progress += (st.kind.interval + sieveStages - 1) / sieveStages
	if st.progress >= st.kind.interval {
		s.finishSifting(tx, pos, st, false)
	}
	showSieve(tx, pos, st)
}

// finishSifting uses up one input block and gives its products: dropped on
// top, or for the auto sieve sifting on its own (toContainers) put into the
// containers beside or below it first.
func (s *session) finishSifting(tx *world.Tx, pos cube.Pos, st *machineState, toContainers bool) {
	st.progress = 0
	slot, in, ok := st.firstInput()
	if !ok {
		return
	}
	_ = st.inv().SetItem(slot, in.Grow(-1))
	s.advance(tx, advSift)
	for _, it := range s.sieveDrops(in) {
		if toContainers {
			if it = s.storeBesideOrBelow(tx, pos, it); it.Empty() {
				continue
			}
		}
		s.dropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 0.6, 0}), it)
	}
}

// sieveTargets are the sides an auto sieve puts its products into: never
// up, where the products drop.
var sieveTargets = []cube.Face{cube.FaceDown, cube.FaceNorth, cube.FaceEast, cube.FaceSouth, cube.FaceWest}

// storeBesideOrBelow puts it into the containers beside or below pos and
// returns what none of them took.
func (s *session) storeBesideOrBelow(tx *world.Tx, pos cube.Pos, it item.Stack) item.Stack {
	for _, f := range sieveTargets {
		c, ok := tx.Block(pos.Side(f)).(block.Container)
		if !ok {
			continue
		}
		n, _ := c.Inventory(tx, pos.Side(f)).AddItem(it)
		if it = it.Grow(-n); it.Empty() {
			break
		}
	}
	return it
}

// runAutoSieve sifts the input of an auto sieve on its own.
func (s *session) runAutoSieve(st *machineState, pos cube.Pos, tx *world.Tx) {
	if _, _, ok := st.firstInput(); !ok {
		st.progress = 0
		return
	}
	if st.progress++; st.progress >= st.kind.interval {
		s.finishSifting(tx, pos, st, true)
	}
}
