package skygear

import (
	"fmt"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestPinSimple records, per scenario, the marks the simple player events of
// handler.go reach (HandleMove down to HandleCommandExecution) and the
// outcome, into testdata/pin/handler-simple.golden. Each scenario runs on a
// fresh session and player. The golden is recorded from the legacy code; it
// may be re-recorded (SKYGEAR_GOLDEN=update) only when the legacy behaviour
// itself is being changed on purpose.
func TestPinSimple(t *testing.T) {
	lines := map[string]string{}
	cancelled := func(ctx *player.Context) func() string {
		return func() string { return fmt.Sprintf("cancelled=%t", ctx.Cancelled()) }
	}
	// scene runs f on a fresh session with the player inside a transaction.
	scene := func(f func(s *session, hd *handler, p *player.Player, tx *world.Tx)) {
		s := testSession(t)
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			f(s, s.Handler(p).(*handler), p, tx)
		})
	}
	// ctxEvent runs a context event: call gets the context.
	ctxEvent := func(name string, setup func(s *session, p *player.Player, tx *world.Tx), call func(hd *handler, ctx *player.Context, p *player.Player)) {
		scene(func(s *session, hd *handler, p *player.Player, tx *world.Tx) {
			if setup != nil {
				setup(s, p, tx)
			}
			ctx := player.NewEventContext(tx, p)
			pinRun(t, lines, name, cancelled(ctx), func() { call(hd, ctx, p) })
		})
	}
	plain := func(name string, call func(hd *handler, ctx *player.Context, p *player.Player)) {
		ctxEvent(name, nil, call)
	}
	expanding := func(s *session, p *player.Player, tx *world.Tx) { expand.Show(s.Session, p) }
	hunting := func(s *session, p *player.Player, tx *world.Tx) {
		if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the hunt did not begin")
		}
	}
	// twoElevators puts two red elevators 12 blocks apart and stands the
	// player on the lower (up) or the upper (down) one.
	twoElevators := func(top bool) func(s *session, p *player.Player, tx *world.Tx) {
		return func(s *session, p *player.Player, tx *world.Tx) {
			x, z := island.Centre+2, island.Centre
			base := island.IslandY + 2
			red := elevator.Elevator{Colour: item.ColourRed()}
			tx.SetBlock(cube.Pos{x, base, z}, red, nil)
			tx.SetBlock(cube.Pos{x, base + 12, z}, red, nil)
			dy := 0
			if top {
				dy = 12
			}
			p.Teleport(mgl64.Vec3{float64(x) + 0.3, float64(base + dy + 1), float64(z) + 0.7})
		}
	}
	move := func(to mgl64.Vec3) func(hd *handler, ctx *player.Context, p *player.Player) {
		return func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandleMove(ctx, to, cube.Rotation{}) }
	}
	sneak := func(after bool) func(hd *handler, ctx *player.Context, p *player.Player) {
		return func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandleToggleSneak(ctx, after) }
	}

	ctxEvent("Move/walk", nil, move(mgl64.Vec3{1.5, 64, 2.5}))
	ctxEvent("Move/expanding", expanding, move(mgl64.Vec3{3, 64, 3}))
	ctxEvent("Move/void", nil, move(mgl64.Vec3{0.5, -10, 2.5}))

	ctxEvent("Sneak/start", nil, sneak(true))
	ctxEvent("Sneak/end", nil, sneak(false))
	ctxEvent("Sneak/expanding", expanding, sneak(true))
	ctxEvent("Sneak/elevator", twoElevators(true), func(hd *handler, ctx *player.Context, p *player.Player) {
		y := p.Position().Y()
		sneak(true)(hd, ctx, p)
		if p.Position().Y() >= y {
			t.Fatal("Sneak/elevator: did not ride down")
		}
	})
	ctxEvent("Sprint/start", nil, func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandleToggleSprint(ctx, true) })
	ctxEvent("Sprint/end", nil, func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandleToggleSprint(ctx, false) })

	for name, setup := range map[string]func(s *session, p *player.Player, tx *world.Tx){"Jump/plain": nil, "Jump/elevator": twoElevators(false)} {
		scene(func(s *session, hd *handler, p *player.Player, tx *world.Tx) {
			if setup != nil {
				setup(s, p, tx)
			}
			y := p.Position().Y()
			pinRun(t, lines, name, nil, func() { hd.HandleJump(p) })
			if rode := p.Position().Y() != y; rode != (setup != nil) {
				t.Fatalf("%s: rode = %t", name, rode)
			}
		})
	}

	damage := func(name string, held, it item.Stack, dmg int) {
		scene(func(s *session, hd *handler, p *player.Player, tx *world.Tx) {
			p.SetHeldItems(held, item.Stack{})
			ctx := player.NewEventContext(tx, p)
			pinRun(t, lines, name, func() string { return fmt.Sprintf("damage=%d", dmg) }, func() { hd.HandleItemDamage(ctx, it, &dmg) })
		})
	}
	pick := item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1)
	hammer := item.NewStack(sgitem.Hammer{}, 1)
	damage("ItemDamage/plain", item.Stack{}, pick, 1)
	damage("ItemDamage/ironHammer", hammer, hammer, 5)
	damage("ItemDamage/lastUse", pick, pick, pick.Durability())

	plain("ItemConsume/plain", func(hd *handler, ctx *player.Context, p *player.Player) {
		hd.HandleItemConsume(ctx, item.NewStack(item.Apple{}, 1))
	})
	plain("ItemPickup/plain", func(hd *handler, ctx *player.Context, p *player.Player) {
		st := item.NewStack(block.Dirt{}, 3)
		hd.HandleItemPickup(ctx, &st)
	})
	plain("PunchAir/plain", func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandlePunchAir(ctx) })
	plain("Chat/plain", func(hd *handler, ctx *player.Context, p *player.Player) {
		msg := "hi"
		hd.HandleChat(ctx, &msg)
	})
	plain("BlockPick/plain", func(hd *handler, ctx *player.Context, p *player.Player) {
		hd.HandleBlockPick(ctx, cube.Pos{}, block.Dirt{})
	})
	plain("HeldSlot/plain", func(hd *handler, ctx *player.Context, p *player.Player) { hd.HandleHeldSlotChange(ctx, 0, 1) })
	plain("Command/plain", func(hd *handler, ctx *player.Context, p *player.Player) {
		hd.HandleCommandExecution(ctx, cmd.Command{}, nil)
	})

	plain("ItemDrop/plain", func(hd *handler, ctx *player.Context, p *player.Player) {
		hd.HandleItemDrop(ctx, item.NewStack(block.Dirt{}, 1))
	})
	ctxEvent("ItemDrop/hunting", hunting, func(hd *handler, ctx *player.Context, p *player.Player) {
		hd.HandleItemDrop(ctx, item.NewStack(block.Dirt{}, 1))
	})

	scene(func(s *session, hd *handler, p *player.Player, tx *world.Tx) {
		pinRun(t, lines, "Death/plain", nil, func() { hd.HandleDeath(p, nil, new(bool)) })
	})
	scene(func(s *session, hd *handler, p *player.Player, tx *world.Tx) {
		p.SetFood(5)
		hd.HandleDeath(p, nil, new(bool))
		p.SetFood(20)
		pos, w := p.Position(), tx.World()
		pinRun(t, lines, "Respawn/afterDeath", func() string { return fmt.Sprintf("food=%d", p.Food()) }, func() { hd.HandleRespawn(p, &pos, &w) })
	})
	scene(func(s *session, hd *handler, p *player.Player, tx *world.Tx) {
		pos, w := p.Position(), tx.World()
		pinRun(t, lines, "Respawn/plain", func() string { return fmt.Sprintf("food=%d", p.Food()) }, func() { hd.HandleRespawn(p, &pos, &w) })
	})

	hurt := func(hd *handler, dmg float64) func(ctx *player.Context, p *player.Player) {
		return func(ctx *player.Context, p *player.Player) {
			var d time.Duration
			hd.HandleHurt(ctx, &dmg, false, &d, nil)
		}
	}
	ctxEvent("Hurt/plain", nil, func(hd *handler, ctx *player.Context, p *player.Player) { hurt(hd, 1)(ctx, p) })
	ctxEvent("Hurt/frozen", func(s *session, p *player.Player, tx *world.Tx) {
		hunting(s, p, tx)
		r := dimension.RunOf(s.Session)
		dimension.GiveGems(r, tx, p, 4)
		if !dimension.Frozen(s.Session, p) {
			t.Fatal("the hunter is not frozen")
		}
	}, func(hd *handler, ctx *player.Context, p *player.Player) { hurt(hd, 1)(ctx, p) })
	ctxEvent("Hurt/fallen", hunting, func(hd *handler, ctx *player.Context, p *player.Player) { hurt(hd, p.Health()+100)(ctx, p) })

	pinGolden(t, "testdata/pin/handler-simple.golden", lines)
	pinCheckReached(t, pinMarksIn(t, "handler.go", "func (h *handler) HandleMove("), lines)
}
