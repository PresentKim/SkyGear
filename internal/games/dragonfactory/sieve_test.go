package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/item/recipe"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestSieveByHand(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindSieve, "")
	other := placeTestMachine(s, machinePos.Add(cube.Pos{1, 0, 0}), KindSieve, "")
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func() {
			st.lastUse, other.lastUse = time.Time{}, time.Time{}
			if !(Machine{Kind: KindSieve, st: st}).Activate(machinePos, cube.FaceUp, tx, p, nil) {
				t.Fatal("not handled")
			}
		}
		_ = other.inv().SetItem(other.kind.input[0], item.NewStack(block.Sand{}, 1))
		p.SetHeldItems(item.NewStack(block.Gravel{}, 5), item.Stack{})
		use()
		if held, _ := p.HeldItems(); held.Count() != 4 {
			t.Fatalf("held after inserting = %v", held)
		}
		if m := tx.Block(machinePos).(Machine); m.content != 1 || m.fill != sieveStages {
			t.Fatalf("shown content %d fill %d", m.content, m.fill)
		}
		// Holding gravel again sifts instead of inserting more.
		for range sieveStages - 1 {
			use()
		}
		if m := tx.Block(machinePos).(Machine); m.fill != 1 {
			t.Fatalf("fill after %d stages = %d", sieveStages-1, m.fill)
		}
		use()
		if _, _, ok := st.firstInput(); ok {
			t.Fatal("gravel not used up")
		}
		if m := tx.Block(machinePos).(Machine); m.content != 0 || m.fill != 1 {
			t.Fatalf("empty sieve with a mesh shows %d/%d, want 0/1", m.content, m.fill)
		}
		if _, _, ok := other.firstInput(); ok {
			t.Fatal("the sieve next to it did not sift along")
		}
		if held, _ := p.HeldItems(); held.Count() != 4 {
			t.Fatalf("sifting used held items: %v", held)
		}
	})
}

// TestSieveNeedsAMesh: a sieve without a mesh takes nothing; using it with a
// mesh fits it (the old one goes back), and the tier picks the drop table.
func TestSieveNeedsAMesh(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	m := newMachine(KindSieve, "", "")
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(machinePos, m, nil) })
	st := m.st
	if st.accepts(item.NewStack(block.Gravel{}, 1)) || st.insert(item.NewStack(block.Gravel{}, 1)) {
		t.Fatal("a sieve without a mesh takes gravel")
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Gravel{}, 3), item.Stack{})
		if !(Machine{Kind: KindSieve, st: st}).Activate(machinePos, cube.FaceUp, tx, p, nil) {
			t.Fatal("using the sieve was not handled")
		}
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Fatalf("gravel went into a sieve without a mesh: %v", held)
		}
		p.SetHeldItems(item.NewStack(Mesh{Tier: "string"}, 2), item.Stack{})
		(Machine{Kind: KindSieve, st: st}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		if st.mesh != "string" {
			t.Fatal("the mesh was not fitted")
		}
		if held, _ := p.HeldItems(); held.Count() != 1 {
			t.Errorf("held meshes after fitting one: %v", held)
		}
		if !st.accepts(item.NewStack(block.Gravel{}, 1)) {
			t.Error("a sieve with a mesh refuses gravel")
		}
	})
	if drops := (Machine{Kind: KindSieve, st: st}).BreakInfo().Drops(nil, nil); len(drops) != 2 {
		t.Errorf("%d drops, want the sieve and its mesh", len(drops))
	}
	back := Machine{Kind: KindSieve}.DecodeNBT(Machine{Kind: KindSieve, st: st}.EncodeNBT()).(Machine)
	if back.st.mesh != "string" {
		t.Error("the mesh is lost through NBT")
	}
}

