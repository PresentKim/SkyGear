package fairy_test

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestGiftsAreBasics: with no research done, a gift is oak logs, oak
// saplings or stone pebbles.
func TestGiftsAreBasics(t *testing.T) {
	s := coretest.Session(t)
	kinds := map[string]bool{}
	for range 200 {
		gift := fairy.RollGift(s.Session)
		if len(gift) != 1 {
			t.Fatalf("gift = %v", gift)
		}
		kinds[naming.ItemKey(gift[0].Item())] = true
	}
	if len(kinds) != 3 || !kinds[naming.ItemKey(fairy.OakLog)] || !kinds[naming.ItemKey(block.Sapling{Type: block.OakSapling()})] || !kinds[naming.ItemKey(sgitem.Pebble{Kind: sgitem.PebbleStone})] {
		t.Errorf("gifts = %v", kinds)
	}
}

// TestFairyVisitsAndGives: a fairy comes when due, gives one gift when
// touched and flies away.
func TestFairyVisitsAndGives(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	s.World.Do(func(tx *world.Tx) {
		fairy.Tick(s.Session, tx)
		if day := s.Clock.DayLength(); fairy.StateOf(s.Session).NextFairy < day*fairy.FairyFrom/100 || fairy.StateOf(s.Session).NextFairy > day*fairy.FairyTo/100 {
			t.Fatalf("next fairy at %v", fairy.StateOf(s.Session).NextFairy)
		}
		fairy.StateOf(s.Session).NextFairy = -1 // due (0 means not scheduled)
		fairy.Tick(s.Session, tx)
		if fairy.LocalOf(s.Session).Fairy == nil {
			t.Fatal("no fairy came when due")
		}
	})
	coretest.Tick(s, 20)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		e, ok := fairy.LocalOf(s.Session).Fairy.Entity(tx)
		if !ok {
			t.Fatal("fairy entity is gone")
		}
		f, _ := fairy.Of(s.Session, e)
		if !f.Gift(tx, s.Session, p) || f.Gift(tx, s.Session, p) {
			t.Fatal("a fairy must give exactly one gift")
		}
	})
	coretest.Tick(s, fairy.FairyLeaveTicks+5)
	if fairy.LocalOf(s.Session).Fairy != nil {
		t.Fatal("the fairy did not fly away")
	}
}

// TestHittingTheFairyGivesTheGift: a left click is the same as a right
// click: it is cancelled (no hit) and gives the gift.
func TestHittingTheFairyGivesTheGift(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		fairy.Spawn(s.Session, tx, p)
	})
	coretest.Tick(s, 2)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		e, _ := fairy.LocalOf(s.Session).Fairy.Entity(tx)
		if e.(*fairy.Fairy).NameTag() != "" {
			t.Errorf("fairy has a name tag %q", e.(*fairy.Fairy).NameTag())
		}
		ctx := player.NewEventContext(tx, p)
		s.Handler(p).HandleAttackEntity(ctx, e, nil, nil, nil)
		if !ctx.Cancelled() || !e.(*fairy.Fairy).Data.Data.(*fairy.FairyData).Gave {
			t.Fatal("hitting the fairy did not give the gift instead of a hit")
		}
	})
}

// TestFairyComesEveryNight: the visits fall in the night window, one each
// day, and a save opened during the window still gets that night's visit.
func TestFairyComesEveryNight(t *testing.T) {
	s := coretest.Session(t)
	day := s.Clock.DayLength()
	fairy.Schedule(s.Session, false)
	for n := range 20 {
		at := fairy.StateOf(s.Session).NextFairy
		if want := time.Duration(n) * day; at < want+day*fairy.FairyFrom/100 || at > want+day*fairy.FairyTo/100 {
			t.Fatalf("visit %d at %v, want in night %d", n, at, n)
		}
		now, _, _ := s.Clock.Now()
		s.Clock.Advance(at - now)
		fairy.Schedule(s.Session, true)
	}
	s = coretest.Session(t)
	s.Clock.Advance(day * 80 / 100)
	fairy.Schedule(s.Session, false)
	if now, _, _ := s.Clock.Now(); fairy.StateOf(s.Session).NextFairy > now+day*5/100 {
		t.Errorf("opened at 80%% of the day, visit at %v (now %v)", fairy.StateOf(s.Session).NextFairy, now)
	}
	s.Clock.Advance(day * 10 / 100)
	fairy.Schedule(s.Session, false)
	if fairy.StateOf(s.Session).NextFairy < day+day*fairy.FairyFrom/100 {
		t.Errorf("opened after the night, visit at %v", fairy.StateOf(s.Session).NextFairy)
	}
}

// TestFairyGiftHasDust: every gift brings two or three fairy dust.
func TestFairyGiftHasDust(t *testing.T) {
	s := coretest.Session(t)
	for range 100 {
		gift := fairy.GiftOf(s.Session)
		last := gift[len(gift)-1]
		if _, ok := last.Item().(sgitem.FairyDust); !ok || last.Count() < fairy.FairyDustMin || last.Count() > fairy.FairyDustMax {
			t.Fatalf("gift = %v", gift)
		}
	}
}

// TestFairyScattersTheGift: left alone for fairyStay, the fairy drops its
// gift about the player and flies away.
func TestFairyScattersTheGift(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// a floor for the gift to land on
		for x := -4; x <= 4; x++ {
			for z := -2; z <= 7; z++ {
				tx.SetBlock(cube.Pos{x, 63, z}, block.Stone{}, nil)
			}
		}
		fairy.Spawn(s.Session, tx, p)
	})
	coretest.Tick(s, int(fairy.FairyStay/(time.Second/20))+fairy.FairyLeaveTicks+10)
	if fairy.LocalOf(s.Session).Fairy != nil {
		t.Fatal("the fairy did not fly away")
	}
	dust := 0
	s.World.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if it, ok := e.(*entity.Ent); ok {
				if b, ok := it.Behaviour().(*entity.ItemBehaviour); ok {
					if _, ok := b.Item().Item().(sgitem.FairyDust); ok {
						dust += b.Item().Count()
					}
				}
			}
		}
	})
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		dust += core.CountPlain(p.Inventory(), sgitem.FairyDust{})
	})
	if dust < fairy.FairyDustMin {
		t.Errorf("%d fairy dust scattered", dust)
	}
}

// TestFairyDustBackfill: a save that played nights before the fairy brought
// dust gets two or three dust a night once; the fairy's own dust ends it.
func TestFairyDustBackfill(t *testing.T) {
	s := coretest.Session(t)
	coretest.LoadAround(s)
	h := coretest.SpawnPlayer(s)
	day := s.Clock.DayLength()
	s.Clock.Advance(4*day + day*fairy.FairyTo/100)
	if n := fairy.PastNights(s.Session); n != 5 {
		t.Fatalf("%d nights past", n)
	}
	coretest.WithPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		fairy.BackfillDust(s.Session, p, tx)
		if n := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); n < 5*fairy.FairyDustMin || n > 5*fairy.FairyDustMax {
			t.Fatalf("%d dust for 5 nights", n)
		}
		fairy.BackfillDust(s.Session, p, tx)
		if n := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); n > 5*fairy.FairyDustMax {
			t.Fatalf("backfilled twice: %d dust", n)
		}
	})
	s = coretest.Session(t)
	s.Clock.Advance(3 * day)
	fairy.GiftOf(s.Session)
	if !fairy.StateOf(s.Session).FairyDust {
		t.Fatal("the fairy's dust did not mark the save")
	}
}
