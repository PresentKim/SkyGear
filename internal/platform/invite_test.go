package platform

import (
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/presentkim/dragonfactory/internal/db"
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

// A fresh invites value must work without any add (this crashed the server).
func TestInvitesZeroValue(t *testing.T) {
	var iv invites
	iv.remove("a", 1)
	if len(iv.list("b")) != 0 {
		t.Fatal("zero value invites not empty")
	}
}

// TestInviteCodeRenewed: a renewed code is what is shown afterwards, even
// from an older copy of the save (a running instance's).
func TestInviteCodeRenewed(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	_, _ = d.TouchAccount("h", "", "h")
	s, _ := d.CreateSave("h", "test", 1, "s")
	pf := &Platform{log: slog.Default(), db: d}
	first, err := pf.inviteCode(s, false)
	if err != nil {
		t.Fatal(err)
	}
	stale, _ := d.Save(s.ID) // the copy an instance holds
	renewed, err := pf.inviteCode(stale, true)
	if err != nil || renewed == first {
		t.Fatalf("renewed %q (err %v), first %q", renewed, err, first)
	}
	if got, _ := pf.inviteCode(stale, false); got != renewed {
		t.Fatalf("shown %q after renewing, want %q", got, renewed)
	}
}
