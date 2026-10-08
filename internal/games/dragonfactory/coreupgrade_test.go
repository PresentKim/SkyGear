package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
)

func TestCoreUpgradeEffects(t *testing.T) {
	s := testSession(t)
	rec := &CoreRecord{Type: "earth", Tier: 1}
	if got := s.coreInterval(rec); got != 60 {
		t.Fatalf("tier 1 interval = %d", got)
	}
	rec.Upgrades = map[string]int{upgradeSpeed: 2}
	if got := s.coreInterval(rec); got != 60*76/100 {
		t.Fatalf("speed 2 interval = %d", got)
	}
	weight := func(rec *CoreRecord, b world.Block) int {
		for _, e := range coreTable(rec) {
			if itemKey(e.Block.(world.Item)) == itemKey(b.(world.Item)) {
				return e.Weight
			}
		}
		return 0
	}
	if weight(rec, block.IronOre{}) != 0 {
		t.Fatal("tier 1 generates iron ore")
	}
	rec.Tier = 2
	if weight(rec, block.IronOre{}) == 0 || s.coreInterval(rec) != 50*76/100 {
		t.Fatalf("tier 2: iron weight %d, interval %d", weight(rec, block.IronOre{}), s.coreInterval(rec))
	}
	coal := weight(rec, block.CoalOre{})
	rec.Upgrades[upgradeLuck] = 2
	if got := weight(rec, block.CoalOre{}); got != coal*2 {
		t.Fatalf("luck 2: coal weight %d, was %d", got, coal)
	}
	if weight(rec, block.Dirt{}) != 20 {
		t.Fatal("luck changed a common block")
	}
	rec.Excluded = []string{itemKey(block.Dirt{})}
	if weight(rec, block.Dirt{}) != 0 {
		t.Fatal("filtered dirt still generated")
	}
}

func TestCoreUpgradeMenu(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		rec := s.state.Cores["earth"]
		menu := func() []ui.Option {
			_, opts, ok := s.coreUpgradeMenu("earth")
			if !ok {
				t.Fatal("no menu")
			}
			return opts
		}
		s.state.CorePoints = 5
		// The tier needs Lv.4: refused, points kept.
		press(t, p, tx, menu(), "§a티어 2로")
		if rec.Tier != 1 || s.state.CorePoints != 5 {
			t.Fatalf("tier %d points %d before Lv.4", rec.Tier, s.state.CorePoints)
		}
		s.state.Level = 4
		press(t, p, tx, menu(), "§a티어 2로")
		if rec.Tier != 2 || s.state.CorePoints != 2 {
			t.Fatalf("tier %d points %d after buying tier 2", rec.Tier, s.state.CorePoints)
		}
		// Speed 0 → 1 costs 1, 1 → 2 costs 2: only the first fits.
		press(t, p, tx, menu(), "§b속도 Lv.0")
		press(t, p, tx, menu(), "§b속도 Lv.1")
		if rec.level(upgradeSpeed) != 1 || s.state.CorePoints != 1 {
			t.Fatalf("speed %d points %d", rec.level(upgradeSpeed), s.state.CorePoints)
		}
		s.state.CorePoints = filterCost
		press(t, p, tx, menu(), "§f생성 필터 해금")
		if !rec.FilterUnlocked || s.state.CorePoints != 0 {
			t.Fatal("filter not unlocked")
		}
	})
}

func TestCoreFilterKeepsOneBlock(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		rec := s.state.Cores["earth"]
		rec.FilterUnlocked = true
		all := len(coreTypes["earth"].table(1))
		// Turning every block off leaves the last one on.
		for range all + 2 {
			opts, _ := s.coreFilterMenu("earth")
			press(t, p, tx, opts, "§a켜짐")
		}
		if n := len(coreTable(rec)); n != 1 {
			t.Fatalf("%d blocks left", n)
		}
		opts, _ := s.coreFilterMenu("earth")
		press(t, p, tx, opts, "§8꺼짐")
		if n := len(coreTable(rec)); n != 2 {
			t.Fatalf("%d blocks after turning one back on", n)
		}
	})
}
