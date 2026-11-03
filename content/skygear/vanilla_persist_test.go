package skygear

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/content/skygear/island"
)

// vanillaContainers are the Dragonfly containers players can craft and use.
var vanillaContainers = []struct {
	name string
	new  func() world.Block
}{
	{"chest", func() world.Block { return block.NewChest() }},
	{"barrel", func() world.Block { return block.NewBarrel() }},
	{"hopper", func() world.Block { return block.NewHopper() }},
	{"furnace", func() world.Block { return block.NewFurnace(cube.North) }},
	{"blast_furnace", func() world.Block { return block.NewBlastFurnace(cube.North) }},
	{"smoker", func() world.Block { return block.NewSmoker(cube.North) }},
}

func vanillaInv(tx *world.Tx, pos cube.Pos) *inventory.Inventory {
	return tx.Block(pos).(block.Container).Inventory(tx, pos)
}

// TestVanillaContainerInvSurvivesReload: items put in a vanilla container,
// by a player (its inventory) or by a hopper or conduit (itemio), in a
// chunk where nothing else changes, are saved without any block update.
func TestVanillaContainerInvSurvivesReload(t *testing.T) {
	// A chunk away from the island, where nothing else changes.
	pos := cube.Pos{100, island.IslandY, 100}
	paths := []struct {
		name string
		put  func(tx *world.Tx) int
	}{
		{"player", func(tx *world.Tx) int {
			_ = vanillaInv(tx, pos).SetItem(0, item.NewStack(item.Wheat{}, 5))
			return 5
		}},
		{"itemio", func(tx *world.Tx) int {
			return itemio.Insert(tx, pos, cube.FaceUp, item.NewStack(item.Wheat{}, 5))
		}},
	}
	for _, c := range vanillaContainers {
		for _, p := range paths {
			t.Run(c.name+"/"+p.name, func(t *testing.T) {
				dir := t.TempDir()
				s, closeAll := openPersisted(t, dir)
				s.World.Do(func(tx *world.Tx) { tx.SetBlock(pos, c.new(), nil) })
				closeAll()

				s, closeAll = openPersisted(t, dir)
				var want int
				s.World.Do(func(tx *world.Tx) { want = p.put(tx) })
				if want == 0 {
					t.Fatal("nothing was put in")
				}
				closeAll()

				s, _ = openPersisted(t, dir)
				s.World.Do(func(tx *world.Tx) {
					got, _ := vanillaInv(tx, pos).Item(0)
					if got.Count() != want {
						t.Errorf("slot 0 after reload = %d, want %d", got.Count(), want)
					}
				})
			})
		}
	}
}

// TestVanillaSmeltingSurvivesReload: what a lit furnace smelts after a
// reload, in a chunk where nothing else changes, is saved without any block
// update.
func TestVanillaSmeltingSurvivesReload(t *testing.T) {
	// A chunk the test loader loads and ticks, away from the island.
	pos := cube.Pos{-20, island.IslandY, 4}
	for i, c := range vanillaContainers[3:] {
		// The smoker cooks only food.
		input := []world.Item{item.RawIron{}, item.RawIron{}, item.Beef{}}[i]
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			s, closeAll := openPersisted(t, dir)
			s.World.Do(func(tx *world.Tx) {
				tx.SetBlock(pos, c.new(), nil)
				inv := vanillaInv(tx, pos)
				_ = inv.SetItem(0, item.NewStack(input, 64))
				_ = inv.SetItem(1, item.NewStack(block.Coal{}, 1))
			})
			tick(s, 5) // lights it: a block update
			closeAll()

			s, closeAll = openPersisted(t, dir)
			tick(s, 600)
			// Read without Inventory, which marks the column modified.
			var want time.Duration
			s.World.Do(func(tx *world.Tx) { want = smeltLeft(tx, pos) })
			closeAll()

			s, _ = openPersisted(t, dir)
			s.World.Do(func(tx *world.Tx) {
				if got := smeltLeft(tx, pos); got != want {
					t.Errorf("fuel left after reload = %v, want %v", got, want)
				}
				// What it smelted after the reload.
				if res, _ := vanillaInv(tx, pos).Item(2); res.Empty() {
					t.Error("no output after reload")
				}
			})
		})
	}
}

// smeltLeft is the fuel time left in the furnace at pos.
func smeltLeft(tx *world.Tx, pos cube.Pos) time.Duration {
	left, _, _ := tx.Block(pos).(interface {
		Durations() (time.Duration, time.Duration, time.Duration)
	}).Durations()
	return left
}
