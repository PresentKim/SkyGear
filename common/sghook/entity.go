package sghook

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// fallingBlockType is the network type of a falling block entity.
const fallingBlockType = "minecraft:falling_block"

func init() {
	// A custom entity type sent as minecraft:falling_block (its
	// NetworkEncodeEntity) carries no block, and the client draws a falling
	// block without one as a purple cube. Dragonfly sets the variant for its
	// own falling blocks and text only, so give the others the air block: a
	// falling block entity then shows nothing but its name tag. The packet
	// is built for this session alone, so it is changed in place. The hook
	// keeps no state.
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		a, ok := pk.(*packet.AddActor)
		if !ok || a.EntityType != fallingBlockType {
			return false
		}
		if _, has := a.EntityMetadata[protocol.EntityDataKeyVariant]; has {
			return false
		}
		var br world.BlockRegistry = world.DefaultBlockRegistry
		if s := c.Session(); s != nil {
			br = *field[world.BlockRegistry](s, "br")
		}
		if a.EntityMetadata == nil {
			a.EntityMetadata = map[uint32]any{}
		}
		a.EntityMetadata[protocol.EntityDataKeyVariant] = int32(br.BlockRuntimeID(block.Air{}))
		return false
	})
}
