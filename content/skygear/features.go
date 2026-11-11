package skygear

import (
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/elevator"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/handwork"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/network"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wand"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Features is every feature of the game, the one place they are listed. The
// order is the layer order: settings and item entry edits apply in it, and
// each feature's Setup runs in it. The subscribers the root package holds
// (legacy_*.go) register themselves in init functions, not through Setup.
var Features = []*core.Feature{
	core.KernelFeature,
	achievement.Feature,
	progress.Feature,
	island.Feature,
	climate.Feature,
	network.Feature,
	tinker.Feature,
	core.AutofillFeature,
	machine.Feature,
	conduit.Feature,
	storage.Feature,
	farm.Feature,
	fairy.Feature,
	dimension.Feature,
	handwork.Feature,
	expand.Feature,
	copier.Feature,
	wand.Feature,
	netstore.Feature,
	elevator.Feature,
	look.Feature,
	screens.Feature,
	itemsFeature,
}

// itemsFeature is the root's own feature: the item entry edit that names an
// item by its translation key and clips the empty wooden bucket at liquids.
var itemsFeature = &core.Feature{
	Name: "items",
	ItemEntry: func(e protocol.ItemEntry) protocol.ItemEntry {
		return sgitem.ClipWoodenBucket(localItemName(e))
	},
}

// A process has one game, so the features are installed once, when the
// package is initialised: every program and test that uses the game has them.
func init() { core.Install(Features...) }
