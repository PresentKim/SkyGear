package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/slotdeco"
)

func referenceSlotDeco(tip string) (string, string) {
	rarity := map[string]int{"§a": 1, "§b": 2, "§d": 3, "§6": 4}
	if rest, ok := strings.CutPrefix(tip, "§k§r"); ok && len(rest) >= len("§a") {
		if r, ok := rarity[rest[:len("§a")]]; ok {
			return rest, fmt.Sprintf("tex:textures/ui/sg/deco_%d_0", r)
		}
	}
	return tip, ""
}

func TestSlotDeco(t *testing.T) {
	tips := []string{"", "plain", "§k§r", "§k§r§", "§k§r§c red", "§k§r§a", "§k§r§k§r§b x", "x§k§r§a"}
	for r := slotdeco.Common; r <= slotdeco.Legendary; r++ {
		for tier := 0; tier <= slotdeco.MaxFront; tier++ {
			tips = append(tips, slotdeco.Decorate(r, tier, "이름"), slotdeco.Decorate(r, tier, "")+"\n§7lore")
		}
	}
	for _, tip := range tips {
		gotTip, gotTex := slotDeco(tip)
		wantTip, wantTex := referenceSlotDeco(tip)
		if gotTip != wantTip || gotTex != wantTex {
			t.Errorf("slotDeco(%q) = %q, %q, want %q, %q", tip, gotTip, gotTex, wantTip, wantTex)
		}
	}
}

func TestOreIcon(t *testing.T) {
	for in, want := range map[string]string{
		"":                          "",
		"20840448":                  "aux:20840448",
		"textures/ui/sg/arrow_left": "tex:textures/ui/sg/arrow_left",
	} {
		if got := oreIcon(in); got != want {
			t.Errorf("oreIcon(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUntoken(t *testing.T) {
	if got := untoken(TabOnToken + "§e도감"); got != "§e도감" {
		t.Errorf("untoken = %q", got)
	}
	if got := untoken(CreateToken + BackToken + "메뉴"); got != "메뉴" {
		t.Errorf("untoken = %q", got)
	}
}

// TestLobbySkin: a screen in the lobby is marked for the lobby's look, has
// the shade and the game's name at the top left and keeps its text colours.
func TestLobbySkin(t *testing.T) {
	old := LobbyHeader
	LobbyHeader = func(*player.Player) (string, string) { return "game", "textures/blocks/grass_side_carried" }
	defer func() { LobbyHeader = old }()
	l := newOreLayout()
	layout := lobbySkin(nil, l.window("title", nil, nil, txt("", "x")))
	root := layout.(map[string]any)
	if root["class"] != "mc-screen mc-lobby" {
		t.Fatalf("class %q", root["class"])
	}
	kids := root["children"].([]any)
	if shade := kids[0].(map[string]any); shade["class"] != "mc-lobby-shade" {
		t.Fatalf("first child %v", shade)
	}
	if head := kids[1].(map[string]any); head["class"] != "mc-lobby-head" {
		t.Fatalf("second child %v", head)
	}
	if win := kids[2].(map[string]any); win["ink"] != "dark" {
		t.Errorf("window ink %v", win["ink"])
	}
}
