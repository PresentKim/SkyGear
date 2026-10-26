package dfitem

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
)

// CrushRecipes maps an input item key to its output.
var CrushRecipes = map[string]world.Item{
	naming.ItemKey(block.Stone{}):       block.Cobblestone{},
	naming.ItemKey(block.Cobblestone{}): block.Gravel{},
	naming.ItemKey(block.Gravel{}):      block.Sand{},
	// Sand crushes into dust, red sand too.
	naming.ItemKey(block.Sand{}):          Dust{},
	naming.ItemKey(block.Sand{Red: true}): Dust{},
	// The stones of the deep, the Nether and the End (Ex Deorum, ATM10: To
	// the Sky).
	naming.ItemKey(block.Deepslate{}):                               CrushedBlock{Kind: CrushedDeepslate},
	naming.ItemKey(block.Deepslate{Type: block.CobbledDeepslate()}): CrushedBlock{Kind: CrushedDeepslate},
	naming.ItemKey(block.Netherrack{}):                              CrushedBlock{Kind: CrushedNetherrack},
	naming.ItemKey(CrushedBlock{Kind: CrushedNetherrack}):           block.Sand{Red: true},
	naming.ItemKey(block.Blackstone{}):                              CrushedBlock{Kind: CrushedBlackstone},
	naming.ItemKey(block.EndStone{}):                                CrushedBlock{Kind: CrushedEndStone},
}
