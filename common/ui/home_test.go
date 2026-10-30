package ui

import (
	"fmt"
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
	defer func() { oreShown = nil }()

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
	if want := fmt.Sprintf("%dx%d", windowW, windowH); got.size != want {
		t.Errorf("size %q, the windows' %q", got.size, want)
	}
	// profile, a, b, next, play, saves, resume
	if len(got.runs) != 7 || got.close == nil {
		t.Errorf("%d pressable, close %v", len(got.runs), got.close != nil)
	}
}

// TestHomeData: 64 colours go in face (no icon), else face is [] and the
// icon is sent; Esc runs close.
func TestHomeData(t *testing.T) {
	var got *oreLayout
	var root any
	oreShown = func(l *oreLayout, r any) { got, root = l, r }
	defer func() { oreShown = nil }()

	face := make([]string, 64)
	for i := range face {
		face[i] = "ff0000"
	}
	h := Home{Profile: HomeProfile{Face: face, Icon: "textures/x", Fill: 2}, Play: HomeButton{Label: "p", Run: nopRun}, Close: nopRun}
	HomeScreen(nil, h)
	_, data := screenRoot(root)
	pr := data["profile"].(map[string]any)
	if len(pr["face"].([]any)) != 64 || pr["icon"] != "" || pr["fill"] != 1.0 || data["manage"] != nil || data["overlay"] != nil {
		t.Errorf("profile %v", data)
	}
	h.Profile.Face = face[:3]
	HomeScreen(nil, h)
	_, data = screenRoot(root)
	pr = data["profile"].(map[string]any)
	if len(pr["face"].([]any)) != 0 || pr["icon"] != "tex:textures/x" || got.close == nil {
		t.Errorf("profile %v", pr)
	}
}

// TestLoadingClose: Esc shows the loading screen again, and the layout is a
// loading one.
func TestLoadingClose(t *testing.T) {
	var got *oreLayout
	var root any
	n := 0
	oreShown = func(l *oreLayout, r any) { got, root = l, r; n++ }
	defer func() { oreShown = nil }()

	LoadingScreen(nil, Loading{Title: "t", Icon: "textures/x"})
	name, data := screenRoot(root)
	if name != "loading" || data["title"] != "t" || data["icon"] != "tex:textures/x" || !got.loading {
		t.Fatalf("%s %v", name, data)
	}
	got.close((*player.Player)(nil), nil)
	if n != 2 {
		t.Errorf("shown %d times", n)
	}
}

// TestConsoleField: the field is cmd<len(lines)> and the run button reads it.
func TestConsoleField(t *testing.T) {
	var got *oreLayout
	var root any
	oreShown = func(l *oreLayout, r any) { got, root = l, r }
	defer func() { oreShown = nil }()

	var ran string
	ConsoleScreen(nil, "c", Console{Lines: []string{"a", "b"}, Submit: "run", Back: nopRun, BackLabel: "back",
		Run: func(_ *player.Player, _ *world.Tx, cmd string) { ran = cmd }})
	name, data := screenRoot(root)
	if name != "console" || data["field"] != "cmd2" || len(data["lines"].([]any)) != 2 {
		t.Fatalf("%s %v", name, data)
	}
	got.values["cmd2"] = "go"
	got.runs[data["submit"].(map[string]any)["id"].(string)](nil, nil)
	if ran != "go" {
		t.Errorf("ran %q", ran)
	}
}
