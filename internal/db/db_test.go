package db

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func open(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestAccountAndSaves(t *testing.T) {
	d := open(t)
	if _, err := d.TouchAccount("u1", "x1", "Alice"); err != nil {
		t.Fatal(err)
	}
	a, err := d.TouchAccount("u1", "x1", "Alice2")
	if err != nil || a.Name != "Alice2" || a.GuideVersion != 0 {
		t.Fatalf("account = %+v, %v", a, err)
	}
	if err := d.SetGuideVersion("u1", 3); err != nil {
		t.Fatal(err)
	}

	s, err := d.CreateSave("u1", "dragonfactory", 1, "first")
	if err != nil {
		t.Fatal(err)
	}
	if s.Day != 1 || s.InstanceTime != 0 {
		t.Fatalf("new save clock = day %d time %v", s.Day, s.InstanceTime)
	}
	if _, err := d.CreateSave("u1", "dragonfactory", 1, "dup"); !errors.Is(err, ErrSlotUsed) {
		t.Fatalf("duplicate slot err = %v", err)
	}
	if r, err := d.MemberRole(s.ID, "u1"); err != nil || r != RoleHost || !r.CanActivate() {
		t.Fatalf("owner role = %v, %v", r, err)
	}

	if err := d.UpdateSaveClock(s.ID, 90*time.Minute, 2, 30*time.Minute); err != nil {
		t.Fatal(err)
	}
	s, _ = d.Save(s.ID)
	if s.InstanceTime != 90*time.Minute || s.Day != 2 || s.DayTime != 30*time.Minute {
		t.Fatalf("clock = %v day %d %v", s.InstanceTime, s.Day, s.DayTime)
	}

	if _, _, err := d.PlayerState(s.ID, "u1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing state err = %v", err)
	}
	if err := d.SetPlayerState(s.ID, "u1", []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if _, st, err := d.PlayerState(s.ID, "u1"); err != nil || len(st) != 3 {
		t.Fatalf("state = %v, %v", st, err)
	}
	if err := d.SetSaveState(s.ID, "level", []byte("1")); err != nil {
		t.Fatal(err)
	}

	saves, err := d.SavesOf("u1", "dragonfactory")
	if err != nil || len(saves) != 1 {
		t.Fatalf("saves = %v, %v", saves, err)
	}
	if err := d.DeleteSave(s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SaveState(s.ID, "level"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("save state after delete err = %v", err)
	}
	if _, err := d.CreateSave("u1", "dragonfactory", 1, "again"); err != nil {
		t.Fatalf("slot not reusable after delete: %v", err)
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	for range 2 {
		d, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		_ = d.Close()
	}
}

func TestCheques(t *testing.T) {
	d := open(t)
	_, _ = d.TouchAccount("u1", "", "a")
	s, _ := d.CreateSave("u1", "g", 1, "s")
	s2, _ := d.CreateSave("u1", "g", 2, "s2")
	if err := d.IssueCheque("c1", s.ID, 500, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.RedeemCheque("c1", s2.ID, "u1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cheque redeemed in another save: %v", err)
	}
	if amt, err := d.RedeemCheque("c1", s.ID, "u1"); err != nil || amt != 500 {
		t.Fatalf("redeem = %d, %v", amt, err)
	}
	if _, err := d.RedeemCheque("c1", s.ID, "u1"); !errors.Is(err, ErrChequeUsed) {
		t.Fatalf("second redeem err = %v", err)
	}
}

func TestFriends(t *testing.T) {
	d := open(t)
	for _, n := range []string{"a", "b", "c"} {
		_, _ = d.TouchAccount(n, "", "Name"+n)
	}
	if acc, err := d.AccountByName("nameB"); err != nil || acc.ID != "b" {
		t.Fatalf("by name = %v, %v", acc, err)
	}
	if _, err := d.SendFriendRequest("a", "a"); !errors.Is(err, ErrSelf) {
		t.Fatal("self request allowed")
	}
	if acc, err := d.SendFriendRequest("a", "b"); err != nil || acc {
		t.Fatal(acc, err)
	}
	reqs, _ := d.FriendRequests("b")
	if len(reqs) != 1 || reqs[0].ID != "a" {
		t.Fatalf("requests = %v", reqs)
	}
	// b asking back accepts.
	if acc, err := d.SendFriendRequest("b", "a"); err != nil || !acc {
		t.Fatalf("mutual request: %v %v", acc, err)
	}
	if f, _ := d.Friends("a"); len(f) != 1 || f[0].ID != "b" {
		t.Fatalf("friends of a = %v", f)
	}
	if f, _ := d.Friends("b"); len(f) != 1 {
		t.Fatal("friendship must be mutual")
	}
	if _, err := d.SendFriendRequest("a", "b"); !errors.Is(err, ErrAlreadyFriends) {
		t.Fatalf("already friends err = %v", err)
	}
	if err := d.Block("c", "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SendFriendRequest("a", "c"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("request to a blocker err = %v", err)
	}
	if err := d.Block("a", "b"); err != nil {
		t.Fatal(err)
	}
	if f, _ := d.Friends("b"); len(f) != 0 {
		t.Fatal("blocking must end the friendship")
	}
	_ = d.Unblock("a", "b")
	if bl, _ := d.BlockedAccounts("a"); len(bl) != 0 {
		t.Fatal("unblock failed")
	}
}

func TestMembersAndInviteCodes(t *testing.T) {
	d := open(t)
	_, _ = d.TouchAccount("h", "", "h")
	_, _ = d.TouchAccount("g", "", "g")
	s, _ := d.CreateSave("h", "g", 1, "s")
	s2, _ := d.CreateSave("h", "g", 2, "s2")
	if err := d.SetInviteCode(s.ID, "ABC123"); err != nil {
		t.Fatal(err)
	}
	if err := d.SetInviteCode(s2.ID, "ABC123"); !errors.Is(err, ErrCodeTaken) {
		t.Fatalf("duplicate code err = %v", err)
	}
	if got, err := d.SaveByInviteCode("ABC123"); err != nil || got.ID != s.ID || got.InviteCode != "ABC123" {
		t.Fatalf("by code = %v %v", got, err)
	}
	_ = d.AddMember(s.ID, "g", RoleGuest)
	if n, _ := d.ActiveMembers(s.ID); n != 2 {
		t.Fatalf("active members = %d", n)
	}
	if j, err := d.JoinedSaves("g"); err != nil || len(j) != 1 || j[0].ID != s.ID {
		t.Fatalf("joined = %v %v", j, err)
	}
	if j, _ := d.JoinedSaves("h"); len(j) != 0 {
		t.Fatal("hosts must not list their own saves as joined")
	}
	_ = d.LeaveMember(s.ID, "h")
	if n, _ := d.ActiveMembers(s.ID); n != 2 {
		t.Fatal("the host cannot leave")
	}
	_ = d.LeaveMember(s.ID, "g")
	if n, _ := d.ActiveMembers(s.ID); n != 1 {
		t.Fatal("leave failed")
	}
	if m, _ := d.Members(s.ID); len(m) != 2 || !m[1].Left {
		t.Fatalf("left members must be kept: %+v", m)
	}
	if _, err := d.MemberRole(s.ID, "g"); !errors.Is(err, ErrNotFound) {
		t.Fatal("a member that left keeps a role")
	}
}
