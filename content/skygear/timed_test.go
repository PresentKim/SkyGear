package skygear

import (
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/timed"
)

// TestStorageTellsItemsApartByNBT: drawers keep only identical items
// together (a time-limited item never merges with a plain one or another
// expiry); machines and research take plain items only.
func TestStorageTellsItemsApartByNBT(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	plainLog := item.NewStack(block.Log{Wood: block.OakWood()}, 32)
	log := timed.For(plainLog, time.Hour)

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
	crusher := machine.NewMachine(machine.KindCrusher, "", "").State
	cobble := item.NewStack(block.Cobblestone{}, 1)
	if crusher.Insert(timed.For(cobble, time.Hour)) || !crusher.Insert(cobble) {
		t.Error("a crusher takes an item with extra NBT or refuses a plain one")
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_, _ = p.Inventory().AddItem(item.NewStack(oakPlanks, 16).WithLore("x"))
		sticks, _ := progress.ByID("stick")
		if _, done := progress.Do(s.Session, p, tx, sticks); done {
			t.Error("a research was done with planks that have extra NBT")
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
	s.World.Do(func(tx *world.Tx) {
		d := Drawer{Facing: cube.North, Locked: true}
		d.insert(timed.For(item.NewStack(item.Apple{}, 10), time.Minute), 10)
		tx.SetBlock(pos, d, nil)
	})
	tick(s, 2)
	timed.Now = func() time.Time { return start.Add(time.Minute) }
	tick(s, 2)
	s.World.Do(func(tx *world.Tx) {
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
	island.AddLost(s.Session, timed.For(item.NewStack(item.Apple{}, 1), time.Minute))
	island.AddLost(s.Session, timed.For(item.NewStack(item.Bread{}, 1), time.Hour))
	if len(island.StateOf(s.Session).Lost) != 2 {
		t.Fatalf("lost = %d", len(island.StateOf(s.Session).Lost))
	}
	timed.Now = func() time.Time { return start.Add(time.Minute) }
	island.AddLost(s.Session, timed.For(item.NewStack(item.Apple{}, 1), -time.Second))
	island.PurgeLost(s.Session)
	if len(island.StateOf(s.Session).Lost) != 1 {
		t.Fatalf("lost after expiry = %d, want only the bread", len(island.StateOf(s.Session).Lost))
	}
}
