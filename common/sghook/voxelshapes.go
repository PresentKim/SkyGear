package sghook

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func init() {
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		if _, ok := pk.(*packet.AvailableActorIdentifiers); ok {
			_ = c.Conn.WritePacket(basicVoxelShapes())
		}
		return false
	})
}

// basicVoxelShapes is the minimal voxel shape registry (minecraft:empty and
// minecraft:unit_cube) sent when the player spawns. Dragonfly sends none and
// 1.26 clients log "Shape registry is full" / "minecraft:unit_cube doesn't
// exist" for it.
func basicVoxelShapes() *packet.VoxelShapes {
	return &packet.VoxelShapes{
		Shapes: []protocol.VoxelShape{
			{Cells: protocol.VoxelCells{}, XCoordinates: []float32{0}, YCoordinates: []float32{0}, ZCoordinates: []float32{0}},
			{Cells: protocol.VoxelCells{XSize: 1, YSize: 1, ZSize: 1, Storage: []uint8{1}}, XCoordinates: []float32{0, 1}, YCoordinates: []float32{0, 1}, ZCoordinates: []float32{0, 1}},
		},
		NameMap: []protocol.VoxelShapeNameEntry{{Name: "minecraft:empty", ID: 0}, {Name: "minecraft:unit_cube", ID: 1}},
	}
}
