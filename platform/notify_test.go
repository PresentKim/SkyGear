package platform

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/presentkim/skygear/platform/db"
)

// TestNotify: friend requests and save invites are one list, newest first,
// each with its actions.
func TestNotify(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, id := range []string{"me", "a", "b"} {
		_, _ = d.TouchAccount(id, "", id)
	}
	if _, err := d.SendFriendRequest("a", "me"); err != nil {
		t.Fatal(err)
	}
	pf := &Platform{log: slog.New(slog.DiscardHandler), db: d}
	time.Sleep(2 * time.Millisecond) // the invite is the newer notice
	isle, _ := d.CreateSave("b", 1, "isle")
	if err := d.AddInvite(isle.ID, "me", "b"); err != nil {
		t.Fatal(err)
	}

	list := pf.notifiesOf("me", "ko")
	if len(list) != 2 || list[0].Key != fmt.Sprintf("invite:%d", isle.ID) || list[1].Key != "friend:a" {
		t.Fatalf("notices %+v", list)
	}
	for _, n := range list {
		if n.Name == "" || n.Feed == "" || len(n.Actions) < 2 {
			t.Errorf("notice %s: %+v", n.Key, n)
		}
	}
	if got := pf.notifiesOf("a", "ko"); len(got) != 0 {
		t.Fatalf("a has %d notices", len(got))
	}
}
