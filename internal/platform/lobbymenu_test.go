package platform

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestLobbyMenuMatchesThePack: the main menu form and the picture of it the
// pack draws on the HUD (ui/df_lobby.json menu) have the same buttons in the
// same order, so closing and opening the form does not change the screen,
// and every label is a key of both language files.
func TestLobbyMenuMatchesThePack(t *testing.T) {
	const pack = "../../assets/resourcepack/dragonfactory/"
	raw, err := os.ReadFile(pack + "ui/df_lobby.json")
	if err != nil {
		t.Fatal(err)
	}
	src := regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAllString(string(raw), "")
	var doc struct {
		Menu struct {
			Controls []map[string]struct {
				Controls []map[string]map[string]string `json:"controls"`
			} `json:"controls"`
		} `json:"menu"`
	}
	if err := json.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatal(err)
	}
	var drawn [][2]string
	for _, c := range doc.Menu.Controls {
		for _, col := range c {
			for _, b := range col.Controls {
				for _, vars := range b {
					drawn = append(drawn, [2]string{vars["$df_launcher_label"], vars["$df_launcher_icon"]})
				}
			}
		}
	}
	opts := (&Platform{}).lobbyMenu()
	if len(drawn) != len(opts) {
		t.Fatalf("the pack draws %d buttons, the form has %d", len(drawn), len(opts))
	}
	for i, o := range opts {
		if drawn[i] != [2]string{o.Label, o.Icon} {
			t.Errorf("button %d: form %q %q, pack %q %q", i, o.Label, o.Icon, drawn[i][0], drawn[i][1])
		}
		for _, lang := range []string{"ko_KR", "en_US"} {
			text, err := os.ReadFile(pack + "texts/" + lang + ".lang")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains("\n"+string(text), "\n"+o.Label+"=") {
				t.Errorf("%s has no %s", lang, o.Label)
			}
		}
	}
}
