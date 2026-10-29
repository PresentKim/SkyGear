package machinerecipe

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/internal/skygear/naming"
	"github.com/presentkim/skygear/internal/skygear/sgitem"
)

// Ingredient is an item with the count a recipe takes.
type Ingredient struct {
	Item  world.Item
	Count int
}

// Lab is one recipe of the laboratory.
type Lab struct {
	// Out is the essence made, OutCount how many at once.
	Out      sgitem.ResearchEssence
	OutCount int
	// In are the ingredients, the glass bottle among them.
	In []Ingredient
	// Ticks is how long a cycle takes.
	Ticks int
}

// labBottle is the glass bottle every recipe uses one of.
var labBottle = Ingredient{item.GlassBottle{}, 1}

// Labs are the recipes, drafts (open-questions.md).
var Labs = []Lab{
	{Out: sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}, OutCount: 2, Ticks: 160, In: []Ingredient{
		labBottle, {item.IronIngot{}, 1}, {item.RedstoneWire{}, 2}}},
	{Out: sgitem.ResearchEssence{Kind: sgitem.EssenceAlchemy}, OutCount: 1, Ticks: 240, In: []Ingredient{
		labBottle, {sgitem.Ingot{Metal: sgitem.MetalBronze}, 1}, {sgitem.Ingot{Metal: sgitem.MetalInvar}, 1}}},
	{Out: sgitem.ResearchEssence{Kind: sgitem.EssenceFarming}, OutCount: 1, Ticks: 240, In: []Ingredient{
		labBottle, {item.Wheat{}, 1}, {block.Carrot{}, 1}, {block.Potato{}, 1}, {item.Beetroot{}, 1}, {block.Pumpkin{}, 1}, {item.MelonSlice{}, 1}}},
	{Out: sgitem.ResearchEssence{Kind: sgitem.EssenceHunting}, OutCount: 1, Ticks: 320, In: []Ingredient{
		labBottle, {item.Gunpowder{}, 2}, {item.Bone{}, 2}, {item.Slimeball{}, 1}, {item.SpiderEye{}, 1}}},
	{Out: sgitem.ResearchEssence{Kind: sgitem.EssenceMagic}, OutCount: 1, Ticks: 320, In: []Ingredient{
		labBottle, {sgitem.Petal{Colour: "red"}, 1}, {sgitem.Petal{Colour: "yellow"}, 1}, {sgitem.Petal{Colour: "blue"}, 1}, {item.LapisLazuli{}, 1}, {item.AmethystShard{}, 1}}},
}

// LabByKind is the recipe of the essence of a kind, nil if there is none.
func LabByKind(kind string) *Lab {
	for i := range Labs {
		if Labs[i].Out.Kind == kind {
			return &Labs[i]
		}
	}
	return nil
}

// Ingredient is the ingredient of the recipe that the item is, if it is one.
func (r *Lab) Ingredient(it item.Stack) (Ingredient, bool) {
	k := naming.ItemKey(it.Item())
	for _, in := range r.In {
		if naming.ItemKey(in.Item) == k {
			return in, true
		}
	}
	return Ingredient{}, false
}

// LabMissing lists what a recipe still lacks of the ingredients.
func LabMissing(r *Lab, have map[string]int) []Ingredient {
	var missing []Ingredient
	for _, in := range r.In {
		if short := in.Count - have[naming.ItemKey(in.Item)]; short > 0 {
			missing = append(missing, Ingredient{in.Item, short})
		}
	}
	return missing
}

// LabIngredientsText lists the ingredients of a recipe in one line.
func LabIngredientsText(lang string, r *Lab) string {
	var parts []string
	for _, in := range r.In {
		if in.Count > 1 {
			parts = append(parts, fmt.Sprintf("%s ×%d", naming.ItemName(lang, item.NewStack(in.Item, 1)), in.Count))
		} else {
			parts = append(parts, naming.ItemName(lang, item.NewStack(in.Item, 1)))
		}
	}
	return strings.Join(parts, ", ")
}
