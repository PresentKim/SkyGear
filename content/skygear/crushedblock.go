package skygear

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

func init() {
	for _, k := range sgitem.CrushedKinds {
		world.RegisterBlock(sgitem.CrushedBlock{Kind: k})
		world.RegisterItem(sgitem.CrushedBlock{Kind: k})
	}
}
