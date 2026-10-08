package dragonfactory

import (
	"fmt"
	"image"
	"image/color"
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

// sieveStages is the number of uses that sift one block by hand.
const sieveStages = 4

// sieveCooldown is the least time between counted uses (clients repeat the
// use packet while the button is held).
const sieveCooldown = 150 * time.Millisecond

// Block states of the sieve: the content shown on the mesh and its height.
const (
	sieveContentState = "dragonfactory:content"
	sieveFillState    = "dragonfactory:fill"
)

// sieveContents are the contents a sieve shows, with their vanilla terrain
// textures (index 0 is empty).
var sieveContents = []struct {
	name string
	item world.Item
}{{"none", nil}, {"gravel", block.Gravel{}}, {"sand", block.Sand{}}, {"dirt", block.Dirt{}}}

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

var (
	sieveMeshTexture   = meshTexture()
	sieveFrameTextures = map[string]image.Image{
		KindSieve:     woodTexture(rgb(176, 132, 82)),
		KindAutoSieve: woodTexture(rgb(150, 150, 160)),
	}
	sieveGeometries = map[string][]byte{KindSieve: sieveModel(KindSieve), KindAutoSieve: sieveModel(KindAutoSieve)}
)

// sieveModel returns the geometries of a sieve kind:
// geometry.dragonfactory_<kind>_<n> with the content n pixels high (0 =
// empty).
func sieveModel(kind string) []byte {
	frame := []box{
		{from: [3]float64{-7, 0, -7}, size: [3]float64{2, 10, 2}},
		{from: [3]float64{5, 0, -7}, size: [3]float64{2, 10, 2}},
		{from: [3]float64{-7, 0, 5}, size: [3]float64{2, 10, 2}},
		{from: [3]float64{5, 0, 5}, size: [3]float64{2, 10, 2}},
		{from: [3]float64{-7, 10, -7}, size: [3]float64{14, 5, 1}},
		{from: [3]float64{-7, 10, 6}, size: [3]float64{14, 5, 1}},
		{from: [3]float64{-7, 10, -6}, size: [3]float64{1, 5, 12}},
		{from: [3]float64{6, 10, -6}, size: [3]float64{1, 5, 12}},
		{from: [3]float64{-6, 10, -6}, size: [3]float64{12, 1, 12}, mat: "mesh"},
	}
	var geos []map[string]any
	for fill := range sieveStages + 1 {
		boxes := append([]box{}, frame...)
		if fill > 0 {
			boxes = append(boxes, box{from: [3]float64{-6, 11, -6}, size: [3]float64{12, float64(fill), 12}, mat: "content"})
		}
		geos = append(geos, geometry(fmt.Sprintf("geometry.dragonfactory_%s_%d", kind, fill), boxes))
	}
	return geoFile(geos...)
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
	contents := make([]any, len(sieveContents))
	for i, c := range sieveContents {
		contents[i] = c.name
	}
	fills := make([]any, sieveStages+1)
	for i := range fills {
		fills[i] = fmt.Sprint(i)
	}
	return map[string][]any{sieveContentState: contents, sieveFillState: fills}
}

func sievePermutations(kind string) []customblock.Permutation {
	var out []customblock.Permutation
	for c := 1; c < len(sieveContents); c++ {
		for fill := 1; fill <= sieveStages; fill++ {
			out = append(out, customblock.Permutation{
				Condition: fmt.Sprintf("q.block_state('%s') == '%s' && q.block_state('%s') == '%d'", sieveContentState, sieveContents[c].name, sieveFillState, fill),
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
	st.progress += st.kind.interval / sieveStages
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

// woodTexture draws planks of colour light for a sieve frame.
func woodTexture(light color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	mid, dark := shade(light, 0.85), shade(light, 0.62)
	for y := range 16 {
		for x := range 16 {
			c := light
			if (x+y*3)%7 == 0 {
				c = mid
			}
			if y%4 == 3 {
				c = dark
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// meshTexture draws a string mesh with transparent holes.
func meshTexture() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	str := color.RGBA{R: 225, G: 220, B: 205, A: 255}
	for y := range 16 {
		for x := range 16 {
			if x%3 == 0 || y%3 == 0 {
				img.Set(x, y, str)
			}
		}
	}
	return img
}
