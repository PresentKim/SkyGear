package ui

import (
	"strings"
	"testing"

	"github.com/presentkim/skygear/common/invui"
)

func TestQuestCanvasDrawsLineThroughLane(t *testing.T) {
	pg := questPage(questData{
		Nodes: []questNodeD{{ID: "a", X: 0, Y: 0, State: QuestDone}, {ID: "b", X: 10.8, Y: 7.2, State: QuestDone}},
		Lines: []questLine{
			{X1: 0, Y1: 0, X2: 5.4, Y2: 0, Cls: "oui-line-seg done"},
			{X1: 5.4, Y1: 0, X2: 5.4, Y2: 7.2, Cls: "oui-line-seg done"},
			{X1: 5.4, Y1: 7.2, X2: 10.8, Y2: 7.2, Cls: "oui-line-seg done"},
		},
	})
	if key := pg.items[invui.SlotKey].CustomName(); key != invui.KeyQuest+"0" {
		t.Fatalf("key = %q", key)
	}
	cols := invui.QuestGrids[0][0]
	left := (cols - 2) / 2
	for _, c := range []struct {
		cell int
		node bool
		code string
	}{
		{left, true, invui.QuestCode(invui.QuestDone, invui.KindLineE+invui.KindGapW+invui.KindGapS, invui.QuestSquare, invui.QuestDone)},
		{cols + left, false, invui.QuestCode(invui.QuestDone, invui.KindGapN+invui.KindGapE, invui.QuestSquare, invui.QuestOpen)},
		{cols + left + 1, true, invui.QuestCode(invui.QuestDone, invui.KindLineW, invui.QuestSquare, invui.QuestDone)},
	} {
		name := pg.items[invui.Cell0+c.cell].CustomName()
		if !strings.HasPrefix(name[invui.FaceBytes:], c.code) || strings.Contains(name, invui.FamSlot) != c.node {
			t.Errorf("cell %d = %q, want code %q", c.cell, name, c.code)
		}
	}
}

func TestTallQuestChapterUsesNarrowCanvas(t *testing.T) {
	var nodes []questNodeD
	for x := range 6 {
		for y := range 15 {
			if (x+y)%4 == 0 {
				nodes = append(nodes, questNodeD{ID: "q", X: float64(x) * 10.8, Y: float64(y) * 7.2, Tip: "t"})
			}
		}
	}
	pg := questPage(questData{Nodes: nodes})
	if key := pg.items[invui.SlotKey].CustomName(); key != invui.KeyQuest+"1" {
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
