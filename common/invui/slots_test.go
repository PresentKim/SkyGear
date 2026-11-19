package invui

import "testing"

func TestEveryLayoutFitsTheWindow(t *testing.T) {
	for _, l := range Layouts() {
		if _, end := Region(l); end > WindowSize {
			t.Errorf("%s ends at %d", l, end)
		}
	}
}

func TestSharedLayoutsKeepTheMainScreen(t *testing.T) {
	hFirst, _ := Region(KeyHome)
	for _, l := range Shared[KeyHome] {
		if _, end := Region(l); l != KeyHome && end > hFirst {
			t.Errorf("%s ends at %d over the main screen from %d", l, end, hFirst)
		}
	}
}

func TestPlacesAreDistinct(t *testing.T) {
	for _, l := range Layouts() {
		seen := map[int]bool{}
		for _, v := range append(append([]int{}, Common...), own[l]...) {
			p, ok := Place(l, v)
			if !ok || seen[p] {
				t.Errorf("%s slot %d at %d", l, v, p)
			}
			seen[p] = true
		}
	}
}
