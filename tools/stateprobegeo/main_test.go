package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

type geoFile struct {
	Geometry []struct {
		Description struct {
			Identifier string `json:"identifier"`
		} `json:"description"`
		Bones []struct {
			Name  string `json:"name"`
			Cubes []struct {
				Origin []float64 `json:"origin"`
				Size   []float64 `json:"size"`
			} `json:"cubes"`
		} `json:"bones"`
	} `json:"minecraft:geometry"`
}

func read(t *testing.T, name string) geoFile {
	t.Helper()
	var g geoFile
	if err := json.Unmarshal(files()[name], &g); err != nil {
		t.Fatal(err)
	}
	return g
}

// TestShapes: 3 × 64 geometries, one for each kind and set of arms, with the
// core and one 2×2 pixel arm for each face of the set, reaching from the core
// to the edge of the block, the core and arms in the lane of the kind.
func TestShapes(t *testing.T) {
	g := read(t, "state_probe_shape.geo.json")
	if len(g.Geometry) != 3*64 {
		t.Fatalf("%d shapes", len(g.Geometry))
	}
	for i, geo := range g.Geometry {
		kind, f := i/64, i%64
		want := 1
		for i := range directions {
			if f&(1<<i) != 0 {
				want++
			}
		}
		if len(geo.Bones) != want || geo.Bones[0].Name != "core" {
			t.Errorf("shape %d has %d bones, want %d", f, len(geo.Bones), want)
		}
		if id := fmt.Sprintf("geometry.skygear.state_probe_shape_%s_%d", kinds[kind].name, f); geo.Description.Identifier != id {
			t.Errorf("geometry %d is %s, want %s", i, geo.Description.Identifier, id)
		}
	}
	// the item arm down of shape 1 runs from the floor to the core
	arm := g.Geometry[64+1].Bones[1]
	if arm.Name != "down" || arm.Cubes[0].Origin[1] != 0 || arm.Cubes[0].Size[1] != 8-coreHalf || arm.Cubes[0].Size[0] != 2 {
		t.Errorf("arm down = %+v", arm)
	}
	// the item arm east of shape 32 runs from the core to the east edge
	arm = g.Geometry[64+32].Bones[1]
	if arm.Name != "east" || arm.Cubes[0].Origin[0] != coreHalf || arm.Cubes[0].Size[0] != 8-coreHalf {
		t.Errorf("arm east = %+v", arm)
	}
	// the energy lane (-3 pixels): the core and the arms are shifted, the arms
	// still end at the edge of the block (up: 16, east: 8)
	east, up := g.Geometry[32], g.Geometry[2]
	if c := east.Bones[1].Cubes[0]; c.Origin[0] != -3+coreHalf || c.Size[0] != 8-(-3+coreHalf) {
		t.Errorf("energy arm east = %+v", c)
	}
	if c := up.Bones[1].Cubes[0]; c.Origin[1]+c.Size[1] != 16 || c.Origin[1] != 8-3+coreHalf || c.Origin[0] != -3-armHalf {
		t.Errorf("energy arm up = %+v", c)
	}
	// the fluid core is shifted +3 on every axis
	if c := g.Geometry[2*64].Bones[0].Cubes[0]; c.Origin[0] != 3-coreHalf || c.Origin[1] != 8+3-coreHalf || c.Origin[2] != 3-coreHalf {
		t.Errorf("fluid core = %+v", c)
	}
}

// TestBones: a geometry for each kind with the core and every arm.
func TestBones(t *testing.T) {
	g := read(t, "state_probe_bones.geo.json")
	if len(g.Geometry) != 3 {
		t.Fatalf("geometries %+v", g.Geometry)
	}
	for k, geo := range g.Geometry {
		if want := "geometry.skygear.state_probe_bones_" + kinds[k].name; geo.Description.Identifier != want {
			t.Fatalf("geometry %d is %s, want %s", k, geo.Description.Identifier, want)
		}
		var names []string
		for _, b := range geo.Bones {
			names = append(names, b.Name)
		}
		want := []string{"core", "down", "up", "north", "south", "west", "east"}
		if len(names) != len(want) {
			t.Fatalf("bones %v", names)
		}
		for i := range want {
			if names[i] != want[i] {
				t.Fatalf("bones %v, want %v", names, want)
			}
		}
	}
}
