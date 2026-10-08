package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func TestHarvesterHarvestsGrownCrops(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, KindHarvester, "")
	wheat, young, melon := machinePos.Add(cube.Pos{2, 0, 0}), machinePos.Add(cube.Pos{-2, 0, 1}), machinePos.Add(cube.Pos{0, 0, 3})
	s.world.Do(func(tx *world.Tx) {
		for _, p := range []cube.Pos{wheat, young} {
			tx.SetBlock(p.Side(cube.FaceDown), block.Farmland{Hydration: 7}, nil)
		}
		tx.SetBlock(wheat, block.WheatSeeds{}, nil)
		tx.SetBlock(young, block.WheatSeeds{}, nil)
		w := tx.Block(wheat).(block.WheatSeeds)
		w.Growth = 7
		tx.SetBlock(wheat, w, nil)
		tx.SetBlock(melon, block.Melon{}, nil)
	})
	tick(s, 2*(machineKinds[KindHarvester].interval+1))
	s.world.Do(func(tx *world.Tx) {
		if w, ok := tx.Block(wheat).(block.WheatSeeds); !ok || w.Growth != 0 {
			t.Fatalf("grown wheat not replanted: %#v", tx.Block(wheat))
		}
		if _, ok := tx.Block(young).(block.WheatSeeds); !ok {
			t.Fatal("young wheat was harvested")
		}
		if _, ok := tx.Block(melon).(block.Air); !ok {
			t.Fatal("melon not cut")
		}
	})
	if outputCount(st, item.Wheat{}) == 0 || outputCount(st, item.MelonSlice{}) == 0 {
		t.Fatalf("outputs: wheat %d melon %d", outputCount(st, item.Wheat{}), outputCount(st, item.MelonSlice{}))
	}
}

// TestHarvestedCropsAreYoung: dropped crops and seeds are new (growth 0),
// not copies of the grown crop that would be planted grown (Dragonfly
// patch).
func TestHarvestedCropsAreYoung(t *testing.T) {
	for _, b := range []world.Block{block.Carrot{}, block.Potato{}, block.WheatSeeds{}, block.BeetrootSeeds{}} {
		grown := b
		switch c := b.(type) {
		case block.Carrot:
			c.Growth = 7
			grown = c
		case block.Potato:
			c.Growth = 7
			grown = c
		case block.WheatSeeds:
			c.Growth = 7
			grown = c
		case block.BeetrootSeeds:
			c.Growth = 7
			grown = c
		}
		for range 20 {
			for _, it := range grown.(block.Breakable).BreakInfo().Drops(item.ToolNone{}, nil) {
				if c, ok := it.Item().(block.Crop); ok && c.GrowthStage() != 0 {
					t.Fatalf("%T drops %T with growth %d", b, it.Item(), c.GrowthStage())
				}
			}
		}
	}
}
