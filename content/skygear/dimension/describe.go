package dimension

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// What an item does in a hunt, the way survivor games put it: the kind and
// grade, one line of what it does, then its numbers in a row ("Damage 4 ·
// Every 1 s"). A level-up card says only what the next level adds (gear.go).
// The numbers come from the same constants as the fight: a weapon relic's at
// its grade and run level 1.

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
	return i18n.T(lang, "dimension.desc.other")
}

// gradeStars is grade as filled stars out of the highest grade.
func gradeStars(grade int) string {
	return "§e" + strings.Repeat("★", grade) + "§8" + strings.Repeat("★", sgitem.RelicMaxGrade-grade)
}

// describeRelic is what a relic of kind does at grade.
func describeRelic(lang, kind string, grade int) string {
	r := sgitem.Relic{Kind: kind}
	g := float64(grade - 1)
	about := "§7" + i18n.T(lang, "relic."+kind+".about")
	if !r.IsWeapon() {
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
		return fmt.Sprintf(i18n.T(lang, "dimension.desc.charm"), gradeStars(grade)) + "\n" + about + "\n" +
			fmt.Sprintf(i18n.T(lang, "relic."+kind+".stats"), num(v))
	}
	dmg := num(weaponDamage(kind) * (1 + 0.25*g))
	sec := func(f float64) string { return fmt.Sprintf(i18n.T(lang, "dimension.stat.sec"), num(f)) }
	blocks := func(f float64) string { return fmt.Sprintf(i18n.T(lang, "dimension.stat.blocks"), num(f)) }
	count := func(n int) string { return fmt.Sprintf(i18n.T(lang, "dimension.stat.count"), n) }
	stats := [][2]string{{"damage", dmg}}
	switch kind {
	case sgitem.RelicBlade:
		stats = append(stats, [2]string{"blades", count(2)}, [2]string{"area", blocks(bladeOuter)})
	case sgitem.RelicCrossbow:
		stats = append(stats, [2]string{"every", sec(1)}, [2]string{"reach", blocks(16)})
	case sgitem.RelicFlameRing:
		stats = append(stats, [2]string{"every", sec(0.5)}, [2]string{"area", blocks(3)})
	case sgitem.RelicLightning:
		stats = append(stats, [2]string{"every", sec(2)}, [2]string{"bolts", count(3)})
	case sgitem.RelicBoomerang:
		stats = append(stats, [2]string{"every", sec(3)}, [2]string{"far", blocks(9)})
	}
	var parts []string
	for _, st := range stats {
		parts = append(parts, "§7"+i18n.T(lang, "dimension.stat."+st[0])+" §f"+st[1])
	}
	return fmt.Sprintf(i18n.T(lang, "dimension.desc.weapon"), gradeStars(grade)) + "\n" + about + "\n" + strings.Join(parts, " §8· ")
}

// weaponDamage is a weapon relic's damage at grade 1, run level 1.
func weaponDamage(kind string) float64 {
	switch kind {
	case sgitem.RelicBlade:
		return bladeDamage
	case sgitem.RelicCrossbow:
		return 6
	case sgitem.RelicFlameRing:
		return 2
	case sgitem.RelicLightning:
		return 8
	case sgitem.RelicBoomerang:
		return 5
	}
	return 0
}

// levelGains is what level of a weapon relic of kind adds over the one
// before: more damage, and for some levels more blades, arrows and so on.
func levelGains(lang, kind string, level int) string {
	gains := []string{i18n.T(lang, "dimension.lv.damage")}
	extra := ""
	switch kind {
	case sgitem.RelicBlade:
		if level == 3 || level == 5 {
			extra = "blades"
		}
	case sgitem.RelicCrossbow:
		if level == 2 || level == 4 {
			extra = "arrows"
		}
	case sgitem.RelicFlameRing:
		extra = "area"
	case sgitem.RelicLightning:
		extra = "bolts"
	case sgitem.RelicBoomerang:
		extra = "far"
	}
	if extra != "" {
		gains = append(gains, i18n.T(lang, "dimension.lv."+extra))
	}
	return "§f" + strings.Join(gains, "\n§f")
}

// num is f with at most one decimal, none when whole.
func num(f float64) string { return strconv.FormatFloat(float64(int(f*10+0.5))/10, 'f', -1, 64) }
