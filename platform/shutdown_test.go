package platform

import (
	"sync/atomic"
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook/hooktest"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type quitCounter struct {
	player.NopHandler
	n *atomic.Int32
}

func (h quitCounter) HandleQuit(*player.Player) { h.n.Add(1) }

// TestShutdownKick: closing the server sends the joined client exactly one
// Disconnect with the shutdown text, and the player's quit handler runs once.
func TestShutdownKick(t *testing.T) {
	cl := hooktest.Join(t, server.Config{}, nil)
	var quits atomic.Int32
	h := cl.Handle()
	<-cl.Server().World().Do(func(tx *world.Tx) {
		e, ok := h.Entity(tx)
		if !ok {
			t.Error("player not in the world")
			return
		}
		e.(*player.Player).Handle(quitCounter{n: &quits})
	}).Done()
	shutdown(cl.Server())

	cl.Wait(func(p packet.Packet) bool { _, ok := p.(*packet.Disconnect); return ok })
	var kicks []*packet.Disconnect
	for _, p := range cl.Packets() {
		if d, ok := p.(*packet.Disconnect); ok {
			kicks = append(kicks, d)
		}
	}
	if len(kicks) != 1 {
		t.Fatalf("got %d Disconnect packets, want 1", len(kicks))
	}
	if kicks[0].Message != "bye" || kicks[0].HideDisconnectionScreen {
		t.Fatalf("Disconnect %+v", kicks[0])
	}
	if n := quits.Load(); n != 1 {
		t.Fatalf("quit handler ran %d times, want 1", n)
	}
}

func shutdown(srv *server.Server) {
	DisconnectAll(srv.Players(nil), func(*player.Player) string { return "bye" })
	_ = srv.Close()
}
