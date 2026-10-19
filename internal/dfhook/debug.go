package dfhook

// Experiments switched on in platform.toml [Experimental].

import (
	"log/slog"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var (
	// SendBasicVoxelShapes sends a minimal voxel shape registry
	// (minecraft:empty and minecraft:unit_cube) when the player spawns.
	// Dragonfly sends none and 1.26 clients log "Shape registry is full" /
	// "minecraft:unit_cube doesn't exist" for it. Whether this matters (e.g.
	// for breaking custom blocks) is unverified. Set it once at startup.
	SendBasicVoxelShapes bool
	// BlockActionDebug logs the block breaking actions clients send, at
	// INFO level. Set it once at startup.
	BlockActionDebug bool
)

func init() {
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		if _, ok := pk.(*packet.AvailableActorIdentifiers); ok && SendBasicVoxelShapes {
			_ = c.Conn.WritePacket(basicVoxelShapes())
		}
		return false
	})
	OnClientPacket(func(_ *Conn, pk packet.Packet) bool {
		if !BlockActionDebug {
			return false
		}
		switch pk := pk.(type) {
		case *packet.PlayerAction:
			logBlockAction(pk.ActionType, pk.BlockFace, pk.BlockPosition)
		case *packet.PlayerAuthInput:
			if actions, ok := pk.BlockActions.Value(); ok && pk.InputData.Load(packet.InputFlagPerformBlockActions) {
				for _, a := range actions {
					logBlockAction(a.Action, a.Face, a.BlockPos)
				}
			}
		}
		return false
	})
}

func basicVoxelShapes() *packet.VoxelShapes {
	return &packet.VoxelShapes{
		Shapes: []protocol.VoxelShape{
			{Cells: protocol.VoxelCells{}, XCoordinates: []float32{0}, YCoordinates: []float32{0}, ZCoordinates: []float32{0}},
			{Cells: protocol.VoxelCells{XSize: 1, YSize: 1, ZSize: 1, Storage: []uint8{1}}, XCoordinates: []float32{0, 1}, YCoordinates: []float32{0, 1}, ZCoordinates: []float32{0, 1}},
		},
		NameMap: []protocol.VoxelShapeNameEntry{{Name: "minecraft:empty", ID: 0}, {Name: "minecraft:unit_cube", ID: 1}},
	}
}

func logBlockAction(action, face int32, pos protocol.BlockPos) {
	switch action {
	case protocol.PlayerActionStartBreak, protocol.PlayerActionContinueDestroyBlock, protocol.PlayerActionAbortBreak,
		protocol.PlayerActionPredictDestroyBlock, protocol.PlayerActionStopBreak, protocol.PlayerActionCrackBreak:
		slog.Info("block action", "action", action, "pos", pos, "face", face)
	}
}
