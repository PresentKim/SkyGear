package sghook

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// TestFallingBlockVariant: a falling block with no block gets air, one
// with a block keeps it, and other entity types and packets are left alone.
func TestFallingBlockVariant(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	f := &fakeConn{}
	c := conn{&Conn{Conn: f}}
	air := int32(world.DefaultBlockRegistry.BlockRuntimeID(block.Air{}))
	send := func(typ string, md map[uint32]any) *packet.AddActor {
		a := &packet.AddActor{EntityType: typ, EntityMetadata: md}
		_ = c.WritePacket(a)
		return a
	}

	if a := send("minecraft:falling_block", map[uint32]any{protocol.EntityDataKeyName: "x"}); a.EntityMetadata[protocol.EntityDataKeyVariant] != air {
		t.Errorf("no variant: got %v, want air %d", a.EntityMetadata[protocol.EntityDataKeyVariant], air)
	}
	if a := send("minecraft:falling_block", nil); a.EntityMetadata[protocol.EntityDataKeyVariant] != air {
		t.Errorf("nil metadata: got %v, want air %d", a.EntityMetadata[protocol.EntityDataKeyVariant], air)
	}
	if a := send("minecraft:falling_block", map[uint32]any{protocol.EntityDataKeyVariant: int32(7)}); a.EntityMetadata[protocol.EntityDataKeyVariant] != int32(7) {
		t.Errorf("variant 7: got %v", a.EntityMetadata[protocol.EntityDataKeyVariant])
	}
	for _, typ := range []string{"minecraft:armor_stand", "skygear:hunt_gem", ""} {
		if a := send(typ, map[uint32]any{}); len(a.EntityMetadata) != 0 {
			t.Errorf("%q got metadata %v", typ, a.EntityMetadata)
		}
	}
	if len(f.out) != 6 {
		t.Errorf("%d packets reached the client, want all 6", len(f.out))
	}
}
