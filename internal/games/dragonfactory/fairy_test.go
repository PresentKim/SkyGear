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

// TestGiftsAreCoreBlocks: a gift is giftCount of oak logs or cobblestone.
func TestGiftsAreCoreBlocks(t *testing.T) {
	s := testSession(t)
	blocks := map[string]bool{}
	for range 200 {
		st := s.rollGift()
		blocks[itemKey(st.Item())] = true
		if st.Count() != giftCount {
			t.Fatalf("gift of %d", st.Count())
		}
	}
	if len(blocks) != 2 || !blocks[itemKey(oakLog)] || !blocks[itemKey(block.Cobblestone{})] {
		t.Errorf("gifts = %v", blocks)
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
