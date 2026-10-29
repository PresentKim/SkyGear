package skygear

import (
	"github.com/presentkim/skygear/content/skygear/codex"
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/progress"

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
	st := placeTestMachine(s, machinePos, machine.KindSieve, "")
	other := placeTestMachine(s, machinePos.Add(cube.Pos{1, 0, 0}), machine.KindSieve, "")
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func() {
			st.LastUse, other.LastUse = time.Time{}, time.Time{}
			if !(machine.Machine{Kind: machine.KindSieve, State: st}).Activate(machinePos, cube.FaceUp, tx, p, nil) {
				t.Fatal("not handled")
			}
		}
		_ = other.Inv().SetItem(other.Def.Input[0], item.NewStack(block.Sand{}, 1))
		p.SetHeldItems(item.NewStack(block.Gravel{}, 5), item.Stack{})
		use()
		if held, _ := p.HeldItems(); held.Count() != 4 {
			t.Fatalf("held after inserting = %v", held)
		}
		if m := tx.Block(machinePos).(machine.Machine); m.Content != 1 || m.Fill != machine.SieveStages {
			t.Fatalf("shown content %d fill %d", m.Content, m.Fill)
		}
		// Holding gravel again sifts instead of inserting more.
		for range machine.SieveStages - 1 {
			use()
		}
		if m := tx.Block(machinePos).(machine.Machine); m.Fill != 1 {
			t.Fatalf("fill after %d stages = %d", machine.SieveStages-1, m.Fill)
		}
		use()
		if _, _, ok := st.FirstInput(); ok {
			t.Fatal("gravel not used up")
		}
		if m := tx.Block(machinePos).(machine.Machine); m.Content != 0 || m.Fill != 1 {
			t.Fatalf("empty sieve with a mesh shows %d/%d, want 0/1", m.Content, m.Fill)
		}
		if _, _, ok := other.FirstInput(); ok {
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
	m := machine.New(machine.KindSieve, "", "")
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(machinePos, m, nil) })
	st := m.State
	if st.Accepts(item.NewStack(block.Gravel{}, 1)) || st.Insert(item.NewStack(block.Gravel{}, 1)) {
		t.Fatal("a sieve without a mesh takes gravel")
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Gravel{}, 3), item.Stack{})
		if !(machine.Machine{Kind: machine.KindSieve, State: st}).Activate(machinePos, cube.FaceUp, tx, p, nil) {
			t.Fatal("using the sieve was not handled")
		}
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Fatalf("gravel went into a sieve without a mesh: %v", held)
		}
		p.SetHeldItems(item.NewStack(sgitem.Mesh{Tier: "string"}, 2), item.Stack{})
		(machine.Machine{Kind: machine.KindSieve, State: st}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		if st.Mesh != "string" {
			t.Fatal("the mesh was not fitted")
		}
		if held, _ := p.HeldItems(); held.Count() != 1 {
			t.Errorf("held meshes after fitting one: %v", held)
		}
		if !st.Accepts(item.NewStack(block.Gravel{}, 1)) {
			t.Error("a sieve with a mesh refuses gravel")
		}
	})
	if drops := (machine.Machine{Kind: machine.KindSieve, State: st}).BreakInfo().Drops(nil, nil); len(drops) != 2 {
		t.Errorf("%d drops, want the sieve and its mesh", len(drops))
	}
	back := machine.Machine{Kind: machine.KindSieve}.DecodeNBT(machine.Machine{Kind: machine.KindSieve, State: st}.EncodeNBT()).(machine.Machine)
	if back.State.Mesh != "string" {
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
			if out := r.Output(); len(out) == 1 && progress.RecipeKey(out[0].Item()) == progress.RecipeKey(k.Block) && len(r.Input()) == 4 {
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
	mid := placeTestMachine(s, at(0, 0), machine.KindSieve, "")
	near := placeTestMachine(s, at(2, 2), machine.KindSieve, "")
	far := placeTestMachine(s, at(3, 0), machine.KindSieve, "")
	for _, st := range []*machine.State{mid, near, far} {
		st.Mesh = ""
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func() {
			(machine.Machine{Kind: machine.KindSieve, State: mid}).Activate(at(0, 0), cube.FaceUp, tx, p, nil)
		}
		p.SetHeldItems(item.NewStack(sgitem.Mesh{Tier: "string"}, 5), item.Stack{})
		use()
		if mid.Mesh != "string" || near.Mesh != "" || far.Mesh != "" {
			t.Fatalf("meshes %q %q %q: only the clicked sieve takes one", mid.Mesh, near.Mesh, far.Mesh)
		}
		near.Mesh, far.Mesh = "string", ""
		p.SetHeldItems(item.Stack{}, item.Stack{})
		p.SetHeldItems(item.NewStack(block.Gravel{}, 5), item.Stack{})
		use()
		_, _, a := mid.FirstInput()
		_, _, b := near.FirstInput()
		_, _, c := far.FirstInput()
		if !a || !b || c {
			t.Fatalf("gravel went in: %v %v %v", a, b, c)
		}
		if held, _ := p.HeldItems(); held.Count() != 3 {
			t.Errorf("held gravel %d, want 3", held.Count())
		}
		if mid.Mesh == "" || near.Mesh == "" {
			t.Error("putting in blocks took a mesh out")
		}
		use()
		if mid.Progress == 0 || near.Progress == 0 || far.Progress != 0 {
			t.Errorf("sifting progress %d %d %d", mid.Progress, near.Progress, far.Progress)
		}
	})
}

