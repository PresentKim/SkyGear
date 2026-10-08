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

	if s.entityContainerOpen() {
		s.ecDebug("client close", "window", pk.WindowID, "type", pk.ContainerType, "opening", s.entityContainerOpening())
		// dragonfactory patch: the client sends a close of window 0xff when a
		// window opens while it is busy (e.g. a form closing); it must not
		// close an entity container. While one is still being opened with
		// retries, closes of its window are acknowledged but ignored too.
		if pk.WindowID == 0xff {
			return nil
		}
		if pk.WindowID != 0 && s.entityContainerOpening() {
			s.writePacket(&packet.ContainerClose{WindowID: pk.WindowID, ContainerType: pk.ContainerType})
			return nil
		}
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
