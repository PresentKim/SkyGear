// Package network holds the networks of a save that blocks share: the
// energy that conduits carry between generators and machines, the items of
// the item conduits and the fluids of the fluid conduits (common/energy,
// common/itemnet, common/fluidnet). Each save has one of each, made on first
// use.
package network

import (
	"github.com/presentkim/skygear/common/energy"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/core"
)

var (
	energyGrid = core.NewPart(energy.NewGrid)
	itemGrid   = core.NewPart(itemnet.NewGrid)
	fluidGrid  = core.NewPart(fluidnet.NewGrid)
)

// Energy shares out the energy of the save's energy conduits.
func Energy(s *core.Session) *energy.Grid { return energyGrid.Of(s) }

// Items moves items over the save's item conduits.
func Items(s *core.Session) *itemnet.Grid { return itemGrid.Of(s) }

// Fluids moves fluids over the save's fluid conduits.
func Fluids(s *core.Session) *fluidnet.Grid { return fluidGrid.Of(s) }
