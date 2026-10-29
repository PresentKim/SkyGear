package skygear

import (
	"testing"

	"github.com/presentkim/skygear/content/skygear/handwork"

	"github.com/presentkim/skygear/content/skygear/island"

	"github.com/presentkim/skygear/content/skygear/core"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestSneakingStickFillsChestWithWhatItHolds(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{1, island.IslandY, 1}
		tx.SetBlock(pos, block.NewChest(), nil)
		chest := tx.Block(pos).(block.Chest).Inventory(tx, pos)
		_ = chest.SetItem(0, item.NewStack(block.Dirt{}, 1))
		_ = chest.SetItem(1, item.NewStack(item.Stick{}, 1))
		stick := item.NewStack(item.Stick{}, 1)
		p.SetHeldItems(stick, item.Stack{})
		_ = p.Inventory().SetItem(5, item.NewStack(block.Dirt{}, 7))
		_ = p.Inventory().SetItem(14, item.NewStack(block.Dirt{}, 40))
		_ = p.Inventory().SetItem(15, item.NewStack(block.Cobblestone{}, 10))
		_ = p.Inventory().SetItem(16, item.NewStack(item.Stick{}, 3))
		if handwork.UseStickOnChest(s.Session, p, tx, pos, stick) {
			t.Fatal("a stick use without sneaking was handled")
		}
		p.StartSneaking()
		if !handwork.UseStickOnChest(s.Session, p, tx, pos, stick) {
			t.Fatal("a sneaking stick use on a chest was not handled")
		}
		if got := core.CountPlain(chest, block.Dirt{}); got != 41 {
			t.Errorf("chest holds %d dirt, want 41", got)
		}
		if got := core.CountPlain(chest, item.Stick{}); got != 4 {
			t.Errorf("chest holds %d sticks, want 4", got)
		}
		if got := core.CountPlain(chest, block.Cobblestone{}); got != 0 {
			t.Errorf("chest took %d cobblestone it did not hold", got)
		}
		if held, _ := p.HeldItems(); held.Count() != 1 {
			t.Errorf("held %v, want the stick", held)
		}
		if it, _ := p.Inventory().Item(5); it.Count() != 7 {
			t.Errorf("hotbar slot 5 holds %v, want the 7 dirt kept", it)
		}
	})
}

func TestThreeSneakingStickUsesSortChest(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		pos := cube.Pos{1, island.IslandY, 1}
		tx.SetBlock(pos, block.NewChest(), nil)
		chest := tx.Block(pos).(block.Chest).Inventory(tx, pos)
		_ = chest.SetItem(3, item.NewStack(block.Stone{}, 40))
		_ = chest.SetItem(8, item.NewStack(block.Dirt{}, 50))
		_ = chest.SetItem(12, item.NewStack(block.Stone{}, 40))
		_ = chest.SetItem(20, item.NewStack(block.Dirt{}, 5))
		stick := item.NewStack(item.Stick{}, 1)
		p.SetHeldItems(stick, item.Stack{})
		p.StartSneaking()
		use := func() {
			delete(handwork.LastStick.Of(s.Session), p.H())
			handwork.UseStickOnChest(s.Session, p, tx, pos, stick)
		}
		use()
		use()
		if it, _ := chest.Item(0); !it.Empty() {
			t.Fatalf("chest sorted after two uses: slot 0 holds %v", it)
		}
		use()
		want := []item.Stack{
			item.NewStack(block.Dirt{}, 55),
			item.NewStack(block.Stone{}, 64),
			item.NewStack(block.Stone{}, 16),
			{},
		}
		for slot, w := range want {
			if got, _ := chest.Item(slot); got.Count() != w.Count() || (!w.Empty() && !got.Comparable(w)) {
				t.Errorf("slot %d holds %v, want %v", slot, got, w)
			}
		}
	})
}
