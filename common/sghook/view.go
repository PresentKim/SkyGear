package sghook

// Client views Dragonfly has no API for: they only send packets to one
// player (camera, HUD, input locks, blocks only that player sees).

import (
	"sync"
	_ "unsafe" // go:linkname

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/player/title"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// cameraPresetFree is the index of the free camera preset sent on spawn.
const cameraPresetFree = 0

func init() {
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		// Sent once as the player spawns: send the camera presets custom
		// cameras use with it. Dragonfly sends none, and camera instructions
		// refer to presets by index. Movement in the free camera is relative
		// to the camera, so with a top-down camera the movement keys move
		// across the screen.
		if _, ok := pk.(*packet.AvailableActorIdentifiers); ok {
			_ = c.Conn.WritePacket(&packet.CameraPresets{Presets: []protocol.CameraPreset{{
				Name:          "minecraft:free",
				ControlScheme: protocol.Option(byte(packet.ControlSchemeCameraRelativeStrafe)),
			}}})
		}
		return false
	})
}

// write queues pk for p's client; players without one are skipped.
func write(p *player.Player, pk packet.Packet) {
	WritePacket(Session(p), pk)
}

// SetCamera puts p's camera at pos looking at the rotation (pitch, yaw in
// degrees), moving there over ease seconds. Movement keys then move
// relative to the camera.
func SetCamera(p *player.Player, pos mgl64.Vec3, pitch, yaw, ease float64) {
	set := protocol.CameraInstructionSet{
		Preset:   cameraPresetFree,
		Position: protocol.Option(vec32(pos)),
		Rotation: protocol.Option(mgl32.Vec2{float32(pitch), float32(yaw)}),
	}
	if ease > 0 {
		set.Ease = protocol.Option(protocol.CameraEase{Type: protocol.EasingTypeLinear, Duration: float32(ease)})
	}
	write(p, &packet.CameraInstruction{Set: protocol.Option(set)})
}

// selfRuntimeID is the runtime ID a client knows its own player by.
const selfRuntimeID = 1

// Animate plays the resource pack animation name once on p's own player for
// p's client only, over whatever the player model is doing (the way the
// script API's playAnimation does). The server's player does not change.
func Animate(p *player.Player, name string) { AnimateEntity(p, selfRuntimeID, name) }

// AnimateEntity is Animate on the entity p's client knows by runtimeID (a
// fake player of ViewFakePlayer).
func AnimateEntity(p *player.Player, runtimeID uint64, name string) {
	write(p, &packet.AnimateEntity{
		Animation:        name,
		NextState:        "default",
		StopCondition:    "query.any_animation_finished",
		Controller:       "__runtime_controller",
		BlendOutTime:     0.2,
		EntityRuntimeIDs: []uint64{runtimeID},
	})
}

// FakePlayer is a player model shown to one client only, in no world: it
// stands where it was put, turned the way it was put, with the skin given.
type FakePlayer struct {
	// RuntimeID is the ID the client knows it by. It must not be one the
	// session gives the world's entities (those count up from 1): a high
	// number is safe.
	RuntimeID uint64
	Pos       mgl64.Vec3
	// Yaw is the way it faces (degrees, 0 south).
	Yaw  float64
	Skin skin.Skin
}

// ViewFakePlayer shows f to p only: the skin goes in the player list for as
// long as the model is added, and out again, so the list shows no one new.
// A model of the same runtime ID shown before is removed first.
func ViewFakePlayer(p *player.Player, f FakePlayer) {
	if Session(p) == session.Nop {
		return
	}
	RemoveFakeEntity(p, f.RuntimeID)
	id := uuid.New()
	write(p, &packet.PlayerList{Entries: []protocol.PlayerListEntry{{
		ActionType:     protocol.PlayerListActionAdd,
		UUID:           id,
		EntityUniqueID: int64(f.RuntimeID),
		BuildPlatform:  int32(protocol.DeviceUnknown),
		Skin:           session_skinToProtocol(f.Skin),
	}}})
	meta := protocol.NewEntityMetadata()
	meta[protocol.EntityDataKeyWidth] = float32(0.6)
	meta[protocol.EntityDataKeyHeight] = float32(1.8)
	meta[protocol.EntityDataKeyScale] = float32(1)
	meta[protocol.EntityDataKeyName] = ""
	meta[protocol.EntityDataKeyAlwaysShowNameTag] = byte(0)
	write(p, &packet.AddPlayer{
		UUID:            id,
		EntityRuntimeID: f.RuntimeID,
		Position:        vec32(f.Pos),
		Yaw:             float32(f.Yaw),
		HeadYaw:         float32(f.Yaw),
		GameType:        packet.GameTypeAdventure,
		EntityMetadata:  meta,
		BuildPlatform:   int32(protocol.DeviceUnknown),
		AbilityData: protocol.AbilityData{
			EntityUniqueID: int64(f.RuntimeID),
			Layers:         []protocol.AbilityLayer{{Type: protocol.AbilityLayerTypeBase, Abilities: protocol.AbilityCount - 1}},
		},
	})
	write(p, &packet.PlayerList{Entries: []protocol.PlayerListEntry{{ActionType: protocol.PlayerListActionRemove, UUID: id}}})
}

// RemoveFakeEntity removes the entity p's client knows by runtimeID that
// only it was shown (ViewFakePlayer).
func RemoveFakeEntity(p *player.Player, runtimeID uint64) {
	write(p, &packet.RemoveActor{EntityUniqueID: int64(runtimeID)})
}

// ClearCamera returns p's camera to normal.
func ClearCamera(p *player.Player) {
	write(p, &packet.CameraInstruction{Clear: protocol.Option(true)})
}

