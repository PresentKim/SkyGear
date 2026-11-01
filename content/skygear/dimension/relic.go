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

// gradeEssences is how many monster essences (of any monster) raising a relic
// from grade g costs: gradeEssences[g-1]. A run gives about three essences
// for four kills (records.go), so the first raise takes some five hunts of the
// first monster, and each next one doubles.
var gradeEssences = []int{320, 640, 1280, 2560}

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
// the materials and the essences of any monster.
func relicCost(kind string, grade int) ([]item.Stack, int) {
	times := max(1, grade)
	var mats []item.Stack
	for _, m := range relicMaterials[kind] {
		mats = append(mats, m.Grow(m.Count()*(times-1)))
	}
	ess := 0
	if grade >= 1 {
		ess = gradeEssences[grade-1]
	}
	return mats, ess
}

// essences counts the monster essences in p's inventory.
func essences(p *player.Player) int {
	n := 0
	for _, it := range p.Inventory().Items() {
		if _, ok := it.Item().(sgitem.Essence); ok {
			n += it.Count()
		}
	}
	return n
}

// afford reports whether p has the materials and essences.
func afford(p *player.Player, mats []item.Stack, ess int) bool {
	for _, m := range mats {
		if core.CountPlain(p.Inventory(), m.Item()) < m.Count() {
			return false
		}
	}
	return essences(p) >= ess
}

// pay takes the materials and essences from p (afford must hold).
func pay(p *player.Player, mats []item.Stack, ess int) {
	for _, m := range mats {
		core.TakePlain(p.Inventory(), m.Item(), m.Count())
	}
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if ess == 0 {
			break
		}
		if _, ok := it.Item().(sgitem.Essence); ok {
			n := min(ess, it.Count())
			_ = inv.SetItem(slot, it.Grow(-n))
			ess -= n
		}
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
	mats, ess := relicCost(sel, grade)
	// the numbers are those of the grade carried, or of grade 1
	var b strings.Builder
	b.WriteString(i18n.T(lang, "relic."+sel+".about") + "\n" + describeRelic(lang, sel, max(grade, 1)) + "\n\n")
	var costs []string
	for _, m := range mats {
		costs = append(costs, fmt.Sprintf("%s ×%d", naming.ItemName(lang, m), m.Count()))
	}
	if ess > 0 {
		costs = append(costs, fmt.Sprintf(i18n.T(lang, "dimension.relic.essences"), ess))
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
	mats, ess := relicCost(kind, grade)
	if !afford(p, mats, ess) {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "dimension.relic.short"))
		return
	}
	pay(p, mats, ess)
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
