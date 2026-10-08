package dragonfactory

import (
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
		if m := tx.Block(machinePos).(Machine); m.content != 0 || m.fill != 0 {
			t.Fatalf("empty sieve shows %d/%d", m.content, m.fill)
		}
		if _, _, ok := other.firstInput(); ok {
			t.Fatal("the sieve next to it did not sift along")
		}
		if held, _ := p.HeldItems(); held.Count() != 4 {
			t.Fatalf("sifting used held items: %v", held)
		}
	})
}
