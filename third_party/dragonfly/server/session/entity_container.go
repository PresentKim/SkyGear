package session

// This file is part of the dragonfactory patch to Dragonfly (third_party/README.md).
// It adds container windows backed by an invisible, client-only entity
// instead of a block, like PocketMine's InvMenu actor menus. Combined with a
// resource pack that reads the size from the window title, the window can
// have any number of slots and opens anywhere.
//
// Opening follows Muqsit/InvMenu (PlayerWindowDispatcher), because the
// client often ignores a ContainerOpen sent right after spawning the entity,
// or while a form or another window is closing:
//
//  1. AddActor (riding the player, so it stays next to them), then a
//     NetworkStackLatency request.
//  2. When the client answers it, the entity exists client-side: send
//     ContainerOpen and the contents.
//  3. Until the client acknowledges the window with a PacketViolationWarning
//     (type -1, severity warning) caused by the excess opens, resend
//     ContainerClose + ContainerOpen every tick (at most entityOpenRetries).
//     Meanwhile, client ContainerClose packets are acknowledged but do not
//     close the window.

import (
	"math/rand/v2"
	"sync"
	"time"

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

const (
	// entityOpenRetries limits the ContainerOpen resends (one per tick).
	entityOpenRetries = 20
	// entityActorTimeout is how long to wait for the NetworkStackLatency
	// answer before opening anyway.
	entityActorTimeout = time.Second
	// rideEntityContainer links the entity to the player as a rider.
	rideEntityContainer = true
)

type entityContainerState int

const (
	ecWaitingActor entityContainerState = iota
	ecOpening
	ecOpen
)

type entityContainer struct {
	runtimeID uint64
	inv       *inventory.Inventory
	pos       cube.Pos
	onClose   func(tx *world.Tx)

	mu        sync.Mutex
	state     entityContainerState
	latencyTS int64
	windowID  byte
	stop      chan struct{}
}

// OpenEntityContainer opens inv in a container window backed by an
// invisible entity spawned at pos for this session only. title is the
// window title. onClose, if not nil, is called once when the window closes
// for any reason (client close, another container opening, disconnect).
// Opening inv while it is already open or opening does nothing.
func (s *Session) OpenEntityContainer(inv *inventory.Inventory, title string, pos mgl64.Vec3, onClose func(tx *world.Tx), tx *world.Tx) {
	if s == Nop {
		return
	}
	if ec := s.entityContainer.Load(); ec != nil && ec.inv == inv {
		return
	}
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
	var links []protocol.EntityLink
	if rideEntityContainer {
		links = []protocol.EntityLink{{RiddenEntityUniqueID: selfEntityRuntimeID, RiderEntityUniqueID: int64(id), Type: protocol.EntityLinkPassenger, Immediate: true}}
	}
	s.writePacket(&packet.AddActor{
		EntityUniqueID:  int64(id),
		EntityRuntimeID: id,
		EntityType:      EntityContainerIdentifier,
		EntityMetadata:  m,
		Position:        vec64To32(pos),
		EntityLinks:     links,
	})

	ec := &entityContainer{runtimeID: id, inv: inv, pos: cube.PosFromVec3(pos), onClose: onClose,
		latencyTS: rand.Int64N(1 << 20), stop: make(chan struct{})}
	s.entityContainer.Store(ec)
	s.writePacket(&packet.NetworkStackLatency{Timestamp: ec.latencyTS, NeedsResponse: true})
	go s.entityContainerTimeout(ec)
}

// entityContainerTimeout opens the window if the client never answers the
// NetworkStackLatency request.
func (s *Session) entityContainerTimeout(ec *entityContainer) {
	select {
	case <-ec.stop:
	case <-s.closeBackground:
	case <-time.After(entityActorTimeout):
		s.openEntityWindow(ec)
	}
}

// handleEntityContainerLatency handles a NetworkStackLatency answer. The
// client answers with the timestamp multiplied by 1, 1000 or 1000000
// depending on the platform.
func (s *Session) handleEntityContainerLatency(ts int64) {
	ec := s.entityContainer.Load()
	if ec == nil {
		return
	}
	if t := ec.latencyTS; ts == t || ts == t*1000 || ts == t*1000000 {
		s.openEntityWindow(ec)
	}
}

// openEntityWindow sends the ContainerOpen of a waiting entity container
// and starts resending it until the client acknowledges it.
func (s *Session) openEntityWindow(ec *entityContainer) {
	ec.mu.Lock()
	if ec.state != ecWaitingActor || s.entityContainer.Load() != ec {
		ec.mu.Unlock()
		return
	}
	ec.state = ecOpening
	ec.windowID = s.nextWindowID()
	ec.mu.Unlock()

	s.containerOpened.Store(true)
	s.openedWindow.Store(ec.inv)
	s.openedPos.Store(&ec.pos)
	s.openedContainerID.Store(uint32(protocol.ContainerTypeContainer))
	s.writePacket(ec.openPacket())
	s.sendInv(ec.inv, uint32(ec.windowID))
	go s.retryEntityWindow(ec)
}

func (ec *entityContainer) openPacket() *packet.ContainerOpen {
	return &packet.ContainerOpen{
		WindowID:                ec.windowID,
		ContainerType:           protocol.ContainerTypeContainer,
		ContainerPosition:       protocol.BlockPos{int32(ec.pos[0]), int32(ec.pos[1]), int32(ec.pos[2])},
		ContainerEntityUniqueID: int64(ec.runtimeID),
	}
}

// retryEntityWindow resends the window every tick until it is acknowledged.
func (s *Session) retryEntityWindow(ec *entityContainer) {
	t := time.NewTicker(time.Second / 20)
	defer t.Stop()
	for range entityOpenRetries {
		select {
		case <-ec.stop:
			return
		case <-s.closeBackground:
			return
		case <-t.C:
		}
		ec.mu.Lock()
		opening := ec.state == ecOpening && s.entityContainer.Load() == ec
		ec.mu.Unlock()
		if !opening {
			return
		}
		s.writePacket(&packet.ContainerClose{WindowID: ec.windowID, ContainerType: protocol.ContainerTypeContainer})
		s.writePacket(ec.openPacket())
	}
	// Never acknowledged: keep the window as it is.
	s.ackEntityWindow(ec)
}

// handleEntityContainerViolation handles a PacketViolationWarning, which the
// client sends for an excess ContainerOpen once the window is open.
func (s *Session) handleEntityContainerViolation(pk *packet.PacketViolationWarning) {
	if pk.Type != -1 || pk.Severity != packet.ViolationSeverityWarning {
		return
	}
	if ec := s.entityContainer.Load(); ec != nil {
		s.ackEntityWindow(ec)
	}
}

// ackEntityWindow marks the window open and sends its contents again.
func (s *Session) ackEntityWindow(ec *entityContainer) {
	ec.mu.Lock()
	if ec.state != ecOpening {
		ec.mu.Unlock()
		return
	}
	ec.state = ecOpen
	ec.mu.Unlock()
	if s.entityContainer.Load() == ec {
		s.sendInv(ec.inv, uint32(ec.windowID))
	}
}

// entityContainerOpening reports whether an entity container is still
// being opened, during which client ContainerClose packets are ignored.
func (s *Session) entityContainerOpening() bool {
	ec := s.entityContainer.Load()
	if ec == nil {
		return false
	}
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return ec.state != ecOpen
}

// CloseOpenedContainer closes the container window the session has open.
func (s *Session) CloseOpenedContainer(tx *world.Tx) {
	s.closeCurrentContainer(tx, false)
}

// closeEntityContainer removes the entity of an open (or opening) entity
// container and runs its close callback. It reports if one was open.
func (s *Session) closeEntityContainer(tx *world.Tx) bool {
	ec := s.entityContainer.Swap(nil)
	if ec == nil {
		return false
	}
	close(ec.stop)
	s.writePacket(&packet.RemoveActor{EntityUniqueID: int64(ec.runtimeID)})
	if ec.onClose != nil {
		ec.onClose(tx)
	}
	return true
}

// NetworkStackLatencyHandler handles NetworkStackLatency answers for
// entity containers (dragonfactory patch).
type NetworkStackLatencyHandler struct{}

func (NetworkStackLatencyHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, _ Controllable) error {
	s.handleEntityContainerLatency(p.(*packet.NetworkStackLatency).Timestamp)
	return nil
}

// PacketViolationWarningHandler handles the client's violation warnings
// (dragonfactory patch): they acknowledge entity container windows.
type PacketViolationWarningHandler struct{}

func (PacketViolationWarningHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, _ Controllable) error {
	s.handleEntityContainerViolation(p.(*packet.PacketViolationWarning))
	return nil
}
