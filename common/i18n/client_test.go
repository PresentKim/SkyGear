package i18n

import (
	"encoding/json"
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
	data, _ := json.Marshal(map[string]string{"title": s})
	var back map[string]string
	if err := json.Unmarshal(PlainBytes(data), &back); err != nil || back["title"] != Plain(s) {
		t.Fatalf("PlainBytes = %s (%v)", PlainBytes(data), err)
	}
}
