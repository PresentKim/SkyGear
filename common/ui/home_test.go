package ui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestHomeScreen: the main screen has the size of the windows, so the menus
// its buttons open replace it in place; every button presses and Esc runs
// its close.
func TestHomeScreen(t *testing.T) {
	var got *oreLayout
	oreShown = func(l *oreLayout, _ any) { got = l }
	oldOre := OreUI
	OreUI = true
	defer func() { oreShown, OreUI = nil, oldOre }()

	run := func(*player.Player, *world.Tx) {}
	HomeScreen(nil, Home{Buttons: []Option{{Label: "a", Run: run}, {Label: "b", Run: run}}, Footer: "name", Close: run})
	if got == nil {
		t.Fatal("nothing shown")
	}
	l := newOreLayout()
	l.window("", nil, nil, nil)
	if got.size != l.size {
		t.Errorf("size %q, the windows' %q", got.size, l.size)
	}
	if len(got.runs) != 2 || got.close == nil {
		t.Errorf("%d pressable, close %v", len(got.runs), got.close != nil)
	}
}
