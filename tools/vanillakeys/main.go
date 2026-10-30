// Command vanillakeys builds from the official texts (Mojang/bedrock-samples
// en_US.lang, ko_KR.lang) content/skygear/locales/{en,ko}.vanilla.json, the names of
// every vanilla block and item as "vanilla.<id>", and naming/vanillakeys.json,
// the client's key of each block and item named by another key than
// "tile.<id>.name" (items, and legacy keys like "tile.planks.oak.name"),
// matched by its English name in Dragonfly.
//
//	go run ./tools/vanillakeys en_US.lang ko_KR.lang content/skygear/naming/vanillakeys.json content/skygear/locales
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/df-mc/dragonfly/server/block" // registers the blocks
	"github.com/df-mc/dragonfly/server/world"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: vanillakeys en_US.lang ko_KR.lang vanillakeys.json locales-dir")
		os.Exit(2)
	}
	en, ko := readLang(os.Args[1]), readLang(os.Args[2])
	byName := map[string]string{}
	for k, v := range en {
		if (strings.HasPrefix(k, "tile.") || strings.HasPrefix(k, "item.")) && strings.HasSuffix(k, ".name") {
			if old, dup := byName[strings.ToLower(v)]; !dup || k < old {
				byName[strings.ToLower(v)] = k
			}
		}
	}
	world.DefaultBlockRegistry.Finalize()
	// ids are every vanilla block and item.
	ids := map[string]bool{}
	for _, b := range world.Blocks() {
		id, _ := b.EncodeBlock()
		ids[id] = true
	}
	for _, it := range world.Items() {
		id, _ := it.EncodeItem()
		ids[id] = true
	}
	keyOf := map[string]string{} // keys other than tile.<id>.name
	names := map[string]map[string]string{"en": {}, "ko": {}}
	var unmapped []string
	var sorted []string
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	for _, id := range sorted {
		n, ok := strings.CutPrefix(id, "minecraft:")
		if !ok {
			continue
		}
		// A double slab is named like its slab, "hard_" glass like the glass.
		base := strings.Replace(strings.TrimPrefix(n, "hard_"), "_double_slab", "_slab", 1)
		key := ""
		for _, k := range []string{"tile." + base + ".name", "item." + base + ".name"} {
			if _, ok := en[k]; ok {
				key = k
				break
			}
		}
		if key == "" {
			key = byName[strings.ToLower(titleCase(base))]
		}
		if key == "" {
			unmapped = append(unmapped, n)
			continue
		}
		if key != "tile."+n+".name" {
			keyOf[n] = key
		}
		names["en"]["vanilla."+n] = en[key]
		if v, ok := ko[key]; ok {
			names["ko"]["vanilla."+n] = v
		}
	}
	fmt.Fprintf(os.Stderr, "%d keys, %d ids named, %d unmapped\n", len(keyOf), len(names["en"]), len(unmapped))
	write(os.Args[3], keyOf)
	for lang, m := range names {
		write(filepath.Join(os.Args[4], lang+".vanilla.json"), m)
	}
}

func readLang(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok || strings.HasPrefix(k, "##") {
			continue
		}
		v, _, _ = strings.Cut(v, "\t#")
		out[k] = strings.TrimRight(v, " \t\r")
	}
	return out
}

func write(path string, v any) {
	data, _ := json.MarshalIndent(v, "", " ")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		panic(err)
	}
}

// titleCase turns "oak_planks" into "Oak Planks".
func titleCase(id string) string {
	parts := strings.Split(id, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
