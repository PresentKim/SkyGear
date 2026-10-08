package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/account"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Core upgrades (docs/design/02-dragonfactory.md §3.1, §3.2): core points
// earned by levelling up buy tiers (more blocks, faster base interval),
// upgrade levels (speed, luck, yield) and the block filter. Costs and
// effects are balancing drafts (docs/design/open-questions.md).

// coreUpgrade is an upgrade a core can take levels of.
type coreUpgrade struct {
	id, name, effect string
}

const (
	upgradeSpeed = "speed"
	upgradeLuck  = "luck"
	upgradeYield = "yield"
)

var coreUpgrades = []coreUpgrade{
	{upgradeSpeed, "속도", "생성 주기 -12%/단계"},
	{upgradeLuck, "행운", "희귀 블록 비중 +50%/단계"},
	{upgradeYield, "산출", "생성할 때 8%/단계 확률로 하나 더(위로 튀어나옴)"},
}

const (
	// maxUpgradeLevel is the highest level of each upgrade.
	maxUpgradeLevel = 3
	// filterCost is the core points that unlock the block filter.
	filterCost = 2
)

// upgradeCost is the core points the next level of an upgrade costs.
func upgradeCost(level int) int { return level + 1 }

func (r *CoreRecord) level(id string) int { return r.Upgrades[id] }

// coreInterval is the generation interval of a core in ticks: the tier's
// base interval shortened by the speed upgrade.
func (s *session) coreInterval(rec *CoreRecord) int {
	t := coreTypes[rec.Type]
	base := t.Tiers[min(max(rec.Tier, 1), len(t.Tiers))-1].Interval
	return max(10, base*(100-12*rec.level(upgradeSpeed))/100)
}

// coreTable returns what a core generates with its tier, luck and filter.
func coreTable(rec *CoreRecord) []WeightedBlock {
	luck := rec.level(upgradeLuck)
	var out []WeightedBlock
	for _, e := range coreTypes[rec.Type].table(max(rec.Tier, 1)) {
		if slices.Contains(rec.Excluded, itemKey(e.Block.(world.Item))) {
			continue
		}
		if e.Rare {
			e.Weight = e.Weight * (2 + luck) / 2
		}
		out = append(out, e)
	}
	return out
}

// generate makes a block above the core at pos, and with the yield upgrade
// sometimes one more that pops out on top.
func (s *session) generate(tx *world.Tx, pos cube.Pos, rec *CoreRecord) {
	table := coreTable(rec)
	if len(table) == 0 {
		return
	}
	b := pick(s.rand, table)
	tx.SetBlock(pos.Side(cube.FaceUp), b, nil)
	if s.rand.IntN(100) < 8*rec.level(upgradeYield) {
		s.dropAt(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 1.7, 0}), item.NewStack(b.(world.Item), 1))
	}
}

// spendPoints takes n core points, telling p if there are not enough.
func (s *session) spendPoints(p *player.Player, n int) bool {
	if s.state.CorePoints < n {
		p.Messagef("§c코어 강화 포인트가 부족합니다. (필요 %d, 보유 %d)", n, s.state.CorePoints)
		return false
	}
	s.state.CorePoints -= n
	return true
}

// showCoreUpgrades shows the upgrades of a core and buys them.
func (s *session) showCoreUpgrades(p *player.Player, typ string) {
	if body, opts, ok := s.coreUpgradeMenu(typ); ok {
		ui.CardMenu(p, coreTypes[typ].Name+" 강화", body, opts...)
	}
}

