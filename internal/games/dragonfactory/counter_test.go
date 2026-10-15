package dragonfactory

import (
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// TestCounterCatalog: every counter achievement has its texts, the titles
// are unique and fit a name tag, and the counts rise within a counter.
func TestCounterCatalog(t *testing.T) {
	titles := map[string]string{}
	keys := map[string]bool{}
	for _, c := range counterCatalog {
		if keys[c.key] {
			t.Errorf("counter %s twice", c.key)
		}
		keys[c.key] = true
		if g := groupText(c.group); strings.HasPrefix(g, "achgroup.") {
			t.Errorf("counter %s: group %s has no name", c.key, c.group)
		}
		last := int64(0)
		for _, n := range c.needs {
			if n <= last {
				t.Errorf("counter %s: count %d is not above %d", c.key, n, last)
			}
			last = n
			id := counterID(c.key, n)
			a, ok := account.Get(id)
			if !ok {
				t.Errorf("%s is not registered", id)
				continue
			}
			for _, text := range []string{a.Name, a.Desc, a.Title} {
				if text == "" || strings.HasPrefix(text, "achievement.") {
					t.Errorf("%s has a missing text: %q", id, text)
				}
			}
			if other, dup := titles[a.Title]; dup {
				t.Errorf("title %q of %s is also %s", a.Title, id, other)
			}
			titles[a.Title] = id
			if n := len([]rune(a.Title)); n > 14 {
				t.Errorf("title %q is %d characters", a.Title, n)
			}
		}
	}
}

func groupText(g string) string { return achievementGroup(g) }

// TestCountersCount: the actions of a player count under their keys.
func TestCountersCount(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		account.Flush()
		account.Init(nil, slog.Default())
		_ = d.Close()
	})
	account.Init(d, slog.Default())
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		acc := instance.AccountID(p)
		if _, err := d.TouchAccount(acc, "", p.Name()); err != nil {
			t.Fatal(err)
		}
		pos := cube.Pos{1, IslandY - 1, 1}
		tx.SetBlock(pos, block.Dirt{}, nil)
		if !s.useStick(p, tx, pos, cube.FaceUp, item.NewStack(item.Stick{}, 1)) {
			t.Fatal("stick on dirt must be used")
		}
		if got := account.Counter(acc, "stick_dirt"); got != 1 {
			t.Fatalf("stick_dirt %d", got)
		}
		if owned := account.Owned(acc); !slices.Contains(owned, counterID("stick_dirt", 1)) {
			t.Fatalf("owned %v", owned)
		}
		countBreak(p, block.Cobblestone{}, true)
		if account.Counter(acc, "hammer_cobble") != 1 || account.Counter(acc, "block_break") != 1 {
			t.Fatal("a hammered cobblestone counts as broken and as crushed")
		}
		countAll(tx, "jump", 3)
		if account.Counter(acc, "jump") != 3 {
			t.Fatal("countAll counts for the players of the world")
		}
	})
}
