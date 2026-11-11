package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
)

func TestSieveTopsUpTheHand(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	var group []*machine.State
	for dx := range 3 {
		group = append(group, placeTestMachine(s, machinePos.Add(cube.Pos{dx, 0, 0}), machine.KindSieve))
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
