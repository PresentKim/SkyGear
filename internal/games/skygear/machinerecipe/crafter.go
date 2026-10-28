package machinerecipe

import (
	"fmt"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/internal/games/skygear/naming"
	"github.com/presentkim/skygear/internal/games/skygear/wintext"
)

// CrafterReq is one ingredient of a recipe: an item (or a tag of items) and
// how many.
type CrafterReq struct {
	Need  recipe.Item
	key   string
	Count int
}

// Crafter is a crafting-table recipe the auto crafter can make.
type Crafter struct {
	ID     string
	Recipe recipe.Recipe
	Out    item.Stack
	OutKey string
	Reqs   []CrafterReq
}

// reqKey identifies an ingredient; equal ingredients of a recipe are added.
func reqKey(in recipe.Item) string {
	switch in := in.(type) {
	case item.Stack:
		name, meta := in.Item().EncodeItem()
		if _, variants := in.Value("variants"); variants {
			return "v:" + name
		}
		return fmt.Sprintf("s:%s:%d", name, meta)
	case recipe.ItemTag:
		return "t:" + in.Tag()
	}
	return ""
}

var (
	crafterOnce sync.Once
	crafterList []*Crafter
	crafterIDs  map[string]*Crafter
)

// Crafters lists the recipes the auto crafter can make: the shaped and
// shapeless ones of the crafting table (their shape does not matter), in
// registration order. The game's recipes are registered in init and the
// vanilla ones when the server is made, before the game, so the list is
// built once.
func Crafters() []*Crafter {
	crafterOnce.Do(buildCrafterRecipes)
	return crafterList
}

// ResetCrafters makes Crafters work the list out again, for tests that
// register recipes after it was first built.
func ResetCrafters() { crafterOnce = sync.Once{} }

func buildCrafterRecipes() {
	seen := map[string]int{}
	var list []*Crafter
	ids := map[string]*Crafter{}
	for _, r := range recipe.Recipes() {
		if r.Block() != "crafting_table" {
			continue
		}
		switch r.(type) {
		case recipe.Shaped, recipe.Shapeless:
		default:
			continue
		}
		outs := r.Output()
		if len(outs) != 1 || outs[0].Empty() {
			continue
		}
		cr := &Crafter{Recipe: r, Out: outs[0]}
		index := map[string]int{}
		for _, in := range r.Input() {
			if in == nil || in.Empty() {
				continue
			}
			k := reqKey(in)
			if i, ok := index[k]; ok {
				cr.Reqs[i].Count += in.Count()
				continue
			}
			index[k] = len(cr.Reqs)
			cr.Reqs = append(cr.Reqs, CrafterReq{Need: in, key: k, Count: in.Count()})
		}
		if len(cr.Reqs) == 0 {
			continue
		}
		base := naming.ItemKey(outs[0].Item())
		cr.OutKey = base
		cr.ID = fmt.Sprintf("%s#%d", base, seen[base])
		seen[base]++
		list = append(list, cr)
		ids[cr.ID] = cr
	}
	crafterList, crafterIDs = list, ids
}

// CrafterByID returns the recipe with an id.
func CrafterByID(id string) *Crafter {
	crafterOnce.Do(buildCrafterRecipes)
	return crafterIDs[id]
}

// MatchReq reports whether a stack is the ingredient (the matching of the
// crafting grid).
func MatchReq(has item.Stack, req CrafterReq) bool {
	if has.Empty() {
		return false
	}
	switch need := req.Need.(type) {
	case item.Stack:
		if _, variants := need.Value("variants"); variants {
			a, _ := has.Item().EncodeItem()
			b, _ := need.Item().EncodeItem()
			return a == b
		}
		return has.Comparable(need)
	case recipe.ItemTag:
		name, _ := has.Item().EncodeItem()
		return need.Contains(name)
	}
	return false
}

// Wants reports whether a recipe uses an item as an ingredient.
func (cr *Crafter) Wants(it item.Stack) bool {
	for _, req := range cr.Reqs {
		if MatchReq(it, req) {
			return true
		}
	}
	return false
}

// ReqName is the name of an ingredient for a tooltip.
func ReqName(lang string, req CrafterReq) string {
	switch need := req.Need.(type) {
	case item.Stack:
		return naming.ItemName(lang, need)
	case recipe.ItemTag:
		return i18n.T(lang, "crafter.anyOf") + strings.TrimPrefix(need.Tag(), "minecraft:")
	}
	return ""
}

// Tip is the tooltip of a recipe: the result and what it takes.
func (cr *Crafter) Tip(lang string) []string {
	lines := []string{"§7" + i18n.T(lang, "crafter.needs")}
	for _, req := range cr.Reqs {
		lines = append(lines, fmt.Sprintf("§f%d× %s", req.Count, ReqName(lang, req)))
	}
	return lines
}

// Preview is the item shown on the recipe slot: the result with what it
// takes.
func (cr *Crafter) Preview() item.Stack {
	lore := []wintext.Text{{wintext.Raw("§7"), wintext.Key("crafter.needs")}}
	for _, req := range cr.Reqs {
		line := wintext.Text{wintext.Raw(fmt.Sprintf("§f%d× ", req.Count))}
		switch need := req.Need.(type) {
		case item.Stack:
			line = append(line, wintext.Item(need))
		case recipe.ItemTag:
			line = append(line, wintext.Key("crafter.anyOf"), wintext.Raw(strings.TrimPrefix(need.Tag(), "minecraft:")))
		}
		lore = append(lore, line)
	}
	return wintext.With(cr.Out, wintext.Text{wintext.Raw("§r§f"), wintext.Item(cr.Out)}, lore...)
}
