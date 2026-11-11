package skygear

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	dfsession "github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// nopConn is the connection of the session that replaces session.Nop for the
// tests (pinNop).
type nopConn struct{}

func (nopConn) Close() error                                               { return nil }
func (nopConn) IdentityData() login.IdentityData                           { return login.IdentityData{DisplayName: "nop"} }
func (nopConn) ClientData() login.ClientData                               { return login.ClientData{} }
func (nopConn) ClientCacheEnabled() bool                                   { return false }
func (nopConn) ChunkRadius() int                                           { return 0 }
func (nopConn) Latency() time.Duration                                     { return 0 }
func (nopConn) Flush() error                                               { return nil }
func (nopConn) RemoteAddr() net.Addr                                       { return &net.UDPAddr{} }
func (nopConn) ReadPacket() (packet.Packet, error)                         { return nil, io.EOF }
func (nopConn) WritePacket(packet.Packet) error                            { return nil }
func (nopConn) StartGameContext(context.Context, minecraft.GameData) error { return nil }

// pinNop: the session a player without a connection uses (dfsession.Nop) has
// no scoreboard state, so RemoveScoreboard, which Leave calls, panics on it.
// Before any test runs, Nop becomes a session built from a connection that
// discards everything; writes to it are still skipped (a packet is written
// only if the session is not Nop).
func init() { dfsession.Nop = dfsession.Config{Log: slog.New(slog.DiscardHandler)}.New(nopConn{}) }

// pinStore is the state store of a pin session: it keeps every document the
// session stores, so a scenario can say what was saved.
type pinStore struct {
	mu   sync.Mutex
	docs [][]byte
}

func (k *pinStore) put(b []byte) error {
	k.mu.Lock()
	k.docs = append(k.docs, append([]byte(nil), b...))
	k.mu.Unlock()
	return nil
}

func (k *pinStore) count() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.docs)
}

func (k *pinStore) last() []byte {
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(k.docs) == 0 {
		return nil
	}
	return k.docs[len(k.docs)-1]
}

