package skygear

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
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
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
		p.SetHeldItems(item.NewStack(farm.PlantSeed{Seed: "ore_iron"}, 2), item.Stack{})
		ctx := &item.UseContext{}
		if !(farm.PlantSeed{Seed: "ore_iron"}).UseOnBlock(ground, cube.FaceUp, mgl64.Vec3{}, tx, p, ctx) {
			t.Fatal("the seed was not planted")
		}
		at := ground.Side(cube.FaceUp)
		plant, ok := tx.Block(at).(farm.MagicPlant)
		if !ok || plant.Kind != "ore_iron" || plant.Bloom {
			t.Fatalf("block %v", tx.Block(at))
		}
		rec := farm.StateOf(s.Session).Plants[farm.CropKey(at)]
		if rec == nil {
			t.Fatal("the plant is not recorded")
		}
		farm.TickPlants(s.Session, tx)
		if tx.Block(at).(farm.MagicPlant).Bloom {
			t.Error("bloomed at once")
		}
		if info := look.Info(s.Session, p, tx, at, tx.Block(at)); info.Item == nil || len(info.Lines) != 1 || !strings.Contains(info.Lines[0], "남은 시간") {
			t.Errorf("the panel of a bud: %+v", info)
		}
		rec.ReadyAt = 0
		farm.TickPlants(s.Session, tx)
		if !tx.Block(at).(farm.MagicPlant).Bloom {
			t.Fatal("did not bloom when its time came")
		}
		if info := look.Info(s.Session, p, tx, at, tx.Block(at)); len(info.Lines) < 2 || !strings.Contains(info.Lines[0], "다 자람") {
			t.Errorf("the panel of a bloom: %q", info.Lines)
		}
		if drops := tx.Block(at).(farm.MagicPlant).BreakInfo().Drops(nil, nil); len(drops) < 2 {
			t.Errorf("a bloom drops %d stacks when broken, want the seed and the yield", len(drops))
		}
		tx.Block(at).(farm.MagicPlant).Activate(at, cube.FaceUp, tx, p, ctx)
		if got := tx.Block(at).(farm.MagicPlant); got.Bloom {
			t.Error("a harvested plant is still in bloom")
		}
		if !progress.Advanced(s.Session, advancement.Bloom) {
			t.Error("harvesting did not do the advancement")
		}
		// without ground it drops
		tx.SetBlock(ground, block.Air{}, nil)
		tx.Block(at).(farm.MagicPlant).NeighbourUpdateTick(at, ground, tx)
		if _, still := tx.Block(at).(farm.MagicPlant); still {
			t.Error("the plant stands in the air")
		}
	})
}

// TestMagicSeedsAndRecipes: the flower seed makes a random colour, every better seed
// has a recipe, and the researches unlock them in order.
func TestMagicSeedsAndRecipes(t *testing.T) {
	if len(farm.PlantKinds) != 16+7+6+7 {
		t.Fatalf("%d kinds", len(farm.PlantKinds))
	}
	have := map[string]bool{}
	for _, r := range recipe.Recipes() {
		for _, o := range r.Output() {
			if seed, ok := o.Item().(farm.PlantSeed); ok {
				have[seed.Seed] = true
			}
		}
	}
	for _, sd := range farm.LivePlantSeeds() {
		if !strings.HasPrefix(sd, "flower_") && !have[sd] {
			t.Errorf("no recipe makes the seed %s", sd)
		}
	}
	unlocked := map[string]string{}
	for _, r := range progress.Tree {
		for _, u := range r.Unlocks {
			unlocked[u] = r.ID
		}
	}
	for _, sd := range farm.LivePlantSeeds() {
		if !strings.HasPrefix(sd, "flower_") && unlocked["skygear:seed_"+sd] == "" {
			t.Errorf("no research unlocks the seed %s", sd)
		}
	}
	for _, k := range farm.LivePlantKinds() {
		if len(k.Yield) == 0 || k.Grow <= 0 {
			t.Errorf("%s: incomplete", k.ID)
		}
	}
}

