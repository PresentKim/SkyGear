package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ContainerCloseHandler handles the ContainerClose packet.
type ContainerCloseHandler struct{}

// Handle ...
func (h *ContainerCloseHandler) Handle(p packet.Packet, s *Session, tx *world.Tx, c Controllable) error {
	pk := p.(*packet.ContainerClose)

	if s.entityContainerOpening() && pk.WindowID != 0 {
		// dragonfactory patch: the client closes windows on its own while an
		// entity container is being opened (e.g. a form closing). Acknowledge
		// it, but keep the window: it is opened again until acknowledged.
		if pk.WindowID != 0xff {
			s.writePacket(&packet.ContainerClose{WindowID: pk.WindowID, ContainerType: pk.ContainerType})
		}
		return nil
	}
	c.MoveItemsToInventory()

	var containerType byte
	switch pk.WindowID {
	case 0:
		// Closing of the normal inventory.
		s.invOpened = false
	case byte(s.openedWindowID.Load()):
		containerType = byte(s.openedContainerID.Load())
		s.closeCurrentContainer(tx, true)
	case 0xff:
		// Sent when an inventory/container is opened at the same time as chat.
		s.invOpened = false
		if s.containerOpened.Load() {
			s.closeCurrentContainer(tx, false)
		}
		return nil
	default:
		containerType = pk.ContainerType
	}
	s.writePacket(&packet.ContainerClose{
		WindowID:      pk.WindowID,
		ContainerType: containerType,
	})
	return nil
}
