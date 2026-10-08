package session

// This file is part of the dragonfactory patch to Dragonfly (third_party/README.md).
// It adds container windows backed by an invisible, client-only entity
// instead of a block, like PocketMine's InvMenu actor menus. Combined with a
// resource pack that reads the size from the window title, the window can
// have any number of slots and opens anywhere.

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// EntityContainerIdentifier is the actor identifier of the invisible entity
// backing entity containers. It is announced in AvailableActorIdentifiers so
// clients accept it; no client entity definition is needed.
const EntityContainerIdentifier = "dragonfly:entity_container"

type entityContainer struct {
	runtimeID uint64
	onClose   func(tx *world.Tx)
}

// OpenEntityContainer opens inv in a container window backed by an
// invisible entity spawned at pos for this session only. title is the
// window title. onClose, if not nil, is called once when the window closes
// for any reason (client close, another container opening, disconnect).
func (s *Session) OpenEntityContainer(inv *inventory.Inventory, title string, pos mgl64.Vec3, onClose func(tx *world.Tx), tx *world.Tx) {
	s.closeCurrentContainer(tx, false)

	s.entityMutex.Lock()
	s.currentEntityRuntimeID++
	id := s.currentEntityRuntimeID
	s.entityMutex.Unlock()

	m := protocol.NewEntityMetadata()
	m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInvisible)
	m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagNoAI)
	m[protocol.EntityDataKeyName] = title
	m[protocol.EntityDataKeyContainerType] = byte(0xff) // ContainerTypeInventory (-1)
	m[protocol.EntityDataKeyContainerSize] = int32(inv.Size())
	s.writePacket(&packet.AddActor{
		EntityUniqueID:  int64(id),
		EntityRuntimeID: id,
		EntityType:      EntityContainerIdentifier,
		EntityMetadata:  m,
		Position:        vec64To32(pos),
	})

	blockPos := cube.PosFromVec3(pos)
	nextID := s.nextWindowID()
	s.containerOpened.Store(true)
	s.openedWindow.Store(inv)
	s.openedPos.Store(&blockPos)
	s.entityContainer.Store(&entityContainer{runtimeID: id, onClose: onClose})
	s.openedContainerID.Store(uint32(protocol.ContainerTypeContainer))
	s.writePacket(&packet.ContainerOpen{
		WindowID:                nextID,
		ContainerType:           protocol.ContainerTypeContainer,
		ContainerPosition:       protocol.BlockPos{int32(blockPos[0]), int32(blockPos[1]), int32(blockPos[2])},
		ContainerEntityUniqueID: int64(id),
	})
	s.sendInv(inv, uint32(nextID))
}

// CloseOpenedContainer closes the container window the session has open.
func (s *Session) CloseOpenedContainer(tx *world.Tx) {
	s.closeCurrentContainer(tx, false)
}

// closeEntityContainer removes the entity of an open entity container and
// runs its close callback. It reports if an entity container was open.
func (s *Session) closeEntityContainer(tx *world.Tx) bool {
	ec := s.entityContainer.Swap(nil)
	if ec == nil {
		return false
	}
	s.writePacket(&packet.RemoveActor{EntityUniqueID: int64(ec.runtimeID)})
	if ec.onClose != nil {
		ec.onClose(tx)
	}
	return true
}
