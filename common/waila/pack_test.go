package waila

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var packToken = regexp.MustCompile(`__waila:[a-z_.0-9]+__`)

// TestPackTokens keeps the pack's title tokens and the ones Go can emit the
// same set. The pack strips every token it knows from the title on each
// rebuild, so a token only one side knows is either a missing strip (the
// token shows as text) or wasted expression work.
func TestPackTokens(t *testing.T) {
	emitted := []string{anchor, lineToken}
	for _, l := range []Layout{LayoutChest, LayoutFurnace, LayoutHopper, LayoutBrewing} {
		emitted = append(emitted, Container{Layout: l}.token())
	}
	// The block state line is switched off (docs/tech/waila.md): the pack
	// still names it, Go never sends it.
	pack := []string{"__waila:block_states__"}

	dir := filepath.Join("..", "..", "assets", "resourcepack", "waila", "ui", "waila")
	text, err := os.ReadFile(filepath.Join(dir, "main.json"))
	if err != nil {
		t.Fatal(err)
	}
	found := packToken.FindAllString(string(text), -1)
	for _, tok := range emitted {
		if !slices.Contains(found, tok) {
			t.Errorf("main.json does not handle %s, which the server sends", tok)
		}
	}
	for _, tok := range found {
		if !slices.Contains(emitted, tok) && !slices.Contains(pack, tok) {
			t.Errorf("main.json handles %s, which the server never sends", tok)
		}
	}
	// Every token but the anchor (stripped earlier by $var_title_text) must
	// be in the A: panel text expression, or it shows as text.
	var strip string
	for _, line := range strings.Split(string(text), "\n") {
		if strings.Contains(line, "'%.19s'") {
			strip = line
		}
	}
	for _, tok := range emitted {
		if tok != anchor && !strings.Contains(strip, tok) {
			t.Errorf("the A: text expression does not strip %s", tok)
		}
	}
}
