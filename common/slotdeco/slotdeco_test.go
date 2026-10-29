package slotdeco

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecorateStripOf(t *testing.T) {
	for _, tc := range []struct {
		r    Rarity
		tier int
	}{{Common, 0}, {Uncommon, 0}, {Rare, 2}, {Epic, 3}, {Legendary, 4}, {Common, 1}, {Common, BadgeWarn}, {Uncommon, BadgeLock}} {
		s := Decorate(tc.r, tc.tier, "정밀 키트")
		if got := Strip(s); got != "정밀 키트" {
			t.Errorf("Strip(%q) = %q", s, got)
		}
		if r, tier := Of(s); r != tc.r || tier != tc.tier {
			t.Errorf("Of(%q) = %v, %v; want %v, %v", s, r, tier, tc.r, tc.tier)
		}
	}
	// no mark is part of another one: a binding that looks for one never
	// finds it in a text that carries a different one
	var marks []string
	for r := Uncommon; r <= Legendary; r++ {
		marks = append(marks, r.Mark())
	}
	for tier := 1; tier <= MaxFront; tier++ {
		marks = append(marks, TierMark(tier))
	}
	for i, a := range marks {
		for j, b := range marks {
			if i != j && strings.Contains(a, b) {
				t.Errorf("mark %q contains mark %q", a, b)
			}
		}
	}
}

func referenceTierMark(tier int) string {
	return fmt.Sprintf("§k§l§r§%d§r", tier)
}

func referenceStrip(s string) string {
	for t := 1; t <= MaxFront; t++ {
		s = strings.ReplaceAll(s, referenceTierMark(t), "")
	}
	for _, m := range rarityMarks[1:] {
		s = strings.ReplaceAll(s, m, "")
	}
	return s
}

func referenceOf(s string) (Rarity, int) {
	var r Rarity
	for i := Uncommon; i <= Legendary; i++ {
		if strings.Contains(s, i.Mark()) {
			r = i
		}
	}
	tier := 0
	for t := 1; t <= MaxFront; t++ {
		if strings.Contains(s, referenceTierMark(t)) {
			tier = t
		}
	}
	return r, tier
}

func TestStripOfMatchSequentialReplace(t *testing.T) {
	for tier := 1; tier <= MaxFront; tier++ {
		if TierMark(tier) != referenceTierMark(tier) {
			t.Errorf("TierMark(%d) = %q", tier, TierMark(tier))
		}
		if !strings.Contains(TierMark(tier), markPrefix) {
			t.Errorf("TierMark(%d) lacks %q", tier, markPrefix)
		}
	}
	for _, m := range rarityMarks[1:] {
		if !strings.Contains(m, markPrefix) {
			t.Errorf("rarity mark %q lacks %q", m, markPrefix)
		}
	}
	inputs := []string{"", "plain", "§aGreen§r", "§l§r§3§r", "§k", "§kraw", "a§k§r§ab" + TierMark(2) + "§k§r§6x" + TierMark(BadgeLock)}
	for r := Common; r <= Legendary; r++ {
		for tier := 0; tier <= MaxFront; tier++ {
			inputs = append(inputs, Decorate(r, tier, "name"), "§k§l§r"+Decorate(r, tier, "§k§r§a")+"§r")
		}
	}
	for _, s := range inputs {
		if got, want := Strip(s), referenceStrip(s); got != want {
			t.Errorf("Strip(%q) = %q, want %q", s, got, want)
		}
		gr, gt := Of(s)
		wr, wt := referenceOf(s)
		if gr != wr || gt != wt {
			t.Errorf("Of(%q) = %v, %v, want %v, %v", s, gr, gt, wr, wt)
		}
	}
}

const packDir = "../packs/slotdeco"

func stripComments(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "//") {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

func readJSON(t *testing.T, path string, v any) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(packDir, path))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(stripComments(string(b))), v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return string(b)
}

// TestPackMatchesMarks: the pack's bindings look for exactly the marks this
// package writes, every file it names exists, and the modifications insert
// controls that are defined.
func TestPackMatchesMarks(t *testing.T) {
	var defs struct {
		UIDefs []string `json:"ui_defs"`
	}
	readJSON(t, "ui/_ui_defs.json", &defs)
	if len(defs.UIDefs) != 1 {
		t.Fatalf("ui_defs lists %v", defs.UIDefs)
	}
	var ui map[string]any
	text := readJSON(t, defs.UIDefs[0], &ui)
	for r := Uncommon; r <= Legendary; r++ {
		if !strings.Contains(text, "'"+r.Mark()+"'") {
			t.Errorf("slotdeco.json does not look for the mark of rarity %d", r)
		}
		for _, name := range []string{"rarity_", "tip_"} {
			if _, err := os.Stat(filepath.Join(packDir, "textures/ui/sg", name+string(rune('0'+int(r)))+".png")); err != nil {
				t.Error(err)
			}
		}
	}
	for tier := 1; tier <= MaxFront; tier++ {
		if !strings.Contains(text, "'"+TierMark(tier)+"'") {
			t.Errorf("slotdeco.json does not look for the mark of front image %d", tier)
		}
		if _, err := os.Stat(filepath.Join(packDir, "textures/ui/sg", "front_"+string(rune('0'+tier))+".png")); err != nil {
			t.Error(err)
		}
	}
	for _, f := range []string{"ui/ui_common.json", "ui/inventory_screen.json"} {
		var mods map[string]struct {
			Modifications []struct {
				Value map[string]any `json:"value"`
			} `json:"modifications"`
		}
		readJSON(t, f, &mods)
		for name, m := range mods {
			for _, mod := range m.Modifications {
				for ref := range mod.Value {
					_, id, _ := strings.Cut(ref, "@sg_slot.")
					if _, ok := ui[id]; !ok {
						t.Errorf("%s: %s inserts %s, which slotdeco.json does not define", f, name, ref)
					}
				}
			}
		}
	}
}

// TestScreenDecoPictures: every rarity has the frame the DDUI screens put
// behind an item slot (common/ui slotDeco: deco_<rarity>_0 of the sgui pack).
func TestScreenDecoPictures(t *testing.T) {
	for r := Uncommon; r <= Legendary; r++ {
		name := "deco_" + string(rune('0'+int(r))) + "_0.png"
		if _, err := os.Stat(filepath.Join("../packs/sgui/textures/ui/sg", name)); err != nil {
			t.Error(err)
		}
	}
}
