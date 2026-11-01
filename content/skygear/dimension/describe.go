package dimension

import (
	"fmt"
	"strconv"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// What an item does in a hunt, with its numbers, for the kit screen and the
// relic table. The numbers come from the same constants as the fight
// (gear.go): a weapon relic's at its grade and run level 1.

// describe is what it does in a hunt, in lines.
func describe(lang string, it item.Stack) string {
	if it.Empty() {
		return ""
	}
	if r, ok := it.Item().(sgitem.Relic); ok {
		return describeRelic(lang, r.Kind, sgitem.RelicGrade(it))
	}
	if a, ok := it.Item().(item.Armour); ok {
		return fmt.Sprintf(i18n.T(lang, "dimension.desc.armour"), num(a.DefencePoints()))
	}
	if w, ok := it.Item().(item.Weapon); ok {
		return fmt.Sprintf(i18n.T(lang, "dimension.desc.melee"), num(w.AttackDamage()), num(float64(swingTicks(it))/20), num(swingReach))
	}
	return i18n.T(lang, "dimension.desc.other")
}

// describeRelic is what a relic of kind does at grade.
func describeRelic(lang, kind string, grade int) string {
	r := sgitem.Relic{Kind: kind}
	g := float64(grade - 1)
	if !r.IsWeapon() {
		head := fmt.Sprintf(i18n.T(lang, "dimension.desc.charm"), grade)
		var v float64
		switch kind {
		case sgitem.RelicLife:
			v = 4 + 2*g
		case sgitem.RelicWind:
			v = 100 * (0.1 + 0.05*g)
		case sgitem.RelicMagnet:
			v = 100 * (0.5 + 0.15*g)
		case sgitem.RelicHourglass:
			v = 100 * (0.08 + 0.03*g)
		case sgitem.RelicLens:
			v = 100 * (0.15 + 0.05*g)
		case sgitem.RelicRegen:
			v = 0.3 + 0.15*g
		}
		return head + "\n" + fmt.Sprintf(i18n.T(lang, "relic."+kind+".stats"), num(v))
	}
	dmg := func(base float64) string { return num(base * (1 + 0.25*g)) }
	stats := i18n.T(lang, "relic."+kind+".stats")
	switch kind {
	case sgitem.RelicBlade:
		stats = fmt.Sprintf(stats, 2, num(bladeInner), num(bladeOuter), dmg(bladeDamage))
	case sgitem.RelicCrossbow:
		stats = fmt.Sprintf(stats, num(16), dmg(6))
	case sgitem.RelicFlameRing:
		stats = fmt.Sprintf(stats, num(3), dmg(2))
	case sgitem.RelicLightning:
		stats = fmt.Sprintf(stats, num(12), 3, dmg(8))
	case sgitem.RelicBoomerang:
		stats = fmt.Sprintf(stats, num(9), dmg(5))
	}
	return fmt.Sprintf(i18n.T(lang, "dimension.desc.weapon"), grade) + "\n" + stats + "\n" + i18n.T(lang, "dimension.desc.growth")
}

// num is f with at most one decimal, none when whole.
func num(f float64) string { return strconv.FormatFloat(float64(int(f*10+0.5))/10, 'f', -1, 64) }
