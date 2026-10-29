package skygear

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// TestEveryCounterIsCounted: each key of the catalog is counted somewhere in
// the game's code, so no achievement is out of reach. A key is counted if the
// code has it as a string, or has a prefix of it that is joined to a kind
// ("crop_"+kind, "place_"+kind, "wave_level_%d" …). The game's code is this
// package and the packages under it, without the catalog.
func TestEveryCounterIsCounted(t *testing.T) {
	var files []string
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".go") {
			files = append(files, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	literals := map[string]bool{}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == filepath.Join("achievement", "catalog.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil {
					literals[s] = true
				}
			}
			return true
		})
	}
	counted := func(key string) bool {
		if literals[key] {
			return true
		}
		for lit := range literals {
			prefix := strings.TrimSuffix(lit, "%d")
			if strings.HasSuffix(prefix, "_") && len(prefix) > 3 && strings.HasPrefix(key, prefix) {
				return true
			}
		}
		return false
	}
	for _, c := range achievement.Counters {
		if !counted(c.Key) {
			t.Errorf("counter %s is not counted anywhere", c.Key)
		}
	}
}

// TestCounterKindsExist: the counters named after a kind of machine, mob,
// crop, mesh, pebble, essence or wave level name one the game has.
func TestCounterKindsExist(t *testing.T) {
	has := func(set []string, v string) bool {
		for _, s := range set {
			if s == v {
				return true
			}
		}
		return false
	}
	var meshes, pebbles, crops []string
	for _, m := range sgitem.MeshTiers {
		meshes = append(meshes, m.Tier)
	}
	for _, p := range sgitem.PebbleKinds {
		pebbles = append(pebbles, p.Kind)
	}
	for k := range cropKinds {
		crops = append(crops, k)
	}
	rules := []struct {
		prefix string
		kinds  []string
	}{
		{"place_", machine.MachineOrder}, {"cycle_", machine.MachineOrder}, {"kill_", sgitem.MobKinds}, {"crop_", crops},
		{"sift_mesh_", meshes}, {"pebble_", pebbles}, {"lab_", sgitem.ResearchEssenceKinds},
	}
	for _, c := range achievement.Counters {
		if rest, ok := strings.CutPrefix(c.Key, "wave_level_"); ok {
			if n, err := strconv.Atoi(rest); err != nil || n < 1 || n > len(waveLevels) {
				t.Errorf("counter %s: the gate has %d levels", c.Key, len(waveLevels))
			}
			continue
		}
		for _, r := range rules {
			// crop_plant, crop_harvest, cycle_total and kill_total count all kinds
			if kind, ok := strings.CutPrefix(c.Key, r.prefix); ok && !has([]string{"plant", "harvest", "total"}, kind) && !has(r.kinds, kind) {
				t.Errorf("counter %s: no %q", c.Key, kind)
			}
		}
	}
	// and the other way round: every machine can be placed for an achievement
	for _, k := range machine.MachineOrder {
		if k == machine.KindSellBox {
			continue
		}
		found := false
		for _, c := range achievement.Counters {
			found = found || c.Key == "place_"+k
		}
		if !found {
			t.Errorf("no achievement for placing the %s", k)
		}
	}
}
