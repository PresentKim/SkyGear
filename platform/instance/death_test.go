package instance

import "testing"

// TestDeathKeepsTheInventoryEverywhere: the platform's handler keeps the
// inventory for every death, in the lobby and in an instance, before the
// game's handler runs. The game relies on it: a hunter that dies in the
// second dimension gets what it carried back in the game's death step, and
// nothing of it may drop there.
func TestDeathKeepsTheInventoryEverywhere(t *testing.T) {
	keep := false
	(&Handler{}).HandleDeath(nil, nil, &keep)
	if !keep {
		t.Fatal("a death does not keep the inventory")
	}
}
