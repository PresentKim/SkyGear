package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/presentkim/skygear/common/invui"
)

func TestPlacesUseSmallestGap(t *testing.T) {
	if got := places([]float64{3, 0, 1.5, 7.5}); !slices.Equal(got, []int{2, 0, 1, 5}) {
		t.Fatalf("places = %v", got)
	}
}

func TestQuestMapDrawsLineBetweenQuests(t *testing.T) {
	pg := questPage(questData{
		Nodes: []questNodeD{{ID: "a", X: 0, Y: 0, State: QuestDone}, {ID: "b", X: 10, Y: 0, State: QuestDone}},
		Lines: []questLine{{X1: 0, Y1: 0, X2: 10, Y2: 0.5, Cls: "oui-line-seg done"}},
	})
	if key := pg.items[invui.SlotKey].CustomName(); !strings.HasPrefix(key, invui.KeyBoard) {
		t.Fatalf("key = %q", key)
	}
	for i, want := range []string{invui.FamSlot, invui.KindLineW + invui.KindLineE + invui.KindDone, invui.FamSlot} {
		if name := pg.items[invui.Cell0+i].CustomName(); !strings.Contains(name, want) {
			t.Errorf("cell %d = %q, want %q", i, name, want)
		}
	}
}

func TestTallQuestChapterTurnsOnGrid(t *testing.T) {
	var nodes []questNodeD
	for x := range 6 {
		for y := range 15 {
			if (x+y)%4 == 0 {
				nodes = append(nodes, questNodeD{ID: "q", X: float64(x) * 10.8, Y: float64(y) * 7.2, Tip: "t"})
			}
		}
	}
	pg := questPage(questData{Nodes: nodes})
	if key := pg.items[invui.SlotKey].CustomName(); !strings.HasPrefix(key, invui.KeyGrid) {
		t.Fatalf("key = %q", key)
	}
	placed := 0
	for i := range invui.Cells {
		if strings.Contains(pg.items[invui.Cell0+i].CustomName(), invui.FamSlot) {
			placed++
		}
	}
	if placed != len(nodes) {
		t.Fatalf("placed %d of %d", placed, len(nodes))
	}
}
