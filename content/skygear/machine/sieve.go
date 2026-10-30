package machine

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
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
)

// Sieves follow the Dorios' Basics add-on: a frame with a mesh and the block
// being sifted lying on it, sinking as it is sifted. The hand sieve
// (KindSieve) takes one block and sifts a stage per use, dropping the
// products on top; the auto sieve takes stacks, sifts on its own and puts
// its products into its output slots.

func IsSieve(kind string) bool {
	return kind == KindSieve || kind == KindAutoSieve || kind == KindMagicSieve
}

// SieveStages is the number of uses that sift one block by hand, and the
// heights the content shows (half a pixel each).
const SieveStages = 8

// sieveCooldown is the least time between counted uses (clients repeat the
// use packet while the button is held).
const sieveCooldown = 150 * time.Millisecond

// SieveLookState is the one block state of a sieve: the content shown on
// the mesh and its height, as lookValue(content, fill). One state with
// values in registration order that also sort alphabetically: with two
// states (content, fill) the client paired runtime IDs with permutations in
// another order, and the hand sieve showed a different block at every use.
const SieveLookState = "skygear:look"

// lookIndex numbers the look of a sieve: the mesh fitted (0 none, 1 to
// len(sgitem.MeshTiers) the tier), the content and the fill; lookValue is its state
// value.
func lookIndex(mesh, content, fill int) int {
	return (mesh*len(SieveContents)+content)*(SieveStages+1) + fill
}

func LookValue(mesh, content, fill int) string {
	return fmt.Sprintf("%03d", lookIndex(mesh, content, fill))
}

// meshLookOf is the look number of the mesh tier fitted (0: none).
func meshLookOf(tier string) int {
	for i, t := range sgitem.MeshTiers {
		if t.Tier == tier {
			return i + 1
		}
	}
	return 0
}

// SieveLooks are the looks a sieve can have besides the bare one: each mesh
// with nothing on it (fill 1) and with each content at each fill.
func SieveLooks() [][3]int {
	var out [][3]int
	for mesh := 1; mesh <= len(sgitem.MeshTiers); mesh++ {
		out = append(out, [3]int{mesh, 0, 1})
		for c := 1; c < len(SieveContents); c++ {
			for fill := 1; fill <= SieveStages; fill++ {
				out = append(out, [3]int{mesh, c, fill})
			}
		}
	}
	return out
}

// SieveContents are the contents a sieve shows, with their vanilla terrain
// textures (index 0 is empty).
var SieveContents = []struct {
	name string
	item world.Item
}{
	{"none", nil}, {"gravel", block.Gravel{}}, {"sand", block.Sand{}}, {"dirt", block.Dirt{}}, {"skygear_dust", sgitem.Dust{}},
	{"red_sand", block.Sand{Red: true}}, {"soul_sand", block.SoulSand{}},
	{"skygear_crushed_deepslate", sgitem.CrushedBlock{Kind: sgitem.CrushedDeepslate}},
	{"skygear_crushed_netherrack", sgitem.CrushedBlock{Kind: sgitem.CrushedNetherrack}},
	{"skygear_crushed_blackstone", sgitem.CrushedBlock{Kind: sgitem.CrushedBlackstone}},
	{"skygear_crushed_end_stone", sgitem.CrushedBlock{Kind: sgitem.CrushedEndStone}},
}

func SieveContentOf(it item.Stack) int {
	if it.Empty() {
		return 0
	}
	for i, c := range SieveContents[1:] {
		if naming.ItemKey(c.item) == naming.ItemKey(it.Item()) {
			return i + 1
		}
	}
	return 0
}

// sieveMaterials are the materials of a sieve look; without a mesh fitted
// the mesh is a transparent texture, so the sieve looks empty.
func sieveMaterials(kind string, content, mesh int) map[string]customblock.Material {
	meshTexture := "skygear_sieve_nomesh"
	if mesh > 0 {
		meshTexture = "skygear_sieve_mesh_" + sgitem.MeshTiers[mesh-1].Tier
	}
	m := map[string]customblock.Material{
		"*":    customblock.NewMaterial("skygear_"+kind, customblock.AlphaTestRenderMethod()),
		"mesh": customblock.NewMaterial(meshTexture, customblock.AlphaTestRenderMethod()),
	}
	if content > 0 {
		m["content"] = customblock.NewMaterial(SieveContents[content].name, customblock.AlphaTestRenderMethod())
	}
	return m
}

func sieveProperties(kind string) customblock.Properties {
	return customblock.Properties{Geometry: "geometry.skygear_" + kind + "_0", Textures: sieveMaterials(kind, 0, 0)}
}

func SieveStates() map[string][]any {
	values := []any{LookValue(0, 0, 0)}
	for _, l := range SieveLooks() {
		values = append(values, LookValue(l[0], l[1], l[2]))
	}
	return map[string][]any{SieveLookState: values}
}

