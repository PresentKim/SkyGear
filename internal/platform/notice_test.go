package platform

import (
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/presentkim/skygear/internal/db"
)

// TestNotices: friend requests and save invites are one list, newest first,
// each with its actions.
func TestNotices(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, id := range []string{"me", "a"} {
		_, _ = d.TouchAccount(id, "", id)
	}
	if _, err := d.SendFriendRequest("a", "me"); err != nil {
		t.Fatal(err)
	}
	pf := &Platform{log: slog.New(slog.DiscardHandler), db: d}
	pf.invites.add("me", invite{saveID: 7, saveName: "isle", from: "b", at: time.Now().Add(time.Minute)})

	list := pf.noticesOf("me", "ko")
	if len(list) != 2 || list[0].Key != "invite:7" || list[1].Key != "friend:a" {
		t.Fatalf("notices %+v", list)
	}
	for _, n := range list {
		if n.Name == "" || n.Feed == "" || len(n.Actions) < 2 {
			t.Errorf("notice %s: %+v", n.Key, n)
		}
	}
	if got := pf.noticesOf("a", "ko"); len(got) != 0 {
		t.Fatalf("a has %d notices", len(got))
	}
}
