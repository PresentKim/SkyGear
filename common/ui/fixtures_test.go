package ui

import (
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
	defer capture(func(_ *oreLayout, root any) { roots = append(roots, root) })()
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
