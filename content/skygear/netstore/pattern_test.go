package netstore

import (
	"os"
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// keysFor is the fake of the key lookup of tag slots; SUPERLITE_REAL=1 uses
// the real item keys of the tag's members.
func fakeKeys(r recipe.Item) []storagenet.Key {
	if os.Getenv("SUPERLITE_REAL") == "1" {
		return []storagenet.Key{KeyOf(item.NewStack(block.Log{Wood: block.OakWood()}, 1))}
	}
	return []storagenet.Key{"oak_log"}
}

func blank() item.Stack { return item.NewStack(Pattern{}, 1) }

// planksID registers a log -> 4 planks recipe and returns its crafter id.
func planksID(t *testing.T) string {
	t.Helper()
	out := item.NewStack(block.Planks{Wood: block.OakWood()}, 4)
	recipe.Register(recipe.NewShapeless([]recipe.Item{recipe.NewItemTag("minecraft:logs", 1)}, out, "crafting_table"))
	machinerecipe.ResetCrafters()
	for _, cr := range machinerecipe.Crafters() {
		if cr.OutKey == naming.ItemKey(out.Item()) && cr.Out.Count() == 4 {
			return cr.ID
		}
	}
	t.Fatal("recipe not listed")
	return ""
}

func TestTagIngredientOptions(t *testing.T) {
	id := planksID(t)
	p, ok := PatternOf(EncodeCrafting(blank(), id), fakeKeys)
	if !ok || p.Processing {
		t.Fatalf("got %v %v", p, ok)
	}
	want := []storagenet.Ingredient{{Count: 1, Options: fakeKeys(nil)}}
	if !reflect.DeepEqual(p.Inputs, want) {
		t.Errorf("inputs %v, want %v", p.Inputs, want)
	}
	planks := item.NewStack(block.Planks{Wood: block.OakWood()}, 1)
	if !reflect.DeepEqual(p.Outputs, []storagenet.Stack{{Key: KeyOf(planks), Count: 4}}) {
		t.Errorf("outputs %v", p.Outputs)
	}
}

func TestBlankIsNoPattern(t *testing.T) {
	if _, ok := PatternOf(blank(), fakeKeys); ok {
		t.Error("blank pattern reads as a pattern")
	}
	if _, ok := PatternOf(item.NewStack(block.Dirt{}, 1), fakeKeys); ok {
		t.Error("dirt reads as a pattern")
	}
	if got := EncodeCrafting(blank(), "no_such#9"); !isBlank(got) {
		t.Error("unknown recipe was written")
	}
}

func TestProcessingPattern(t *testing.T) {
	ore, ingot := item.NewStack(item.RawIron{}, 1), item.NewStack(item.IronIngot{}, 1)
	p, ok := PatternOf(EncodeProcessing(blank(), []item.Stack{ore, {}}, []item.Stack{ingot}), fakeKeys)
	if !ok || !p.Processing {
		t.Fatalf("got %v %v", p, ok)
	}
	if !reflect.DeepEqual(p.Inputs, []storagenet.Ingredient{{Count: 1, Options: []storagenet.Key{KeyOf(ore)}}}) {
		t.Errorf("inputs %v", p.Inputs)
	}
	if !reflect.DeepEqual(p.Outputs, []storagenet.Stack{{Key: KeyOf(ingot), Count: 1}}) {
		t.Errorf("outputs %v", p.Outputs)
	}
	if got := EncodeProcessing(blank(), []item.Stack{ore}, nil); !isBlank(got) {
		t.Error("a pattern with no output was written")
	}
}

func TestPatternIDStable(t *testing.T) {
	id := planksID(t)
	a, _ := PatternOf(EncodeCrafting(blank(), id), fakeKeys)
	b, _ := PatternOf(EncodeCrafting(blank(), id), fakeKeys)
	if a.ID == "" || a.ID != b.ID {
		t.Errorf("ids %q %q", a.ID, b.ID)
	}
	ore, ingot := item.NewStack(item.RawIron{}, 1), item.NewStack(item.IronIngot{}, 1)
	c, _ := PatternOf(EncodeProcessing(blank(), []item.Stack{ore}, []item.Stack{ingot}), fakeKeys)
	d, _ := PatternOf(EncodeProcessing(blank(), []item.Stack{ore}, []item.Stack{ingot.Grow(1)}), fakeKeys)
	if c.ID == a.ID || c.ID == d.ID {
		t.Errorf("different patterns share an id: %q %q %q", a.ID, c.ID, d.ID)
	}
}
