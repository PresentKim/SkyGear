package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

func compact(level int) world.Item {
	if level == 0 {
		return block.Cobblestone{}
	}
	return sgitem.CompactBlock{Kind: "cobblestone", Level: level}
}

// TestCompactRecipesAndResearch: each level crafts from nine of the one
// below and back, and the researches open both ways.
func TestCompactRecipesAndResearch(t *testing.T) {
	r, ok := progress.ByID("compact_cobblestone")
	if !ok {
		t.Fatal("no compact_cobblestone research")
	}
	for l := 1; l <= 4; l++ {
		name, _ := compact(l).EncodeItem()
		found := false
		for _, u := range r.Unlocks {
			found = found || u == name
		}
		if !found {
			t.Errorf("%s not opened by the research", name)
		}
	}
	soil, ok := progress.ByID("compact_soil")
	if !ok {
		t.Fatal("no compact_soil research")
	}
	for _, n := range []string{"gravel", "sand", "dirt", "skygear:compressed_dirt_2"} {
		found := false
		for _, u := range soil.Unlocks {
			found = found || u == n
		}
		if !found {
			t.Errorf("%s not opened by compact_soil", n)
		}
	}
	if b := (sgitem.CompactBlock{Kind: "sand", Level: 2}).Below(); b != (sgitem.CompactBlock{Kind: "sand", Level: 1}) {
		t.Errorf("below sand 2 = %v", b)
	}
	if b := (sgitem.CompactBlock{Kind: "gravel", Level: 1}).Below(); b != (block.Gravel{}) {
		t.Errorf("below gravel 1 = %v", b)
	}
}
