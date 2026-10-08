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
