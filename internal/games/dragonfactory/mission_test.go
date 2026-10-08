package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestMissionDeliveryAndLevelUp(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(s, h, func(p *player.Player, tx *world.Tx) {
		s.money[p.UUID().String()] = 0
		_, _ = p.Inventory().AddItem(item.NewStack(block.Cobblestone{}, 50))
		_, _ = p.Inventory().AddItem(item.NewStack(block.Dirt{}, 10))
		if n := s.deliverInventory(p, tx); n != 50 {
			t.Fatalf("delivered %d, want 50 cobblestone only", n)
		}
		if s.state.Level != 1 || s.needed(block.Cobblestone{}) != 14 {
			t.Fatalf("level %d needed %d", s.state.Level, s.needed(block.Cobblestone{}))
		}
		_, _ = p.Inventory().AddItem(item.NewStack(block.Cobblestone{}, 64))
		s.deliverInventory(p, tx)
		if s.state.Level != 2 {
			t.Fatalf("level = %d after completing mission 1", s.state.Level)
		}
		if p.Inventory().ContainsItem(item.NewStack(block.Cobblestone{}, 51)) || !p.Inventory().ContainsItem(item.NewStack(block.Cobblestone{}, 50)) {
			t.Fatal("only the needed 14 cobblestone must be taken")
		}
		if s.Money(p) != Levels[0].Reward.Money || s.state.CorePoints != Levels[0].Reward.CorePoints {
			t.Fatalf("reward: money %d points %d", s.Money(p), s.state.CorePoints)
		}
		if !p.Inventory().ContainsItem(item.NewStack(block.Dirt{}, 10)) {
			t.Fatal("dirt was taken before mission 2 asked for it")
		}
	})
}

func TestDeliveryBoxFromHopper(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	box, hopper := cube.Pos{2, 64, 2}, cube.Pos{2, 65, 2}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(box, DeliveryBox{}, nil)
		tx.SetBlock(hopper, block.NewHopper(), nil)
		inv := tx.Block(hopper).(block.Hopper).Inventory(tx, hopper)
		_, _ = inv.AddItem(item.NewStack(block.Cobblestone{}, 5))
		_, _ = inv.AddItem(item.NewStack(item.Apple{}, 2))
	})
	tick(s, 20*10)
	s.world.Do(func(tx *world.Tx) {
		if got := s.state.Delivered[itemKey(block.Cobblestone{})]; got != 5 {
			t.Fatalf("hopper delivered %d cobblestone, want 5", got)
		}
		inv := tx.Block(hopper).(block.Hopper).Inventory(tx, hopper)
		if !inv.ContainsItem(item.NewStack(item.Apple{}, 2)) {
			t.Fatal("items the mission does not need must stay in the hopper")
		}
	})
}

func TestLevelsAreValid(t *testing.T) {
	for i, l := range Levels {
		if len(l.Requirements) == 0 {
			t.Errorf("level %d has no requirements", i+1)
		}
		for _, r := range l.Requirements {
			if r.Count <= 0 {
				t.Errorf("level %d requirement %s count %d", i+1, itemKey(r.Item), r.Count)
			}
		}
	}
}