// TestSieveTablesGiveOreChunks: the string mesh gives iron chunks only
// (copper from the flint mesh, gold from the iron mesh), with chances high
// enough to see them, and four chunks of an ore make the ore block (recipes
// registered).
func TestSieveTablesGiveOreChunks(t *testing.T) {
	chance := func(tier string, in world.Item, want world.Item) int {
		n := 0
		for _, d := range sieveTables[tier][itemKey(in)] {
			if itemKey(d.Item.Item()) == itemKey(want) {
				n += d.Chance
			}
		}
		return n
	}
	first := map[string]string{OreIron: "string", OreCopper: "flint", OreGold: "iron"}
	for _, k := range oreKinds {
		for tier := range sieveTables {
			sum := chance(tier, block.Gravel{}, OreChunk{Ore: k.ore}) + chance(tier, block.Sand{}, OreChunk{Ore: k.ore})
			if (tier == "string" || tier == "flint" && k.ore == OreGold) && k.ore != OreIron && sum != 0 {
				t.Errorf("the %s mesh gives %s chunks", tier, k.ore)
			}
			if tier == first[k.ore] && sum < 10 {
				t.Errorf("%s chunks are too rare for the %s mesh", k.ore, tier)
			}
		}
		found := false
		for _, r := range recipe.Recipes() {
			if out := r.Output(); len(out) == 1 && recipeKey(out[0].Item()) == recipeKey(k.block) && len(r.Input()) == 4 {
				found = true
			}
		}
		if !found {
			t.Errorf("no recipe makes %T from four chunks", k.block)
		}
	}
	for in, drops := range sieveTables["string"] {
		total := 0
		for _, d := range drops {
			total += d.Chance
		}
		if total < 45 {
			t.Errorf("%s: the drops add up to %d%%, a sift should usually give something", in, total)
		}
	}
}

// TestSieveActsOn3x3: fitting a mesh and putting in a block work on every
// sieve in the 3×3 around the clicked one on its level, not beyond it.
func TestSieveActsOn3x3(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	at := func(dx, dz int) cube.Pos { return machinePos.Add(cube.Pos{dx, 0, dz}) }
	mid := placeTestMachine(s, at(0, 0), KindSieve, "")
	near := placeTestMachine(s, at(1, 1), KindSieve, "")
	far := placeTestMachine(s, at(2, 0), KindSieve, "")
	for _, st := range []*machineState{mid, near, far} {
		st.mesh = ""
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func() {
			(Machine{Kind: KindSieve, st: mid}).Activate(at(0, 0), cube.FaceUp, tx, p, nil)
		}
		p.SetHeldItems(item.NewStack(Mesh{Tier: "string"}, 5), item.Stack{})
		use()
		if mid.mesh != "string" || near.mesh != "" || far.mesh != "" {
			t.Fatalf("meshes %q %q %q: only the clicked sieve takes one", mid.mesh, near.mesh, far.mesh)
		}
		near.mesh, far.mesh = "string", ""
		p.SetHeldItems(item.Stack{}, item.Stack{})
		p.SetHeldItems(item.NewStack(block.Gravel{}, 5), item.Stack{})
		use()
		_, _, a := mid.firstInput()
		_, _, b := near.firstInput()
		_, _, c := far.firstInput()
		if !a || !b || c {
			t.Fatalf("gravel went in: %v %v %v", a, b, c)
		}
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Errorf("held gravel %d, want 3", held.Count())
		}
		if mid.mesh == "" || near.mesh == "" {
			t.Error("putting in blocks took a mesh out")
		}
		use()
		if mid.progress == 0 || near.progress == 0 || far.progress != 0 {
			t.Errorf("sifting progress %d %d %d", mid.progress, near.progress, far.progress)
		}
	})
}

// TestEmptyHandTakesTheMeshOut: using an empty sieve sneaking with an empty hand
// gives the mesh back (only that sieve's); a filled sieve sifts instead.
func TestEmptyHandTakesTheMeshOut(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	a := placeTestMachine(s, machinePos, KindSieve, "")
	b := placeTestMachine(s, machinePos.Add(cube.Pos{1, 0, 0}), KindSieve, "")
	a.mesh, b.mesh = "iron", "iron"
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.Stack{}, item.Stack{})
		(Machine{Kind: KindSieve, st: a}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		if a.mesh == "" {
			t.Fatal("an empty hand without sneaking took the mesh out")
		}
		p.StartSneaking()
		(Machine{Kind: KindSieve, st: a}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		if a.mesh != "" || b.mesh != "iron" {
			t.Fatalf("meshes after taking out: %q %q", a.mesh, b.mesh)
		}
		got := 0
		for _, st := range p.Inventory().Items() {
			if m, ok := st.Item().(Mesh); ok && m.Tier == "iron" {
				got += st.Count()
			}
		}
		if got != 1 {
			t.Errorf("the player got %d iron meshes back", got)
		}
		b.insert(item.NewStack(block.Gravel{}, 1))
		p.StartSneaking()
		(Machine{Kind: KindSieve, st: b}).Activate(machinePos.Add(cube.Pos{1, 0, 0}), cube.FaceUp, tx, p, nil)
		if b.mesh != "iron" {
			t.Error("a filled sieve gave its mesh up")
		}
	})
}
