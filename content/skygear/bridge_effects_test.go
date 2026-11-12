package skygear

import (
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

// bridgeAccount opens an account store for the test (closed with it) and
// returns the database.
func bridgeAccount(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		account.Flush()
		account.Init(nil, slog.Default())
		_ = d.Close()
	})
	account.Init(d, slog.Default())
	return d
}

// TestBridgeCounters pins what the order goldens cannot: every player event
// counts under its own key, by its own amount, exactly once, and the toggles
// count only when they start.
func TestBridgeCounters(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		hd := s.Handler(p)
		ctx := func() *player.Context { return player.NewEventContext(tx, p) }
		count := func(key string, want int64, f func()) {
			t.Helper()
			before := account.Counter(acc, key)
			f()
			if got := account.Counter(acc, key) - before; got != want {
				t.Errorf("%s counted %d, want %d", key, got, want)
			}
		}
		count("sprint", 1, func() { hd.HandleToggleSprint(ctx(), true) })
		count("sprint", 0, func() { hd.HandleToggleSprint(ctx(), false) })
		count("sneak", 1, func() { hd.HandleToggleSneak(ctx(), true) })
		count("sneak", 0, func() { hd.HandleToggleSneak(ctx(), false) })
		count("jump", 1, func() { hd.HandleJump(p) })
		count("punch_air", 1, func() { hd.HandlePunchAir(ctx()) })
		count("chat", 1, func() { msg := "hi"; hd.HandleChat(ctx(), &msg) })
		count("block_pick", 1, func() { hd.HandleBlockPick(ctx(), cube.Pos{}, block.Dirt{}) })
		count("hotbar", 1, func() { hd.HandleHeldSlotChange(ctx(), 0, 1) })
		count("command", 1, func() { hd.HandleCommandExecution(ctx(), cmd.Command{}, nil) })
		count("item_drop", 1, func() { hd.HandleItemDrop(ctx(), item.NewStack(block.Dirt{}, 1)) })
		count("eat", 1, func() { hd.HandleItemConsume(ctx(), item.NewStack(item.Apple{}, 1)) })
		count("pickup", 3, func() {
			st := item.NewStack(block.Dirt{}, 3)
			hd.HandleItemPickup(ctx(), &st)
		})
		count("hurt", 1, func() {
			dmg, d := 1.0, time.Duration(0)
			hd.HandleHurt(ctx(), &dmg, false, &d, nil)
		})
		count("death", 1, func() { hd.HandleDeath(p, nil, new(bool)) })
		count("block_place", 1, func() { hd.HandleBlockPlace(ctx(), cube.Pos{0, 60, 0}, block.Dirt{}) })
		count("block_break", 1, func() {
			var drops []item.Stack
			var xp int
			tx.SetBlock(cube.Pos{1, 60, 0}, block.Stone{}, nil)
			hd.HandleBlockBreak(ctx(), cube.Pos{1, 60, 0}, &drops, &xp)
		})
		count("play_seconds", 1, func() { s.Second(tx) })
	})
}

// TestBridgeCollection: Second hands the inventory names progress.checkHeld
// took to progress.collection, which grants the first-possession achievement.
func TestBridgeCollection(t *testing.T) {
	d := bridgeAccount(t)
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Inventory().AddItem(item.NewStack(item.Bone{}, 1)); err != nil {
			t.Fatal(err)
		}
		if slices.Contains(account.Owned(acc), "sg_have_bone") {
			t.Fatal("granted before the second")
		}
		s.Second(tx)
		if !slices.Contains(account.Owned(acc), "sg_have_bone") {
			t.Fatalf("a held bone must grant sg_have_bone, owned %v", account.Owned(acc))
		}
	})
}

// TestBridgeFastCountPerSession: each session counts its own Fast calls.
func TestBridgeFastCountPerSession(t *testing.T) {
	a, b := testSession(t), testSession(t)
	for _, s := range []*session{a, a, a, b} {
		s.World.Do(func(tx *world.Tx) { s.Fast(tx) })
	}
	if x, y := network.FastCount(a.Session), network.FastCount(b.Session); x != 3 || y != 1 {
		t.Fatalf("fast counts %d and %d, want 3 and 1", x, y)
	}
}

// TestBridgeKeptFoodLeave: the food a player died with is dropped when the
// player leaves, not kept for the next player of the session.
func TestBridgeKeptFoodLeave(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		hd := s.Handler(p)
		p.SetFood(4)
		hd.HandleDeath(p, nil, new(bool))
		if got, ok := tinker.KeptFood(s.Session)[h]; !ok || got != 4 {
			t.Fatalf("kept food %d %v, want 4", got, ok)
		}
		s.Leave(p)
		if _, ok := tinker.KeptFood(s.Session)[h]; ok {
			t.Fatal("kept food survived the player's leaving")
		}
	})
}
