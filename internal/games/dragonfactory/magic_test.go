package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// TestMagicPlantLifecycle: a seed planted on dirt makes a bud, it blooms when its
// time has come, a harvest gives the yield and sets it back to a bud, and it
// drops when its ground goes.
func TestMagicPlantLifecycle(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	ground := machinePos
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(ground, block.Dirt{}, nil)
		tx.SetBlock(ground.Side(cube.FaceUp), block.Air{}, nil)
		p.SetHeldItems(item.NewStack(PlantSeed{Seed: "ore_iron"}, 2), item.Stack{})
		ctx := &item.UseContext{}
		if !(PlantSeed{Seed: "ore_iron"}).UseOnBlock(ground, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) {
			t.Fatal("the seed was not planted")
		}
		at := ground.Side(cube.FaceUp)
		plant, ok := tx.Block(at).(MagicPlant)
		if !ok || plant.Kind != "ore_iron" || plant.Bloom {
			t.Fatalf("block %v", tx.Block(at))
		}
		rec := s.state.Plants[cropKey(at)]
		if rec == nil {
			t.Fatal("the plant is not recorded")
		}
		s.tickPlants(tx)
		if tx.Block(at).(MagicPlant).Bloom {
			t.Error("bloomed at once")
		}
		rec.ReadyAt = 0
		s.tickPlants(tx)
		if !tx.Block(at).(MagicPlant).Bloom {
			t.Fatal("did not bloom when its time came")
		}
		if drops := tx.Block(at).(MagicPlant).BreakInfo().Drops(nil, nil); len(drops) < 2 {
			t.Errorf("a bloom drops %d stacks when broken, want the seed and the yield", len(drops))
		}
		tx.Block(at).(MagicPlant).Activate(at, cube.FaceUp, tx, p, ctx)
		if got := tx.Block(at).(MagicPlant); got.Bloom {
			t.Error("a harvested plant is still in bloom")
		}
		if !s.advanced(advBloom) {
			t.Error("harvesting did not do the advancement")
		}
		// without ground it drops
		tx.SetBlock(ground, block.Air{}, nil)
		tx.Block(at).(MagicPlant).NeighbourUpdateTick(at, ground, tx)
		if _, still := tx.Block(at).(MagicPlant); still {
			t.Error("the plant stands in the air")
		}
	})
}

// TestMagicSeedsAndRecipes: the flower seed makes a random colour, every better seed
// has a recipe, and the researches unlock them in order.
func TestMagicSeedsAndRecipes(t *testing.T) {
	if len(plantKinds) != 16+7+6+7 {
		t.Fatalf("%d kinds", len(plantKinds))
	}
	have := map[string]bool{}
	for _, r := range recipe.Recipes() {
		for _, o := range r.Output() {
			if seed, ok := o.Item().(PlantSeed); ok {
				have[seed.Seed] = true
			}
		}
	}
	for _, sd := range plantSeeds {
		if !strings.HasPrefix(sd, "flower_") && !have[sd] {
			t.Errorf("no recipe makes the seed %s", sd)
		}
	}
	unlocked := map[string]string{}
	for _, r := range researchTree {
		for _, u := range r.Unlocks {
			unlocked[u] = r.ID
		}
	}
	for _, sd := range plantSeeds {
		if !strings.HasPrefix(sd, "flower_") && unlocked["dragonfactory:seed_"+sd] == "" {
			t.Errorf("no research unlocks the seed %s", sd)
		}
	}
	for _, k := range plantKinds {
		if len(k.yield) == 0 || k.grow <= 0 {
			t.Errorf("%s: incomplete", k.id)
		}
	}
}