// await waits until n documents are stored (the writer of Flush is a
// goroutine; Close stores before it returns).
func (k *pinStore) await(t *testing.T, n int) {
	t.Helper()
	for range 5000 {
		if k.count() >= n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("only %d of %d documents stored", k.count(), n)
}

// reload is a session made from the last stored document.
func (k *pinStore) reload() *core.Session {
	return core.New(core.Config{Log: slog.Default(), State: k.last()})
}

// pinLifeSession is testSession with a counting state store.
func pinLifeSession(t *testing.T) (*session, *pinStore) {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := d.TouchAccount("host", "", "host"); err != nil {
		t.Fatal(err)
	}
	save, err := d.CreateSave("host", 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(EntityTypes())}.New()
	t.Cleanup(func() { _ = w.Close() })
	g := &Game{log: slog.Default(), db: d}
	st := &pinStore{}
	c := core.New(core.Config{
		World: w, Clock: instance.NewClock(0, 1, 0, 20*time.Minute),
		SaveID: save.ID, SaveName: save.Name, HostID: "host", Log: slog.Default(),
		Store: st.put,
	})
	c.Rand = rand.New(rand.NewPCG(1, 2))
	s := &session{Lifecycle: core.Lifecycle{Session: c}, g: g}
	s.state = savedState.Of(c)
	t.Cleanup(c.Close)
	g.sessions.Store(w, s)
	core.Register(c)
	t.Cleanup(func() { core.Unregister(c) })
	activeGame = g
	registerItemIO()
	w.Handle(&core.WorldHandler{S: s.Session})
	return s, st
}

// TestPinLifecycle records, per scenario, the marks the session paths reach
// (Join, Leave, NewDay, Second, Fast, Autosave, Close and the world handler's
// ColumnLoad, ColumnUnload and EntityDespawn) and the outcome that shows what
// each step did, into testdata/pin/lifecycle.golden. Scenarios that are one
// series (Second/01.., Fast/01..) run on one session in order, so the golden
// shows on which calls a step runs. The golden is recorded from the legacy
// code; it may be re-recorded (SKYGEAR_GOLDEN=update) only when the legacy
// behaviour itself is being changed on purpose.
func TestPinLifecycle(t *testing.T) {
	lines := map[string]string{}
	pinWorld.Store(true)
	defer pinWorld.Store(false)

	// retire removes the players from the world before it closes: a
	// delayed instance.PlaceAt must not meet the closing world.
	retire := func(s *session, hs ...*world.EntityHandle) {
		t.Cleanup(func() {
			s.World.Do(func(tx *world.Tx) {
				for _, h := range hs {
					if e, ok := h.Entity(tx); ok {
						tx.RemoveEntity(e)
					}
				}
			})
		})
	}
	// fresh makes a session over a loaded world with n players.
	fresh := func(n int) (*session, *pinStore, []*world.EntityHandle) {
		s, st := pinLifeSession(t)
		loadAround(s)
		var hs []*world.EntityHandle
		for range n {
			hs = append(hs, spawnPlayer(s))
		}
		retire(s, hs...)
		return s, st, hs
	}
	// in runs f in one transaction of the session (no player needed).
	in := func(s *session, f func(tx *world.Tx)) {
		s.World.Do(f)
	}
	players := func(tx *world.Tx) int {
		n := 0
		for range tx.Players() {
			n++
		}
		return n
	}
	slot8 := func(p *player.Player) string {
		it, _ := p.Inventory().Item(8)
		if it.Empty() {
			return "none"
		}
		name, _ := it.Item().EncodeItem()
		return name
	}
	mode := func(p *player.Player) string { return fmt.Sprintf("%T", p.GameMode()) }
	// lost adds n stacks to the lost-and-found box; the first one is older
	// than a day.
	lost := func(s *session, n int) {
		if n > 0 {
			island.AddLost(s.Session, item.NewStack(block.Dirt{}, 1))
		}
		if n > 1 {
			island.AddLost(s.Session, item.NewStack(block.Stone{}, 2))
		}
		for _, l := range island.StateOf(s.Session).Lost {
			if l.Count == 1 {
				l.LastDropped = -2 * s.Clock.DayLength()
			}
		}
	}
	lostN := func(s *session) int { return len(island.StateOf(s.Session).Lost) }
	// lostItems counts the items in the box (a drop merges into its entry).
	lostItems := func(s *session) int {
		n := 0
		for _, l := range island.StateOf(s.Session).Lost {
			n += l.Count
		}
		return n
	}

	// --- Join ---
	for _, c := range []struct {
		name  string
		first bool
		extra int // players already present
	}{{"Join/first", true, 0}, {"Join/again", false, 0}, {"Join/withOther", false, 1}} {
		s, _, hs := fresh(1 + c.extra)
		withPlayer(t, s, hs[c.extra], func(p *player.Player, tx *world.Tx) {
			pinRun(t, lines, c.name, func() string {
				return fmt.Sprintf("slot8=%s players=%d", slot8(p), players(tx))
			}, func() { s.Join(p, c.first) })
		})
	}
	{
		s, _, hs := fresh(1)
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			p.SetGameMode(world.GameModeSpectator)
			pinRun(t, lines, "Join/spectator", func() string { return "mode=" + mode(p) }, func() { s.Join(p, false) })
		})
	}
	{
		s, _, hs := fresh(1)
		lost(s, 2)
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			pinRun(t, lines, "Join/purge", func() string { return fmt.Sprintf("lost=%d", lostN(s)) }, func() { s.Join(p, false) })
		})
	}

	// --- Leave ---
	{
		s, _, hs := fresh(1)
		for h, f := range map[*world.EntityHandle]int{hs[0]: 3} {
			keptFood.Of(s.Session)[h] = f
		}
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			pinRun(t, lines, "Leave/plain", func() string {
				return fmt.Sprintf("diedFood=%d players=%d", len(keptFood.Of(s.Session)), players(tx))
			}, func() { s.Leave(p) })
		})
	}
	{
		s, _, hs := fresh(2)
		for h, f := range map[*world.EntityHandle]int{hs[0]: 3, hs[1]: 5} {
			keptFood.Of(s.Session)[h] = f
		}
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			pinRun(t, lines, "Leave/withOther", func() string {
				return fmt.Sprintf("diedFood=%d players=%d", len(keptFood.Of(s.Session)), players(tx))
			}, func() { s.Leave(p) })
		})
	}
	{
		s, _, hs := fresh(1)
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
				t.Fatal("the hunt did not begin")
			}
			pinRun(t, lines, "Leave/hunting", func() string {
				return fmt.Sprintf("hunting=%t run=%t inArena=%t", dimension.Hunting(s.Session, p), dimension.RunOf(s.Session) != nil, core.InArena(p.Position()))
			}, func() { s.Leave(p) })
		})
	}
	{
		s, _, hs := fresh(1)
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			expand.Show(s.Session, p)
			pinRun(t, lines, "Leave/expanding", func() string {
				_, ok := expand.Of(s.Session, p)
				return fmt.Sprintf("expanding=%t mode=%s", ok, mode(p))
			}, func() { s.Leave(p) })
		})
	}

	// --- NewDay ---
	for _, c := range []struct {
		name string
		n    int
		lost int
	}{{"NewDay/noPlayers", 0, 2}, {"NewDay/onePlayer", 1, 2}, {"NewDay/twoPlayers", 2, 0}} {
		s, _, _ := fresh(c.n)
		lost(s, c.lost)
		in(s, func(tx *world.Tx) {
			pinRun(t, lines, c.name, func() string {
				return fmt.Sprintf("lost=%d players=%d", lostN(s), players(tx))
			}, func() { s.NewDay(tx, 2) })
		})
	}

	// --- Second ---
	// Four calls on one session: the state is stored on the first (never
	// stored yet), not on the second (stored a moment ago, the interval is an
	// hour), on the third (the interval is one nanosecond) and not on the
	// fourth (nothing changed).
	{
		s, st, _ := fresh(1)
		dirty := func() string { return fmt.Sprintf("dirty=%t", s.Dirty()) }
		stored := 0
		call := func(name string, interval time.Duration, mark bool) {
			s.FlushInterval = interval
			if mark {
				s.MarkDirty()
			}
			in(s, func(tx *world.Tx) {
				before := dirty()
				pinRun(t, lines, name, func() string { return before + ">" + dirty() }, func() { s.Second(tx) })
			})
			// A flush hands the document to the writer: wait for it.
			if !s.Dirty() && mark {
				stored++
				st.await(t, stored)
			}
		}
		call("Second/01", time.Hour, true)
		call("Second/02", time.Hour, true)
		call("Second/03", time.Nanosecond, true)
		call("Second/04", time.Nanosecond, false)
	}
	{
		s, _, _ := fresh(0)
		in(s, func(tx *world.Tx) {
			pinRun(t, lines, "Second/noPlayers", func() string { return fmt.Sprintf("players=%d", players(tx)) }, func() { s.Second(tx) })
		})
	}
	{
		s, _, _ := fresh(2)
		in(s, func(tx *world.Tx) {
			pinRun(t, lines, "Second/twoPlayers", func() string { return fmt.Sprintf("players=%d", players(tx)) }, func() { s.Second(tx) })
		})
	}
	// A held log completes an advancement in this very Second, and the
	// change is stored by the FlushIfDue at its end.
	{
		s, st, hs := fresh(1)
		s.FlushInterval = time.Nanosecond
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			_, _ = p.Inventory().AddItem(item.NewStack(block.Log{Wood: block.OakWood()}, 1))
			pinRun(t, lines, "Second/advancement", func() string {
				return fmt.Sprintf("advanced=%t dirty=%t", progress.Advanced(s.Session, "wood"), s.Dirty())
			}, func() { s.Second(tx) })
		})
		if s.Dirty() || !progress.Advanced(s.Session, "wood") {
			lines["Second/advancement"] += " stored=none"
		} else {
			st.await(t, 1)
			lines["Second/advancement"] += fmt.Sprintf(" stored=%t", progress.Advanced(st.reload(), "wood"))
		}
	}

	// --- Fast ---
	// Eleven calls: an item step is every 5th call (itemnet.StepTicks 20 over
	// four ticks), a fluid step every 2nd (fluidnet.StepTicks 8); the cycle is
	// 10 calls long, the 11th starts it again.
	{
		s, _, _ := fresh(1)
		for i := 1; i <= 11; i++ {
			in(s, func(tx *world.Tx) {
				pinRun(t, lines, fmt.Sprintf("Fast/%02d", i), func() string { return fmt.Sprintf("fastCount=%d", *fastCount.Of(s.Session)) }, func() { s.Fast(tx) })
			})
		}
	}
	{
		s, _, _ := fresh(0)
		in(s, func(tx *world.Tx) {
			pinRun(t, lines, "Fast/noPlayers", func() string { return fmt.Sprintf("fastCount=%d", *fastCount.Of(s.Session)) }, func() { s.Fast(tx) })
		})
	}
	{
		s, _, _ := fresh(2)
		in(s, func(tx *world.Tx) {
			pinRun(t, lines, "Fast/twoPlayers", func() string { return fmt.Sprintf("fastCount=%d players=%d", *fastCount.Of(s.Session), players(tx)) }, func() { s.Fast(tx) })
		})
	}
	// A player in the hunting ground without a hunt is sent back to the island.
	{
		s, _, hs := fresh(1)
		withPlayer(t, s, hs[0], func(p *player.Player, tx *world.Tx) {
			p.Teleport(mgl64.Vec3{float64(core.ArenaCentre[0]) + 0.5, float64(core.ArenaCentre[1]), float64(core.ArenaCentre[2]) + 0.5})
			if !core.InArena(p.Position()) {
				t.Fatal("the player is not in the hunting ground")
			}
			pinRun(t, lines, "Fast/watchdog", func() string { return fmt.Sprintf("inArena=%t", core.InArena(p.Position())) }, func() { s.Fast(tx) })
		})
	}

	// --- Autosave and Close ---
	a := cube.Pos{3, island.IslandY, 3}
	// withSaved puts a disk with seven items and a conduit whose cell is
	// stale (the grid holds 150) into the session.
	type saved struct {
		id  string
		buf *conduit.BufCell
	}
	key := netstore.KeyOf(item.NewStack(block.Stone{}, 1))
	withSaved := func(s *session) saved {
		id := netstore.Disks(s.Session).NewID()
		if netstore.Disks(s.Session).Open(id, 1024).Insert(key, 7, false) != 7 {
			t.Fatal("insert")
		}
		buf := &conduit.BufCell{Energy: 150}
		s.World.Do(func(tx *world.Tx) {
			tx.SetBlock(a, conduit.Bundle{Energy: 1, Buf: buf}, nil)
		})
		s.World.Do(func(tx *world.Tx) {
			tx.SetBlock(cube.Pos{5, island.IslandY, 5}, storage.Drawer{Facing: cube.North, Item: item.NewStack(block.Stone{}, 1), Count: 3}, nil)
		})
		tick(s, 40)
		if len(storage.LocalOf(s.Session).Displays) == 0 || len(conduit.LocalOf(s.Session).Visuals) == 0 {
			t.Fatalf("displays %d, visuals %d before the call", len(storage.LocalOf(s.Session).Displays), len(conduit.LocalOf(s.Session).Visuals))
		}
		buf.Energy = 0
		return saved{id, buf}
	}
	diskIn := func(c *core.Session, id string) int64 { return netstore.Disks(c).Open(id, 1024).Contents()[key] }
	for _, c := range []struct {
		name string
		n    int
		disk bool
	}{{"Autosave/empty", 0, false}, {"Autosave/disk", 0, true}, {"Autosave/onePlayer", 1, true}, {"Autosave/twoPlayers", 2, true}} {
		s, st, _ := fresh(c.n)
		var sv saved
		if c.disk {
			sv = withSaved(s)
		}
		in(s, func(tx *world.Tx) {
			pinRun(t, lines, c.name, func() string {
				if !c.disk {
					return fmt.Sprintf("dirty=%t", s.Dirty())
				}
				return fmt.Sprintf("dirty=%t cell=%d", s.Dirty(), sv.buf.Energy)
			}, func() { s.Autosave(tx) })
		})
		st.await(t, 1)
		out := fmt.Sprintf(" stored=%d", st.count())
		if c.disk {
			out += fmt.Sprintf(" disk=%d", diskIn(st.reload(), sv.id))
		}
		lines[c.name] += out
	}
	for _, c := range []struct {
		name string
		n    int
		disk bool
	}{{"Close/empty", 0, false}, {"Close/disk", 0, true}, {"Close/onePlayer", 1, true}, {"Close/twoPlayers", 2, true}} {
		s, st, _ := fresh(c.n)
		var sv saved
		if c.disk {
			sv = withSaved(s)
		}
		in(s, func(tx *world.Tx) {
			pinRun(t, lines, c.name, func() string {
				_, reg := core.OfWorld(s.World)
				_, mine := s.g.sessionOf(s.World)
				out := fmt.Sprintf("stored=%d registered=%t game=%t visuals=%d displays=%d", st.count(), reg, mine, len(conduit.LocalOf(s.Session).Visuals), len(storage.LocalOf(s.Session).Displays))
				if c.disk {
					out += fmt.Sprintf(" cell=%d disk=%d", sv.buf.Energy, diskIn(st.reload(), sv.id))
				}
				return out
			}, func() { s.Close(tx) })
		})
	}

	// --- the world handler ---
	// A bundle is in the grids after its column loads and out of them after
	// it unloads.
	{
		s, _, _ := fresh(0)
		h := &core.WorldHandler{S: s.Session}
		in(s, func(tx *world.Tx) {
			b := conduit.Bundle{Energy: 1, Buf: &conduit.BufCell{Energy: 50}}
			tx.SetBlock(a, b, nil)
			blocks := map[cube.Pos]world.Block{a: b}
			ingrid := func() string {
				_, _, ok := network.Energy(s.Session).Buffer(a)
				return fmt.Sprintf("inGrid=%t", ok)
			}
			pinRun(t, lines, "ColumnUnload/bundle", ingrid, func() { h.HandleColumnUnload(tx, world.ChunkPos{}, blocks) })
			pinRun(t, lines, "ColumnLoad/bundle", ingrid, func() { h.HandleColumnLoad(tx, world.ChunkPos{}, blocks) })
			pinRun(t, lines, "ColumnUnload/empty", ingrid, func() { h.HandleColumnUnload(tx, world.ChunkPos{}, nil) })
			pinRun(t, lines, "ColumnLoad/empty", ingrid, func() { h.HandleColumnLoad(tx, world.ChunkPos{}, nil) })
		})
	}
	{
		s, _, hs := fresh(1)
		h := &core.WorldHandler{S: s.Session}
		lowest := float64(world.Overworld.Range()[0])
		drop := func(name string, y float64, ent func(tx *world.Tx, y float64) world.Entity) {
			in(s, func(tx *world.Tx) {
				e := ent(tx, y)
				before := lostItems(s)
				pinRun(t, lines, name, func() string { return fmt.Sprintf("lostItems=%d", lostItems(s)-before) }, func() { h.HandleEntityDespawn(tx, e) })
			})
		}
		stack := item.NewStack(block.Cobblestone{}, 5)
		itemAt := func(tx *world.Tx, y float64) world.Entity {
			return tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: mgl64.Vec3{0.5, y, 0.5}}, stack))
		}
		drop("EntityDespawn/void", lowest-10, itemAt)
		drop("EntityDespawn/pickedUp", 64, itemAt)
		drop("EntityDespawn/player", lowest-10, func(tx *world.Tx, y float64) world.Entity {
			var out world.Entity
			for e := range tx.Players() {
				out = e
			}
			return out
		})
		_ = hs
	}

	lines["Start"] = pinStart(t)

	pinGolden(t, filepath.Join("testdata", "pin", "lifecycle.golden"), lines)
	var names []string
	for _, file := range []string{"game.go", "legacy_network.go"} {
		names = append(names, pinMarksIn(t, file, "package skygear")...)
	}
	for _, sub := range core.Subscriptions() {
		switch sub.Hook {
		case "Second", "Fast", "Load", "Start", "Join", "Leave", "NewDay", "Autosave", "Close", "ColumnLoad", "ColumnUnload", "EntityDespawn":
			names = append(names, sub.Name)
		}
	}
	pinCheckReached(t, names, lines)
}

