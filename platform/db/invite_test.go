package db

import (
	"errors"
	"testing"
	"time"
)

func TestInvites(t *testing.T) {
	d := open(t)
	for _, id := range []string{"h", "g", "x"} {
		_, _ = d.TouchAccount(id, "", id)
	}
	s, _ := d.CreateSave("h", 1, "isle")
	s2, _ := d.CreateSave("h", 2, "other")
	if err := d.AddInvite(s.ID, "g", "h"); err != nil {
		t.Fatal(err)
	}
	if err := d.AddInvite(s.ID, "g", "x"); err != nil { // replaces
		t.Fatal(err)
	}
	_ = d.AddInvite(s2.ID, "g", "h")
	list, err := d.Invites("g", time.Now().Add(-time.Hour))
	if err != nil || len(list) != 2 {
		t.Fatalf("invites = %+v, %v", list, err)
	}
	for _, in := range list {
		if in.SaveID == s.ID && (in.From.ID != "x" || in.SaveName != "isle") {
			t.Fatalf("replaced invite = %+v", in)
		}
	}
	_ = d.RemoveInvite(s2.ID, "g")
	if list, _ := d.Invites("g", time.Now().Add(-time.Hour)); len(list) != 1 {
		t.Fatalf("after remove = %+v", list)
	}
	// expired invites are dropped
	if list, _ := d.Invites("g", time.Now().Add(time.Hour)); len(list) != 0 {
		t.Fatalf("expired kept = %+v", list)
	}
	// deleting the save deletes its invites
	_ = d.AddInvite(s.ID, "g", "h")
	_ = d.DeleteSave(s.ID)
	if list, _ := d.Invites("g", time.Now().Add(-time.Hour)); len(list) != 0 {
		t.Fatalf("invite to deleted save = %+v", list)
	}
}

func TestCloseFriends(t *testing.T) {
	d := open(t)
	for _, id := range []string{"a", "b", "c"} {
		_, _ = d.TouchAccount(id, "", id)
	}
	if err := d.SetCloseFriend("a", "b", true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("close friend without friendship err = %v", err)
	}
	_, _ = d.SendFriendRequest("a", "b")
	_ = d.AcceptFriend("b", "a")
	if err := d.SetCloseFriend("a", "b", true); err != nil {
		t.Fatal(err)
	}
	if ok, _ := d.IsCloseFriend("a", "b"); !ok {
		t.Fatal("a -> b not close")
	}
	if ok, _ := d.IsCloseFriend("b", "a"); ok {
		t.Fatal("close friendship must be one way")
	}
	if m, _ := d.CloseFriends("a"); !m["b"] || len(m) != 1 {
		t.Fatalf("close friends of a = %v", m)
	}
	if hosts, _ := d.CloseFriendOf("b"); len(hosts) != 1 || hosts[0].ID != "a" {
		t.Fatalf("close friend of b = %+v", hosts)
	}
	_ = d.RemoveFriend("a", "b")
	if hosts, _ := d.CloseFriendOf("b"); len(hosts) != 0 {
		t.Fatalf("close friendship kept after removal = %+v", hosts)
	}
}

func TestBlockDropsInvites(t *testing.T) {
	d := open(t)
	for _, id := range []string{"h", "g"} {
		_, _ = d.TouchAccount(id, "", id)
	}
	s, _ := d.CreateSave("h", 1, "isle")
	_ = d.AddInvite(s.ID, "g", "h")
	if ok, _ := d.HasInvite(s.ID, "g", time.Now().Add(-time.Hour)); !ok {
		t.Fatal("invite missing")
	}
	if err := d.Block("h", "g"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := d.HasInvite(s.ID, "g", time.Now().Add(-time.Hour)); ok {
		t.Fatal("invite kept after block")
	}
}
