package skygear

import (
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/core"

	"github.com/df-mc/dragonfly/server/item/recipe"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
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
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(machinePos, m, nil) })
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
		p.SetHeldItems(item.NewStack(sgitem.Mesh{Tier: "string"}, 2), item.Stack{})
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

// TestSieveTablesGiveOreChunks: gravel gives every ore's chunks from the
// string mesh on, as ATM10: To the Sky does,
// sand none, and four chunks of an ore make the ore block (recipes
// registered).
func TestSieveTablesGiveOreChunks(t *testing.T) {
	chance := func(tier string, in world.Item, want world.Item) int {
		n := 0
		for _, d := range sievetable.Tables[tier][naming.ItemKey(in)] {
			if naming.ItemKey(d.Item.Item()) == naming.ItemKey(want) {
				n += d.Permille
			}
		}
		return n
	}
	want := map[string]int{sgitem.OreIron: 100, sgitem.OreCopper: 80, sgitem.OreGold: 30, sgitem.MetalTin: 60, sgitem.MetalLead: 40, sgitem.MetalSilver: 40, sgitem.MetalNickel: 40}
	for _, k := range sgitem.OreKinds {
		if got := chance("string", block.Gravel{}, sgitem.OreChunk{Ore: k.Ore}); got != want[k.Ore] {
			t.Errorf("%s chunks from gravel on the string mesh: %d permille, want %d", k.Ore, got, want[k.Ore])
		}
		for tier := range sievetable.Tables {
			if chance(tier, block.Sand{}, sgitem.OreChunk{Ore: k.Ore}) != 0 {
				t.Errorf("sand gives %s chunks on the %s mesh", k.Ore, tier)
			}
		}
		if k.Block == nil {
			continue
		}
		found := false
		for _, r := range recipe.Recipes() {
			if out := r.Output(); len(out) == 1 && recipeKey(out[0].Item()) == recipeKey(k.Block) && len(r.Input()) == 4 {
				found = true
			}
		}
		if !found {
			t.Errorf("no recipe makes %T from four chunks", k.Block)
		}
	}
	for in, drops := range sievetable.Tables["string"] {
		total := 0
		for _, d := range drops {
			total += d.Permille
		}
		// crushed end stone is the thinnest (ATM10 without the chorus Bedrock servers lack)
		if total < 80 {
			t.Errorf("%s: the drops add up to %d per mille, a sift should often give something", in, total)
		}
	}
}

// TestSieveActsOn5x5: fitting a mesh and putting in a block work on every
// sieve in the 5×5 around the clicked one on its level, not beyond it.
func TestSieveActsOn5x5(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	at := func(dx, dz int) cube.Pos { return machinePos.Add(cube.Pos{dx, 0, dz}) }
	mid := placeTestMachine(s, at(0, 0), KindSieve, "")
	near := placeTestMachine(s, at(2, 2), KindSieve, "")
	far := placeTestMachine(s, at(3, 0), KindSieve, "")
	for _, st := range []*machineState{mid, near, far} {
		st.mesh = ""
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func() {
			(Machine{Kind: KindSieve, st: mid}).Activate(at(0, 0), cube.FaceUp, tx, p, nil)
		}
		p.SetHeldItems(item.NewStack(sgitem.Mesh{Tier: "string"}, 5), item.Stack{})
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

func TestSieveTopsUpTheHand(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	var group []*machineState
	for dx := range 3 {
		group = append(group, placeTestMachine(s, machinePos.Add(cube.Pos{dx, 0, 0}), KindSieve, ""))
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Gravel{}, 1), item.Stack{})
		_ = p.Inventory().SetItem(20, item.NewStack(block.Gravel{}, 10))
		(Machine{Kind: KindSieve, st: group[0]}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		for i, st := range group {
			if _, _, filled := st.firstInput(); !filled {
				t.Errorf("sieve %d got no gravel", i)
			}
		}
		held, _ := p.HeldItems()
		if naming.ItemKey(held.Item()) != naming.ItemKey(block.Gravel{}) || held.Count() != 8 {
			t.Errorf("held %d %s, want the 8 gravel left", held.Count(), naming.ItemKey(held.Item()))
		}
		if total := core.CountPlain(p.Inventory(), block.Gravel{}); total != 8 {
			t.Errorf("%d gravel in the inventory, want 8", total)
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
			if m, ok := st.Item().(sgitem.Mesh); ok && m.Tier == "iron" {
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

// TestSieveShowsItsMesh: each mesh tier has its own look, so a sieve shows the
// mesh it has fitted; the looks are numbered without gaps or repeats.
func TestSieveShowsItsMesh(t *testing.T) {
	seen := map[string]bool{lookValue(0, 0, 0): true}
	for _, l := range sieveLooks() {
		v := lookValue(l[0], l[1], l[2])
		if seen[v] {
			t.Fatalf("look %v repeats the state value %s", l, v)
		}
		seen[v] = true
	}
	if len(seen) != 1+len(sgitem.MeshTiers)*(1+(len(sieveContents)-1)*sieveStages) {
		t.Fatalf("%d looks", len(seen))
	}
	for i, tier := range sgitem.MeshTiers {
		st := newMachine(KindSieve, "", "").st
		st.mesh = tier.Tier
		if mesh, content, fill := sieveLook(st); mesh != i+1 || content != 0 || fill != 1 {
			t.Fatalf("%s mesh look %d/%d/%d", tier.Tier, mesh, content, fill)
		}
	}
	if mesh, _, _ := sieveLook(newMachine(KindSieve, "", "").st); mesh != 0 {
		t.Fatal("a sieve without a mesh shows one")
	}
	// the state values sort in the order of the looks
	vals := sieveStates()[sieveLookState]
	for i := 1; i < len(vals); i++ {
		if vals[i-1].(string) >= vals[i].(string) {
			t.Fatalf("state values %v and %v are not in order", vals[i-1], vals[i])
		}
	}
}

// TestSieveTakesCrushedBlocks: every crushed block, soul sand and red sand is
// sifted, shows on the mesh, and its pebbles make their stone.
func TestSieveTakesCrushedBlocks(t *testing.T) {
	for _, in := range codexSieveInputs() {
		if len(sievetable.Tables["string"][naming.ItemKey(in)]) == 0 {
			t.Errorf("%s is not sifted on the string mesh", naming.ItemKey(in))
		}
		if sieveContentOf(item.NewStack(in, 1)) == 0 {
			t.Errorf("%s does not show on the mesh", naming.ItemKey(in))
		}
	}
	for _, k := range sgitem.PebbleKinds {
		found := false
		for _, r := range recipe.Recipes() {
			if out := r.Output(); len(out) == 1 && naming.ItemKey(out[0].Item()) == naming.ItemKey(k.Block) && len(r.Input()) == 4 {
				found = true
			}
		}
		if !found {
			t.Errorf("no recipe makes %s from four %s pebbles", naming.ItemKey(k.Block), k.Kind)
		}
	}
}