// pinStart runs the real Game.Start under a real instance manager: one player
// enters a new save. Only the names Game.Start writes are kept (the
// instance's own ticks write others meanwhile). The trace is set before the
// manager exists and reset after CloseAll, so no instance goroutine meets the
// change.
func pinStart(t *testing.T) string {
	t.Helper()
	startNames := map[string]bool{"farm.loadCrops": true, "island.keepLoaded": true, "climate.apply": true}
	var k pinSink
	var mu sync.Mutex
	core.Trace = func(name string) {
		if startNames[name] {
			mu.Lock()
			k.add(name)
			mu.Unlock()
		}
	}
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			core.Trace = nil
		}
	}
	defer stop()

	dir := t.TempDir()
	d, err := db.Open(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()
	g := &Game{log: slog.New(slog.DiscardHandler), db: d, release: "test"}
	activeGame = g
	registerItemIO()
	lobby := world.Config{Entities: entity.DefaultRegistry.Config().New(EntityTypes())}.New()
	defer func() { _ = lobby.Close() }()
	m := instance.NewManager(slog.New(slog.DiscardHandler), d, instance.Config{
		Dir: filepath.Join(dir, "saves"), DayLength: time.Hour, LoadRadius: 2,
		Entities: entity.DefaultRegistry.Config().New(EntityTypes()),
	}, instance.Lobby{World: lobby, Spawn: mgl64.Vec3{0, 100, 0}}, g)
	closed := false
	closeAll := func() {
		if !closed {
			closed = true
			m.CloseAll()
		}
	}
	defer closeAll()

	id := uuid.New()
	if _, err := d.TouchAccount(id.String(), "", "pin"); err != nil {
		t.Fatal(err)
	}
	save, err := d.CreateSave(id.String(), 1, "pin")
	if err != nil {
		t.Fatal(err)
	}
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "pin", UUID: id, Position: mgl64.Vec3{0, 100, 0}})
	<-lobby.Do(func(tx *world.Tx) { m.Join(tx.AddEntity(h).(*player.Player)) }).Done()
	entered := make(chan error, 1)
	h.Do(func(_ *world.Tx, e world.Entity) { entered <- m.Enter(e.(*player.Player), save.ID) })
	select {
	case err := <-entered:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Enter did not return")
	}
	for range 5000 {
		mu.Lock()
		n := len(k.names)
		mu.Unlock()
		if n >= len(startNames) && m.Location(id.String()) != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	closeAll()
	stop()
	return strings.Join(k.take(), " ") + " | "
}

// TestPinSettings records the keys of account.Settings in registration order
// into testdata/pin/settings.golden. Re-record it
// (SKYGEAR_GOLDEN=update) only when a setting is added, removed or moved on
// purpose.
func TestPinSettings(t *testing.T) {
	lines := map[string]string{}
	for i, st := range account.Settings() {
		lines[fmt.Sprintf("%02d", i)] = fmt.Sprintf(" | %s default=%t group=%s", st.Key, st.Default, st.Group)
	}
	pinGolden(t, filepath.Join("testdata", "pin", "settings.golden"), lines)
}
