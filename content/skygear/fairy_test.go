package skygear

import (
	"testing"
	"time"

	"github.com/presentkim/skygear/content/skygear/fairy"

	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestOldResearchNoteIsKept: an old research note only says it is no
// longer used.
func TestOldResearchNoteIsKept(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		ctx := &item.UseContext{}
		if (sgitem.ResearchNote{}).Use(tx, p, ctx) || ctx.CountSub != 0 {
			t.Fatal("an old research note was used up")
		}
	})
}

// TestGiftsAreBasics: with no research done, a gift is oak logs, oak
// saplings or stone pebbles.
func TestGiftsAreBasics(t *testing.T) {
	s := testSession(t)
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

func TestGiftsComeFromDoneResearch(t *testing.T) {
	s := testSession(t)
	progress.StateOf(s.Session).Researched = map[string]bool{}
	var reach func(id string)
	reach = func(id string) {
		progress.StateOf(s.Session).Researched[id] = true
		r, _ := progress.ByID(id)
		for _, n := range r.Needs {
			reach(n)
		}
	}
	reach("generator")
	generator, _ := progress.ByID("generator")
	costs, rewards := 0, 0
	for range 1000 {
		gift := fairy.RollGift(s.Session)
		switch {
		case gift[0].Comparable(progress.Reward(generator)[0]):
			rewards++
		case gift[0].Comparable(fairy.HalfCost(generator)[0]):
			costs++
		}
	}
	if rewards == 0 || costs == 0 || rewards > costs {
		t.Errorf("%d reward gifts and %d cost gifts in 1000", rewards, costs)
	}
}

func TestGiftsSkipProjectsAndResearchDoneOutOfOrder(t *testing.T) {
	s := testSession(t)
	progress.StateOf(s.Session).Researched = map[string]bool{"project_dragon": true, "iron_tools": true, "stick": true}
	for range 500 {
		gift := fairy.RollGift(s.Session)
		for _, st := range gift {
			switch st.Item().(type) {
			case item.Diamond, item.NetheriteIngot, item.Emerald, item.IronIngot:
				t.Fatalf("gift %v from research done out of order", gift)
			}
		}
	}
	stick, _ := progress.ByID("stick")
	if !fairy.ReachedResearch(s.Session, stick) {
		t.Error("stick, done and needing nothing, is not reached")
	}
	project, _ := progress.ByID("project_dragon")
	if fairy.ReachedResearch(s.Session, project) {
		t.Error("project_dragon counted as reached without its needs")
	}
}

func TestDeeperResearchIsDeeper(t *testing.T) {
	stick, _ := progress.ByID("stick")
	generator, _ := progress.ByID("generator")
	if fairy.ResearchDepth(stick) != 0 || fairy.ResearchDepth(generator) <= fairy.ResearchDepth(stick) {
		t.Errorf("depths: stick %d, generator %d", fairy.ResearchDepth(stick), fairy.ResearchDepth(generator))
	}
}

// TestFairyVisitsAndGives: a fairy comes when due, gives one gift when
// touched and flies away.
func TestFairyVisitsAndGives(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
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
	tick(s, 20)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		e, ok := fairy.LocalOf(s.Session).Fairy.Entity(tx)
		if !ok {
			t.Fatal("fairy entity is gone")
		}
		f, _ := fairy.Of(s.Session, e)
		if !f.Gift(tx, s.Session, p) || f.Gift(tx, s.Session, p) {
			t.Fatal("a fairy must give exactly one gift")
		}
	})
	tick(s, fairy.FairyLeaveTicks+5)
	if fairy.LocalOf(s.Session).Fairy != nil {
		t.Fatal("the fairy did not fly away")
	}
}

// TestHittingTheFairyGivesTheGift: a left click is the same as a right
// click: it is cancelled (no hit) and gives the gift.
func TestHittingTheFairyGivesTheGift(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		fairy.Spawn(s.Session, tx, p)
	})
	tick(s, 2)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		e, _ := fairy.LocalOf(s.Session).Fairy.Entity(tx)
		if e.(*fairy.Fairy).NameTag() != "" {
			t.Errorf("fairy has a name tag %q", e.(*fairy.Fairy).NameTag())
		}
		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleAttackEntity(ctx, e, nil, nil, nil)
		if !ctx.Cancelled() || !e.(*fairy.Fairy).Data.Data.(*fairy.FairyData).Gave {
			t.Fatal("hitting the fairy did not give the gift instead of a hit")
		}
	})
}

// TestFairyComesEveryNight: the visits fall in the night window, one each
// day, and a save opened during the window still gets that night's visit.
func TestFairyComesEveryNight(t *testing.T) {
	s := testSession(t)
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
	s = testSession(t)
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
	s := testSession(t)
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
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// a floor for the gift to land on
		for x := -4; x <= 4; x++ {
			for z := -2; z <= 7; z++ {
				tx.SetBlock(cube.Pos{x, 63, z}, block.Stone{}, nil)
			}
		}
		fairy.Spawn(s.Session, tx, p)
	})
	tick(s, int(fairy.FairyStay/(time.Second/20))+fairy.FairyLeaveTicks+10)
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
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		dust += core.CountPlain(p.Inventory(), sgitem.FairyDust{})
	})
	if dust < fairy.FairyDustMin {
		t.Errorf("%d fairy dust scattered", dust)
	}
}

// TestFairyDustBackfill: a save that played nights before the fairy brought
// dust gets two or three dust a night once; the fairy's own dust ends it.
func TestFairyDustBackfill(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	day := s.Clock.DayLength()
	s.Clock.Advance(4*day + day*fairy.FairyTo/100)
	if n := fairy.PastNights(s.Session); n != 5 {
		t.Fatalf("%d nights past", n)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		fairy.BackfillDust(s.Session, p, tx)
		if n := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); n < 5*fairy.FairyDustMin || n > 5*fairy.FairyDustMax {
			t.Fatalf("%d dust for 5 nights", n)
		}
		fairy.BackfillDust(s.Session, p, tx)
		if n := core.CountPlain(p.Inventory(), sgitem.FairyDust{}); n > 5*fairy.FairyDustMax {
			t.Fatalf("backfilled twice: %d dust", n)
		}
	})
	s = testSession(t)
	s.Clock.Advance(3 * day)
	fairy.GiftOf(s.Session)
	if !fairy.StateOf(s.Session).FairyDust {
		t.Fatal("the fairy's dust did not mark the save")
	}
}