func sievePermutations(kind string) []customblock.Permutation {
	var out []customblock.Permutation
	for _, l := range SieveLooks() {
		mesh, c, fill := l[0], l[1], l[2]
		out = append(out, customblock.Permutation{
			Condition: fmt.Sprintf("q.block_state('%s') == '%s'", SieveLookState, LookValue(mesh, c, fill)),
			Properties: customblock.Properties{
				Geometry: fmt.Sprintf("geometry.skygear_%s_%d", kind, fill*min(c, 1)),
				Textures: sieveMaterials(kind, c, mesh),
			},
		})
	}
	return out
}

// SieveLook returns the mesh, content and fill a sieve shows for its state:
// no mesh (0, 0, 0), a mesh and nothing on it (mesh, 0, 1).
func SieveLook(st *State) (mesh, content, fill int) {
	if mesh = meshLookOf(st.Mesh); mesh == 0 {
		return 0, 0, 0
	}
	_, in, ok := st.FirstInput()
	if !ok {
		return mesh, 0, 1
	}
	if content = SieveContentOf(in); content == 0 {
		return mesh, 0, 1
	}
	fill = SieveStages - st.Progress*SieveStages/st.Def.Interval
	return mesh, content, max(1, min(SieveStages, fill))
}

// showSieve sets the sieve block to show its content when it changed.
func showSieve(tx *world.Tx, pos cube.Pos, st *State) {
	m, ok := tx.Block(pos).(Machine)
	if !ok || m.State != st {
		return
	}
	mesh, content, fill := SieveLook(st)
	if m.MeshLook == mesh && m.Content == content && m.Fill == fill {
		return
	}
	m.MeshLook, m.Content, m.Fill = mesh, content, fill
	tx.SetBlock(pos, m, nil)
}

// sieveDrops rolls the products of sifting one in.
func sieveDrops(s *core.Session, in item.Stack, mesh string) []item.Stack {
	var out []item.Stack
	for _, d := range sievetable.Tables[mesh][naming.ItemKey(in.Item())] {
		// every item of the drop's stack is its own roll
		n := 0
		for range d.Item.Count() {
			if s.Rand.IntN(1000) < d.Permille {
				n++
			}
		}
		if n > 0 {
			out = append(out, d.Item.Grow(n-d.Item.Count()))
		}
	}
	return out
}

// sieveHolds reports whether a held item is something a sieve sifts.
func sieveHolds(it item.Stack) bool {
	if it.Empty() || !core.Plain(it) {
		return false
	}
	_, ok := sievetable.Tables["string"][naming.ItemKey(it.Item())]
	return ok
}

// sieveReach is how far from the clicked sieve other sieves on the same
// level work along: the 5×5 around it.
const sieveReach = 2

// sieveAt is a sieve block of the game with its state.
type sieveAt struct {
	pos   cube.Pos
	State *State
}

// sievesAround are the sieves in the 5×5 on the level of pos, the clicked
// one first.
func sievesAround(tx *world.Tx, pos cube.Pos, clicked *State) []sieveAt {
	out := []sieveAt{{pos, clicked}}
	for x := -sieveReach; x <= sieveReach; x++ {
		for z := -sieveReach; z <= sieveReach; z++ {
			other := pos.Add(cube.Pos{x, 0, z})
			if m, ok := tx.Block(other).(Machine); ok && other != pos && IsSieve(m.Kind) && m.State != nil {
				out = append(out, sieveAt{other, m.State})
			}
		}
	}
	return out
}

// useSieve handles using a sieve: fitting a mesh, putting in a block (the
// hand sieve takes one, the auto sieve the held stack) or sifting a stage.
// Putting in and sifting work on every sieve in the 5×5 around the clicked
// one at once; the mesh only on the clicked one. When the hand holds fewer
// blocks than the empty hand sieves of the group take, more of the same
// block comes into the hand from the inventory, and a hand left empty is
// filled again with it.
func useSieve(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, st *State) {
	held, off := p.HeldItems()
	group := sievesAround(tx, pos, st)
	// Fitting and taking out a mesh work on the clicked sieve only.
	if _, ok := held.Item().(sgitem.Mesh); ok {
		equipMesh(s, p, tx, pos, st, held, off)
		return
	}
	if held.Empty() && p.Sneaking() && takeMesh(s, p, tx, pos, st) {
		return
	}
	if st.Mesh == "" && sieveHolds(held) {
		p.SendTip(i18n.T(i18n.Of(p), "sieve.needMesh"))
		return
	}
	if sieveHolds(held) {
		proto := held.Grow(1 - held.Count())
		topUpHand(p, proto, emptyHandSieves(group, proto))
		put := 0
		for _, sv := range group {
			if held, off = p.HeldItems(); held.Empty() {
				break
			}
			put += putIntoSieve(s, p, tx, sv, held, off)
		}
		if held, _ = p.HeldItems(); held.Empty() {
			topUpHand(p, proto, proto.MaxCount())
		}
		if put > 0 {
			return
		}
	}
	if st.Def.autoSieve {
		// it sifts on its own, a click only opens its window
		st.OpenWindow(p)
		return
	}
	swung := false
	for _, sv := range group {
		if sv.State.Def.autoSieve {
			continue
		}
		if _, in, ok := sv.State.FirstInput(); ok && time.Since(sv.State.LastUse) >= sieveCooldown {
			if !swung {
				p.SwingArm()
				swung = true
			}
			if siftStage(s, tx, sv.pos, sv.State, in) {
				achievement.Count(p, "sift_hand", 1)
			}
		}
	}
}

