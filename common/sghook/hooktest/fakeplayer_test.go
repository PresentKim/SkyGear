package hooktest

import (
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// TestViewFakePlayer: the fake player goes in the player list, is added at
// its place facing its way under its runtime ID, and leaves the list again;
// animating and removing it name that runtime ID.
func TestViewFakePlayer(t *testing.T) {
	cl := Join(t, server.Config{}, nil)
	const id = 1 << 40
	pos := mgl64.Vec3{0.5, 64, 0.5}
	<-cl.Server().World().Do(func(tx *world.Tx) {
		e, _ := cl.Handle().Entity(tx)
		p := e.(*player.Player)
		sghook.ViewFakePlayer(p, sghook.FakePlayer{RuntimeID: id, Pos: pos, Yaw: 0, Skin: p.Skin()})
		sghook.AnimateEntity(p, id, "animation.test")
		sghook.RemoveFakeEntity(p, id)
	}).Done()
	// one RemoveActor goes before the model (a model shown before), one
	// after the animation
	for range 2 {
		cl.Wait(func(pk packet.Packet) bool {
			r, ok := pk.(*packet.RemoveActor)
			return ok && r.EntityUniqueID == id
		})
	}
	var adds, removes int
	var added *packet.AddPlayer
	for _, pk := range cl.Packets() {
		switch pk := pk.(type) {
		case *packet.PlayerList:
			for _, en := range pk.Entries {
				if en.EntityUniqueID == id && en.ActionType == protocol.PlayerListActionAdd {
					adds++
				}
				if added != nil && en.UUID == added.UUID && en.ActionType == protocol.PlayerListActionRemove {
					removes++
				}
			}
		case *packet.AddPlayer:
			if pk.EntityRuntimeID == id {
				added = pk
			}
		case *packet.AnimateEntity:
			if len(pk.EntityRuntimeIDs) != 1 || pk.EntityRuntimeIDs[0] != id {
				t.Errorf("animation of %v, want %d", pk.EntityRuntimeIDs, id)
			}
		}
	}
	if added == nil || adds != 1 || removes != 1 {
		t.Fatalf("AddPlayer %v, list adds %d, removes %d", added, adds, removes)
	}
	if added.Position != [3]float32{0.5, 64, 0.5} || added.Yaw != 0 || added.HeadYaw != 0 {
		t.Fatalf("added at %v yaw %v/%v", added.Position, added.Yaw, added.HeadYaw)
	}
}
