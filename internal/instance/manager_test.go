package instance

import (
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/timed"
)

type testGame struct{ joins, firsts, closes int }

func (*testGame) ID() string   { return "test" }
func (*testGame) Name() string { return "Test" }
func (*testGame) Init(tx *world.Tx, _ *Instance) {
	tx.SetBlock(cube.Pos{0, 63, 0}, block.Stone{}, nil)
}
func (g *testGame) Start(*Instance) Session { return &testSession{g: g} }

type testSession struct{ g *testGame }

func (*testSession) Spawn() mgl64.Vec3 { return mgl64.Vec3{0.5, 64, 0.5} }
func (s *testSession) Join(p *player.Player, first bool) {
	s.g.joins++
	if first {
		s.g.firsts++
		_, _ = p.Inventory().AddItem(item.NewStack(item.Diamond{}, 1))
	}
}
func (*testSession) Leave(*player.Player)                  {}
func (*testSession) Handler(*player.Player) player.Handler { return nil }
func (*testSession) NewDay(*world.Tx, int)                 {}
func (*testSession) Autosave(*world.Tx)                    {}
func (s *testSession) Close(*world.Tx)                     { s.g.closes++ }

type fixture struct {
	m     *Manager
	d     *db.DB
	lobby *world.World
	game  *testGame
}

func setup(t *testing.T) fixture {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	lobby := world.Config{Entities: entity.DefaultRegistry}.New()
	t.Cleanup(func() { _ = lobby.Close() })
	m := NewManager(slog.Default(), d, Config{
		Dir: filepath.Join(dir, "saves"), DayLength: time.Hour, LoadRadius: 2, Entities: entity.DefaultRegistry,
	}, Lobby{World: lobby, Spawn: mgl64.Vec3{0, 100, 0}, Arrive: func(p *player.Player) {
		_, _ = p.Inventory().AddItem(item.NewStack(item.Compass{}, 1))
	}})
	g := &testGame{}
	m.Register(g)
	return fixture{m: m, d: d, lobby: lobby, game: g}
}

func (f fixture) spawn(t *testing.T, name string) *world.EntityHandle {
	t.Helper()
	id := uuid.New()
	if _, err := f.d.TouchAccount(id.String(), "", name); err != nil {
		t.Fatal(err)
	}
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: name, UUID: id, Position: mgl64.Vec3{0, 100, 0}})
	<-f.lobby.Do(func(tx *world.Tx) { f.m.Join(tx.AddEntity(h).(*player.Player)) }).Done()
	return h
}

// in runs f with p in whatever world it currently is, waiting for transfers.
func in(t *testing.T, h *world.EntityHandle, f func(p *player.Player)) {
	t.Helper()
	time.Sleep(100 * time.Millisecond)
	done := make(chan struct{})
	h.Do(func(_ *world.Tx, e world.Entity) {
		defer close(done)
		f(e.(*player.Player))
	})
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for player")
	}
}

func TestEnterLeaveRestoresPerSaveState(t *testing.T) {
	f := setup(t)
	host := f.spawn(t, "host")
	acc := host.UUID().String()
	save, err := f.d.CreateSave(acc, "test", 1, "s")
	if err != nil {
		t.Fatal(err)
	}

	in(t, host, func(p *player.Player) {
		if !p.Inventory().ContainsItem(item.NewStack(item.Compass{}, 1)) {
			t.Error("lobby item missing")
		}
		if err := f.m.Enter(p, save.ID); err != nil {
			t.Fatal(err)
		}
	})
	in(t, host, func(p *player.Player) {
		if p.Tx().World() == f.lobby {
			t.Fatal("player still in lobby")
		}
		if p.Inventory().ContainsItem(item.NewStack(item.Compass{}, 1)) {
			t.Error("lobby item leaked into the instance")
		}
		if !p.Inventory().ContainsItem(item.NewStack(item.Diamond{}, 1)) {
			t.Error("first-join item missing")
		}
		_, _ = p.Inventory().AddItem(item.NewStack(item.Apple{}, 5))
		p.Teleport(mgl64.Vec3{3.5, 64, 3.5})
		f.m.ToLobby(p)
	})
	in(t, host, func(p *player.Player) {
		if p.Tx().World() != f.lobby {
			t.Fatal("player not back in lobby")
		}
		if p.Inventory().ContainsItem(item.NewStack(item.Apple{}, 5)) {
			t.Error("instance item leaked into the lobby")
		}
	})
	waitInactive(t, f.m, save.ID)
	if f.game.closes != 1 {
		t.Fatalf("session closed %d times, want 1", f.game.closes)
	}

	in(t, host, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); err != nil {
			t.Fatal(err)
		}
	})
	in(t, host, func(p *player.Player) {
		if !p.Inventory().ContainsItem(item.NewStack(item.Apple{}, 5)) {
			t.Error("instance inventory not restored")
		}
		// Only x/z: there is no block under (3, 63, 3), so the player starts falling.
		if pos := p.Position(); pos.X() != 3.5 || pos.Z() != 3.5 {
			t.Errorf("position not restored: %v", p.Position())
		}
		if _, ok := p.Tx().Block(cube.Pos{0, 63, 0}).(block.Stone); !ok {
			t.Error("world built by Init was not saved")
		}
	})
	if f.game.firsts != 1 || f.game.joins != 2 {
		t.Fatalf("joins=%d firsts=%d, want 2 and 1", f.game.joins, f.game.firsts)
	}
	f.m.CloseAll()
	if f.m.Active(save.ID) != nil {
		t.Fatal("instance still active after CloseAll")
	}
}