func (s *session) coreUpgradeMenu(typ string) (string, []ui.Option, bool) {
	rec := s.state.Cores[typ]
	if rec == nil {
		return "", nil, false
	}
	t := coreTypes[typ]
	var b strings.Builder
	fmt.Fprintf(&b, "§f%s §7· 티어 %d · 코어 강화 포인트 §b%d\n", t.Name, rec.Tier, s.state.CorePoints)
	fmt.Fprintf(&b, "§7생성 주기 %.1f초\n", float64(s.coreInterval(rec))/20)
	var opts []ui.Option
	if rec.Tier < len(t.Tiers) {
		next := t.Tiers[rec.Tier]
		label := fmt.Sprintf("§a티어 %d로\n§7%d포인트 · Lv.%d", rec.Tier+1, next.Points, next.Level)
		var adds []string
		for _, e := range next.Add {
			adds = append(adds, itemName(item.NewStack(e.Block.(world.Item), 1)))
		}
		fmt.Fprintf(&b, "§7다음 티어: 기본 주기 %.1f초, 추가 블록 %s\n", float64(next.Interval)/20, strings.Join(adds, "·"))
		shown := rec.Tier
		opts = append(opts, s.opt(label, iconCores, func(p *player.Player, _ *world.Tx) {
			// The form may be stale: another player may have bought it.
			switch {
			case rec.Tier != shown:
			case s.state.Level < next.Level:
				p.Messagef("§cLv.%d부터 올릴 수 있습니다.", next.Level)
			case s.spendPoints(p, next.Points):
				rec.Tier++
				s.storeState()
				account.Grant(p, achCoreTier)
				p.Messagef("§a%s가 티어 %d가 되었습니다.", t.Name, rec.Tier)
			}
			s.showCoreUpgrades(p, typ)
		}))
	}
	for _, u := range coreUpgrades {
		lv := rec.level(u.id)
		fmt.Fprintf(&b, "§f%s Lv.%d/%d §7%s\n", u.name, lv, maxUpgradeLevel, u.effect)
		if lv >= maxUpgradeLevel {
			opts = append(opts, s.opt(fmt.Sprintf("§8%s Lv.%d\n§8최대", u.name, lv), iconLocked, func(p *player.Player, _ *world.Tx) {
				s.showCoreUpgrades(p, typ)
			}))
			continue
		}
		cost := upgradeCost(lv)
		opts = append(opts, s.opt(fmt.Sprintf("§b%s Lv.%d → %d\n§7%d포인트", u.name, lv, lv+1, cost), iconCores, func(p *player.Player, _ *world.Tx) {
			// The form may be stale: another player may have bought it.
			if rec.level(u.id) == lv && s.spendPoints(p, cost) {
				if rec.Upgrades == nil {
					rec.Upgrades = map[string]int{}
				}
				rec.Upgrades[u.id]++
				s.storeState()
				p.Messagef("§a%s %s Lv.%d", t.Name, u.name, rec.Upgrades[u.id])
			}
			s.showCoreUpgrades(p, typ)
		}))
	}
	if rec.FilterUnlocked {
		opts = append(opts, s.opt("§f생성 필터\n§7만들 블록 고르기", iconMission, func(p *player.Player, _ *world.Tx) { s.showCoreFilter(p, typ) }))
	} else {
		opts = append(opts, s.opt(fmt.Sprintf("§f생성 필터 해금\n§7%d포인트", filterCost), iconLocked, func(p *player.Player, _ *world.Tx) {
			if !rec.FilterUnlocked && s.spendPoints(p, filterCost) {
				rec.FilterUnlocked = true
				s.storeState()
				p.Message("§a생성 필터를 해금했습니다.")
			}
			s.showCoreUpgrades(p, typ)
		}))
	}
	opts = append(opts, s.opt("§7← 코어 관리", iconBack, func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) }))
	return b.String(), opts, true
}

// showCoreFilter toggles the blocks a core may generate. At least one block
// always stays on.
func (s *session) showCoreFilter(p *player.Player, typ string) {
	if opts, ok := s.coreFilterMenu(typ); ok {
		ui.CardMenu(p, coreTypes[typ].Name+" 필터", "§7누르면 그 블록을 만들지 않거나 다시 만듭니다.", opts...)
	}
}

func (s *session) coreFilterMenu(typ string) ([]ui.Option, bool) {
	rec := s.state.Cores[typ]
	if rec == nil || !rec.FilterUnlocked {
		return nil, false
	}
	var opts []ui.Option
	all := coreTypes[typ].table(max(rec.Tier, 1))
	for _, e := range all {
		key := itemKey(e.Block.(world.Item))
		st := item.NewStack(e.Block.(world.Item), 1)
		on := !slices.Contains(rec.Excluded, key)
		label := "§a켜짐 §f" + itemName(st)
		if !on {
			label = "§8꺼짐 §7" + itemName(st)
		}
		opts = append(opts, s.opt(label, iconOf(st), func(p *player.Player, _ *world.Tx) {
			if on == slices.Contains(rec.Excluded, key) {
				// Stale form: someone else toggled it already.
			} else if on {
				if len(rec.Excluded)+1 >= len(all) {
					p.Message("§c블록을 하나 이상은 남겨야 합니다.")
				} else {
					rec.Excluded = append(rec.Excluded, key)
				}
			} else {
				rec.Excluded = slices.DeleteFunc(rec.Excluded, func(k string) bool { return k == key })
			}
			s.storeState()
			s.showCoreFilter(p, typ)
		}))
	}
	opts = append(opts, s.opt("§7← 강화", iconBack, func(p *player.Player, _ *world.Tx) { s.showCoreUpgrades(p, typ) }))
	return opts, true
}
