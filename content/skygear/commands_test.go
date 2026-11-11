package skygear

import (
	"slices"
	"testing"

	"github.com/presentkim/skygear/content/skygear/core"
)

// TestFeatureCommands: the commands the features add, by feature order: the
// machines' statistics, then the farm's sapling.
func TestFeatureCommands(t *testing.T) {
	var names []string
	for _, c := range core.Commands() {
		names = append(names, c.Name())
	}
	want := []string{"통계", "stats", "묘목", "sapling"}
	if !slices.Equal(names, want) {
		t.Errorf("feature commands %v, want %v", names, want)
	}
}
