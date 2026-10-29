package tinker

// The tinker table: a click opens a menu to make a tool (kind, head metal,
// handle) or to put modifiers on the tool in hand.

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
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// EquipToolFor puts the first tool in p's inventory that fits b in p's hand,
// moving what p held to that tool's slot, when the held item does not fit b.
func EquipToolFor(p *player.Player, b world.Block) {
	held, off := p.HeldItems()
	if fitsBlock(b, held) {
		return
	}
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		if st.Empty() || !fitsBlock(b, st) {
			continue
		}
		if sgitem.IsHammer(st) {
			// A hammer turns the drops into something else: never pick it
			// for the player.
			continue
		}
		_ = inv.SetItem(slot, held)
		p.SetHeldItems(st, off)
		return
	}
}

func ShowTable(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.body"),
		ui.Option{Label: i18n.T(lang, "tinker.make"), Run: func(p *player.Player, tx *world.Tx) { showToolKinds(s, p, tx, pos) }},
		ui.Option{Label: i18n.T(lang, "tinker.modify"), Run: func(p *player.Player, tx *world.Tx) { showModifiers(s, p, tx, pos) }},
	)
}

// showToolKinds lets the player pick the kind of tool to make.
func showToolKinds(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, k := range ToolKinds {
		slots = append(slots, ui.ItemSlot{
			Icon: ui.ItemIcon(Tool{Kind: k, Head: "iron", Handle: "wood"}),
			Tip:  "§f" + i18n.T(lang, "tool."+k) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.headCost"), ToolHeadCost[k]),
			Run:  func(p *player.Player, tx *world.Tx) { showToolHeads(s, p, tx, pos, k) },
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.make")), i18n.T(lang, "tinker.pickKind"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { ShowTable(s, p, tx, pos) }))
}

// showToolHeads lists the head metals open to the save.
func showToolHeads(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind string) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, h := range HeadMaterials {
		if !progress.Researched(s, h.Research) && !progress.LocalOf(s).RecipesFree {
			continue
		}
		have := core.CountPlain(p.Inventory(), h.Item)
		need := ToolHeadCost[kind]
		mark := "§f"
		if have < need {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(h.Item),
			Count: mark + fmt.Sprint(have),
			Tip: "§f" + i18n.T(lang, "tmat."+h.ID) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.headStats"), h.Dur, h.Eff, h.Attack+ToolAttackBonus[kind]) +
				"\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.need"), need),
			Run: func(p *player.Player, tx *world.Tx) { showToolHandles(s, p, tx, pos, kind, h) },
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tool."+kind)), i18n.T(lang, "tinker.pickHead"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { showToolKinds(s, p, tx, pos) }))
}

// showToolHandles lists the handles and makes the tool on a pick.
func showToolHandles(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind string, head HeadMaterial) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, hd := range HandleMaterials {
		have := core.CountPlain(p.Inventory(), hd.Item)
		mark := "§f"
		if have < 1 {
			mark = "§c"
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  ui.ItemIcon(hd.Item),
			Count: mark + fmt.Sprint(have),
			Tip:   "§f" + i18n.T(lang, "thandle."+hd.ID) + "\n§7" + fmt.Sprintf(i18n.T(lang, "tinker.handleStats"), hd.DurMul*100, hd.SpeedMul*100, hd.Attack),
			Run: func(p *player.Player, tx *world.Tx) {
				MakeTool(s, p, tx, pos, kind, head, hd)
				showToolHandles(s, p, tx, pos, kind, head)
			},
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tmat."+head.ID)+" "+i18n.T(lang, "tool."+kind)), i18n.T(lang, "tinker.pickHandle"), slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { showToolHeads(s, p, tx, pos, kind) }))
}

// MakeTool takes the materials and gives the tool; it does nothing when they are
// missing.
func MakeTool(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, kind string, head HeadMaterial, hd HandleMaterial) bool {
	inv := p.Inventory()
	need := ToolHeadCost[kind]
	if core.CountPlain(inv, head.Item) < need || core.CountPlain(inv, hd.Item) < 1 {
		return false
	}
	core.TakePlain(inv, head.Item, need)
	core.TakePlain(inv, hd.Item, 1)
	tool := DescribeTool(item.NewStack(Tool{Kind: kind, Head: head.ID, Handle: hd.ID}, 1))
	s.GiveItem(p, tx, p.Position(), tool)
	progress.Advance(s, tx, advancement.ToolMade)
	achievement.Count(p, "tool_made", 1)
	tx.PlaySound(pos.Vec3Centre(), sound.AnvilUse{})
	return true
}

// showModifiers lists the modifiers for the tool in hand.
func showModifiers(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	lang := i18n.Of(p)
	held, _ := p.HeldItems()
	t, ok := held.Item().(Tool)
	if !ok {
		ui.Menu(p, ui.SubTitle(i18n.T(lang, "tinker.title"), i18n.T(lang, "tinker.modify")), i18n.T(lang, "tinker.holdTool"),
			ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { ShowTable(s, p, tx, pos) }))
		return
	}
	mods := ToolMods(held)
	lvl := ToolLevel(ToolXP(held))
	var slots []ui.ItemSlot
	for _, m := range ToolModifiers {
		cur := mods[m.ID]
		have := core.CountPlain(p.Inventory(), m.Item)
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
				ApplyModifier(s, p, tx, pos, m)
				showModifiers(s, p, tx, pos)
			},
		})
	}
	ui.ItemGrid(p, ui.SubTitle(i18n.T(lang, "tinker.title"), t.Name()+fmt.Sprintf(" Lv.%d", lvl)), fmt.Sprintf(i18n.T(lang, "tinker.slots"), ModsUsed(mods), ToolSlots(lvl)), slots,
		ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) { ShowTable(s, p, tx, pos) }))
}

