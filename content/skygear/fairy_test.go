package skygear

import (
	"testing"
	"time"

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
		gift := s.rollGift()
		if len(gift) != 1 {
			t.Fatalf("gift = %v", gift)
		}
		kinds[naming.ItemKey(gift[0].Item())] = true
	}
	if len(kinds) != 3 || !kinds[naming.ItemKey(oakLog)] || !kinds[naming.ItemKey(block.Sapling{Type: block.OakSapling()})] || !kinds[naming.ItemKey(sgitem.Pebble{Kind: sgitem.PebbleStone})] {
		t.Errorf("gifts = %v", kinds)
	}
}

func TestGiftsComeFromDoneResearch(t *testing.T) {
	s := testSession(t)
	s.state.Researched = map[string]bool{}
	var reach func(id string)
	reach = func(id string) {
		s.state.Researched[id] = true
		r, _ := researchByID(id)
		for _, n := range r.Needs {
			reach(n)
		}
	}
	reach("generator")
	generator, _ := researchByID("generator")
	costs, rewards := 0, 0
	for range 1000 {
		gift := s.rollGift()
		switch {
		case gift[0].Comparable(researchReward(generator)[0]):
			rewards++
		case gift[0].Comparable(halfCost(generator)[0]):
			costs++
		}
	}
	if rewards == 0 || costs == 0 || rewards > costs {
		t.Errorf("%d reward gifts and %d cost gifts in 1000", rewards, costs)
	}
}

func TestGiftsSkipProjectsAndResearchDoneOutOfOrder(t *testing.T) {
	s := testSession(t)
	s.state.Researched = map[string]bool{"project_dragon": true, "iron_tools": true, "stick": true}
	for range 500 {
		gift := s.rollGift()
		for _, st := range gift {
			switch st.Item().(type) {
			case item.Diamond, item.NetheriteIngot, item.Emerald, item.IronIngot:
				t.Fatalf("gift %v from research done out of order", gift)
			}
		}
	}
	stick, _ := researchByID("stick")
	if !s.reachedResearch(stick) {
		t.Error("stick, done and needing nothing, is not reached")
	}
	project, _ := researchByID("project_dragon")
	if s.reachedResearch(project) {
		t.Error("project_dragon counted as reached without its needs")
	}
}

func TestDeeperResearchIsDeeper(t *testing.T) {
	stick, _ := researchByID("stick")
	generator, _ := researchByID("generator")
	if researchDepth(stick) != 0 || researchDepth(generator) <= researchDepth(stick) {
		t.Errorf("depths: stick %d, generator %d", researchDepth(stick), researchDepth(generator))
	}
}

// TestFairyVisitsAndGives: a fairy comes when due, gives one gift when
// touched and flies away.
func TestFairyVisitsAndGives(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	s.World.Do(func(tx *world.Tx) {
		s.tickFairy(tx)
		if day := s.Clock.DayLength(); s.state.NextFairy < day*fairyFrom/100 || s.state.NextFairy > day*fairyTo/100 {
			t.Fatalf("next fairy at %v", s.state.NextFairy)
		}
		s.state.NextFairy = -1 // due (0 means not scheduled)
		s.tickFairy(tx)
		if s.fairy == nil {
			t.Fatal("no fairy came when due")
		}
	})
	tick(s, 20)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		e, ok := s.fairy.Entity(tx)
		if !ok {
			t.Fatal("fairy entity is gone")
		}
		f, _ := s.fairyOf(e)
		if !f.gift(tx, s, p) || f.gift(tx, s, p) {
			t.Fatal("a fairy must give exactly one gift")
		}
	})
	tick(s, fairyLeaveTicks+5)
	if s.fairy != nil {
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
		s.spawnFairy(tx, p)
	})
	tick(s, 2)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		e, _ := s.fairy.Entity(tx)
		if e.(*Fairy).NameTag() != "" {
			t.Errorf("fairy has a name tag %q", e.(*Fairy).NameTag())
		}
		ctx := player.NewEventContext(tx, p)
		(&handler{s: s}).HandleAttackEntity(ctx, e, nil, nil, nil)
		if !ctx.Cancelled() || !e.(*Fairy).data.Data.(*fairyData).gave {
			t.Fatal("hitting the fairy did not give the gift instead of a hit")
		}
	})
}

// TestFairyComesEveryNight: the visits fall in the night window, one each
// day, and a save opened during the window still gets that night's visit.
func TestFairyComesEveryNight(t *testing.T) {
	s := testSession(t)
	day := s.Clock.DayLength()
	s.scheduleFairy(false)
	for n := range 20 {
		at := s.state.NextFairy
		if want := time.Duration(n) * day; at < want+day*fairyFrom/100 || at > want+day*fairyTo/100 {
			t.Fatalf("visit %d at %v, want in night %d", n, at, n)
		}
		now, _, _ := s.Clock.Now()
		s.Clock.Advance(at - now)
		s.scheduleFairy(true)
	}
	s = testSession(t)
	s.Clock.Advance(day * 80 / 100)
	s.scheduleFairy(false)
	if now, _, _ := s.Clock.Now(); s.state.NextFairy > now+day*5/100 {
		t.Errorf("opened at 80%% of the day, visit at %v (now %v)", s.state.NextFairy, now)
	}
	s.Clock.Advance(day * 10 / 100)
	s.scheduleFairy(false)
	if s.state.NextFairy < day+day*fairyFrom/100 {
		t.Errorf("opened after the night, visit at %v", s.state.NextFairy)
	}
}

// TestFairyGiftHasDust: every gift brings two or three fairy dust.
func TestFairyGiftHasDust(t *testing.T) {
	s := testSession(t)
	for range 100 {
		gift := s.fairyGift()
		last := gift[len(gift)-1]
		if _, ok := last.Item().(sgitem.FairyDust); !ok || last.Count() < fairyDustMin || last.Count() > fairyDustMax {
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
		s.spawnFairy(tx, p)
	})
	tick(s, int(fairyStay/(time.Second/20))+fairyLeaveTicks+10)
	if s.fairy != nil {
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
		dust += countPlain(p.Inventory(), sgitem.FairyDust{})
	})
	if dust < fairyDustMin {
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
	s.Clock.Advance(4*day + day*fairyTo/100)
	if n := s.pastNights(); n != 5 {
		t.Fatalf("%d nights past", n)
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		s.backfillFairyDust(p, tx)
		if n := countPlain(p.Inventory(), sgitem.FairyDust{}); n < 5*fairyDustMin || n > 5*fairyDustMax {
			t.Fatalf("%d dust for 5 nights", n)
		}
		s.backfillFairyDust(p, tx)
		if n := countPlain(p.Inventory(), sgitem.FairyDust{}); n > 5*fairyDustMax {
			t.Fatalf("backfilled twice: %d dust", n)
		}
	})
	s = testSession(t)
	s.Clock.Advance(3 * day)
	s.fairyGift()
	if !s.state.FairyDust {
		t.Fatal("the fairy's dust did not mark the save")
	}
}
