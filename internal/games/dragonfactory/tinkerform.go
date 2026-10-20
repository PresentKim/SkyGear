package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
)

// The tinker table: a click opens a menu to make a tool (kind, head metal,
// handle) or to put modifiers on the tool in hand.

func (s *session) showTinker(p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.body"),
		ui.Option{Label: i18n.T(lang, "tinker.make"), Run: func(p *player.Player, tx *world.Tx) { s.showToolKinds(p, tx, pos) }},
		ui.Option{Label: i18n.T(lang, "tinker.modify"), Run: func(p *player.Player, tx *world.Tx) { s.showModifiers(p, tx, pos) }},
	)
}

// showToolKinds lets the player pick the kind of tool to make.
func (s *session) showToolKinds(p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, k := range toolKinds {
		slots = append(slots, ui.ItemSlot{
			Icon: ui.ItemIcon(TinkerTool{Kind: k, Head: "iron", Handle: "wood"}),
			Tip:  "§f" + i18n.T(lang, "tool."+k) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.headCost"), toolHeadCost[k]),
			Run:  func(p *player.Player, tx *world.Tx) { s.showToolHeads(p, tx, pos, k) },
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.make")), i18n.T(lang, "tinker.pickKind"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showTinker(p, tx, pos) }))
}

// showToolHeads lists the head metals open to the save.
func (s *session) showToolHeads(p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, h := range headMaterials {
		if !s.researched(h.research) && !s.recipesFree {
			continue
		}
		have := countPlain(p.Inventory(), h.item)
		need := toolHeadCost[kind]
		mark := "§f"
		if have < need {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(h.item),
			Count: mark + fmt.Sprint(have),
			Tip: "§f" + i18n.T(lang, "tmat."+h.id) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.headStats"), h.dur, h.eff, h.attack+toolAttackBonus[kind]) +
				"\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.need"), need),
			Run: func(p *player.Player, tx *world.Tx) { s.showToolHandles(p, tx, pos, kind, h) },
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tool."+kind)), i18n.T(lang, "tinker.pickHead"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showToolKinds(p, tx, pos) }))
}

// showToolHandles lists the handles and makes the tool on a pick.
func (s *session) showToolHandles(p *player.Player, tx *world.Tx, pos cube.Pos, kind string, head headMaterial) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, hd := range handleMaterials {
		have := countPlain(p.Inventory(), hd.item)
		mark := "§f"
		if have < 1 {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(hd.item),
			Count: mark + fmt.Sprint(have),
			Tip:   "§f" + i18n.T(lang, "thandle."+hd.id) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.handleStats"), hd.durMul*100, hd.speedMul*100, hd.attack),
			Run: func(p *player.Player, tx *world.Tx) {
				s.makeTool(p, tx, pos, kind, head, hd)
				s.showToolHandles(p, tx, pos, kind, head)
			},
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tmat."+head.id)+" "+i18n.T(lang, "tool."+kind)), i18n.T(lang, "tinker.pickHandle"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showToolHeads(p, tx, pos, kind) }))
}

// makeTool takes the materials and gives the tool; it does nothing when they are
// missing.
func (s *session) makeTool(p *player.Player, tx *world.Tx, pos cube.Pos, kind string, head headMaterial, hd handleMaterial) bool {
	inv := p.Inventory()
	need := toolHeadCost[kind]
	if countPlain(inv, head.item) < need || countPlain(inv, hd.item) < 1 {
		return false
	}
	takePlain(inv, head.item, need)
	takePlain(inv, hd.item, 1)
	tool := describeTool(item.NewStack(TinkerTool{Kind: kind, Head: head.id, Handle: hd.id}, 1))
	s.giveItem(p, tx, p.Position(), tool)
	s.advance(tx, advToolMade)
	count(p, "tool_made", 1)
	tx.PlaySound(pos.Vec3Centre(), sound.AnvilUse{})
	return true
}

// showModifiers lists the modifiers for the tool in hand.
func (s *session) showModifiers(p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	held, _ := p.HeldItems()
	t, ok := held.Item().(TinkerTool)
	if !ok {
		ui.Menu(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.modify")), i18n.T(lang, "tinker.holdTool"),
			ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showTinker(p, tx, pos) }))
		return
	}
	mods := toolMods(held)
	lvl := toolLevel(toolXP(held))
	var slots []ui.ItemSlot
	for _, m := range toolModifiers {
		cur := mods[m.id]
		have := countPlain(p.Inventory(), m.item)
		mark := "§f"
		if have < m.cost {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(m.item),
			Count: mark + fmt.Sprint(have),
			Tip: "§f" + i18n.T(lang, "tmod."+m.id) + " §7" + fmt.Sprintf("%d/%d", cur, m.max) +
				"\n§7" + i18n.T(lang, "tmodinfo."+m.id) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.need"), m.cost),
			Run: func(p *player.Player, tx *world.Tx) {
				s.applyModifier(p, tx, pos, m)
				s.showModifiers(p, tx, pos)
			},
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), t.Name()+fmt.Sprintf(" Lv.%d", lvl)), fmt.Sprintf(i18n.T(lang, "tinker.slots"), modsUsed(mods), toolSlots(lvl)), slots,
		ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showTinker(p, tx, pos) }))
}

