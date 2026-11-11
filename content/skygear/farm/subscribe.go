package farm

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// subscribe registers the farm's steps and its composter adapter.
func subscribe() {
	subscribeSneak()
	subscribeSecond()
	subscribeLoad()
	subscribeBlocks()
	subscribeUse()
	registerComposterIO()
	registerLook()
	registerKinds()
}

// registerComposterIO lets itemio reach a composter (a vanilla block) through
// ComposterIO, in the session of the world the block is in.
func registerComposterIO() {
	itemio.RegisterAdapter(func(tx *world.Tx, _ cube.Pos, b world.Block) (itemio.Inserter, itemio.Extractor) {
		if _, ok := b.(block.Composter); ok {
			if s := core.At(tx); s != nil {
				return ComposterIO{S: s}, ComposterIO{S: s}
			}
		}
		return nil, nil
	})
}

// Sneaking grows saplings nearby by chance.
func subscribeSneak() {
	core.Sneak.On(40, "farm.crouchGrow", func(e core.PlayerEvent) {
		CrouchGrow(e.S, e.P, e.Tx)
	})
}

// The crops, barrels, composters and plants grow every second.
func subscribeSecond() {
	core.Second.On(10, "farm.growCrops", func(e core.Tick) {
		GrowCrops(e.S, e.Tx)
	})
	core.Second.On(30, "farm.barrels", func(e core.Tick) {
		TickBarrels(e.S, e.Tx)
		TickComposters(e.S, e.Tx)
	})
	core.Second.On(90, "farm.plants", func(e core.Tick) {
		TickPlants(e.S, e.Tx)
	})
}

// The crops are loaded before the world transaction of the session start.
func subscribeLoad() {
	core.Load.On(10, "farm.loadCrops", func(e core.Tick) {
		LoadCrops(e.S)
	})
}

// Leaves pulled apart with a crook give more saplings, now and then a
// silkworm (silkworm leaves drop their string themselves); a broken composter
// and a broken barrel give back what they hold. A placed crop is planted.
func subscribeBlocks() {
	core.BlockBreak.On(50, "farm.break", func(e core.BlockBreakEvent) {
		if c, ok := sgitem.CrookOf(e.Held); ok {
			if l, ok := e.Block.(block.Leaves); ok {
				*e.Drops = CrookLeafDrops(l, c, e.S.Rand.Float64)
			}
		}
		switch b := e.Block.(type) {
		case block.Composter:
			BreakComposter(e.S, e.Pos, b, e.Drops)
		case Barrel:
			BreakBarrel(e.S, e.Pos, e.Drops)
		}
	})
	core.BlockPlace.On(50, "farm.plant", func(e core.BlockPlaceEvent) {
		if kind, _, ok := CropOf(e.Block); ok {
			achievement.Count(e.P, "crop_plant", 1)
			PlantCrop(e.S, e.Pos, kind)
		}
	})
}

// Using a composter or a crop with a free hand acts on it; sneaking uses the
// item as usual (placing blocks, bone meal). A crop shows its time or
// harvests, and places nothing.
func subscribeUse() {
	core.UseOnBlock.Handle(120, "farm.composter", func(e core.UseOnBlockEvent) bool {
		if !e.P.Sneaking() && UseComposter(e.S, e.P, e.Tx, e.Pos, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnBlock.Handle(150, "farm.crop", func(e core.UseOnBlockEvent) bool {
		if !e.P.Sneaking() && UseCrop(e.S, e.P, e.Tx, e.Pos) {
			e.Cancel()
			return true
		}
		return false
	})
}