// emptyHandSieves counts the hand sieves of a group that would take one of
// it now (empty and with a mesh that sifts it).
func emptyHandSieves(group []sieveAt, it item.Stack) int {
	n := 0
	for _, sv := range group {
		if _, _, filled := sv.State.FirstInput(); !filled && !sv.State.Def.autoSieve && sv.State.Accepts(it) {
			n++
		}
	}
	return n
}

// topUpHand brings more of proto from the rest of p's inventory into its main
// hand until the hand holds want of it (a stack at most). The hand must hold
// proto or nothing. The hand is emptied while the other slots are searched,
// so its own slot is never counted twice.
func topUpHand(p *player.Player, proto item.Stack, want int) {
	held, off := p.HeldItems()
	if !held.Empty() && !held.Comparable(proto) {
		return
	}
	want = min(want, proto.MaxCount())
	if held.Count() >= want {
		return
	}
	got := held
	p.SetHeldItems(item.Stack{}, off)
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if got.Count() >= want {
			break
		}
		if it.Empty() || !it.Comparable(proto) {
			continue
		}
		n := min(it.Count(), want-got.Count())
		_ = inv.SetItem(slot, it.Grow(-n))
		if got.Empty() {
			got = it.Grow(n - it.Count())
		} else {
			got = got.Grow(n)
		}
	}
	p.SetHeldItems(got, off)
}

// putIntoSieve puts the held item into one sieve (the hand sieve takes one
// block while it is empty, the auto sieve the held stack) and returns how
// many it put.
func putIntoSieve(s *core.Session, p *player.Player, tx *world.Tx, sv sieveAt, held, off item.Stack) int {
	st := sv.State
	_, _, filled := st.FirstInput()
	if !st.Accepts(held) || (filled && !st.Def.autoSieve) {
		return 0
	}
	n := 1
	if st.Def.autoSieve {
		n = held.Count()
	}
	put := 0
	for put < n && st.Insert(held) {
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
func siftStage(s *core.Session, tx *world.Tx, pos cube.Pos, st *State, in item.Stack) (finished bool) {
	st.LastUse = time.Now()
	if b, ok := in.Item().(world.Block); ok {
		tx.PlaySound(pos.Vec3Centre(), sound.BlockBreaking{Block: b})
	}
	st.Progress += (st.Def.Interval + SieveStages - 1) / SieveStages
	if st.Progress >= st.Def.Interval {
		finishSifting(s, tx, pos, st, false)
		finished = true
	}
	showSieve(tx, pos, st)
	return finished
}

// finishSifting uses up one input block and gives its products: dropped on
// top of a hand sieve, or into the output slots of an auto sieve (toOutput),
// what does not fit waits in pending.
func finishSifting(s *core.Session, tx *world.Tx, pos cube.Pos, st *State, toOutput bool) {
	st.Progress = 0
	slot, in, ok := st.FirstInput()
	if !ok {
		return
	}
	_ = st.Inv().SetItem(slot, in.Grow(-1))
	progress.Advance(s, tx, advancement.Sift)
	achievement.CountAll(tx, "sift_mesh_"+st.Mesh, 1)
	drops := SiftDrops(s, st, in)
	for _, it := range drops {
		if !toOutput {
			recordMade(s, it)
			s.DropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 0.6, 0}), it)
		} else if st.RoomFor([]item.Stack{it}) {
			st.Emit(it)
		} else {
			st.pending = append(st.pending, it)
		}
	}
}

// outputHasRoom reports whether an output slot is empty or not yet full.
func outputHasRoom(st *State) bool {
	for _, slot := range st.Def.Output {
		if it, _ := st.Inv().Item(slot); it.Empty() || it.Count() < it.MaxCount() {
			return true
		}
	}
	return false
}

// runAutoSieve sifts the input of an auto sieve on its own.
func runAutoSieve(s *core.Session, st *State, pos cube.Pos, tx *world.Tx) {
	// products that did not fit go first; while any is left the sieve waits
	for len(st.pending) > 0 {
		if !st.RoomFor(st.pending[:1]) {
			return
		}
		st.Emit(st.pending[0])
		st.pending = st.pending[1:]
	}
	if _, _, ok := st.FirstInput(); !ok {
		st.Progress = 0
		return
	}
	if !outputHasRoom(st) {
		st.Stuck = StuckOutput
		return
	}
	ChargeSieve(s, st)
	if !st.TickWork() {
		return
	}
	if st.Progress >= st.Def.Interval {
		finishSifting(s, tx, pos, st, true)
		st.Done++
	}
}
