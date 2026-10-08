package spike

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl64"
)

func init() { world.DefaultBlockRegistry.Finalize() }

func newWorld(t *testing.T, dir string) *world.World {
	t.Helper()
	conf := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}
	if dir != "" {
		db, err := mcdb.Config{}.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		conf.Provider = db
	}
	return conf.New()
}

func tick(w *world.World, n int) {
	for range n {
		w.AdvanceTick()
	}
}

// A custom block with a block entity is ticked every world tick while a
// loader (with no player) keeps its chunk loaded, and its NBT survives a
// save/close/reopen cycle of an mcdb world.
func TestMachineTicksWithoutPlayersAndPersists(t *testing.T) {
	dir := t.TempDir()
	pos := cube.Pos{100, 64, 100} // chunk (6, 6): away from the loader's own chunk

	w := newWorld(t, dir)
	var l *world.Loader
	w.Do(func(tx *world.Tx) {
		l = world.NewLoader(8, w, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 1000)
		tx.SetBlock(pos, Machine{}, nil)
	})
	w.SetTickRange(8)
	tick(w, 40)

	var ticks int64
	w.Do(func(tx *world.Tx) { ticks = tx.Block(pos).(Machine).Ticks })
	if ticks < 30 {
		t.Fatalf("machine ticked %d times in 40 world ticks, want ~40", ticks)
	}
	w.Do(func(tx *world.Tx) { l.Close(tx) })
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	w = newWorld(t, dir)
	defer w.Close()
	w.Do(func(tx *world.Tx) {
		m, ok := tx.Block(pos).(Machine)
		if !ok {
			t.Fatalf("block after reload = %T, want Machine", tx.Block(pos))
		}
		if m.Ticks != ticks {
			t.Fatalf("Ticks after reload = %d, want %d", m.Ticks, ticks)
		}
	})
}

// In a real (asynchronous) world, block entities are only ticked in chunks
// within the tick range of a loader's chunk; synchronous test worlds tick
// every loaded chunk instead. Owned chunks must therefore be covered by
// loader position + tick range for factories to run.
func TestTickRangeLimitsMachines(t *testing.T) {
	w := world.Config{Entities: entity.DefaultRegistry}.New()
	defer w.Close()
	near, far := cube.Pos{16 * 3, 64, 0}, cube.Pos{16 * 20, 64, 0}
	<-w.Do(func(tx *world.Tx) {
		l := world.NewLoader(4, w, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 100)
		tx.SetBlock(near, Machine{}, nil)
		tx.SetBlock(far, Machine{}, nil)
	}).Done()
	w.SetTickRange(4)
	time.Sleep(time.Second)
	<-w.Do(func(tx *world.Tx) {
		if m := tx.Block(near).(Machine); m.Ticks == 0 {
			t.Error("machine 3 chunks away was not ticked with tick range 4")
		}
		if m := tx.Block(far).(Machine); m.Ticks != 0 {
			t.Errorf("machine 20 chunks away ticked %d times with tick range 4", m.Ticks)
		}
	}).Done()
}

type despawnRecorder struct {
	world.NopHandler
	void []item.Stack
}

func (h *despawnRecorder) HandleEntityDespawn(tx *world.Tx, e world.Entity) {
	ent, ok := e.(*entity.Ent)
	if !ok {
		return
	}
	ib, ok := ent.Behaviour().(*entity.ItemBehaviour)
	if !ok {
		return
	}
	if e.Position().Y() < float64(tx.Range()[0]) {
		h.void = append(h.void, ib.Item())
	}
}

// Item entities that fall into the void are closed by Dragonfly, which fires
// HandleEntityDespawn with the entity still below the world range. Items that
// are picked up or despawn normally are above it, so the lost-and-found box
// can tell them apart.
func TestVoidItemsAreDetectable(t *testing.T) {
	w := newWorld(t, "")
	defer w.Close()
	h := &despawnRecorder{}
	w.Handle(h)
	w.Do(func(tx *world.Tx) {
		l := world.NewLoader(2, w, world.NopViewer{})
		l.Move(tx, mgl64.Vec3{})
		l.Load(tx, 100)
		tx.SetBlock(cube.Pos{0, 63, 0}, block.Stone{}, nil)
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{0.5, -60, 0.5}}, item.NewStack(item.Diamond{}, 3)))
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{0.5, 64, 0.5}}, item.NewStack(item.Apple{}, 1)))
	})
	tick(w, 200)
	if len(h.void) != 1 || h.void[0].Count() != 3 {
		t.Fatalf("void items = %v, want 3 diamonds only", h.void)
	}
}

// Item stacks (with enchantments, names and custom NBT values) round-trip
// through item.WriteNBT/ReadNBT, which is what per-instance inventories,
// the lost-and-found box and cheques will be stored with.
func TestItemStackRoundTrip(t *testing.T) {
	in := item.NewStack(item.Pickaxe{Tier: item.ToolTierDiamond}, 1).
		WithCustomName("Core Breaker").
		WithEnchantments(item.NewEnchantment(enchantment.Efficiency, 3)).
		WithValue("cheque_id", "c-123").
		Damage(10)
	out := item.ReadNBT(item.WriteNBT(in, true), nil)
	if !in.Equal(out) || out.Count() != 1 || out.Durability() != in.Durability() {
		t.Fatalf("round trip mismatch:\n in=%v\nout=%v", in, out)
	}
	if v, _ := out.Value("cheque_id"); v != "c-123" {
		t.Fatalf("custom value lost: %v", v)
	}
	m := item.ReadNBT(item.WriteNBT(item.NewStack(Machine{}, 2), true), nil)
	if _, ok := m.Item().(Machine); !ok || m.Count() != 2 {
		t.Fatalf("custom item round trip = %v", m)
	}
}

// A player entity handle can be removed from one world and added to another;
// its inventory travels with the handle, so per-instance inventories must be
// swapped explicitly on transfer.
func TestPlayerMovesBetweenWorlds(t *testing.T) {
	lobby, inst := newWorld(t, ""), newWorld(t, "")
	defer lobby.Close()
	defer inst.Close()

	var h *world.EntityHandle
	lobby.Do(func(tx *world.Tx) {
		h = world.EntitySpawnOpts{Position: mgl64.Vec3{0, 64, 0}}.New(player.Type, player.Config{Name: "spike"})
		p := tx.AddEntity(h).(*player.Player)
		_, _ = p.Inventory().AddItem(item.NewStack(item.Compass{}, 1))
	})
	lobby.Do(func(tx *world.Tx) {
		e, ok := h.Entity(tx)
		if !ok {
			t.Fatal("player not in lobby")
		}
		tx.RemoveEntity(e)
	})
	inst.Do(func(tx *world.Tx) {
		p := tx.AddEntityAt(h, mgl64.Vec3{8, 80, 8}).(*player.Player)
		if p.Tx().World() != inst {
			t.Fatal("player not in instance world")
		}
		if p.Position() != (mgl64.Vec3{8, 80, 8}) {
			t.Fatalf("position = %v", p.Position())
		}
		if !p.Inventory().ContainsItem(item.NewStack(item.Compass{}, 1)) {
			t.Fatal("inventory did not travel with the handle")
		}
	})
}
