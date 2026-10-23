package ui

import "testing"

func TestOreIcon(t *testing.T) {
	for in, want := range map[string]string{
		"":                          "",
		"20840448":                  "aux:20840448",
		"textures/ui/df/arrow_left": "tex:textures/ui/df/arrow_left",
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
	if got := untoken(CardToken + UntitledToken + "메뉴"); got != "메뉴" {
		t.Errorf("untoken = %q", got)
	}
}