// ApplyModifier adds a level of a modifier to the tool in hand, if the tool has
// a free slot, the modifier is not at its limit and the materials are there.
func ApplyModifier(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, m ToolModifier) bool {
	held, off := p.HeldItems()
	if _, ok := held.Item().(Tool); !ok {
		return false
	}
	mods := ToolMods(held)
	if mods[m.ID] >= m.Max || ModsUsed(mods) >= ToolSlots(ToolLevel(ToolXP(held))) || !core.TakePlain(p.Inventory(), m.Item, m.Cost) {
		return false
	}
	mods[m.ID]++
	p.SetHeldItems(DescribeTool(WithToolMods(held, mods)), off)
	progress.Advance(s, tx, advancement.ToolMod)
	achievement.Count(p, "tool_mod", 1)
	tx.PlaySound(pos.Vec3Centre(), sound.AnvilUse{})
	return true
}

// ToolOf returns the tinkered tool in p's hand.
func ToolOf(p *player.Player) (item.Stack, bool) {
	held, _ := p.HeldItems()
	_, ok := held.Item().(Tool)
	return held, ok
}

// GainXP adds xp to the tool in hand; on a level up it says so.
func GainXP(s *core.Session, p *player.Player, n int) {
	held, ok := ToolOf(p)
	if !ok {
		return
	}
	next, up := WithToolXP(held, n)
	_, off := p.HeldItems()
	p.SetHeldItems(next, off)
	if up {
		achievement.Count(p, "tool_levelup", 1)
		progress.Advance(s, p.Tx(), advancement.ToolLevel)
		if ToolLevel(ToolXP(next)) >= 5 {
			progress.Advance(s, p.Tx(), advancement.Tool5)
		}
		p.Messagef(i18n.T(i18n.Of(p), "tinker.levelUp"), naming.ItemName(i18n.Of(p), next), ToolLevel(ToolXP(next)))
	}
}

// Reinforce gives a reinforced tool a chance to take back the durability a use
// cost.
func Reinforce(s *core.Session, p *player.Player) {
	held, ok := ToolOf(p)
	if !ok {
		return
	}
	if l := ToolMods(held)["reinforced"]; l > 0 && held.Durability() < held.MaxDurability() && s.Rand.IntN(100) < 14*l {
		_, off := p.HeldItems()
		p.SetHeldItems(held.WithDurability(held.Durability()+1), off)
	}
}

// TickEffects gives the players haste from a tool they hold, every second
// for the next few seconds.
func TickEffects(s *core.Session, tx *world.Tx) {
	for e := range tx.Players() {
		p := e.(*player.Player)
		haste := 0
		if held, ok := ToolOf(p); ok {
			haste = ToolMods(held)["haste"]
		}
		if haste > 0 {
			p.AddEffect(effect.NewAmbient(effect.Haste, haste, 3*time.Second))
		}
	}
}

// veinLimit is the most blocks one vein mining breaks, the mined one
// included. Each block past the mined one also costs veinFoodCost food
// points: a full food bar (20) pays for veinLimit blocks.
const (
	VeinLimit    = 256
	veinFoodCost = 20.0 / VeinLimit
)

// VeinBlocks are the blocks of the same kind as the one at start connected
// to it face to face (with diagonal also edge to edge and corner to corner),
// start first, up to limit; none when the hammer does not crush it. canBuild
// filters the blocks a player may break.
func VeinBlocks(tx *world.Tx, start cube.Pos, limit int, diagonal bool, canBuild func(cube.Pos) bool) []cube.Pos {
	b := tx.Block(start)
	it, ok := b.(world.Item)
	if _, crushable := sgitem.HammerDrop(b); !ok || !crushable {
		return nil
	}
	name, meta := it.EncodeItem()
	same := func(pos cube.Pos) bool {
		other, ok := tx.Block(pos).(world.Item)
		if !ok {
			return false
		}
		otherName, otherMeta := other.EncodeItem()
		return otherName == name && otherMeta == meta && canBuild(pos)
	}
	out := []cube.Pos{start}
	seen := map[cube.Pos]bool{start: true}
	neighbours := veinNeighbours(diagonal)
	for i := 0; i < len(out) && len(out) < limit; i++ {
		for _, d := range neighbours {
			n := out[i].Add(d)
			if seen[n] || n.OutOfBounds(tx.Range()) {
				continue
			}
			seen[n] = true
			if same(n) {
				if out = append(out, n); len(out) >= limit {
					break
				}
			}
		}
	}
	return out
}

// VeinBudget is how many blocks past the mined one p can break with held.
func VeinBudget(s *core.Session, p *player.Player, held item.Stack) int {
	return max(0, min(veinToolBudget(s, held), veinFoodBudget(s, p)))
}

// PayVeinFood takes the food of n vein mined blocks from p.
func PayVeinFood(s *core.Session, p *player.Player, n int) {
	owed := VeinFood.Of(s)[p.H()] + float64(n)*veinFoodCost
	whole := int(owed + 1e-9)
	VeinFood.Of(s)[p.H()] = max(0, owed-float64(whole))
	if whole > 0 {
		p.SetFood(p.Food() - whole)
	}
}

// VeinFood is the fraction of a food point each player owes for vein
// mining.
var VeinFood = core.NewPlayers[float64]()
