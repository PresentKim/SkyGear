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