// TestHarvesterPicksBloomingPlants: the auto harvester picks a blooming
// magic plant into its output and leaves a bud; a bud is left alone.
func TestHarvesterPicksBloomingPlants(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	st := placeTestMachine(s, machinePos, machine.KindHarvester)
	bloom, bud := machinePos.Add(cube.Pos{2, 0, 0}), machinePos.Add(cube.Pos{-2, 0, 0})
	s.World.Do(func(tx *world.Tx) {
		for _, at := range []cube.Pos{bloom, bud} {
			tx.SetBlock(at.Side(cube.FaceDown), block.Dirt{}, nil)
		}
		tx.SetBlock(bloom, farm.MagicPlant{Kind: "ore_iron", Bloom: true}, nil)
		tx.SetBlock(bud, farm.MagicPlant{Kind: "ore_iron"}, nil)
	})
	tick(s, 2*(machine.Kinds[machine.KindHarvester].Interval+1))
	s.World.Do(func(tx *world.Tx) {
		if mp, ok := tx.Block(bloom).(farm.MagicPlant); !ok || mp.Bloom {
			t.Fatalf("the bloom was not picked: %#v", tx.Block(bloom))
		}
	})
	if machine.OutputCount(st, sgitem.OreChunk{Ore: sgitem.OreIron}) == 0 && machine.OutputCount(st, farm.PlantSeed{Seed: "ore_iron"}) == 0 {
		t.Error("the harvester has no yield of the plant")
	}
}

// TestRetiredEssenceBlooms: an essence bloom of an old island turns into a
// redstone bloom as it was, its seeds into redstone bloom seeds; no recipe
// makes essence bloom seeds, flower seeds come from essences and fairy dust,
// and no sieve finds monster essences.
func TestRetiredEssenceBlooms(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	at := machinePos.Side(cube.FaceUp)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		tx.SetBlock(machinePos, block.Dirt{}, nil)
		tx.SetBlock(at, farm.MagicPlant{Kind: "ess_zombie", Bloom: true}, nil)
		farm.StateOf(s.Session).Plants = map[string]*farm.PlantRecord{farm.CropKey(at): {Kind: "ess_zombie"}}
		farm.TickPlants(s.Session, tx)
		if b, ok := tx.Block(at).(farm.MagicPlant); !ok || b.Kind != "gem_redstone" || !b.Bloom {
			t.Errorf("the essence bloom became %v", tx.Block(at))
		}
		if k := farm.StateOf(s.Session).Plants[farm.CropKey(at)].Kind; k != "gem_redstone" {
			t.Errorf("its record says %s", k)
		}
		_, _ = p.Inventory().AddItem(item.NewStack(farm.PlantSeed{Seed: "ess_blaze"}, 3))
		farm.RetireSeeds(p)
		if n := core.CountPlain(p.Inventory(), farm.PlantSeed{Seed: "gem_redstone"}); n != 3 {
			t.Errorf("%d redstone bloom seeds for 3 essence bloom seeds", n)
		}
	})
	flower := false
	for _, r := range recipe.Recipes() {
		for _, o := range r.Output() {
			seed, ok := o.Item().(farm.PlantSeed)
			if !ok {
				continue
			}
			if strings.HasPrefix(seed.Seed, "ess_") {
				t.Errorf("a recipe makes %s", seed.Seed)
			}
			if seed.Seed == "flower_red" {
				for _, in := range r.Input() {
					if st, ok := in.(item.Stack); ok {
						if _, ess := st.Item().(sgitem.Essence); ess {
							flower = true
						}
					}
				}
			}
		}
	}
	if !flower {
		t.Error("no recipe makes a red flower seed from essences")
	}
	for tier, table := range sievetable.Tables {
		for in, drops := range table {
			for _, d := range drops {
				if _, ok := d.Item.Item().(sgitem.Essence); ok {
					t.Errorf("the %s mesh finds an essence in %s", tier, in)
				}
			}
		}
	}
}
