package core

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Who is the session, the player and the player's transaction. It is all a
// player event carries that Dragonfly does not let a handler cancel (Jump,
// Death, Respawn): such an event has no Cancel.
type Who struct {
	S  *Session
	P  *player.Player
	Tx *world.Tx
}

// Actor is what a player event that can be cancelled carries: Who and the
// Dragonfly context of the event.
//
// Cancel cancels the event: the action the player asked for does not happen
// in Dragonfly, and the subscribers that follow are not called for it, except
// those registered with the HandleCancelled option (CancelHook). Cancelled
// tells the state. A subscriber cancels only when the action or the later
// subscribers must not happen; otherwise it does its work and leaves the
// event alone.
type Actor struct {
	Who
	Ctx *player.Context
}

// Cancel cancels the event. It does nothing for an event built without a
// context (a test).
func (a Actor) Cancel() {
	if a.Ctx != nil {
		a.Ctx.Cancel()
	}
}

// Cancelled reports whether the event is cancelled: by a subscriber that ran
// before, or by Dragonfly.
func (a Actor) Cancelled() bool {
	return a.Ctx != nil && a.Ctx.Cancelled()
}

// cancelState is where the event's cancel state is kept, nil for an event
// built without a context.
func (a Actor) cancelState() *world.Context {
	if a.Ctx == nil {
		return nil
	}
	return a.Ctx.Context
}

// Pointer fields (Drops, XP, Damage, Pos, Pick) are the pointers Dragonfly
// passed to the handler, not copies: a subscriber that writes through one
// changes what Dragonfly does, exactly as a direct handler would. The other
// fields (Held, Off, Block, Item, From) are copies taken when the event was
// built: when a subscriber changes the hand, the world or the player's place,
// later subscribers still see the old value and must ask the player or the Tx
// for the new one.

// StartBreakEvent is a player starting to break the block at Pos.
type StartBreakEvent struct {
	Actor
	Pos cube.Pos
}

// BlockBreakEvent is a player breaking the block at Pos. Block is the block
// there before it breaks and Held the player's main-hand item.
type BlockBreakEvent struct {
	Actor
	Pos   cube.Pos
	Block world.Block
	Held  item.Stack
	Drops *[]item.Stack
	XP    *int
}

// BlockPlaceEvent is a player placing Block at Pos. Held is the player's
// main-hand item.
type BlockPlaceEvent struct {
	Actor
	Pos   cube.Pos
	Block world.Block
	Held  item.Stack
}

// UseOnBlockEvent is a player using Held on the block at Pos, on Face, at
// Click.
type UseOnBlockEvent struct {
	Actor
	Pos   cube.Pos
	Face  cube.Face
	Click mgl64.Vec3
	Held  item.Stack
}

// EntityEvent is a player using an item on, or attacking, Target.
type EntityEvent struct {
	Actor
	Target world.Entity
}

// UseEvent is a player using the item in the air. Held and Off are the main
// and the off hand.
type UseEvent struct {
	Actor
	Held, Off item.Stack
}

// MoveEvent is a player moving from From to Pos, turned to Rot. From is
// where the player stood when the event was built: a subscriber that moves
// the player (a teleport) does not change it.
type MoveEvent struct {
	Actor
	From mgl64.Vec3
	Pos  mgl64.Vec3
	Rot  cube.Rotation
}

// PlayerEvent is a cancellable event that has no arguments of its own (Sneak,
// Sprint, PunchAir, Chat, BlockPick, HeldSlot, Command).
type PlayerEvent struct {
	Actor
}

// NoticeEvent is a player event Dragonfly only gives notice of (Jump, Death):
// it cannot be cancelled.
type NoticeEvent struct {
	Who
}

// ItemEvent is a player consuming, picking up or dropping Item. For ItemPickup
// Pick is the stack Dragonfly will add, which a subscriber may change; it is
// nil for the other hooks.
type ItemEvent struct {
	Actor
	Item item.Stack
	Pick *item.Stack
}

// ItemDamageEvent is Item about to lose *Damage durability.
type ItemDamageEvent struct {
	Actor
	Item   item.Stack
	Damage *int
}

