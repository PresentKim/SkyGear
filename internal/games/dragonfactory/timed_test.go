package dragonfactory

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/timed"
)

// TestStorageTellsItemsApartByNBT: drawers keep only identical items
// together (a time-limited item never merges with a plain one or another
// expiry); recipes, missions and the shop take plain items only.
func TestStorageTellsItemsApartByNBT(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	plainLog := item.NewStack(block.Log{Wood: block.OakWood()}, 32)
	log := timed.For(plainLog, time.Hour)
	named := plainLog.WithCustomName("특별한 원목")

	d := Drawer{}
	if d.insert(log, 32) != 32 {
		t.Fatal("an empty drawer refuses a timed item")
	}
	if d.accepts(plainLog) || d.accepts(timed.For(plainLog, 2*time.Hour)) {
		t.Error("a drawer of timed logs takes plain logs or another expiry")
	}
	if (Drawer{Item: plainLog.Grow(-31), Count: 1}).accepts(log) {
		t.Error("a drawer of plain logs takes timed logs")
	}
	sellBox := newMachine(KindSellBox, "", "").st
	if sellBox.insert(log) || sellBox.insert(named) {
		t.Error("a sell box takes an item with extra NBT")
	}
	if !sellBox.insert(plainLog) {
		t.Error("a sell box refuses a plain log")
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, _ = p.Inventory().AddItem(log)
		_, _ = p.Inventory().AddItem(named)
		if n := s.deliverInventory(p, tx); n != 0 {
			t.Errorf("delivered %d logs with extra NBT", n)
		}
		if n := s.sell(p, log); n != 0 {
			t.Errorf("sold %d timed logs", n)
		}
		if lines, _ := s.quoteSale(p); len(lines) != 0 {
			t.Errorf("sale quote lists items with extra NBT: %v", lines)
		}
	})
}

// TestExpiredDrawerEmpties: a drawer of time-limited items empties itself
// when they expire.
func TestExpiredDrawerEmpties(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	start := time.Now()
	timed.Now = func() time.Time { return start }
	t.Cleanup(func() { timed.Now = time.Now })
	pos := cube.Pos{3, 65, 3}
	s.world.Do(func(tx *world.Tx) {
		d := Drawer{Facing: cube.North, Locked: true}
		d.insert(timed.For(item.NewStack(item.Apple{}, 10), time.Minute), 10)
		tx.SetBlock(pos, d, nil)
	})
	tick(s, 2)
	timed.Now = func() time.Time { return start.Add(time.Minute) }
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		if d := tx.Block(pos).(Drawer); !d.Item.Empty() || d.Count != 0 {
			t.Fatalf("expired drawer holds %v x%d", d.Item, d.Count)
		}
	})
}

func TestExpiredItemsLeaveLostAndFound(t *testing.T) {
	s := testSession(t)
	start := time.Now()
	timed.Now = func() time.Time { return start }
	t.Cleanup(func() { timed.Now = time.Now })
	s.addLost(timed.For(item.NewStack(item.Apple{}, 1), time.Minute))
	s.addLost(timed.For(item.NewStack(item.Bread{}, 1), time.Hour))
	if len(s.state.Lost) != 2 {
		t.Fatalf("lost = %d", len(s.state.Lost))
	}
	timed.Now = func() time.Time { return start.Add(time.Minute) }
	s.addLost(timed.For(item.NewStack(item.Apple{}, 1), -time.Second))
	s.purgeLost()
	if len(s.state.Lost) != 1 {
		t.Fatalf("lost after expiry = %d, want only the bread", len(s.state.Lost))
	}
}
