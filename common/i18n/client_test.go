package i18n

import (
	"testing"
	"time"
)

func TestPlain(t *testing.T) {
	name := ClientKey("item.diamond.name", "§b다이아몬드")
	s := "§f" + name + " ×3, " + ClientDate(time.Unix(1700000000, 0), "2023-11-14")
	if got, want := Plain(s), "§f§b다이아몬드 ×3, 2023-11-14"; got != want {
		t.Fatalf("Plain = %q, want %q", got, want)
	}
	if got := ClientKey("", "x"); got != "x" {
		t.Fatalf("ClientKey without key = %q", got)
	}
}
