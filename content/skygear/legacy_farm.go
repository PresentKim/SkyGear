package skygear

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Sneaking grows saplings nearby by chance.
func init() {
	core.Sneak.On(40, "farm.crouchGrow", func(e core.PlayerEvent) {
		farm.CrouchGrow(e.S, e.P, e.Tx)
	})
}

// The crops, barrels, composters and plants grow every second.
func init() {
	core.Second.On(10, "farm.growCrops", func(e core.Tick) {
		farm.GrowCrops(e.S, e.Tx)
	})
	core.Second.On(30, "farm.barrels", func(e core.Tick) {
		farm.TickBarrels(e.S, e.Tx)
		farm.TickComposters(e.S, e.Tx)
	})
	core.Second.On(90, "farm.plants", func(e core.Tick) {
		farm.TickPlants(e.S, e.Tx)
	})
}

// The crops are loaded before the world transaction of the session start.
func init() {
	core.Load.On(10, "farm.loadCrops", func(e core.Tick) {
		farm.LoadCrops(e.S)
	})
}

// Leaves pulled apart with a crook give more saplings, now and then a
// silkworm (silkworm leaves drop their string themselves); a broken composter
// and a broken barrel give back what they hold. A placed crop is planted.
func init() {
	core.BlockBreak.On(50, "farm.break", func(e core.BlockBreakEvent) {
		if c, ok := sgitem.CrookOf(e.Held); ok {
			if l, ok := e.Block.(block.Leaves); ok {
				*e.Drops = farm.CrookLeafDrops(l, c, e.S.Rand.Float64)
			}
		}
		switch b := e.Block.(type) {
		case block.Composter:
			farm.BreakComposter(e.S, e.Pos, b, e.Drops)
		case farm.Barrel:
			farm.BreakBarrel(e.S, e.Pos, e.Drops)
		}
	})
	core.BlockPlace.On(50, "farm.plant", func(e core.BlockPlaceEvent) {
		if kind, _, ok := farm.CropOf(e.Block); ok {
			achievement.Count(e.P, "crop_plant", 1)
			farm.PlantCrop(e.S, e.Pos, kind)
		}
	})
}

// Using a composter or a crop with a free hand acts on it; sneaking uses the
// item as usual (placing blocks, bone meal). A crop shows its time or
// harvests, and places nothing.
func init() {
	core.UseOnBlock.Handle(120, "farm.composter", func(e core.UseOnBlockEvent) bool {
		if !e.P.Sneaking() && farm.UseComposter(e.S, e.P, e.Tx, e.Pos, e.Held) {
			e.Cancel()
			return true
		}
		return false
	})
	core.UseOnBlock.Handle(150, "farm.crop", func(e core.UseOnBlockEvent) bool {
		if !e.P.Sneaking() && farm.UseCrop(e.S, e.P, e.Tx, e.Pos) {
			e.Cancel()
			return true
		}
		return false
	})
}
