package dimension

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// The relics: made and raised a grade at the relic table (after the research
// "relics"), the spinning blade also on a crafting table (after the research
// "dimension"). A hunter goes in with the loadout it picked in the second
// dimension's window, or its best relics when it picked none. Values are
// drafts.

// RelicsResearch is the research that opens the relic table.
const RelicsResearch = "relics"

// relicMaterials is what a relic of grade 1 costs; each grade up costs them
// times the grade it comes from, and essences.
var relicMaterials = map[string][]item.Stack{
	sgitem.RelicBlade:     {item.NewStack(item.IronIngot{}, 8), item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 2)},
	sgitem.RelicCrossbow:  {item.NewStack(block.String{}, 8), item.NewStack(item.IronIngot{}, 4), item.NewStack(item.Flint{}, 4)},
	sgitem.RelicFlameRing: {item.NewStack(item.BlazePowder{}, 4), item.NewStack(item.IronIngot{}, 4)},
	sgitem.RelicLightning: {item.NewStack(item.CopperIngot{}, 8), item.NewStack(item.RedstoneWire{}, 8)},
	sgitem.RelicBoomerang: {item.NewStack(sgitem.Ingot{Metal: sgitem.MetalInvar}, 4), item.NewStack(item.LapisLazuli{}, 8)},
	sgitem.RelicLife:      {item.NewStack(item.GoldenApple{}, 1), item.NewStack(item.GoldIngot{}, 4)},
	sgitem.RelicWind:      {item.NewStack(block.String{}, 8), item.NewStack(sgitem.Ingot{Metal: sgitem.MetalElectrum}, 4)},
	sgitem.RelicMagnet:    {item.NewStack(item.IronIngot{}, 8), item.NewStack(item.RedstoneWire{}, 4)},
	sgitem.RelicHourglass: {item.NewStack(sgitem.FairyHourglass{}, 1)},
	sgitem.RelicLens:      {item.NewStack(block.Glass{}, 8), item.NewStack(item.GoldIngot{}, 2)},
	sgitem.RelicRegen:     {item.NewStack(sgitem.Petal{Colour: "pink"}, 8), item.NewStack(item.GoldIngot{}, 2)},
}

// Raising a relic takes the essences of its own monsters: its early one for
// every grade, and from grade 3 its late one too, so every relic can be
// raised a little soon and only the hunts far along raise it all the way.
// A run gives about three essences for four kills (records.go): a raise is
// two or three hunts of that monster.

// relicMobs are each relic's early and late monster.
var relicMobs = map[string][2]string{
	sgitem.RelicBlade:     {sgitem.MobZombie, sgitem.MobWither},
	sgitem.RelicCrossbow:  {sgitem.MobSkeleton, sgitem.MobPiglin},
	sgitem.RelicFlameRing: {sgitem.MobCreeper, sgitem.MobBlaze},
	sgitem.RelicLightning: {sgitem.MobCreeper, sgitem.MobWitch},
	sgitem.RelicBoomerang: {sgitem.MobDrowned, sgitem.MobEnder},
	sgitem.RelicLife:      {sgitem.MobZombie, sgitem.MobWitch},
	sgitem.RelicWind:      {sgitem.MobSpider, sgitem.MobEnder},
	sgitem.RelicMagnet:    {sgitem.MobSlime, sgitem.MobPiglin},
	sgitem.RelicHourglass: {sgitem.MobSlime, sgitem.MobEnder},
	sgitem.RelicLens:      {sgitem.MobSpider, sgitem.MobBlaze},
	sgitem.RelicRegen:     {sgitem.MobSlime, sgitem.MobWither},
}

// gradeEssences are how many essences of the early and the late monster
// raising a relic from grade g costs: gradeEssences[g-1].
var gradeEssences = [][2]int{{150, 0}, {300, 0}, {400, 200}, {600, 400}}

func init() {
	for _, kind := range append(append([]string(nil), sgitem.RelicWeapons...), sgitem.RelicCharms...) {
		world.RegisterItem(sgitem.Relic{Kind: kind})
	}
	// the first relic on a crafting table: iron, steel and a stick
	recipe.Register(recipe.NewShapeless([]recipe.Item{
		item.NewStack(item.IronIngot{}, 1), item.NewStack(item.IronIngot{}, 1), item.NewStack(item.IronIngot{}, 1),
		item.NewStack(item.IronIngot{}, 1), item.NewStack(item.IronIngot{}, 1), item.NewStack(item.IronIngot{}, 1),
		item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 1), item.NewStack(sgitem.Ingot{Metal: sgitem.MetalSteel}, 1),
		item.NewStack(item.Stick{}, 1),
	}, sgitem.RelicStack(sgitem.RelicBlade, 1), "crafting_table"))
	machine.Kinds[machine.KindRelicTable].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
		if !s.Debounced(p) {
			ShowRelicTable(s, p, "")
		}
		return true
	}
}