// HurtEvent is a player about to take *Damage.
type HurtEvent struct {
	Actor
	Damage *float64
}

// RespawnEvent is a player respawning at *Pos. It cannot be cancelled.
type RespawnEvent struct {
	Who
	Pos *mgl64.Vec3
}

// Tick is a lifecycle event of the whole session. Tx is the world's
// transaction; it is nil for Load and may be nil for Autosave and Close in
// tests.
type Tick struct {
	S  *Session
	Tx *world.Tx
}

// DayEvent is the instance day changing to Day.
type DayEvent struct {
	S   *Session
	Tx  *world.Tx
	Day int
}

// Presence is a player joining (First: for the first time in the save) or
// leaving the save.
type Presence struct {
	S     *Session
	P     *player.Player
	Tx    *world.Tx
	First bool
}

// ColumnEvent is a column loaded into or unloaded from the world. Blocks is
// the column's own block entity map: what a subscriber writes there is saved.
type ColumnEvent struct {
	S      *Session
	Tx     *world.Tx
	Blocks map[cube.Pos]world.Block
}

// DespawnEvent is an entity leaving the world.
type DespawnEvent struct {
	S      *Session
	Tx     *world.Tx
	Entity world.Entity
}

// Player hooks, one per method of PlayerHandler. Their subscribers run tier
// by tier. A CancelHook is an event Dragonfly lets a handler cancel; Jump,
// Death and Respawn are not.
var (
	StartBreak   = NewCancelHook[StartBreakEvent]("StartBreak")
	BlockBreak   = NewCancelHook[BlockBreakEvent]("BlockBreak")
	BlockPlace   = NewCancelHook[BlockPlaceEvent]("BlockPlace")
	UseOnBlock   = NewCancelHook[UseOnBlockEvent]("UseOnBlock")
	UseOnEntity  = NewCancelHook[EntityEvent]("UseOnEntity")
	AttackEntity = NewCancelHook[EntityEvent]("AttackEntity")
	Use          = NewCancelHook[UseEvent]("Use")
	Move         = NewCancelHook[MoveEvent]("Move")
	// Sneak and Sprint are emitted when sneaking or sprinting starts only.
	Sneak       = NewCancelHook[PlayerEvent]("Sneak")
	Sprint      = NewCancelHook[PlayerEvent]("Sprint")
	Jump        = NewHook[NoticeEvent]("Jump")
	ItemDamage  = NewCancelHook[ItemDamageEvent]("ItemDamage")
	ItemConsume = NewCancelHook[ItemEvent]("ItemConsume")
	ItemPickup  = NewCancelHook[ItemEvent]("ItemPickup")
	ItemDrop    = NewCancelHook[ItemEvent]("ItemDrop")
	Death       = NewHook[NoticeEvent]("Death")
	Respawn     = NewHook[RespawnEvent]("Respawn")
	Hurt        = NewCancelHook[HurtEvent]("Hurt")
	PunchAir    = NewCancelHook[PlayerEvent]("PunchAir")
	Chat        = NewCancelHook[PlayerEvent]("Chat")
	BlockPick   = NewCancelHook[PlayerEvent]("BlockPick")
	HeldSlot    = NewCancelHook[PlayerEvent]("HeldSlot")
	Command     = NewCancelHook[PlayerEvent]("Command")
)

// Session and world hooks, emitted by Lifecycle and WorldHandler. None of
// their events can be cancelled: every subscriber is called.
var (
	// Load is emitted before the save's first transaction, Start inside it
	// (Tx is nil for Load).
	Load          = NewHook[Tick]("Load")
	Start         = NewHook[Tick]("Start")
	Join          = NewHook[Presence]("Join")
	Leave         = NewHook[Presence]("Leave")
	NewDay        = NewHook[DayEvent]("NewDay")
	Second        = NewHook[Tick]("Second")
	Fast          = NewHook[Tick]("Fast")
	Autosave      = NewHook[Tick]("Autosave")
	Close         = NewHook[Tick]("Close")
	ColumnLoad    = NewHook[ColumnEvent]("ColumnLoad")
	ColumnUnload  = NewHook[ColumnEvent]("ColumnUnload")
	EntityDespawn = NewHook[DespawnEvent]("EntityDespawn")
)
