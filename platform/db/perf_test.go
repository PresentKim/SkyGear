package db

import (
	"strings"
	"testing"
)

func TestSynchronousNormal(t *testing.T) {
	d := open(t)
	var v int
	if err := d.sql.QueryRow(`PRAGMA synchronous`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", v)
	}
}

func plan(t *testing.T, d *DB, q string, args ...any) string {
	t.Helper()
	rows, err := d.sql.Query(`EXPLAIN QUERY PLAN `+q, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		out = append(out, detail)
	}
	return strings.Join(out, "\n")
}

func TestPerfIndexes(t *testing.T) {
	d := open(t)
	p := plan(t, d, `SELECT `+saveColumns+`
		FROM saves s JOIN save_members m ON m.save_id = s.id
		WHERE m.account_id = ? AND m.left_at IS NULL AND m.role != ? ORDER BY s.last_played DESC`, "a", RoleHost)
	if !strings.Contains(p, "USING INDEX") && !strings.Contains(p, "save_members_account") {
		t.Fatalf("JoinedSaves plan has no index:\n%s", p)
	}
	p = plan(t, d, `SELECT `+accountColumns+` FROM friend_requests r JOIN accounts a ON a.id = r.from_id WHERE r.to_id = ? ORDER BY r.created_at`, "a")
	if !strings.Contains(p, "USING INDEX") {
		t.Fatalf("FriendRequests-to plan has no index:\n%s", p)
	}
}

func TestSettingsOneQuery(t *testing.T) {
	d := open(t)
	if _, err := d.TouchAccount("u1", "x", "A"); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{"a": "1", "b": "0", "c": "x"} {
		if err := d.SetSetting("u1", k, v); err != nil {
			t.Fatal(err)
		}
	}
	got, err := d.Settings("u1")
	if err != nil || len(got) != 3 || got["a"] != "1" || got["b"] != "0" || got["c"] != "x" {
		t.Fatalf("Settings = %v, %v", got, err)
	}
}

func TestMembersTieBreakInsertionOrder(t *testing.T) {
	d := open(t)
	_, _ = d.TouchAccount("h", "", "h")
	_, _ = d.TouchAccount("a", "", "a")
	s, _ := d.CreateSave("h", 1, "s")
	if _, err := d.sql.Exec(`UPDATE save_members SET joined_at = 1000 WHERE save_id = ?`, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.sql.Exec(`INSERT INTO save_members (save_id, account_id, role, joined_at) VALUES (?, 'a', ?, 1000)`, s.ID, RoleGuest); err != nil {
		t.Fatal(err)
	}
	m, err := d.Members(s.ID)
	if err != nil || len(m) != 2 || m[0].Role != RoleHost {
		t.Fatalf("members = %+v %v", m, err)
	}
}