func TestGuestNeedsHost(t *testing.T) {
	f := setup(t)
	host, guest := f.spawn(t, "host"), f.spawn(t, "guest")
	save, _ := f.d.CreateSave(host.UUID().String(), "test", 1, "s")

	in(t, guest, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); !errors.Is(err, ErrNotMember) {
			t.Fatalf("non-member enter err = %v", err)
		}
	})
	if err := addGuest(f.d, save.ID, guest.UUID().String()); err != nil {
		t.Fatal(err)
	}
	in(t, guest, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); !errors.Is(err, ErrHostOffline) {
			t.Fatalf("guest without host err = %v", err)
		}
	})
	in(t, host, func(p *player.Player) { _ = f.m.Enter(p, save.ID) })
	in(t, guest, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); err != nil {
			t.Fatalf("guest with host err = %v", err)
		}
	})
	in(t, host, func(p *player.Player) { f.m.ToLobby(p) })
	waitInactive(t, f.m, save.ID)
	in(t, guest, func(p *player.Player) {
		if p.Tx().World() != f.lobby {
			t.Fatal("guest not returned to lobby when the host left")
		}
	})
	if _, blob, err := f.d.PlayerState(save.ID, guest.UUID().String()); err != nil || blob == nil {
		t.Fatalf("guest state not stored: %v", err)
	}
}

func waitInactive(t *testing.T, m *Manager, saveID int64) {
	t.Helper()
	for range 100 {
		if m.Active(saveID) == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("instance did not deactivate")
}

// The clock time of a short session must be stored on deactivation.
func TestClockStoredOnDeactivate(t *testing.T) {
	f := setup(t)
	host := f.spawn(t, "host")
	save, _ := f.d.CreateSave(host.UUID().String(), "test", 1, "s")
	in(t, host, func(p *player.Player) { _ = f.m.Enter(p, save.ID) })
	time.Sleep(1500 * time.Millisecond)
	in(t, host, func(p *player.Player) { f.m.ToLobby(p) })
	waitInactive(t, f.m, save.ID)
	s, _ := f.d.Save(save.ID)
	if s.InstanceTime < 1500*time.Millisecond {
		t.Fatalf("stored instance time %v, want >= 1.5s", s.InstanceTime)
	}
}

func TestLockStateBlocksEnter(t *testing.T) {
	f := setup(t)
	host := f.spawn(t, "host")
	acc := host.UUID().String()
	save, _ := f.d.CreateSave(acc, "test", 1, "s")
	unlock, ok := f.m.LockState(save.ID, acc)
	if !ok {
		t.Fatal("lock refused for a player in the lobby")
	}
	if _, again := f.m.LockState(save.ID, acc); again {
		t.Fatal("a state was locked twice")
	}
	in(t, host, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); !errors.Is(err, ErrStateLocked) {
			t.Fatalf("enter while locked err = %v", err)
		}
	})
	unlock()
	in(t, host, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); err != nil {
			t.Fatal(err)
		}
	})
	in(t, host, func(p *player.Player) {
		if _, ok := f.m.LockState(save.ID, acc); ok {
			t.Fatal("locked the stored state of a player inside the save")
		}
	})
	f.m.CloseAll()
}

func TestKeepLoadedOutsideRadius(t *testing.T) {
	f := setup(t)
	host := f.spawn(t, "host")
	save, _ := f.d.CreateSave(host.UUID().String(), "test", 1, "s")
	in(t, host, func(p *player.Player) { _ = f.m.Enter(p, save.ID) })
	inst := f.m.Active(save.ID)
	if inst == nil {
		t.Fatal("not active")
	}
	<-inst.World.Do(func(tx *world.Tx) {
		inst.KeepLoaded(tx, 1, 1) // inside the radius: nothing to do
		inst.KeepLoaded(tx, 6, -3)
		inst.KeepLoaded(tx, 6, -3)
		inst.KeepLoaded(tx, 2, 2) // in the radius-2 square, outside its circle
	}).Done()
	if len(inst.extra) != 2 {
		t.Fatalf("extra loaders = %d, want 2", len(inst.extra))
	}
	in(t, host, func(p *player.Player) { f.m.ToLobby(p) })
	waitInactive(t, f.m, save.ID)
}

// TestTimedItemsExpireInInstance: the instance removes a time-limited item
// from a player within a second of its real-time expiry.
func TestTimedItemsExpireInInstance(t *testing.T) {
	f := setup(t)
	host := f.spawn(t, "host")
	save, err := f.d.CreateSave(host.UUID().String(), "test", 1, "s")
	if err != nil {
		t.Fatal(err)
	}
	in(t, host, func(p *player.Player) {
		if err := f.m.Enter(p, save.ID); err != nil {
			t.Fatal(err)
		}
	})
	in(t, host, func(p *player.Player) {
		_, _ = p.Inventory().AddItem(timed.For(item.NewStack(item.Apple{}, 3), time.Second))
		_, _ = p.Inventory().AddItem(timed.For(item.NewStack(item.Bread{}, 1), time.Hour))
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		gone := false
		in(t, host, func(p *player.Player) {
			gone = !p.Inventory().ContainsItemFunc(1, func(st item.Stack) bool { _, ok := st.Item().(item.Apple); return ok })
			if gone && !p.Inventory().ContainsItemFunc(1, func(st item.Stack) bool { _, ok := st.Item().(item.Bread); return ok }) {
				t.Fatal("the unexpired item was removed too")
			}
		})
		if gone {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("expired item still carried")
		}
		time.Sleep(200 * time.Millisecond)
	}
}
