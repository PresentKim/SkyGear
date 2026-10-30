// Command slotmarks writes the slot decoration marks (common/slotdeco) into
// the item names of the pack's language files from decor.ItemDecor;
// plain items lose them. Run it after changing ItemDecor or adding items:
//
//	go run ./tools/slotmarks assets/resourcepack/skygear
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/slotdeco"
	_ "github.com/presentkim/skygear/content/skygear" // the items
	"github.com/presentkim/skygear/content/skygear/decor"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: slotmarks <pack dir>")
		os.Exit(2)
	}
	type mark struct {
		r    slotdeco.Rarity
		tier int
	}
	marks := map[string]mark{}
	for _, it := range world.CustomItems() {
		id, _ := it.EncodeItem()
		r, tier := decor.ItemDecor(it)
		marks["item."+id+".name"] = mark{r, tier}
	}
	for _, file := range []string{"ko_KR.lang", "en_US.lang"} {
		path := filepath.Join(os.Args[1], "texts", file)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		lines := strings.Split(string(data), "\n")
		changed := 0
		for i, line := range lines {
			key, value, ok := strings.Cut(line, "=")
			m, known := marks[key]
			if !ok || !known {
				continue
			}
			if out := key + "=" + slotdeco.Decorate(m.r, m.tier, slotdeco.Strip(value)); out != line {
				lines[i] = out
				changed++
			}
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(file, changed, "lines changed")
	}
}