// relicKinds are every relic, the weapons first.
func relicKinds() []string {
	return append(append([]string(nil), sgitem.RelicWeapons...), sgitem.RelicCharms...)
}

// relicCost is what raising a relic of kind from grade (0: making it) costs:
// the materials and the essences of its monsters.
func relicCost(kind string, grade int) []item.Stack {
	times := max(1, grade)
	var mats []item.Stack
	for _, m := range relicMaterials[kind] {
		mats = append(mats, m.Grow(m.Count()*(times-1)))
	}
	if grade >= 1 {
		for i, n := range gradeEssences[grade-1] {
			if n > 0 {
				mats = append(mats, item.NewStack(sgitem.Essence{Mob: relicMobs[kind][i]}, n))
			}
		}
	}
	return mats
}

// afford reports whether p has all of cost.
func afford(p *player.Player, cost []item.Stack) bool {
	for _, m := range cost {
		if core.CountPlain(p.Inventory(), m.Item()) < m.Count() {
			return false
		}
	}
	return true
}

// pay takes cost from p (afford must hold).
func pay(p *player.Player, cost []item.Stack) {
	for _, m := range cost {
		core.TakePlain(p.Inventory(), m.Item(), m.Count())
	}
}

// ShowRelicTable is the relic table: every relic with what the hunter has of
// it, and making or raising the chosen one.
func ShowRelicTable(s *core.Session, p *player.Player, sel string) {
	lang := i18n.Of(p)
	if sel == "" {
		sel = sgitem.RelicBlade
	}
	var rows []ui.ListRow
	for _, kind := range relicKinds() {
		info := "§8-"
		if g, ok := carried(p, kind); ok {
			info = strings.Repeat("§e★", g)
		}
		rows = append(rows, ui.ListRow{
			Label: ui.ListRowText(i18n.T(lang, "relic."+kind), info), Icon: formicon.Of(item.NewStack(sgitem.Relic{Kind: kind}, 1)), Selected: kind == sel,
			Run: func(p *player.Player, _ *world.Tx) { ShowRelicTable(s, p, kind) },
		})
	}
	grade, has := carried(p, sel)
	mats := relicCost(sel, grade)
	// the numbers are those of the grade carried, or of grade 1
	var b strings.Builder
	b.WriteString(describeRelic(lang, sel, max(grade, 1)) + "\n\n")
	// each cost with what the hunter has of it, red while short
	var costs []string
	for _, m := range mats {
		have, col := core.CountPlain(p.Inventory(), m.Item()), "§a"
		if have < m.Count() {
			col = "§c"
		}
		costs = append(costs, fmt.Sprintf("%s %s%d§7/%d", naming.ItemName(lang, m), col, have, m.Count()))
	}
	var actions []ui.Option
	switch {
	case !progress.Researched(s, RelicsResearch):
		b.WriteString(i18n.T(lang, "dimension.relic.locked"))
	case has && grade >= sgitem.RelicMaxGrade:
		b.WriteString(i18n.T(lang, "dimension.relic.max"))
	default:
		key := "dimension.relic.make"
		if has {
			key = "dimension.relic.raise"
		}
		fmt.Fprintf(&b, i18n.T(lang, key+".cost"), strings.Join(costs, ", "))
		actions = append(actions, s.Opt(fmt.Sprintf(i18n.T(lang, key), grade+1), "", func(p *player.Player, tx *world.Tx) {
			craftRelic(s, p, tx, sel)
			ShowRelicTable(s, p, sel)
		}))
	}
	ui.ListSplitMenu(p, i18n.T(lang, "machine.relictable.name"), ui.ListSplit{Rows: rows, Body: b.String(), Actions: actions})
}

// craftRelic makes kind or raises the one p carries a grade, if p pays.
func craftRelic(s *core.Session, p *player.Player, tx *world.Tx, kind string) {
	grade, has := carried(p, kind)
	if !progress.Researched(s, RelicsResearch) || grade >= sgitem.RelicMaxGrade {
		return
	}
	mats := relicCost(kind, grade)
	if !afford(p, mats) {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "dimension.relic.short"))
		return
	}
	pay(p, mats)
	next := sgitem.RelicStack(kind, grade+1)
	if has {
		inv := p.Inventory()
		for slot, it := range inv.Slots() {
			if r, ok := it.Item().(sgitem.Relic); ok && r.Kind == kind && sgitem.RelicGrade(it) == grade {
				_ = inv.SetItem(slot, next)
				break
			}
		}
	} else {
		s.GiveItem(p, tx, p.Position(), next)
	}
	tx.PlaySound(p.Position(), sound.AnvilUse{})
}
