package skygear

import (
	"slices"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// TestFeatureCommands: the commands the features add, by feature order: the
// machines' statistics, the farm's sapling, then the screens' commands.
func TestFeatureCommands(t *testing.T) {
	var names []string
	for _, c := range core.Commands() {
		names = append(names, c.Name())
	}
	want := []string{"통계", "stats", "묘목", "sapling", "menu", "sg", "메뉴발급", "menuitem", "도감", "codex", "도움말", "sghelp"}
	if !slices.Equal(names, want) {
		t.Errorf("feature commands %v, want %v", names, want)
	}
}