func TestSieveTopsUpTheHand(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	var group []*machine.State
	for dx := range 3 {
		group = append(group, placeTestMachine(s, machinePos.Add(cube.Pos{dx, 0, 0}), machine.KindSieve, ""))
	}
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Gravel{}, 1), item.Stack{})
		_ = p.Inventory().SetItem(20, item.NewStack(block.Gravel{}, 10))
		(machine.Machine{Kind: machine.KindSieve, State: group[0]}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		for i, st := range group {
			if _, _, filled := st.FirstInput(); !filled {
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
	a := placeTestMachine(s, machinePos, machine.KindSieve, "")
	b := placeTestMachine(s, machinePos.Add(cube.Pos{1, 0, 0}), machine.KindSieve, "")
	a.Mesh, b.Mesh = "iron", "iron"
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.Stack{}, item.Stack{})
		(machine.Machine{Kind: machine.KindSieve, State: a}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		if a.Mesh == "" {
			t.Fatal("an empty hand without sneaking took the mesh out")
		}
		p.StartSneaking()
		(machine.Machine{Kind: machine.KindSieve, State: a}).Activate(machinePos, cube.FaceUp, tx, p, nil)
		if a.Mesh != "" || b.Mesh != "iron" {
			t.Fatalf("meshes after taking out: %q %q", a.Mesh, b.Mesh)
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
		b.Insert(item.NewStack(block.Gravel{}, 1))
		p.StartSneaking()
		(machine.Machine{Kind: machine.KindSieve, State: b}).Activate(machinePos.Add(cube.Pos{1, 0, 0}), cube.FaceUp, tx, p, nil)
		if b.Mesh != "iron" {
			t.Error("a filled sieve gave its mesh up")
		}
	})
}

// TestSieveShowsItsMesh: each mesh tier has its own look, so a sieve shows the
// mesh it has fitted; the looks are numbered without gaps or repeats.
func TestSieveShowsItsMesh(t *testing.T) {
	seen := map[string]bool{machine.LookValue(0, 0, 0): true}
	for _, l := range machine.SieveLooks() {
		v := machine.LookValue(l[0], l[1], l[2])
		if seen[v] {
			t.Fatalf("look %v repeats the state value %s", l, v)
		}
		seen[v] = true
	}
	if len(seen) != 1+len(sgitem.MeshTiers)*(1+(len(machine.SieveContents)-1)*machine.SieveStages) {
		t.Fatalf("%d looks", len(seen))
	}
	for i, tier := range sgitem.MeshTiers {
		st := machine.New(machine.KindSieve, "", "").State
		st.Mesh = tier.Tier
		if mesh, content, fill := machine.SieveLook(st); mesh != i+1 || content != 0 || fill != 1 {
			t.Fatalf("%s mesh look %d/%d/%d", tier.Tier, mesh, content, fill)
		}
	}
	if mesh, _, _ := machine.SieveLook(machine.New(machine.KindSieve, "", "").State); mesh != 0 {
		t.Fatal("a sieve without a mesh shows one")
	}
	// the state values sort in the order of the looks
	vals := machine.SieveStates()[machine.SieveLookState]
	for i := 1; i < len(vals); i++ {
		if vals[i-1].(string) >= vals[i].(string) {
			t.Fatalf("state values %v and %v are not in order", vals[i-1], vals[i])
		}
	}
}

// TestSieveTakesCrushedBlocks: every crushed block, soul sand and red sand is
// sifted, shows on the mesh, and its pebbles make their stone.
func TestSieveTakesCrushedBlocks(t *testing.T) {
	for _, in := range codex.SieveInputs() {
		if len(sievetable.Tables["string"][naming.ItemKey(in)]) == 0 {
			t.Errorf("%s is not sifted on the string mesh", naming.ItemKey(in))
		}
		if machine.SieveContentOf(item.NewStack(in, 1)) == 0 {
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
