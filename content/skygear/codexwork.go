package skygear

import (
	"fmt"

	"github.com/presentkim/skygear/content/skygear/handwork"

	"github.com/presentkim/skygear/content/skygear/farm"

	"github.com/presentkim/skygear/content/skygear/textfmt"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Cards of what no machine table lists: the hand work of the start (a stick
// on the ground, the hammer, stripping logs, tilling dirt, the composter) and
// block processes (the crucible, the barrels). Item pages find them by their
// slots (codexWorkCards).

// codexHandCards are the hand work cards, the tool the header.
func (s *session) codexHandCards(lang string) []codexCard {
	var cards []codexCard
	stick := item.NewStack(item.Stick{}, 1)
	total := 0
	for _, k := range sgitem.PebbleKinds {
		total += k.Weight
	}
	var pebbles []ui.BoardCell
	for _, k := range sgitem.PebbleKinds {
		if k.Weight == 0 {
			continue
		}
		st := item.NewStack(sgitem.Pebble{Kind: k.Kind}, 1)
		if k.Weight == total {
			pebbles = append(pebbles, s.codexCell(lang, st, ""))
		} else {
			pebbles = append(pebbles, s.chanceCell(lang, st, k.Weight*1000/total))
		}
	}
	for _, ground := range []item.Stack{item.NewStack(block.Grass{}, 1), item.NewStack(block.Dirt{}, 1)} {
		cards = append(cards, machineCard(lang, stick, []ui.BoardCell{s.codexCell(lang, ground, "", i18n.T(lang, "codex.card.stickUse"))}, pebbles))
	}
	hammer := item.NewStack(sgitem.Hammer{Stone: true}, 1)
	for _, in := range crushInputs() {
		b, ok := in.(world.Block)
		if !ok {
			continue
		}
		if out, ok := sgitem.HammerDrop(b); ok {
			cards = append(cards, machineCard(lang, hammer, []ui.BoardCell{s.codexCell(lang, item.NewStack(in, 1), "", i18n.T(lang, "codex.card.hammer"))},
				[]ui.BoardCell{s.codexCell(lang, out, countLabel(out.Count()))}))
		}
	}
	axe := item.NewStack(item.Axe{Tier: item.ToolTierStone}, 1)
	fiber := s.codexCell(lang, item.NewStack(sgitem.BarkFiber{}, 1), fmt.Sprintf("§f%d%%", handwork.FiberChance),
		fmt.Sprintf(i18n.T(lang, "codex.card.simDrop"), textfmt.Range(1, 2), handwork.FiberChance))
	cards = append(cards, machineCard(lang, axe, []ui.BoardCell{s.codexCell(lang, item.NewStack(block.Log{Wood: block.OakWood()}, 1), "", i18n.T(lang, "codex.card.strip"))}, []ui.BoardCell{fiber}))
	hoe := item.NewStack(item.Hoe{Tier: item.ToolTierWood}, 1)
	seeds := s.codexCell(lang, item.NewStack(sgitem.GrassSeeds{}, 1), fmt.Sprintf("§f%d%%", handwork.GrassSeedChance), fmt.Sprintf(i18n.T(lang, "codex.card.chance"), handwork.GrassSeedChance))
	cards = append(cards, machineCard(lang, hoe, []ui.BoardCell{s.codexCell(lang, item.NewStack(block.Dirt{}, 1), "", i18n.T(lang, "codex.card.till"))}, []ui.BoardCell{seeds}))
	composter := item.NewStack(block.Composter{}, 1)
	plants := s.codexCell(lang, item.NewStack(block.Leaves{Type: block.OakLeaves()}, 1), countLabel(farm.MatterCapacity), i18n.T(lang, "codex.card.anyPlant"))
	cards = append(cards, machineCard(lang, composter, []ui.BoardCell{plants}, []ui.BoardCell{s.codexCell(lang, item.NewStack(block.Dirt{}, 1), "")}))
	return cards
}

// codexWorkCards are the cards an item page finds by their slots: the hand
// work and the processes of the crucible and the barrels.
func (s *session) codexWorkCards(lang string) []codexCard {
	cards := s.codexHandCards(lang)
	for _, pr := range codexProcesses() {
		_, barrel := pr.head.Item().(farm.Barrel)
		if m, ok := pr.head.Item().(machine.Machine); barrel || ok && m.Kind == machine.KindCrucible {
			cards = append(cards, pr.cards(s, lang)...)
		}
	}
	return cards
}

// codexWorkFor are the work cards that make it (out) or that it goes into:
// an input slot, or the tool of the header.
func (s *session) codexWorkFor(lang string, it item.Stack, out bool) []codexCard {
	icon := formicon.Of(item.NewStack(it.Item(), 1))
	var found []codexCard
	for _, c := range s.codexWorkCards(lang) {
		ins, outs := cardSlots(c)
		switch {
		case out && outs[icon]:
		case !out && (ins[icon] || formicon.Of(c.head) == icon):
		default:
			continue
		}
		found = append(found, c)
	}
	return found
}

// cardSlots are the icons of a card's slots left of its arrow (the inputs)
// and right of it (the outputs).
func cardSlots(c codexCard) (ins, outs map[string]bool) {
	ins, outs = map[string]bool{}, map[string]bool{}
	arrow := -1
	for _, row := range c.rows {
		for x, cell := range row {
			if cell.IsArrow() {
				arrow = x
			}
		}
	}
	for _, row := range c.rows {
		for x, cell := range row {
			sl, ok := cell.SlotOf()
			if !ok {
				continue
			}
			if x < arrow {
				ins[sl.Icon] = true
			} else {
				outs[sl.Icon] = true
			}
		}
	}
	return ins, outs
}
