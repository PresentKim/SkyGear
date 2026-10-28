package skygear

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/games/skygear/advancement"
	"github.com/presentkim/skygear/internal/games/skygear/naming"
	"github.com/presentkim/skygear/internal/games/skygear/tinker"
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
	for _, k := range tinker.ToolKinds {
		slots = append(slots, ui.ItemSlot{
			Icon: ui.ItemIcon(tinker.Tool{Kind: k, Head: "iron", Handle: "wood"}),
			Tip:  "§f" + i18n.T(lang, "tool."+k) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.headCost"), tinker.ToolHeadCost[k]),
			Run:  func(p *player.Player, tx *world.Tx) { s.showToolHeads(p, tx, pos, k) },
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.make")), i18n.T(lang, "tinker.pickKind"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showTinker(p, tx, pos) }))
}

// showToolHeads lists the head metals open to the save.
func (s *session) showToolHeads(p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, h := range tinker.HeadMaterials {
		if !s.researched(h.Research) && !s.recipesFree {
			continue
		}
		have := countPlain(p.Inventory(), h.Item)
		need := tinker.ToolHeadCost[kind]
		mark := "§f"
		if have < need {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(h.Item),
			Count: mark + fmt.Sprint(have),
			Tip: "§f" + i18n.T(lang, "tmat."+h.ID) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.headStats"), h.Dur, h.Eff, h.Attack+tinker.ToolAttackBonus[kind]) +
				"\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.need"), need),
			Run: func(p *player.Player, tx *world.Tx) { s.showToolHandles(p, tx, pos, kind, h) },
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tool."+kind)), i18n.T(lang, "tinker.pickHead"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showToolKinds(p, tx, pos) }))
}

// showToolHandles lists the handles and makes the tool on a pick.
func (s *session) showToolHandles(p *player.Player, tx *world.Tx, pos cube.Pos, kind string, head tinker.HeadMaterial) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, hd := range tinker.HandleMaterials {
		have := countPlain(p.Inventory(), hd.Item)
		mark := "§f"
		if have < 1 {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(hd.Item),
			Count: mark + fmt.Sprint(have),
			Tip:   "§f" + i18n.T(lang, "thandle."+hd.ID) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.handleStats"), hd.DurMul*100, hd.SpeedMul*100, hd.Attack),
			Run: func(p *player.Player, tx *world.Tx) {
				s.makeTool(p, tx, pos, kind, head, hd)
				s.showToolHandles(p, tx, pos, kind, head)
			},
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tmat."+head.ID)+" "+i18n.T(lang, "tool."+kind)), i18n.T(lang, "tinker.pickHandle"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showToolHeads(p, tx, pos, kind) }))
}

// makeTool takes the materials and gives the tool; it does nothing when they are
// missing.
func (s *session) makeTool(p *player.Player, tx *world.Tx, pos cube.Pos, kind string, head tinker.HeadMaterial, hd tinker.HandleMaterial) bool {
	inv := p.Inventory()
	need := tinker.ToolHeadCost[kind]
	if countPlain(inv, head.Item) < need || countPlain(inv, hd.Item) < 1 {
		return false
	}
	takePlain(inv, head.Item, need)
	takePlain(inv, hd.Item, 1)
	tool := tinker.DescribeTool(item.NewStack(tinker.Tool{Kind: kind, Head: head.ID, Handle: hd.ID}, 1))
	s.giveItem(p, tx, p.Position(), tool)
	s.advance(tx, advancement.ToolMade)
	count(p, "tool_made", 1)
	tx.PlaySound(pos.Vec3Centre(), sound.AnvilUse{})
	return true
}

// showModifiers lists the modifiers for the tool in hand.
func (s *session) showModifiers(p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	held, _ := p.HeldItems()
	t, ok := held.Item().(tinker.Tool)
	if !ok {
		ui.Menu(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.modify")), i18n.T(lang, "tinker.holdTool"),
			ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showTinker(p, tx, pos) }))
		return
	}
	mods := tinker.ToolMods(held)
	lvl := tinker.ToolLevel(tinker.ToolXP(held))
	var slots []ui.ItemSlot
	for _, m := range tinker.ToolModifiers {
		cur := mods[m.ID]
		have := countPlain(p.Inventory(), m.Item)
		mark := "§f"
		if have < m.Cost {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(m.Item),
			Count: mark + fmt.Sprint(have),
			Tip: "§f" + i18n.T(lang, "tmod."+m.ID) + " §7" + fmt.Sprintf("%d/%d", cur, m.Max) +
				"\n§7" + i18n.T(lang, "tmodinfo."+m.ID) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.need"), m.Cost),
			Run: func(p *player.Player, tx *world.Tx) {
				s.applyModifier(p, tx, pos, m)
				s.showModifiers(p, tx, pos)
			},
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), t.Name()+fmt.Sprintf(" Lv.%d", lvl)), fmt.Sprintf(i18n.T(lang, "tinker.slots"), tinker.ModsUsed(mods), tinker.ToolSlots(lvl)), slots,
		ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { s.showTinker(p, tx, pos) }))
}

// applyModifier adds a level of a modifier to the tool in hand, if the tool has
// a free slot, the modifier is not at its limit and the materials are there.
func (s *session) applyModifier(p *player.Player, tx *world.Tx, pos cube.Pos, m tinker.ToolModifier) bool {
	held, off := p.HeldItems()
	if _, ok := held.Item().(tinker.Tool); !ok {
		return false
	}
	mods := tinker.ToolMods(held)
	if mods[m.ID] >= m.Max || tinker.ModsUsed(mods) >= tinker.ToolSlots(tinker.ToolLevel(tinker.ToolXP(held))) || !takePlain(p.Inventory(), m.Item, m.Cost) {
		return false
	}
	mods[m.ID]++
	p.SetHeldItems(tinker.DescribeTool(tinker.WithToolMods(held, mods)), off)
	s.advance(tx, advancement.ToolMod)
	count(p, "tool_mod", 1)
	tx.PlaySound(pos.Vec3Centre(), sound.AnvilUse{})
	return true
}

// toolOf returns the tinkered tool in p's hand.
func toolOf(p *player.Player) (item.Stack, bool) {
	held, _ := p.HeldItems()
	_, ok := held.Item().(tinker.Tool)
	return held, ok
}

// gainToolXP adds xp to the tool in hand; on a level up it says so.
func (s *session) gainToolXP(p *player.Player, n int) {
	held, ok := toolOf(p)
	if !ok {
		return
	}
	next, up := tinker.WithToolXP(held, n)
	_, off := p.HeldItems()
	p.SetHeldItems(next, off)
	if up {
		count(p, "tool_levelup", 1)
		s.advance(p.Tx(), advancement.ToolLevel)
		if tinker.ToolLevel(tinker.ToolXP(next)) >= 5 {
			s.advance(p.Tx(), advancement.Tool5)
		}
		p.Messagef(i18n.T(i18n.Of(p), "tinker.levelUp"), naming.ItemName(i18n.Of(p), next), tinker.ToolLevel(tinker.ToolXP(next)))
	}
}

// reinforce gives a reinforced tool a chance to take back the durability a use
// cost.
func (s *session) reinforce(p *player.Player) {
	held, ok := toolOf(p)
	if !ok {
		return
	}
	if l := tinker.ToolMods(held)["reinforced"]; l > 0 && held.Durability() < held.MaxDurability() && s.rand.IntN(100) < 14*l {
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
			haste = tinker.ToolMods(held)["haste"]
		}
		if haste > 0 {
			p.AddEffect(effect.NewAmbient(effect.Haste, haste, 3*time.Second))
		}
	}
}
