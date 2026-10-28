package db

import (
	"database/sql"
	"errors"
	"os"
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

	s, err := d.CreateSave("u1", 1, "first")
	if err != nil {
		t.Fatal(err)
	}
	if s.Day != 1 || s.InstanceTime != 0 {
		t.Fatalf("new save clock = day %d time %v", s.Day, s.InstanceTime)
	}
	if _, err := d.CreateSave("u1", 1, "dup"); !errors.Is(err, ErrSlotUsed) {
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

	saves, err := d.SavesOf("u1")
	if err != nil || len(saves) != 1 {
		t.Fatalf("saves = %v, %v", saves, err)
	}
	if err := d.DeleteSave(s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SaveState(s.ID, "level"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("save state after delete err = %v", err)
	}
	if _, err := d.CreateSave("u1", 1, "again"); err != nil {
		t.Fatalf("slot not reusable after delete: %v", err)
	}
}

func TestCopySave(t *testing.T) {
	d := open(t)
	for _, id := range []string{"owner", "admin"} {
		if _, err := d.TouchAccount(id, "x-"+id, id); err != nil {
			t.Fatal(err)
		}
	}
	src, err := d.CreateSave("owner", 1, "island")
	if err != nil {
		t.Fatal(err)
	}
	_ = d.UpdateSaveClock(src.ID, time.Hour, 5, time.Minute)
	_ = d.SetSaveState(src.ID, "level", []byte("7"))
	_ = d.SetPlayerState(src.ID, "owner", []byte{4, 2})
	cp, err := d.CopySave(src.ID, "admin", 2, "island copy")
	if err != nil {
		t.Fatal(err)
	}
	if cp.Owner != "admin" || cp.Slot != 2 || cp.Name != "island copy" || cp.Day != 5 || cp.InstanceTime != time.Hour {
		t.Fatalf("copy = %+v", cp)
	}
	if v, err := d.SaveState(cp.ID, "level"); err != nil || string(v) != "7" {
		t.Fatalf("copied state = %q, %v", v, err)
	}
	if _, st, err := d.PlayerState(cp.ID, "admin"); err != nil || len(st) != 2 {
		t.Fatalf("copied player state = %v, %v", st, err)
	}
	if r, err := d.MemberRole(cp.ID, "admin"); err != nil || r != RoleHost {
		t.Fatalf("copy host = %v, %v", r, err)
	}
	if _, err := d.MemberRole(cp.ID, "owner"); err == nil {
		t.Error("the source owner is a member of the copy")
	}
	if _, err := d.CopySave(src.ID, "admin", 2, "again"); !errors.Is(err, ErrSlotUsed) {
		t.Fatalf("copy into a used slot err = %v", err)
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

func TestSetPlayerStatesKeepsMoney(t *testing.T) {
	d := open(t)
	_, _ = d.TouchAccount("h", "", "h")
	_, _ = d.TouchAccount("g", "", "g")
	s, _ := d.CreateSave("h", 1, "s")
	_ = d.SetMoney(s.ID, "h", 7)
	if err := d.SetPlayerStates(s.ID, map[string][]byte{"h": {1}, "g": {2}}); err != nil {
		t.Fatal(err)
	}
	if money, state, err := d.PlayerState(s.ID, "h"); err != nil || money != 7 || string(state) != "\x01" {
		t.Fatalf("host = %d %v %v", money, state, err)
	}
	if _, state, err := d.PlayerState(s.ID, "g"); err != nil || string(state) != "\x02" {
		t.Fatalf("guest = %v %v", state, err)
	}
}

func TestMembersAndInviteCodes(t *testing.T) {
	d := open(t)
	_, _ = d.TouchAccount("h", "", "h")
	_, _ = d.TouchAccount("g", "", "g")
	s, _ := d.CreateSave("h", 1, "s")
	s2, _ := d.CreateSave("h", 2, "s2")
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

func TestAchievementsAndCosmetics(t *testing.T) {
	d := open(t)
	if _, err := d.TouchAccount("a", "", "A"); err != nil {
		t.Fatal(err)
	}
	if n, err := d.GrantAchievement("a", "x"); err != nil || !n {
		t.Fatalf("first grant: %v %v", n, err)
	}
	if n, _ := d.GrantAchievement("a", "x"); n {
		t.Fatal("second grant reported new")
	}
	_, _ = d.GrantAchievement("a", "y")
	if ids, _ := d.Achievements("a"); len(ids) != 2 {
		t.Fatalf("achievements %v", ids)
	}
	if err := d.SetTitle("a", "x"); err != nil {
		t.Fatal(err)
	}
	if err := d.SetNameColor("a", "gold"); err != nil {
		t.Fatal(err)
	}
	if a, _ := d.Account("a"); a.Title != "x" || a.NameColor != "gold" {
		t.Fatalf("account %+v", a)
	}
}

func TestBackupCopiesData(t *testing.T) {
	d := open(t)
	if _, err := d.TouchAccount("a", "", "A"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "copy.sqlite")
	if err := d.Backup(path); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		t.Fatalf("backup file: %v %v", fi, err)
	}
	c, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if a, err := c.Account("a"); err != nil || a.Name != "A" {
		t.Fatalf("backup account %+v %v", a, err)
	}
}

func TestBackupInMemory(t *testing.T) {
	d, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	path := filepath.Join(t.TempDir(), "copy.sqlite")
	if err := d.Backup(path); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		t.Fatalf("backup file: %v %v", fi, err)
	}
}

func TestMigrationDropsGameTypeAndKeepsSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite", fileDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range migrations[:6] {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`PRAGMA user_version = 6`,
		`INSERT INTO accounts (id, name, created_at, last_seen) VALUES ('h', 'host', 0, 0)`,
		`INSERT INTO saves (game_type, owner, slot, name, invite_code, day, created_at, last_played) VALUES ('skygear', 'h', 2, 'island', 'ABC', 5, 0, 0)`,
		`INSERT INTO save_members (save_id, account_id, role, joined_at) VALUES (1, 'h', 'host', 0)`,
		`INSERT INTO save_state (save_id, key, value) VALUES (1, 'k', x'01')`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = old.Close()

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	saves, err := d.SavesOf("h")
	if err != nil || len(saves) != 1 || saves[0].Slot != 2 || saves[0].Day != 5 || saves[0].InviteCode != "ABC" {
		t.Fatalf("saves = %+v, %v", saves, err)
	}
	if r, err := d.MemberRole(saves[0].ID, "h"); err != nil || r != RoleHost {
		t.Fatalf("member kept = %v, %v", r, err)
	}
	if v, err := d.SaveState(saves[0].ID, "k"); err != nil || len(v) != 1 {
		t.Fatalf("state kept = %v, %v", v, err)
	}
	if _, err := d.CreateSave("h", 2, "dup"); !errors.Is(err, ErrSlotUsed) {
		t.Fatalf("slot still unique: err = %v", err)
	}
	if err := d.DeleteSave(saves[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.MemberRole(saves[0].ID, "h"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("members cascade after migration: err = %v", err)
	}
}
