package core

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/creative"
	"github.com/df-mc/dragonfly/server/world"
)

func creativeOf(slot int, it world.Item) CreativeItem {
	return CreativeItem{Slot: slot, Category: creative.ItemsCategory(), Item: it}
}

// TestCreativeItemsSortBySlot: the list is ordered by slot whoever owns the
// slot, and the items of one slot keep their owner's order.
func TestCreativeItemsSortBySlot(t *testing.T) {
	ResetHooks()
	Install(
		&Feature{Name: "a", Creative: func() []CreativeItem {
			return []CreativeItem{creativeOf(30, item.Stick{}), creativeOf(10, item.Diamond{}), creativeOf(30, item.Apple{})}
		}},
		&Feature{Name: "b"},
		&Feature{Name: "c", Creative: func() []CreativeItem { return []CreativeItem{creativeOf(20, item.Coal{})} }},
	)
	got := CreativeItems([]CreativeItem{creativeOf(40, block.Stone{}), creativeOf(15, block.Dirt{})})
	want := []world.Item{item.Diamond{}, block.Dirt{}, item.Coal{}, item.Stick{}, item.Apple{}, block.Stone{}}
	if len(got) != len(want) {
		t.Fatalf("%d items, want %d", len(got), len(want))
	}
	for i, it := range got {
		if it.Item != want[i] {
			t.Errorf("item %d is %T, want %T", i, it.Item, want[i])
		}
	}
}

// TestCreativeSlotOfTwoOwnersPanics: a slot belongs to one owner; the panic
// names both.
func TestCreativeSlotOfTwoOwnersPanics(t *testing.T) {
	for name, own := range map[string][]CreativeItem{
		"two features":         nil,
		"feature and the list": {creativeOf(20, block.Stone{})},
	} {
		t.Run(name, func(t *testing.T) {
			ResetHooks()
			fs := []*Feature{{Name: "a", Creative: func() []CreativeItem { return []CreativeItem{creativeOf(20, item.Stick{})} }}}
			if own == nil {
				fs = append(fs, &Feature{Name: "b", Creative: func() []CreativeItem { return []CreativeItem{creativeOf(20, item.Coal{})} }})
			}
			Install(fs...)
			defer func() {
				r, _ := recover().(string)
				if !strings.Contains(r, "creative slot 20") || !strings.Contains(r, "feature a") {
					t.Errorf("panic %q, want one naming slot 20 and feature a", r)
				}
			}()
			CreativeItems(own)
			t.Error("no panic")
		})
	}
}
