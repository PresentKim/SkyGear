package sgitem

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/skygear/common/i18n"
)

// Relics are the gear of the second dimension's hunts (dimension package):
// weapons that fight on their own and charms that make the hunter stronger.
// A relic has a grade from 1 to RelicMaxGrade, kept in the stack.

// The relics.
const (
	RelicBlade     = "spin_blade"
	RelicCrossbow  = "crossbow"
	RelicFlameRing = "flame_ring"
	RelicLightning = "lightning_staff"
	RelicBoomerang = "frost_boomerang"

	RelicLife      = "life_charm"
	RelicWind      = "wind_boots"
	RelicMagnet    = "magnet_ring"
	RelicHourglass = "hourglass_charm"
	RelicLens      = "magnifier"
	RelicRegen     = "regen_necklace"
)

// RelicWeapons and RelicCharms are the relics of each sort, in order.
var (
	RelicWeapons = []string{RelicBlade, RelicCrossbow, RelicFlameRing, RelicLightning, RelicBoomerang}
	RelicCharms  = []string{RelicLife, RelicWind, RelicMagnet, RelicHourglass, RelicLens, RelicRegen}
)

// RelicMaxGrade is the highest grade of a relic.
const RelicMaxGrade = 5

// relicGradeKey is the stack value of a relic's grade.
const relicGradeKey = "relic_grade"

// Relic is a relic item.
type Relic struct{ Kind string }

func (r Relic) EncodeItem() (string, int16) { return "skygear:relic_" + r.Kind, 0 }

func (r Relic) Name() string { return i18n.T(i18n.Default, "relic."+r.Kind) }

func (Relic) Category() category.Category { return category.Equipment() }

func (Relic) MaxCount() int { return 1 }

// IsWeapon reports whether the relic fights (else it is a charm).
func (r Relic) IsWeapon() bool {
	for _, w := range RelicWeapons {
		if w == r.Kind {
			return true
		}
	}
	return false
}

// RelicGrade is the grade of a relic stack (1 when unset).
func RelicGrade(st item.Stack) int {
	if g, ok := st.Value(relicGradeKey); ok {
		if n, ok := g.(int); ok && n >= 1 {
			return min(n, RelicMaxGrade)
		}
		if n, ok := g.(int32); ok && n >= 1 {
			return min(int(n), RelicMaxGrade)
		}
		if n, ok := g.(int64); ok && n >= 1 {
			return min(int(n), RelicMaxGrade)
		}
	}
	return 1
}

// RelicStack is a relic of kind at grade, named with its grade.
func RelicStack(kind string, grade int) item.Stack {
	grade = max(1, min(grade, RelicMaxGrade))
	r := Relic{Kind: kind}
	return item.NewStack(r, 1).WithValue(relicGradeKey, grade).
		WithCustomName(fmt.Sprintf("§r§d%s §e%s", r.Name(), stars(grade))).
		WithLore(i18n.T(i18n.Default, "relic."+kind+".about"))
}

// stars draws a grade as filled stars.
func stars(n int) string {
	s := ""
	for range n {
		s += "★"
	}
	return s
}
