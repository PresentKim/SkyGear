package skygear

import (
	"log/slog"
	"math/rand/v2"
	"path/filepath"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

// openPersisted opens a session over a world saved in dir; closing it (the
// returned func) writes the modified chunks like a server shutdown.
func openPersisted(t *testing.T, dir string) (*session, func()) {
	t.Helper()
	d, err := db.Open(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.TouchAccount("host", "", "host"); err != nil {
		t.Fatal(err)
	}
	// The second open finds the save of the first.
	save, err := d.CreateSave("host", 1, "test")
	if err != nil {
		saves, lerr := d.SavesOf("host")
		if lerr != nil || len(saves) == 0 {
			t.Fatal(err)
		}
		save = saves[0]
	}
	prov, err := mcdb.Config{Log: slog.Default()}.Open(filepath.Join(dir, "world"))
	if err != nil {
		t.Fatal(err)
	}
	w := world.Config{Synchronous: true, Provider: prov, Entities: entity.DefaultRegistry.Config().New(EntityTypes())}.New()
	g := &Game{log: slog.Default(), db: d}
	c := core.New(core.Config{
		World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute),
		SaveID: save.ID, SaveName: save.Name, HostID: "host", Log: slog.Default(),
	})
	c.Rand = rand.New(rand.NewPCG(1, 2))
	s := &session{Session: c, g: g}
	s.state = savedState.Of(c)
	g.sessions.Store(w, s)
	core.Register(c)
	activeGame = g
	registerItemIO()
	w.Handle(&worldHandler{s: s})
	loadAround(s)
	closed := false
	closeAll := func() {
		if closed {
			return
		}
		closed = true
		core.Unregister(c)
		c.Close()
		_ = w.Close()
		_ = d.Close()
	}
	t.Cleanup(closeAll)
	return s, closeAll
}

// TestSieveProgressSurvivesReload: a machine that only works (no block
// update) keeps its progress and output over a save and a reload.
func TestSieveProgressSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	pos := cube.Pos{3, island.IslandY, 3}

	s, closeAll := openPersisted(t, dir)
	m := machine.New(machine.KindAutoSieve)
	m.State.Unlimited = true
	_ = m.State.Inv().SetItem(m.State.Def.Input[0], item.NewStack(block.Gravel{}, 30))
	s.World.Do(func(tx *world.Tx) { tx.SetBlock(pos, m, nil) })
	closeAll()

	s, closeAll = openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		st := tx.Block(pos).(machine.Machine).State
		// The flag does not survive the NBT round trip (it reads back as a
		// byte), so set it again; this is not a block change.
		st.Unlimited = true
	})
	tick(s, 130)
	var progress, left, out int
	s.World.Do(func(tx *world.Tx) {
		st := tx.Block(pos).(machine.Machine).State
		progress = st.Progress
		left = st.Inv().Slots()[st.Def.Input[0]].Count()
		out = st.Inv().Slots()[st.Def.Output[0]].Count()
	})
	if left == 30 && progress == 0 {
		t.Fatal("the machine did not work")
	}
	closeAll()

	s, _ = openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		st := tx.Block(pos).(machine.Machine).State
		if st.Progress != progress {
			t.Errorf("progress after reload = %d, want %d", st.Progress, progress)
		}
		if got := st.Inv().Slots()[st.Def.Input[0]].Count(); got != left {
			t.Errorf("input after reload = %d, want %d", got, left)
		}
		if got := st.Inv().Slots()[st.Def.Output[0]].Count(); got != out {
			t.Errorf("output after reload = %d, want %d", got, out)
		}
	})
}

// TestFlowHopperInvSurvivesReload: the items a flow hopper moved are
// saved without any block update.
func TestFlowHopperInvSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	top, hop := cube.Pos{2, island.IslandY + 4, 2}, cube.Pos{2, island.IslandY + 3, 2}
	bottom := cube.Pos{2, island.IslandY + 2, 2}

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		tx.SetBlock(top, block.NewChest(), nil)
		_, _ = tx.Block(top).(block.Chest).Inventory(tx, top).AddItem(item.NewStack(item.Wheat{}, 8))
		tx.SetBlock(hop, hopper.NewFlowHopper("wood_hopper", cube.FaceDown), nil)
		tx.SetBlock(bottom, block.Stone{}, nil)
	})
	closeAll()

	s, closeAll = openPersisted(t, dir)
	tick(s, 100)
	var want int
	s.World.Do(func(tx *world.Tx) {
		want = tx.Block(hop).(hopper.FlowHopper).Inv.Slots()[0].Count()
	})
	if want == 0 {
		t.Fatal("the hopper took nothing")
	}
	closeAll()

	s, _ = openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		if got := tx.Block(hop).(hopper.FlowHopper).Inv.Slots()[0].Count(); got != want {
			t.Errorf("hopper items after reload = %d, want %d", got, want)
		}
	})
}
