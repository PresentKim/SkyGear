package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestResearchNoteRevealsHints: a note reveals the hint of a hidden recipe
// of an open tier and is used up; with nothing left it is kept.
func TestResearchNoteRevealsHints(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// Level 1: only tier 1 is open, whose one hidden recipe is smooth stone.
		ctx := &item.UseContext{}
		if !(ResearchNote{}).Use(tx, p, ctx) || ctx.CountSub != 1 || !s.state.Hinted["smooth_stone"] {
			t.Fatalf("first note: hinted %v, used %d", s.state.Hinted, ctx.CountSub)
		}
		ctx = &item.UseContext{}
		if (ResearchNote{}).Use(tx, p, ctx) || ctx.CountSub != 0 {
			t.Fatal("a note was used up with nothing left to reveal")
		}
		s.state.Level = 10
		for range 2 {
			(ResearchNote{}).Use(tx, p, &item.UseContext{})
		}
		if len(s.state.Hinted) != 3 {
			t.Fatalf("hinted %v after opening every tier", s.state.Hinted)
		}
	})
}

// TestGiftsComeFromOpenTiers: a gift is a research note or some of a block
// a known recipe of an open tier makes.
func TestGiftsComeFromOpenTiers(t *testing.T) {
	s := testSession(t)
	notes, blocks := 0, map[string]bool{}
	for range 2000 {
		st := s.rollGift()
		if _, ok := st.Item().(ResearchNote); ok {
			notes++
			continue
		}
		blocks[itemKey(st.Item())] = true
		if st.Count() != giftCounts[1] {
			t.Fatalf("tier 1 gift of %d", st.Count())
		}
	}
	if notes < 100 || notes > 320 {
		t.Errorf("%d notes in 2000 gifts, want about 10%%", notes)
	}
	if blocks[itemKey(block.Stone{})] || blocks[itemKey(block.Mud{})] || !blocks[itemKey(oakLog)] {
		t.Errorf("gifts at level 1 = %v", blocks)
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
