package skygear

import (
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/screens"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"golang.org/x/text/language"
)

// TestStrandedPlayerKeepsBackfilledDust: the server stops while a player is
// in the second dimension, so the save still says the player is there: what
// the player carried is put away in the save and the player holds the kit of
// the run. When the server runs again and the player joins, the player gets
// back what it carried before the join steps of the other features hand
// anything out, so the fairy dust the save is owed stays in the inventory
// the player keeps.
func TestStrandedPlayerKeepsBackfilledDust(t *testing.T) {
	// The first run of the server: a player goes into the second dimension
	// carrying three stone, and the server stops.
	var (
		mu  sync.Mutex
		doc []byte
	)
	first := coretest.SessionOver(t, nil, func(b []byte) error {
		mu.Lock()
		defer mu.Unlock()
		doc = slices.Clone(b)
		return nil
	})
	coretest.LoadAround(first)
	h := coretest.SpawnPlayer(first)
	carried := item.NewStack(block.Stone{}, 3)
	var (
		id     uuid.UUID
		kit    []item.Stack
		armour []item.Stack
	)
	coretest.WithPlayer(t, first, h, func(p *player.Player, tx *world.Tx) {
		first.Join(p, true)
		_ = p.Inventory().SetItem(0, carried)
		if !dimension.Begin(first.Session, tx, p, sgitem.MobSlime) {
			t.Fatal("the run did not begin")
		}
		if n := core.CountPlain(p.Inventory(), block.Stone{}); n != 0 {
			t.Fatalf("the player still carries %d stone in the second dimension", n)
		}
		id, kit, armour = p.UUID(), slices.Clone(p.Inventory().Slots()), slices.Clone(p.Armour().Slots())
	})
	first.Session.Close() // the server stops: the save is stored as it is
	mu.Lock()
	saved := slices.Clone(doc)
	mu.Unlock()
	if len(saved) == 0 {
		t.Fatal("the first run stored no state document")
	}

	// The server runs again, nights later, on a save that never got its dust.
	const nights = 3
	second := coretest.SessionOver(t, saved, nil)
	coretest.LoadAround(second)
	day := second.Clock.DayLength()
	second.Clock.Advance((nights-1)*day + day*fairy.FairyTo/100)
	if n := fairy.PastNights(second.Session); n != nights {
		t.Fatalf("%d nights past, want %d", n, nights)
	}
	const seed1, seed2 = 1, 2
	second.Rand = rand.New(rand.NewPCG(seed1, seed2))
	r := rand.New(rand.NewPCG(seed1, seed2))
	want := 0
	for range nights {
		want += fairy.FairyDustMin + r.IntN(fairy.FairyDustMax-fairy.FairyDustMin+1)
	}
	if want == 0 {
		t.Fatal("the save is owed no dust")
	}

	// The player comes back as the server stored it: holding the kit.
	inv := inventory.New(36, nil)
	for slot, it := range kit {
		_ = inv.SetItem(slot, it)
	}
	arm := inventory.NewArmour(nil)
	arm.Set(armour[0], armour[1], armour[2], armour[3])
	back := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{
		Name: "p", UUID: id, Locale: language.Korean, Position: mgl64.Vec3{0.5, 64, 2.5},
		Inventory: inv, Armour: arm,
	})
	second.World.Do(func(tx *world.Tx) { tx.AddEntity(back) })
	coretest.WithPlayer(t, second, back, func(p *player.Player, tx *world.Tx) {
		if dimension.Hunting(second.Session, p) {
			t.Fatal("a run is under way after the server started again")
		}
		second.Join(p, false)
		if got, _ := p.Inventory().Item(0); !got.Equal(carried) {
			t.Errorf("slot 0 holds %v after the join, want the three stone the player carried", got)
		}
		if it, _ := p.Inventory().Item(8); !screens.IsMenuItem(it) {
			t.Errorf("slot 8 holds %v, want the menu item", it)
		}
		for _, it := range p.Armour().Slots() {
			if !it.Empty() {
				t.Errorf("the player still wears %v of the kit", it)
			}
		}
		if got := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); got != want {
			t.Errorf("%d fairy dust in the inventory after the join, want %d", got, want)
		}

		// A second join gives nothing more and takes nothing away.
		second.Join(p, false)
		if got := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); got != want {
			t.Errorf("%d fairy dust after joining again, want %d", got, want)
		}
		if got, _ := p.Inventory().Item(0); !got.Equal(carried) {
			t.Errorf("slot 0 holds %v after joining again", got)
		}
	})
}
