package platform

import (
	"strings"
	"testing"
	"time"
)

func TestInviteCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		c := newCode()
		if len(c) != 6 || strings.Trim(c, codeAlphabet) != "" {
			t.Fatalf("bad code %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 190 {
		t.Fatalf("codes repeat too often: %d unique of 200", len(seen))
	}
}

func TestInvitesReplaceAndExpire(t *testing.T) {
	var iv invites
	iv.add("a", invite{saveID: 1, at: time.Now()})
	iv.add("a", invite{saveID: 1, saveName: "again", at: time.Now()})
	iv.add("a", invite{saveID: 2, at: time.Now().Add(-inviteTTL - time.Second)})
	l := iv.list("a")
	if len(l) != 1 || l[0].saveName != "again" {
		t.Fatalf("invites = %+v, want only the newest invite to save 1", l)
	}
	iv.remove("a", 1)
	if len(iv.list("a")) != 0 {
		t.Fatal("remove failed")
	}
}
