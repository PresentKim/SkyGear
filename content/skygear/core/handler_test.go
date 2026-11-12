package core

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"golang.org/x/text/language"
)

func init() { world.DefaultBlockRegistry.Finalize() }

// fix is a session over a synchronous world with one player, inside the
// player's transaction.
type fix struct {
	s   *Session
	w   *world.World
	p   *player.Player
	tx  *world.Tx
	ctx *player.Context
	h   *PlayerHandler
	wh  *WorldHandler
	l   Lifecycle
}

// withFix runs f in a world transaction with a player, over a fresh kernel.
func withFix(t *testing.T, store func([]byte) error, f func(x *fix)) {
	t.Helper()
	ResetHooks()
	t.Cleanup(ResetHooks)
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	s := New(Config{World: w, Store: store})
	Register(s)
	t.Cleanup(func() { Unregister(s) })
	id := uuid.New()
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "p", UUID: id, Locale: language.English, Position: mgl64.Vec3{0.5, 64, 2.5}})
	done := false
	w.Do(func(tx *world.Tx) {
		tx.AddEntity(h)
		e, _ := h.Entity(tx)
		p := e.(*player.Player)
		p.SetHeldItems(stack, off)
		f(&fix{s: s, w: w, p: p, tx: tx, ctx: player.NewEventContext(tx, p),
			h: &PlayerHandler{S: s}, wh: &WorldHandler{S: s}, l: Lifecycle{s}})
		done = true
	})
	if !done {
		t.Fatal("transaction did not finish (panicked)")
	}
}

// row is one hook: call makes its event through the handler or lifecycle
// method, arm subscribes a recorder and returns the check of what it saw.
type row struct {
	name  string
	call  func(x *fix)
	alloc func(x *fix) // call, when call does more than emit
	arm   func() func(t *testing.T, x *fix)
	nop   func()
}

func rowOf[E any](name string, h *Hook[E], call func(x *fix), check func(t *testing.T, x *fix, e E)) row {
	return row{
		name: name,
		call: call,
		arm: func() func(*testing.T, *fix) {
			var got []E
			h.On(10, "t.rec", func(e E) { got = append(got, e) })
			return func(t *testing.T, x *fix) {
				t.Helper()
				if len(got) != 1 {
					t.Fatalf("%d events emitted, want 1", len(got))
				}
				check(t, x, got[0])
			}
		},
		nop: func() { h.On(10, "t.nop", func(E) {}) },
	}
}

