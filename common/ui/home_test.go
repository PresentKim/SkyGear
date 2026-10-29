package ui

import (
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// TestHomeScreen: the main screen has the size of the windows, so the menus
// its buttons open replace it in place; every button presses, an arrow
// without a run does not, and Esc runs its close.
func TestHomeScreen(t *testing.T) {
	var got *oreLayout
	oreShown = func(l *oreLayout, _ any) { got = l }
	oldOre := OreUI
	OreUI = true
	defer func() { oreShown, OreUI = nil, oldOre }()

	run := func(*player.Player, *world.Tx) {}
	HomeScreen(nil, Home{
		Profile: HomeProfile{Name: "name", Fill: 2, Run: run},
		Corner:  []HomeButton{{Label: "a", Badge: 1, Run: run}},
		Side:    []HomeButton{{Label: "b", Run: run}},
		Save:    HomeSave{Name: "save", Next: run},
		Play:    HomeButton{Label: "play", Run: run},
		Manage:  HomeButton{Label: "saves", Run: run},
		Overlay: &HomeOverlay{Title: "menu", Buttons: []HomeButton{{Label: "resume", Run: run}}},
		Close:   run,
	})
	if got == nil {
		t.Fatal("nothing shown")
	}
	l := newOreLayout()
	l.window("", nil, nil, nil)
	if got.size != l.size {
		t.Errorf("size %q, the windows' %q", got.size, l.size)
	}
	// profile, a, b, next, play, saves, resume
	if len(got.runs) != 7 || got.close == nil {
		t.Errorf("%d pressable, close %v", len(got.runs), got.close != nil)
	}
}