// SetCameraFOV sets the field of view of p's camera in degrees.
func SetCameraFOV(p *player.Player, fov float64) {
	write(p, &packet.CameraInstruction{FieldOfView: protocol.Option(protocol.CameraInstructionFieldOfView{FieldOfView: float32(fov)})})
}

// ClearCameraFOV returns p's field of view to their own setting.
func ClearCameraFOV(p *player.Player) {
	write(p, &packet.CameraInstruction{FieldOfView: protocol.Option(protocol.CameraInstructionFieldOfView{Clear: true})})
}

// hudElements are all HUD elements of SetHud (packet.HudElement*).
var hudElements = []int32{
	packet.HudElementPaperDoll, packet.HudElementArmour, packet.HudElementToolTips,
	packet.HudElementTouchControls, packet.HudElementCrosshair, packet.HudElementHotBar,
	packet.HudElementHealth, packet.HudElementProgressBar, packet.HudElementHunger,
	packet.HudElementAirBubbles, packet.HudElementHorseHealth, packet.HudElementStatusEffects,
	packet.HudElementItemText,
}

// HideHUD hides every vanilla HUD element of p's client (hotbar, health,
// crosshair, ...), or shows them all again when hide is false. Resource
// pack HUD parts (the sidebar) are not vanilla elements and stay.
func HideHUD(p *player.Player, hide bool) {
	v := int32(packet.HudVisibilityReset)
	if hide {
		v = packet.HudVisibilityHide
	}
	write(p, &packet.SetHud{Elements: hudElements, Visibility: v})
}

// LockInputs locks p's client inputs in locks (packet.ClientInputLock*
// flags); 0 unlocks them all.
func LockInputs(p *player.Player, locks uint32) {
	write(p, &packet.UpdateClientInputLocks{Locks: locks})
}

// ForgetHiddenEntities makes every entity p hid with HideEntity visible
// again from then on, without spawning them now: p is about to leave the
// world where they were hidden (the new world shows its entities as its
// chunks arrive). It must run in p's transaction.
func ForgetHiddenEntities(p *player.Player) {
	s := Session(p)
	if s == session.Nop {
		return
	}
	mu := field[sync.RWMutex](s, "entityMutex")
	mu.Lock()
	clear(*field[map[uuid.UUID]struct{}](s, "hiddenEntities"))
	mu.Unlock()
}

// ViewFakeBlock shows b at pos to p only. The world is unchanged; show the
// real block (tx.Block) the same way to undo it.
func ViewFakeBlock(p *player.Player, pos cube.Pos, b world.Block) {
	s := Session(p)
	if s == session.Nop {
		return
	}
	blockPos := protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])}
	session_writePacket(s, &packet.UpdateBlock{
		Position:          blockPos,
		NewBlockRuntimeID: (*field[world.BlockRegistry](s, "br")).BlockRuntimeID(b),
		Flags:             packet.BlockUpdateNetwork,
	})
	if v, ok := b.(world.NBTer); ok {
		if data := v.EncodeNBT(); data != nil {
			data["x"], data["y"], data["z"] = int32(pos.X()), int32(pos.Y()), int32(pos.Z())
			session_writePacket(s, &packet.BlockActorData{Position: blockPos, NBTData: data})
		}
	}
}

// ViewFakeBlockLayer shows b at pos to p only on one of the block's layers
// (0 the block, 1 the liquid layer), with no block entity data. The world
// is unchanged; show the real layer the same way to undo it.
func ViewFakeBlockLayer(p *player.Player, pos cube.Pos, b world.Block, layer uint32) {
	s := Session(p)
	if s == session.Nop {
		return
	}
	session_writePacket(s, &packet.UpdateBlock{
		Position:          protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])},
		NewBlockRuntimeID: (*field[world.BlockRegistry](s, "br")).BlockRuntimeID(b),
		Flags:             packet.BlockUpdateNetwork,
		Layer:             layer,
	})
}

// ViewChunkCentre makes p's client keep and draw the chunks around pos
// instead of around the player until it next moves to another chunk, for
// chunks shown with ViewFakeChunk right after a teleport (they are dropped
// if outside the client's chunk centre).
func ViewChunkCentre(p *player.Player, pos mgl64.Vec3) {
	s := Session(p)
	if s == session.Nop {
		return
	}
	session_writePacket(s, &packet.NetworkChunkPublisherUpdate{
		Position: protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])},
		Radius:   uint32(*field[int32](s, "chunkRadius")) << 4,
	})
}

// SendTitle is p.SendTitle with the subtitle sent before the title: the
// client shows a subtitle with the next title, so one sent after its title
// (as Dragonfly does) comes a title late. If rawJSON, the title text is raw
// JSON text ({"rawtext":[{"text":"…"},{"translate":"tile.stone.name"}]})
// that the client translates in its own language.
func SendTitle(p *player.Player, t title.Title, rawJSON bool) {
	s := Session(p)
	if s == session.Nop {
		return
	}
	s.SetTitleDurations(t.FadeInDuration(), t.Duration(), t.FadeOutDuration())
	if t.Subtitle() != "" {
		s.SendSubtitle(t.Subtitle())
	}
	action := int32(packet.TitleActionSetTitle)
	if rawJSON {
		action = packet.TitleActionTitleTextObject
	}
	session_writePacket(s, &packet.SetTitle{ActionType: action, Text: t.Text()})
	if t.ActionText() != "" {
		s.SendActionBarMessage(t.ActionText())
	}
}

func vec32(v mgl64.Vec3) mgl32.Vec3 {
	return mgl32.Vec3{float32(v[0]), float32(v[1]), float32(v[2])}
}

//go:linkname session_skinToProtocol github.com/df-mc/dragonfly/server/session.skinToProtocol
func session_skinToProtocol(skin.Skin) protocol.Skin
