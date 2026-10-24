package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestOldResearchNoteIsKept: an old research note only says it is no
// longer used.
func TestOldResearchNoteIsKept(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		ctx := &item.UseContext{}
		if (ResearchNote{}).Use(tx, p, ctx) || ctx.CountSub != 0 {
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
		kinds[itemKey(gift[0].Item())] = true
	}
	if len(kinds) != 3 || !kinds[itemKey(oakLog)] || !kinds[itemKey(block.Sapling{Type: block.OakSapling()})] || !kinds[itemKey(Pebble{Kind: PebbleStone})] {
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
	s.world.Do(func(tx *world.Tx) {
		s.tickFairy(tx)
		if s.state.NextFairy < fairyMinGap*s.clock.DayLength() || s.state.NextFairy > fairyMaxGap*s.clock.DayLength() {
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
