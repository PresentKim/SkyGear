package dimension

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/sghook/hooktest"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// popupClient joins a client to a server that knows the mark entity types
// and registers a core session with a run, so marks tick the way a hunt
// ticks them.
func popupClient(t *testing.T) (*hooktest.Client, *Run) {
	t.Helper()
	conf := server.Config{Entities: entity.DefaultRegistry.Config().New(append(entity.DefaultRegistry.Types(), GemType, PopupType))}
	c := hooktest.Join(t, conf, nil)
	s := core.New(core.Config{World: c.Server().World()})
	core.Register(s)
	t.Cleanup(func() { core.Unregister(s) })
	r := &Run{Alive: map[*world.EntityHandle]bool{}, sess: s}
	runPart.Of(s).Run = r
	return c, r
}

// do runs f in a transaction of the server's world and waits for it.
func do(t *testing.T, c *hooktest.Client, f func(tx *world.Tx, p *player.Player)) {
	t.Helper()
	task := c.Server().World().Do(func(tx *world.Tx) {
		e, ok := c.Handle().Entity(tx)
		if !ok {
			t.Error("player not in the world")
			return
		}
		f(tx, e.(*player.Player))
	})
	if err := task.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// closeMark closes the mark of h.
func closeMark(h *world.EntityHandle) func(*world.Tx, *player.Player) {
	return func(tx *world.Tx, _ *player.Player) {
		if e, ok := h.Entity(tx); ok {
			_ = e.(*mark).Close()
		}
	}
}

func isAddActor(typ string) func(packet.Packet) bool {
	return func(pk packet.Packet) bool {
		a, ok := pk.(*packet.AddActor)
		return ok && a.EntityType == typ
	}
}

// norm returns a copy of pk of the entity (unique and runtime ID the same)
// with that ID zeroed, and the ID; ok is false for a packet about no
// entity. Runtime IDs differ from process to process, nothing else of a
// packet is changed.
func norm(t *testing.T, pk packet.Packet) (id uint64, out packet.Packet, ok bool) {
	t.Helper()
	switch p := pk.(type) {
	case *packet.AddActor:
		if uint64(p.EntityUniqueID) != p.EntityRuntimeID {
			t.Errorf("AddActor unique ID %d, runtime ID %d", p.EntityUniqueID, p.EntityRuntimeID)
		}
		c := *p
		c.EntityUniqueID, c.EntityRuntimeID = 0, 0
		return p.EntityRuntimeID, &c, true
	case *packet.SetActorData:
		c := *p
		c.EntityRuntimeID = 0
		return p.EntityRuntimeID, &c, true
	case *packet.MoveActorAbsolute:
		c := *p
		c.EntityRuntimeID = 0
		return p.EntityRuntimeID, &c, true
	case *packet.RemoveActor:
		c := *p
		c.EntityUniqueID = 0
		return uint64(p.EntityUniqueID), &c, true
	}
	return 0, nil, false
}

// blockNamed returns pk with the variant of an add-actor packet, a block
// runtime ID, written as the block's name. The runtime ID of a block depends
// on which blocks the program registers, so it differs between this test
// binary and the game and changes when this package's imports do; the name
// does not.
func blockNamed(pk packet.Packet) packet.Packet {
	a, ok := pk.(*packet.AddActor)
	if !ok {
		return pk
	}
	rid, ok := a.EntityMetadata[protocol.EntityDataKeyVariant].(int32)
	if !ok {
		return pk
	}
	b, _ := world.DefaultBlockRegistry.BlockByRuntimeID(uint32(rid))
	name, _ := b.EncodeBlock()
	c := *a
	c.EntityMetadata = maps.Clone(a.EntityMetadata)
	c.EntityMetadata[protocol.EntityDataKeyVariant] = "block " + name
	return &c
}

// TestPopupAddActor pins every packet a client is sent for the damage
// numbers of a hunt and, for contrast, for a gem (a mark that is not
// floating text): the add-actor packet with all its metadata, the moves
// while a number rises and its removal. The game starts a damage number
// only with (*Run).popup and never changes its text, so no metadata update
// is sent later (any would be recorded too).
//
// Only the entity IDs are normalised (zeroed in a copy of each packet);
// the other fields, the position included, are recorded as sent.
func TestPopupAddActor(t *testing.T) {
	c, r := popupClient(t)
	var gem *world.EntityHandle
	do(t, c, func(tx *world.Tx, p *player.Player) {
		at := p.Position().Add(mgl64.Vec3{0, 1, 0})
		r.popup(tx, at, 5)
		r.popup(tx, at.Add(mgl64.Vec3{2, 0, 0}), 12.5)
		gem = world.EntitySpawnOpts{Position: at.Add(mgl64.Vec3{0, 0, 2})}.New(GemType, markConfig{run: r})
		tx.AddEntity(gem)
	})
	// Both numbers rise popupTicks-1 times, then close.
	var ids []uint64
	var gemID uint64
	for range 2 {
		pk := c.Wait(isAddActor("minecraft:falling_block"))
		ids = append(ids, pk.(*packet.AddActor).EntityRuntimeID)
	}
	gemID = c.Wait(isAddActor("skygear:hunt_gem")).(*packet.AddActor).EntityRuntimeID
	for range 2 {
		c.Wait(func(pk packet.Packet) bool {
			p, ok := pk.(*packet.RemoveActor)
			return ok && slices.Contains(ids, uint64(p.EntityUniqueID))
		})
	}
	do(t, c, closeMark(gem))
	c.Wait(func(pk packet.Packet) bool {
		p, ok := pk.(*packet.RemoveActor)
		return ok && uint64(p.EntityUniqueID) == gemID
	})

	entries, named := map[string]string{}, map[string]string{}
	seq := map[string]int{} // packets so far per entity: the order between entities is not fixed
	for _, pk := range c.Packets() {
		id, cp, ok := norm(t, pk)
		if !ok {
			continue
		}
		var name string
		switch {
		case id == gemID:
			name = "gem"
		case slices.Contains(ids, id):
			name = fmt.Sprintf("popup%d", slices.Index(ids, id)+1)
		default:
			continue
		}
		seq[name]++
		key := fmt.Sprintf("%s/%03d %T", name, seq[name], pk)
		entries[key] = hooktest.Canon(cp)
		named[key] = hooktest.Canon(blockNamed(cp))
	}
	t.Logf("%d packets pinned", len(entries))
	hooktest.Golden(t, "testdata/popup.golden", entries)
	// The same packets with the variant's block by name: unlike its runtime
	// ID, the name does not depend on the blocks this test binary registers.
	hooktest.Golden(t, "testdata/popup.byname.golden", named)

	// The number's metadata holds the air variant.
	air := int32(world.DefaultBlockRegistry.BlockRuntimeID(block.Air{}))
	for _, pk := range c.Packets() {
		if a, ok := pk.(*packet.AddActor); ok && a.EntityType == "minecraft:falling_block" {
			if v, ok := a.EntityMetadata[protocol.EntityDataKeyVariant]; !ok || v != air {
				t.Errorf("variant %v (%v), want %d", v, ok, air)
			}
		}
		if a, ok := pk.(*packet.AddActor); ok && a.EntityType == "skygear:hunt_gem" {
			if _, ok := a.EntityMetadata[protocol.EntityDataKeyVariant]; ok {
				t.Error("gem has a variant")
			}
		}
	}
}

// TestPopupResend: a number sent again (the viewer comes back into view,
// changes dimension, or joins late: the session builds a new add-actor
// packet each time) is still sent as air, and with the same metadata.
func TestPopupResend(t *testing.T) {
	c, r := popupClient(t)
	var h *world.EntityHandle
	do(t, c, func(tx *world.Tx, p *player.Player) {
		h = world.EntitySpawnOpts{Position: p.Position().Add(mgl64.Vec3{0, 1, 0}), NameTag: damageText(7)}.New(PopupType, markConfig{run: r, left: 1 << 30})
		tx.AddEntity(h)
	})
	first := c.Wait(isAddActor("minecraft:falling_block")).(*packet.AddActor)
	do(t, c, func(tx *world.Tx, p *player.Player) {
		e, _ := h.Entity(tx)
		s := sghook.Session(p)
		s.HideEntity(e)
		s.ViewEntity(e)
	})
	second := c.Wait(isAddActor("minecraft:falling_block")).(*packet.AddActor)
	do(t, c, closeMark(h))
	if first == second {
		t.Fatal("the second add-actor packet is the first")
	}
	air := int32(world.DefaultBlockRegistry.BlockRuntimeID(block.Air{}))
	for i, a := range []*packet.AddActor{first, second} {
		if v := a.EntityMetadata[protocol.EntityDataKeyVariant]; v != air {
			t.Errorf("send %d: variant %v, want %d", i+1, v, air)
		}
		if hooktest.Canon(a.EntityMetadata) != hooktest.Canon(first.EntityMetadata) {
			t.Errorf("send %d: metadata differs from the first send", i+1)
		}
	}
}
