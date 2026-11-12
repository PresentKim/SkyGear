package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/copier"
	"github.com/presentkim/skygear/content/skygear/dimension"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/platform/account"
)

// The copier, the construction wands, the hammers and the teleport stone do
// their work on the air in their own Use, which Dragonfly calls after the
// handler's subscribers. These tests use the items through Dragonfly
// (Player.UseItem); the pin scenarios call the handler.

// airUse is what one real use of held on the air did: whether the item acted
// at all (it opened its screen or did its work, which arms the session's
// debounce), whether the copier in hand lost its card, and whether diagonal
// vein mining was switched.
type airUse struct{ acted, cleared, toggled bool }

func useOnTheAir(s *session, p *player.Player, held item.Stack, sneak bool) airUse {
	p.SetHeldItems(held, item.Stack{})
	if sneak {
		p.StartSneaking()
	} else {
		p.StopSneaking()
	}
	s.ForgetOpen(p)
	_, hadCard := copier.CardOf(held)
	diagonal := account.On(p, tinker.VeinDiagonalSetting)
	p.UseItem()
	now, _ := p.HeldItems()
	_, hasCard := copier.CardOf(now)
	return airUse{acted: s.Opened(p), cleared: hadCard && !hasCard, toggled: account.On(p, tinker.VeinDiagonalSetting) != diagonal}
}

func filledCopier() item.Stack {
	card := &copier.ConfigCard{Kind: copier.CardSide, Side: 4, Sides: []*copier.ConduitCard{{Mode: int(itemnet.Input), Extract: 5, Insert: 9}}}
	return copier.WithCard(item.NewStack(sgitem.ConfigCopier{}, 1), card, "ko")
}

// airCases are the items used on the air and what each use does for a player
// on the island.
var airCases = []struct {
	name  string
	held  func() item.Stack
	sneak bool
	want  airUse
}{
	{"a filled copier", filledCopier, false, airUse{acted: true, cleared: true}},
	{"a wand, sneaking", func() item.Stack { return item.NewStack(sgitem.NewWand(sgitem.WandStone), 1) }, true, airUse{acted: true}},
	{"a wand, standing", func() item.Stack { return item.NewStack(sgitem.NewWand(sgitem.WandStone), 1) }, false, airUse{}},
	{"the infinity wand, sneaking", func() item.Stack { return item.NewStack(sgitem.NewWand(sgitem.WandInfinity), 1) }, true, airUse{acted: true}},
	{"the iron hammer, sneaking", func() item.Stack { return item.NewStack(sgitem.Hammer{}, 1) }, true, airUse{acted: true, toggled: true}},
	{"the iron hammer, standing", func() item.Stack { return item.NewStack(sgitem.Hammer{}, 1) }, false, airUse{}},
	{"the stone hammer, sneaking", func() item.Stack { return item.NewStack(sgitem.Hammer{Stone: true}, 1) }, true, airUse{}},
	{"the teleport stone", func() item.Stack { return item.NewStack(sgitem.DimensionStone{}, 1) }, false, airUse{acted: true}},
}

// TestItemsUsedOnTheAir: each item does its work when a player on the island
// uses it on the air, and in island expansion mode, where nothing refuses a
// use on the air.
func TestItemsUsedOnTheAir(t *testing.T) {
	for _, mode := range []string{"on the island", "in expansion mode"} {
		for _, c := range airCases {
			s := testSession(t)
			island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
			loadAround(s)
			h := spawnPlayer(s)
			withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
				p.Handle(s.Handler(p))
				if mode == "in expansion mode" {
					expand.Show(s.Session, p)
				}
				if got := useOnTheAir(s, p, c.held(), c.sneak); got != c.want {
					t.Errorf("%s, %s: the use did %+v, want %+v", c.name, mode, got, c.want)
				}
			})
		}
	}
}

// TestHunterUsesNothingOnTheAir: in the second dimension a use on the air is
// refused whatever the hunter holds but the menu item (dimension.hunting
// cancels it), so Dragonfly never calls the item's Use: no item acts.
func TestHunterUsesNothingOnTheAir(t *testing.T) {
	for _, c := range airCases {
		s := testSession(t)
		island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
		loadAround(s)
		h := spawnPlayer(s)
		withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
			p.Handle(s.Handler(p))
			if !dimension.Begin(s.Session, tx, p, sgitem.MobSlime) {
				t.Fatal("the run did not begin")
			}
			if got := useOnTheAir(s, p, c.held(), c.sneak); got != (airUse{}) {
				t.Errorf("%s: a hunter's use did %+v, want nothing", c.name, got)
			}
		})
	}
}

// TestAirItemsAreUsable: the items used on the air are item.Usable, which is
// also what tells a client to send the use at all (clientdef.ItemEntry gives
// such an item minecraft:use_modifiers).
func TestAirItemsAreUsable(t *testing.T) {
	for _, it := range []world.Item{
		sgitem.ConfigCopier{}, sgitem.Wand{Tier: sgitem.WandStone}, sgitem.InfinityWand{},
		sgitem.Hammer{}, sgitem.Hammer{Stone: true}, sgitem.DimensionStone{},
	} {
		if _, ok := it.(item.Usable); !ok {
			t.Errorf("%T is not usable on the air", it)
		}
	}
}
