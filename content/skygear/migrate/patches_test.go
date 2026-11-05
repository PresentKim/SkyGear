package migrate

import "testing"

func TestDrawerPatchKeepsResearchIDs(t *testing.T) {
	r := newRenamer(Patches)
	for in, want := range map[string]string{
		"skygear:storage_controller": "skygear:drawer_controller",
		"skygear:storage_link:0":     "skygear:drawer_trim:0",
		"place_storage_link":         "place_drawer_trim",
		"storage_controller":         "storage_controller",
		"storage_link":               "storage_link",
		"skygear:storage_disk_1k":    "skygear:storage_disk_1k",
	} {
		if got := r.rep(in); got != want {
			t.Errorf("rep(%q) = %q, want %q", in, got, want)
		}
	}
}