// applyModifier adds a level of a modifier to the tool in hand, if the tool has
// a free slot, the modifier is not at its limit and the materials are there.
func (s *session) applyModifier(p *player.Player, tx *world.Tx, pos cube.Pos, m toolModifier) bool {
	held, off := p.HeldItems()
	if _, ok := held.Item().(TinkerTool); !ok {
		return false
	}
	mods := toolMods(held)
	if mods[m.id] >= m.max || modsUsed(mods) >= toolSlots(toolLevel(toolXP(held))) || !takePlain(p.Inventory(), m.item, m.cost) {
		return false
	}
	mods[m.id]++
	p.SetHeldItems(describeTool(withToolMods(held, mods)), off)
	s.advance(tx, advToolMod)
	count(p, "tool_mod", 1)
	tx.PlaySound(pos.Vec3Centre(), sound.AnvilUse{})
	return true
}

// toolOf returns the tinkered tool in p's hand.
func toolOf(p *player.Player) (item.Stack, bool) {
	held, _ := p.HeldItems()
	_, ok := held.Item().(TinkerTool)
	return held, ok
}

// gainToolXP adds xp to the tool in hand; on a level up it says so.
func (s *session) gainToolXP(p *player.Player, n int) {
	held, ok := toolOf(p)
	if !ok {
		return
	}
	next, up := withToolXP(held, n)
	_, off := p.HeldItems()
	p.SetHeldItems(next, off)
	if up {
		count(p, "tool_levelup", 1)
		s.advance(p.Tx(), advToolLevel)
		if toolLevel(toolXP(next)) >= 5 {
			s.advance(p.Tx(), advTool5)
		}
		p.Messagef(i18n.T(i18n.Of(p), "tinker.levelUp"), itemName(next), toolLevel(toolXP(next)))
	}
}

// reinforce gives a reinforced tool a chance to take back the durability a use
// cost.
func (s *session) reinforce(p *player.Player) {
	held, ok := toolOf(p)
	if !ok {
		return
	}
	if l := toolMods(held)["reinforced"]; l > 0 && held.Durability() < held.MaxDurability() && s.rand.IntN(100) < 14*l {
		_, off := p.HeldItems()
		p.SetHeldItems(held.WithDurability(held.Durability()+1), off)
	}
}

// tickToolEffects gives the players haste from a tool they hold, every second
// for the next few seconds.
func (s *session) tickToolEffects(tx *world.Tx) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		haste := 0
		if held, ok := toolOf(p); ok {
			haste = toolMods(held)["haste"]
		}
		if haste > 0 {
			p.AddEffect(effect.NewAmbient(effect.Haste, haste, 3*time.Second))
		}
	}
}
