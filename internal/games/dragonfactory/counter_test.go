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
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/achievement"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// TestCounterCatalog: every counter achievement has its texts, the titles
// are unique and fit a name tag, and the counts rise within a counter.
func TestCounterCatalog(t *testing.T) {
	titles := map[string]string{}
	keys := map[string]bool{}
	for _, c := range achievement.Counters {
		if keys[c.Key] {
			t.Errorf("counter %s twice", c.Key)
		}
		keys[c.Key] = true
		if g := groupText(c.Group); strings.HasPrefix(g, "achgroup.") {
			t.Errorf("counter %s: group %s has no name", c.Key, c.Group)
		}
		last := int64(0)
		for _, n := range c.Needs {
			if n <= last {
				t.Errorf("counter %s: count %d is not above %d", c.Key, n, last)
			}
			last = n
			id := achievement.CounterID(c.Key, n)
			a, ok := account.Get(id)
			if !ok {
				t.Errorf("%s is not registered", id)
				continue
			}
			title := a.Title(i18n.Default)
			for _, text := range []string{a.Name(i18n.Default), a.Desc(i18n.Default), title} {
				if text == "" || strings.HasPrefix(text, "achievement.") {
					t.Errorf("%s has a missing text: %q", id, text)
				}
			}
			if other, dup := titles[title]; dup {
				t.Errorf("title %q of %s is also %s", title, id, other)
			}
			titles[title] = id
			if n := len([]rune(title)); n > 14 {
				t.Errorf("title %q is %d characters", title, n)
			}
		}
	}
}

func groupText(g string) string { return i18n.T(i18n.Default, achievement.Group(g)) }

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
		if owned := account.Owned(acc); !slices.Contains(owned, achievement.CounterID("stick_dirt", 1)) {
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
