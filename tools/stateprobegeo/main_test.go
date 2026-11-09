package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const blocks = "../../assets/resourcepack/skygear/models/blocks"

// TestCommittedFilesAreCurrent: the geometry files in the pack are what the
// tool writes.
func TestCommittedFilesAreCurrent(t *testing.T) {
	for name, want := range files() {
		got, err := os.ReadFile(filepath.Join(blocks, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s is out of date: go run ./tools/stateprobegeo assets/resourcepack/skygear/models/blocks", name)
		}
	}
}

type fourFile struct {
	FormatVersion string `json:"format_version"`
	Geometry      []struct {
		Description struct {
			Identifier string `json:"identifier"`
		} `json:"description"`
		Bones []struct {
			Name  string `json:"name"`
			Cubes []struct {
				Origin []float64 `json:"origin"`
				Size   []float64 `json:"size"`
				UV     map[string]struct {
					MaterialInstance string `json:"material_instance"`
				} `json:"uv"`
			} `json:"cubes"`
		} `json:"bones"`
	} `json:"minecraft:geometry"`
}

// TestPairGeometries: one geometry for each of pair2's two blocks, with the
// core and six arms of each of its two kinds named <kind>_<bone>, every face
// drawn with the material instance of its kind.
func TestPairGeometries(t *testing.T) {
	var g fourFile
	if err := json.Unmarshal(files()["state_probe_four.geo.json"], &g); err != nil {
		t.Fatal(err)
	}
	if len(g.Geometry) != 2 {
		t.Fatalf("%d geometries", len(g.Geometry))
	}
	want := [][]int{{0, 1}, {2, 3}}
	for i, geo := range g.Geometry {
		id := fmt.Sprintf("geometry.skygear.state_probe_two_%s_%s", kinds[want[i][0]].name, kinds[want[i][1]].name)
		if geo.Description.Identifier != id {
			t.Fatalf("geometry %d is %s, want %s", i, geo.Description.Identifier, id)
		}
		var names []string
		for _, k := range want[i] {
			for _, b := range []string{"core", "down", "up", "north", "south", "west", "east"} {
				names = append(names, kinds[k].name+"_"+b)
			}
		}
		if len(geo.Bones) != len(names) {
			t.Fatalf("%s: %d bones, want %v", id, len(geo.Bones), names)
		}
		for j, b := range geo.Bones {
			if b.Name != names[j] {
				t.Fatalf("%s: bone %d is %s, want %s", id, j, b.Name, names[j])
			}
			kind, _, _ := strings.Cut(b.Name, "_")
			if len(b.Cubes) != 1 || len(b.Cubes[0].UV) != 6 {
				t.Fatalf("%s %s: cubes %+v", id, b.Name, b.Cubes)
			}
			for side, f := range b.Cubes[0].UV {
				if f.MaterialInstance != kind {
					t.Fatalf("%s %s %s: material instance %q", id, b.Name, side, f.MaterialInstance)
				}
			}
		}
	}
	energyItem, fluidNetwork := g.Geometry[0], g.Geometry[1]
	// the item arm down runs from the floor to the core
	if c := energyItem.Bones[8].Cubes[0]; energyItem.Bones[8].Name != "item_down" || c.Origin[1] != 0 || c.Size[1] != 8-coreHalf || c.Size[0] != 2 {
		t.Errorf("item arm down = %+v", c)
	}
	// the item arm east runs from the core to the east edge
	if c := energyItem.Bones[13].Cubes[0]; energyItem.Bones[13].Name != "item_east" || c.Origin[0] != coreHalf || c.Size[0] != 8-coreHalf {
		t.Errorf("item arm east = %+v", c)
	}
	// the energy lane (-3 pixels): the core and the arms are shifted, the arms
	// still end at the edge of the block (up: 16, east: 8)
	if c := energyItem.Bones[5].Cubes[0]; energyItem.Bones[5].Name != "energy_west" || c.Origin[0] != -8 || c.Origin[0]+c.Size[0] != -3-coreHalf {
		t.Errorf("energy arm west = %+v", c)
	}
	if c := energyItem.Bones[2].Cubes[0]; c.Origin[1]+c.Size[1] != 16 || c.Origin[1] != 8-3+coreHalf || c.Origin[0] != -3-armHalf {
		t.Errorf("energy arm up = %+v", c)
	}
	// the fluid core is shifted +3 on every axis
	if c := fluidNetwork.Bones[0].Cubes[0]; c.Origin[0] != 3-coreHalf || c.Origin[1] != 8+3-coreHalf || c.Origin[2] != 3-coreHalf {
		t.Errorf("fluid core = %+v", c)
	}
	// the network core sits at (+6, -6, +6) pixels from the middle
	if c := fluidNetwork.Bones[7].Cubes[0]; c.Origin[0] != 6-coreHalf || c.Origin[1] != 8-6-coreHalf || c.Origin[2] != 6-coreHalf {
		t.Errorf("network core = %+v", c)
	}
	// the network arm east ends at the block edge
	if c := fluidNetwork.Bones[13].Cubes[0]; c.Origin[0] != 6+coreHalf || c.Origin[0]+c.Size[0] != 8 {
		t.Errorf("network arm east = %+v", c)
	}
}

// TestLanesDoNotMeet: in every geometry no cube of one kind overlaps a cube of
// the other (positive volume in common), so two kinds in one block never draw
// through each other; and the lanes of the first three kinds are as the
// entity's.
func TestLanesDoNotMeet(t *testing.T) {
	var g fourFile
	if err := json.Unmarshal(files()["state_probe_four.geo.json"], &g); err != nil {
		t.Fatal(err)
	}
	pairs := 0
	for _, geo := range g.Geometry {
		for _, a := range geo.Bones {
			for _, b := range geo.Bones {
				ka, _, _ := strings.Cut(a.Name, "_")
				kb, _, _ := strings.Cut(b.Name, "_")
				if ka >= kb {
					continue
				}
				ca, cb := a.Cubes[0], b.Cubes[0]
				meet := true
				for i := range 3 {
					if ca.Origin[i]+ca.Size[i] <= cb.Origin[i] || cb.Origin[i]+cb.Size[i] <= ca.Origin[i] {
						meet = false
					}
				}
				if meet {
					t.Errorf("%s: %s %+v meets %s %+v", geo.Description.Identifier, a.Name, ca, b.Name, cb)
				}
				pairs++
			}
		}
	}
	if pairs != 2*7*7 {
		t.Errorf("%d pairs of bones checked", pairs)
	}
	for i, want := range [][3]float64{{-3, -3, -3}, {0, 0, 0}, {3, 3, 3}, {6, -6, 6}} {
		if kinds[i].shift != want {
			t.Errorf("lane %s is %v, want %v", kinds[i].name, kinds[i].shift, want)
		}
	}
}
