package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

func fixtureScreens() map[string]func() { return Fixtures(nil) }

var fixtureNames = FixtureNames

// runFixture shows fixture name and returns the roots it showed.
func runFixture(t *testing.T, name string) (roots []any) {
	t.Helper()
	show, ok := fixtureScreens()[name]
	if !ok {
		t.Fatalf("no fixture %q", name)
	}
	oreShown = func(_ *oreLayout, root any) { roots = append(roots, root) }
	defer func() { oreShown = nil }()
	show()
	return roots
}

// TestFixturesCoverAll checks the fixtures are exactly the screens the
// baseline lists.
func TestFixturesCoverAll(t *testing.T) {
	var got []string
	for name := range fixtureScreens() {
		got = append(got, name)
	}
	sort.Strings(got)
	want := slices.Clone(fixtureNames)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("fixtures = %v, want %v", got, want)
	}
}

// TestFixturesShowOnce checks every fixture shows exactly one screen, with
// no player.
func TestFixturesShowOnce(t *testing.T) {
	for _, name := range fixtureNames {
		roots := runFixture(t, name)
		if len(roots) != 1 {
			t.Errorf("%s showed %d roots, want 1", name, len(roots))
		}
	}
}

// TestFixturesKeepLobby checks no fixture leaves InLobby or LobbyHeader set.
func TestFixturesKeepLobby(t *testing.T) {
	for _, name := range fixtureNames {
		runFixture(t, name)
	}
	if InLobby != nil || LobbyHeader != nil {
		t.Error("a lobby fixture left InLobby or LobbyHeader set")
	}
}

// TestDumpFixtures writes every fixture as <name>.json in $OREUI_FIXTURES:
// {"root": the root showScreen was given}.
func TestDumpFixtures(t *testing.T) {
	dir := os.Getenv("OREUI_FIXTURES")
	if dir == "" {
		t.Skip("OREUI_FIXTURES not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range fixtureNames {
		roots := runFixture(t, name)
		if len(roots) != 1 {
			t.Fatalf("%s showed %d roots, want 1", name, len(roots))
		}
		data, err := json.MarshalIndent(map[string]any{"root": roots[0]}, "", " ")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
