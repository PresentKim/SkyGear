package account

import (
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/db"
)

func TestGrantAndDisplayName(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	Init(d, slog.Default())
	Register(Achievement{ID: "test_a", Name: "시험", Desc: "시험 업적", Title: "시험관"})

	id := uuid.New()
	if _, err := d.TouchAccount(id.String(), "", "Hana"); err != nil {
		t.Fatal(err)
	}
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	t.Cleanup(func() { _ = w.Close() })
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "Hana", UUID: id})
	<-w.Do(func(tx *world.Tx) {
		p := tx.AddEntity(h).(*player.Player)
		Grant(p, "test_a")
		Grant(p, "test_a")
		if got := Owned(id.String()); len(got) != 1 {
			t.Fatalf("owned %v", got)
		}
		_ = d.SetTitle(id.String(), "test_a")
		_ = d.SetNameColor(id.String(), "gold")
		Refresh(p)
		if n := DisplayName(p); !strings.Contains(n, "[시험관]") || !strings.Contains(n, "§6Hana") {
			t.Fatalf("display name %q", n)
		}
		if p.NameTag() != DisplayName(p) {
			t.Fatalf("name tag %q", p.NameTag())
		}
	}).Done()
}

func TestCounterTiers(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	Init(d, slog.Default())
	Register(Achievement{ID: "c_one", Name: "하나", Desc: "d", Title: "첫째"}, Achievement{ID: "c_ten", Name: "열", Desc: "d", Title: "열째"})
	RegisterCounter("test_count", Tier{1, "c_one"}, Tier{10, "c_ten"})

	id := uuid.New()
	if _, err := d.TouchAccount(id.String(), "", "Hana"); err != nil {
		t.Fatal(err)
	}
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	t.Cleanup(func() { _ = w.Close() })
	h := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "Hana", UUID: id})
	<-w.Do(func(tx *world.Tx) {
		p := tx.AddEntity(h).(*player.Player)
		Count(p, "test_count", 1)
		Count(p, "unregistered", 5)
		if got := Owned(id.String()); len(got) != 1 || got[0] != "c_one" {
			t.Fatalf("owned after 1: %v", got)
		}
		Count(p, "test_count", 8)
		if len(Owned(id.String())) != 1 {
			t.Fatalf("owned after 9: %v", Owned(id.String()))
		}
		Count(p, "test_count", 5)
		if len(Owned(id.String())) != 2 {
			t.Fatalf("owned after 14: %v", Owned(id.String()))
		}
		if have, need, ok := Progress(id.String(), "c_ten"); !ok || have != 14 || need != 10 {
			t.Fatalf("progress %d %d %v", have, need, ok)
		}
		Flush()
		stored, _ := d.Counters(id.String())
		if stored["test_count"] != 14 || len(stored) != 1 {
			t.Fatalf("stored %v", stored)
		}
		Forget(p)
		if got := Counter(id.String(), "test_count"); got != 14 {
			t.Fatalf("reloaded %d", got)
		}
	}).Done()
}
