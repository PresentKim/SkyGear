package ui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestShowcaseScreens: the pick, result and equip screens send their parts,
// and pressing a card, an equipment slot or a bag slot runs what it was
// given.
func TestShowcaseScreens(t *testing.T) {
	pressed := ""
	run := func(name string) func(*player.Player, *world.Tx) {
		return func(*player.Player, *world.Tx) { pressed = name }
	}
	ls, roots := shown(func() {
		PickCards(nil, "T", "b", []PickCard{{Title: "A", Rarity: "rare", Run: run("a")}, {Title: "B", Run: run("b")}})
	})
	name, d := screenRoot(roots[0])
	cards := listOf(t, d, "cards")
	if name != "pick" || len(cards) != 2 || cards[0].(map[string]any)["rarity"] != "rare" {
		t.Fatalf("pick %s %v", name, d)
	}
	ls[0].runs[cards[1].(map[string]any)["id"].(string)](nil, nil)
	if pressed != "b" {
		t.Errorf("pressing the second card ran %q", pressed)
	}

	_, roots = shown(func() {
		Result(nil, ResultScreen{Headline: "H", Win: true, Stars: [3]bool{true, true}, New: [3]bool{false, true},
			Rewards: []ItemSlot{{Icon: "textures/x", Count: "3"}}, OK: Option{Label: "ok", Run: run("ok")}})
	})
	name, d = screenRoot(roots[0])
	stars := listOf(t, d, "stars")
	if name != "result" || d["win"] != true || len(stars) != 3 || stars[1].(map[string]any)["new"] != true || len(listOf(t, d, "rewards")) != 1 {
		t.Errorf("result %s %v", name, d)
	}

	ls, roots = shown(func() {
		Equip(nil, EquipScreen{Groups: []EquipGroup{{Heading: "h", Slots: []EquipSlot{{Empty: "textures/e", Item: ItemSlot{Run: run("slot")}}}}},
			Bag: []ItemSlot{{Icon: "textures/x", Run: run("bag")}}})
	})
	name, d = screenRoot(roots[0])
	groups := listOf(t, d, "groups")
	slot := listOf(t, groups[0].(map[string]any), "slots")[0].(map[string]any)
	if name != "equip" || slot["empty"] != "tex:textures/e" {
		t.Fatalf("equip %s %v", name, d)
	}
	ls[0].runs[listOf(t, d, "bag")[0].(map[string]any)["id"].(string)](nil, nil)
	if pressed != "bag" {
		t.Errorf("pressing the bag slot ran %q", pressed)
	}
}
