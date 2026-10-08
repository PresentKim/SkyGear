package platform

import (
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func newDrop(p *player.Player, s item.Stack) *world.EntityHandle {
	return entity.NewItem(world.EntitySpawnOpts{Position: p.Position()}, s)
}
