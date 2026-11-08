package main

import (
	"encoding/json"
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

// TestShapes: 64 geometries, one for each set of arms, with the core and one
// 2×2 pixel arm for each face of the set, reaching from the core to the edge
// of the block.
func TestShapes(t *testing.T) {
	g := read(t, "state_probe_shape.geo.json")
	if len(g.Geometry) != 64 {
		t.Fatalf("%d shapes", len(g.Geometry))
	}
	for f, geo := range g.Geometry {
		want := 1
		for i := range directions {
			if f&(1<<i) != 0 {
				want++
			}
		}
		if len(geo.Bones) != want || geo.Bones[0].Name != "core" {
			t.Errorf("shape %d has %d bones, want %d", f, len(geo.Bones), want)
		}
	}
	// the arm down of shape 1 runs from the floor to the core
	arm := g.Geometry[1].Bones[1]
	if arm.Name != "down" || arm.Cubes[0].Origin[1] != 0 || arm.Cubes[0].Size[1] != 8-coreHalf || arm.Cubes[0].Size[0] != 2 {
		t.Errorf("arm down = %+v", arm)
	}
	// the arm east of shape 32 runs from the core to the east edge
	arm = g.Geometry[32].Bones[1]
	if arm.Name != "east" || arm.Cubes[0].Origin[0] != coreHalf || arm.Cubes[0].Size[0] != 8-coreHalf {
		t.Errorf("arm east = %+v", arm)
	}
}

func TestBones(t *testing.T) {
	g := read(t, "state_probe_bones.geo.json")
	if len(g.Geometry) != 1 || g.Geometry[0].Description.Identifier != "geometry.skygear.state_probe_bones" {
		t.Fatalf("geometries %+v", g.Geometry)
	}
	var names []string
	for _, b := range g.Geometry[0].Bones {
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