func is[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func sameStack(t *testing.T, what string, got, want item.Stack) {
	t.Helper()
	if !got.Equal(want) || got.Count() != want.Count() {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func actorIs(t *testing.T, x *fix, a Actor, ctx bool) {
	t.Helper()
	is(t, "S", a.S, x.s)
	is(t, "P", a.P, x.p)
	is(t, "Tx", a.Tx, x.tx)
	if ctx {
		is(t, "Ctx", a.Ctx, x.ctx)
	} else {
		is(t, "Ctx", a.Ctx, (*player.Context)(nil))
	}
}

var (
	pos                 = cube.Pos{3, 64, -2}
	stone   world.Block = block.Stone{}
	stack               = item.NewStack(item.Stick{}, 3)
	off                 = item.NewStack(item.Arrow{}, 2)
	vec                 = mgl64.Vec3{1.5, 70, -2.5}
	rot                 = cube.Rotation{90, 15}
	columns             = map[cube.Pos]world.Block{pos: stone}
)

func rows() []row {
	var drops = []item.Stack{stack}
	var xp = 7
	dmg := 4
	hurt := 2.5
	respawn := mgl64.Vec3{9, 9, 9}
	picked := stack
	return []row{
		rowOf("StartBreak", StartBreak, func(x *fix) { x.h.HandleStartBreak(x.ctx, pos) },
			func(t *testing.T, x *fix, e StartBreakEvent) { actorIs(t, x, e.Actor, true); is(t, "Pos", e.Pos, pos) }),
		rowOf("BlockBreak", BlockBreak, func(x *fix) { x.h.HandleBlockBreak(x.ctx, pos, &drops, &xp) },
			func(t *testing.T, x *fix, e BlockBreakEvent) {
				actorIs(t, x, e.Actor, true)
				is(t, "Pos", e.Pos, pos)
				is(t, "Block", e.Block, x.tx.Block(pos))
				sameStack(t, "Held", e.Held, stack)
				is(t, "Drops", e.Drops, &drops)
				is(t, "XP", e.XP, &xp)
			}),
		rowOf("BlockPlace", BlockPlace, func(x *fix) { x.h.HandleBlockPlace(x.ctx, pos, stone) },
			func(t *testing.T, x *fix, e BlockPlaceEvent) {
				actorIs(t, x, e.Actor, true)
				is(t, "Pos", e.Pos, pos)
				is(t, "Block", e.Block, stone)
				sameStack(t, "Held", e.Held, stack)
			}),
		rowOf("UseOnBlock", UseOnBlock, func(x *fix) { x.h.HandleItemUseOnBlock(x.ctx, pos, cube.FaceEast, vec) },
			func(t *testing.T, x *fix, e UseOnBlockEvent) {
				actorIs(t, x, e.Actor, true)
				is(t, "Pos", e.Pos, pos)
				is(t, "Face", e.Face, cube.FaceEast)
				is(t, "Click", e.Click, vec)
				sameStack(t, "Held", e.Held, stack)
			}),
		rowOf("UseOnEntity", UseOnEntity, func(x *fix) { x.h.HandleItemUseOnEntity(x.ctx, x.p) },
			func(t *testing.T, x *fix, e EntityEvent) {
				actorIs(t, x, e.Actor, true)
				is[world.Entity](t, "Target", e.Target, x.p)
			}),
		rowOf("AttackEntity", AttackEntity, func(x *fix) { x.h.HandleAttackEntity(x.ctx, x.p, nil, nil, nil) },
			func(t *testing.T, x *fix, e EntityEvent) {
				actorIs(t, x, e.Actor, true)
				is[world.Entity](t, "Target", e.Target, x.p)
			}),
		rowOf("Use", Use, func(x *fix) { x.h.HandleItemUse(x.ctx) },
			func(t *testing.T, x *fix, e UseEvent) {
				actorIs(t, x, e.Actor, true)
				sameStack(t, "Held", e.Held, stack)
				sameStack(t, "Off", e.Off, off)
			}),
		rowOf("Move", Move, func(x *fix) { x.h.HandleMove(x.ctx, vec, rot) },
			func(t *testing.T, x *fix, e MoveEvent) {
				actorIs(t, x, e.Actor, true)
				is(t, "From", e.From, x.p.Position())
				is(t, "Pos", e.Pos, vec)
				is(t, "Rot", e.Rot, rot)
			}),
		rowOf("Sneak", Sneak, func(x *fix) { x.h.HandleToggleSneak(x.ctx, true) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
		rowOf("Sprint", Sprint, func(x *fix) { x.h.HandleToggleSprint(x.ctx, true) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
		rowOf("Jump", Jump, func(x *fix) { x.h.HandleJump(x.p) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, false) }),
		rowOf("ItemDamage", ItemDamage, func(x *fix) { x.h.HandleItemDamage(x.ctx, stack, &dmg) },
			func(t *testing.T, x *fix, e ItemDamageEvent) {
				actorIs(t, x, e.Actor, true)
				sameStack(t, "Item", e.Item, stack)
				is(t, "Damage", e.Damage, &dmg)
			}),
		rowOf("ItemConsume", ItemConsume, func(x *fix) { x.h.HandleItemConsume(x.ctx, stack) },
			func(t *testing.T, x *fix, e ItemEvent) {
				actorIs(t, x, e.Actor, true)
				sameStack(t, "Item", e.Item, stack)
				is(t, "Pick", e.Pick, (*item.Stack)(nil))
			}),
		rowOf("ItemPickup", ItemPickup, func(x *fix) { x.h.HandleItemPickup(x.ctx, &picked) },
			func(t *testing.T, x *fix, e ItemEvent) {
				actorIs(t, x, e.Actor, true)
				sameStack(t, "Item", e.Item, stack)
				is(t, "Pick", e.Pick, &picked)
			}),
		rowOf("ItemDrop", ItemDrop, func(x *fix) { x.h.HandleItemDrop(x.ctx, stack) },
			func(t *testing.T, x *fix, e ItemEvent) {
				actorIs(t, x, e.Actor, true)
				sameStack(t, "Item", e.Item, stack)
			}),
		rowOf("Death", Death, func(x *fix) { x.h.HandleDeath(x.p, nil, nil) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, false) }),
		rowOf("Respawn", Respawn, func(x *fix) { x.h.HandleRespawn(x.p, &respawn, nil) },
			func(t *testing.T, x *fix, e RespawnEvent) {
				actorIs(t, x, e.Actor, false)
				is(t, "Pos", e.Pos, &respawn)
			}),
		rowOf("Hurt", Hurt, func(x *fix) { x.h.HandleHurt(x.ctx, &hurt, false, nil, nil) },
			func(t *testing.T, x *fix, e HurtEvent) {
				actorIs(t, x, e.Actor, true)
				is(t, "Damage", e.Damage, &hurt)
			}),
		rowOf("PunchAir", PunchAir, func(x *fix) { x.h.HandlePunchAir(x.ctx) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
		rowOf("Chat", Chat, func(x *fix) { x.h.HandleChat(x.ctx, nil) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
		rowOf("BlockPick", BlockPick, func(x *fix) { x.h.HandleBlockPick(x.ctx, pos, stone) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
		rowOf("HeldSlot", HeldSlot, func(x *fix) { x.h.HandleHeldSlotChange(x.ctx, 1, 2) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),
		rowOf("Command", Command, func(x *fix) { x.h.HandleCommandExecution(x.ctx, cmd.Command{}, nil) },
			func(t *testing.T, x *fix, e PlayerEvent) { actorIs(t, x, e.Actor, true) }),

		rowOf("Load", Load, func(x *fix) { Load.Emit(Tick{S: x.s}) },
			func(t *testing.T, x *fix, e Tick) { is(t, "S", e.S, x.s); is(t, "Tx", e.Tx, (*world.Tx)(nil)) }),
		rowOf("Start", Start, func(x *fix) { Start.Emit(Tick{S: x.s, Tx: x.tx}) },
			func(t *testing.T, x *fix, e Tick) { is(t, "S", e.S, x.s); is(t, "Tx", e.Tx, x.tx) }),
		rowOf("Join", Join, func(x *fix) { x.l.Join(x.p, true) },
			func(t *testing.T, x *fix, e Presence) {
				is(t, "S", e.S, x.s)
				is(t, "P", e.P, x.p)
				is(t, "Tx", e.Tx, x.tx)
				is(t, "First", e.First, true)
			}),
		rowOf("Leave", Leave, func(x *fix) { x.l.Leave(x.p) },
			func(t *testing.T, x *fix, e Presence) {
				is(t, "S", e.S, x.s)
				is(t, "P", e.P, x.p)
				is(t, "Tx", e.Tx, x.tx)
				is(t, "First", e.First, false)
			}),
		withAlloc(rowOf("NewDay", NewDay, func(x *fix) { x.l.NewDay(x.tx, 12) },
			func(t *testing.T, x *fix, e DayEvent) {
				is(t, "S", e.S, x.s)
				is(t, "Tx", e.Tx, x.tx)
				is(t, "Day", e.Day, 12)
			}),
			func(x *fix) { NewDay.Emit(DayEvent{S: x.s, Tx: x.tx, Day: 12}) }),
		rowOf("Second", Second, func(x *fix) { x.l.Second(x.tx) },
			func(t *testing.T, x *fix, e Tick) { is(t, "S", e.S, x.s); is(t, "Tx", e.Tx, x.tx) }),
		rowOf("Fast", Fast, func(x *fix) { x.l.Fast(x.tx) },
			func(t *testing.T, x *fix, e Tick) { is(t, "S", e.S, x.s); is(t, "Tx", e.Tx, x.tx) }),
		withAlloc(rowOf("Autosave", Autosave, func(x *fix) { x.l.Autosave(x.tx) },
			func(t *testing.T, x *fix, e Tick) { is(t, "S", e.S, x.s); is(t, "Tx", e.Tx, x.tx) }),
			func(x *fix) { Autosave.Emit(Tick{S: x.s, Tx: x.tx}) }),
		withAlloc(rowOf("Close", Close, func(x *fix) { x.l.Close(x.tx) },
			func(t *testing.T, x *fix, e Tick) { is(t, "S", e.S, x.s); is(t, "Tx", e.Tx, x.tx) }),
			func(x *fix) { Close.Emit(Tick{S: x.s, Tx: x.tx}) }),
		rowOf("ColumnLoad", ColumnLoad, func(x *fix) { x.wh.HandleColumnLoad(x.tx, world.ChunkPos{1, 2}, columns) },
			func(t *testing.T, x *fix, e ColumnEvent) {
				is(t, "S", e.S, x.s)
				is(t, "Tx", e.Tx, x.tx)
				if len(e.Blocks) != 1 || e.Blocks[pos] != stone {
					t.Errorf("Blocks = %v", e.Blocks)
				}
			}),
		rowOf("ColumnUnload", ColumnUnload, func(x *fix) { x.wh.HandleColumnUnload(x.tx, world.ChunkPos{1, 2}, columns) },
			func(t *testing.T, x *fix, e ColumnEvent) {
				is(t, "S", e.S, x.s)
				is(t, "Tx", e.Tx, x.tx)
				if len(e.Blocks) != 1 {
					t.Errorf("Blocks = %v", e.Blocks)
				}
			}),
		rowOf("EntityDespawn", EntityDespawn, func(x *fix) { x.wh.HandleEntityDespawn(x.tx, x.p) },
			func(t *testing.T, x *fix, e DespawnEvent) {
				is(t, "S", e.S, x.s)
				is(t, "Tx", e.Tx, x.tx)
				is[world.Entity](t, "Entity", e.Entity, x.p)
			}),
	}
}

func withAlloc(r row, alloc func(x *fix)) row { r.alloc = alloc; return r }

// TestEveryHandlerMethodEmitsItsHook: each handler or lifecycle method emits
// its hook once, with the event's fields equal to the arguments.
func TestEveryHandlerMethodEmitsItsHook(t *testing.T) {
	if n := len(rows()); n != 35 {
		t.Fatalf("%d rows, want 35 (23 player, 12 session and world hooks)", n)
	}
	for _, r := range rows() {
		t.Run(r.name, func(t *testing.T) {
			withFix(t, nil, func(x *fix) {
				check := r.arm()
				r.call(x)
				check(t, x)
			})
		})
	}
	if got := len(Subscriptions()); got != 0 {
		t.Errorf("%d subscriptions left", got)
	}
}

// TestHooksAreDeclaredInOrder: the 35 hooks list in the order of the plan.
func TestHooksAreDeclaredInOrder(t *testing.T) {
	ResetHooks()
	t.Cleanup(ResetHooks)
	var want []string
	for _, r := range rows() {
		want = append(want, r.name)
		r.nop()
	}
	subs := Subscriptions()
	if len(subs) != len(want) {
		t.Fatalf("%d subscriptions, want %d", len(subs), len(want))
	}
	for i, s := range subs {
		if s.Hook != want[i] {
			t.Errorf("hook %d is %q, want %q", i, s.Hook, want[i])
		}
	}
}

// TestEventsEmitAllocateNothing: through the real handler call, with a subscriber
// that takes the event, no event type allocates.
func TestEventsEmitAllocateNothing(t *testing.T) {
	for _, r := range rows() {
		t.Run(r.name, func(t *testing.T) {
			withFix(t, nil, func(x *fix) {
				r.nop()
				call := r.call
				if r.alloc != nil {
					call = r.alloc
				}
				call(x)
				n := testing.AllocsPerRun(200, func() { call(x) })
				t.Logf("%s: %v allocs/op", r.name, n)
				if n != 0 {
					t.Errorf("%v allocs per call, want 0", n)
				}
			})
		})
	}
}

// TestSneakAndSprintEmitOnStartOnly: nothing happens when sneaking or
// sprinting ends.
func TestSneakAndSprintEmitOnStartOnly(t *testing.T) {
	withFix(t, nil, func(x *fix) {
		var sneaks, sprints int
		Sneak.On(10, "t.sneak", func(PlayerEvent) { sneaks++ })
		Sprint.On(10, "t.sprint", func(PlayerEvent) { sprints++ })
		x.h.HandleToggleSneak(x.ctx, false)
		x.h.HandleToggleSprint(x.ctx, false)
		if sneaks != 0 || sprints != 0 {
			t.Fatalf("ending emitted: %d sneaks, %d sprints", sneaks, sprints)
		}
		x.h.HandleToggleSneak(x.ctx, true)
		x.h.HandleToggleSprint(x.ctx, true)
		if sneaks != 1 || sprints != 1 {
			t.Fatalf("starting emitted: %d sneaks, %d sprints", sneaks, sprints)
		}
	})
}

// TestCancelAndStopAreIndependent: a subscriber may cancel without stopping
// and stop without cancelling, and neither implies the other.
func TestCancelAndStopAreIndependent(t *testing.T) {
	for _, c := range []struct{ cancel, stop bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		withFix(t, nil, func(x *fix) {
			ran := false
			UseOnBlock.Handle(10, "t.first", func(e UseOnBlockEvent) bool {
				if c.cancel {
					e.Cancel()
				}
				return c.stop
			})
			UseOnBlock.On(20, "t.second", func(UseOnBlockEvent) { ran = true })
			x.h.HandleItemUseOnBlock(x.ctx, pos, cube.FaceUp, vec)
			if got := x.ctx.Cancelled(); got != c.cancel {
				t.Errorf("cancel=%v stop=%v: cancelled = %v", c.cancel, c.stop, got)
			}
			if ran == c.stop {
				t.Errorf("cancel=%v stop=%v: second subscriber ran = %v", c.cancel, c.stop, ran)
			}
		})
	}
}

// TestCancelWithoutContext: Cancel on an event Dragonfly cannot cancel does
// nothing.
func TestCancelWithoutContext(t *testing.T) {
	withFix(t, nil, func(x *fix) {
		Jump.On(10, "t.cancel", func(e PlayerEvent) { e.Cancel() })
		x.h.HandleJump(x.p)
	})
}

// TestEventsCarryDragonflyPointers: a subscriber's write through a pointer
// field reaches the variable Dragonfly passed.
func TestEventsCarryDragonflyPointers(t *testing.T) {
	withFix(t, nil, func(x *fix) {
		BlockBreak.On(10, "t.drops", func(e BlockBreakEvent) { *e.Drops = nil; *e.XP = 99 })
		ItemDamage.On(10, "t.damage", func(e ItemDamageEvent) { *e.Damage = 0 })
		Hurt.On(10, "t.hurt", func(e HurtEvent) { *e.Damage = 0.5 })
		Respawn.On(10, "t.respawn", func(e RespawnEvent) { *e.Pos = mgl64.Vec3{1, 2, 3} })
		ItemPickup.On(10, "t.pick", func(e ItemEvent) { *e.Pick = item.Stack{} })

		drops, xp := []item.Stack{stack}, 1
		x.h.HandleBlockBreak(x.ctx, pos, &drops, &xp)
		if drops != nil || xp != 99 {
			t.Errorf("BlockBreak: drops %v, xp %d", drops, xp)
		}
		dmg := 5
		x.h.HandleItemDamage(x.ctx, stack, &dmg)
		is(t, "item damage", dmg, 0)
		h := 3.0
		x.h.HandleHurt(x.ctx, &h, false, nil, nil)
		is(t, "hurt", h, 0.5)
		at := mgl64.Vec3{}
		x.h.HandleRespawn(x.p, &at, nil)
		is(t, "respawn", at, mgl64.Vec3{1, 2, 3})
		picked := stack
		x.h.HandleItemPickup(x.ctx, &picked)
		if !picked.Empty() {
			t.Errorf("pickup: %v", picked)
		}
	})
}

// TestLifecycleKernelStepsRunLast: the flush after Second and Autosave, and
// the final store and unregistering after Close, come after the subscribers.
func TestLifecycleKernelStepsRunLast(t *testing.T) {
	var mu sync.Mutex
	var stored int
	store := func([]byte) error { mu.Lock(); stored++; mu.Unlock(); return nil }
	storedNow := func() int { mu.Lock(); defer mu.Unlock(); return stored }
	t.Run("Second", func(t *testing.T) {
		withFix(t, store, func(x *fix) {
			x.s.FlushInterval = time.Nanosecond
			x.s.MarkDirty()
			var during bool
			Second.On(10, "t.step", func(Tick) { during = x.s.Dirty() })
			x.l.Second(x.tx)
			if !during {
				t.Error("Second flushed before its subscribers")
			}
			if x.s.Dirty() {
				t.Error("Second did not flush after its subscribers")
			}
			x.s.Close()
		})
	})
	t.Run("NewDay", func(t *testing.T) {
		withFix(t, store, func(x *fix) {
			var seen []string
			NewDay.On(10, "t.step", func(DayEvent) {})
			Trace = func(name string) { seen = append(seen, name) }
			defer func() { Trace = nil }()
			x.l.NewDay(x.tx, 3)
			if got := strings.Join(seen, " "); got != "t.step game.newDay" {
				t.Errorf("NewDay traced %q, want its subscribers then game.newDay", got)
			}
		})
	})
	t.Run("Autosave", func(t *testing.T) {
		withFix(t, store, func(x *fix) {
			x.s.MarkDirty()
			var during bool
			Autosave.On(10, "t.step", func(Tick) { during = x.s.Dirty() })
			x.l.Autosave(x.tx)
			if !during {
				t.Error("Autosave flushed before its subscribers")
			}
			if x.s.Dirty() {
				t.Error("Autosave did not flush after its subscribers")
			}
			x.s.Close()
		})
	})
	t.Run("Close", func(t *testing.T) {
		withFix(t, store, func(x *fix) {
			before := storedNow()
			var registered bool
			var storedDuring int
			Close.On(10, "t.step", func(e Tick) {
				_, registered = OfWorld(x.w)
				storedDuring = storedNow()
			})
			x.l.Close(x.tx)
			if !registered || storedDuring != before {
				t.Errorf("Close stored or unregistered before its subscribers (registered %v, stores %d, before %d)", registered, storedDuring, before)
			}
			if storedNow() == before {
				t.Error("Close did not store the session")
			}
			if _, ok := OfWorld(x.w); ok {
				t.Error("Close left the session registered")
			}
		})
	})
}

// TestLifecycleHandler: Handler returns a PlayerHandler over the session.
func TestLifecycleHandler(t *testing.T) {
	withFix(t, nil, func(x *fix) {
		h, ok := x.l.Handler(x.p).(*PlayerHandler)
		if !ok || h.S != x.s {
			t.Fatalf("Handler = %#v", h)
		}
	})
}
