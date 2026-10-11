package hud

import "testing"

func TestRowsUnique(t *testing.T) {
	var b Board
	b.Add(Quest, "a")
	b.Add(Quest, "a")
	b.Add(TopLeft, "a")
	rows := b.Rows()
	if rows[0] != Quest+"a" || rows[1] != Quest+"a§r" || rows[2] != TopLeft+"a" {
		t.Fatalf("rows = %q", rows)
	}
	if Strip(rows[1]) != "a" || Strip(TopLeft+"Hour") != "Hour" {
		t.Fatalf("strip = %q", Strip(rows[1]))
	}
}

func TestRowsLimit(t *testing.T) {
	var b Board
	for range 20 {
		b.Add(Quest, "x")
	}
	if len(b.Rows()) != maxRows {
		t.Fatalf("%d rows", len(b.Rows()))
	}
}
