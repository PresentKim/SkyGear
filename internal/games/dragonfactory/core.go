package dragonfactory

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Dimension cores (docs/design/02-dragonfactory.md §3): a core is an engine
// of a tier; the block under it (its base) and the blocks around that base
// choose what it makes (corerecipe.go). A core makes its block above itself
// whenever that space is empty, every interval of its tier. Any number of
// cores can be placed. Tier speed upgrades (core points) speed up all cores
// of a tier.

// CoreType is a kind of core item: a tier.
type CoreType struct {
	ID, Name string
	Tier     int
}

// coreTiers are the base generation intervals (ticks) by tier. Drafts.
var coreTiers = map[int]int{1: 60, 2: 100, 3: 160, 4: 240}

// coreTypes are all core types by ID (old IDs map to their replacements);
// coreOrder is the registration order. New types go at the end: the index
// is part of the block hash. The types before "t1" are the cores of earlier
// designs (one resource each), kept so old worlds load; they turn into the
// tier core that replaces them (legacyCores).
var coreTypes, coreOrder = buildCoreTypes()

// legacyCores maps old core types to the tier cores that replace them.
var legacyCores = map[string]string{
	"earth": "t1", "life": "t2",
	"wood": "t1", "stone": "t1", "dirt": "t1", "sand": "t1", "gravel": "t1",
	"coal": "t2", "clay": "t2", "pumpkin": "t2", "melon": "t2", "hay": "t2",
	"iron": "t3", "copper": "t3", "kelp": "t3", "mud": "t3", "gold": "t4",
}

func buildCoreTypes() (map[string]*CoreType, []string) {
	types := map[string]*CoreType{}
	order := []string{"earth", "life", "wood", "stone", "dirt", "sand", "gravel", "coal", "clay", "pumpkin", "melon", "hay", "iron", "copper", "kelp", "mud", "gold"}
	for _, t := range []*CoreType{
		{"t1", "[1티어] 코어", 1},
		{"t2", "[2티어] 코어", 2},
		{"t3", "[3티어] 코어", 3},
		{"t4", "[4티어] 코어", 4},
	} {
		types[t.ID] = t
		order = append(order, t.ID)
	}
	for old, now := range legacyCores {
		types[old] = types[now]
	}
	return types, order
}

func init() {
	for _, id := range coreOrder {
		world.RegisterBlock(Core{Type: id})
		world.RegisterItem(Core{Type: id})
	}
}

// startCore is the core type and position of a new save; its base is an
// oak log.
var (
	startCore    = "t1"
	startCorePos = cube.Pos{0, IslandY, 0}
)

// Core is a core block. Progress counts ticks towards the next block.
type Core struct {
	Type     string
	Progress int
	// Recipe is the recipe being made; a new one restarts the progress.
	Recipe string
}

var coreHash = block.NextHash()

func (c Core) ctype() *CoreType { return coreTypes[c.Type] }

func (c Core) EncodeItem() (string, int16)           { return "dragonfactory:core_" + c.Type, 0 }
func (c Core) EncodeBlock() (string, map[string]any) { return "dragonfactory:core_" + c.Type, nil }
func (c Core) Hash() (uint64, uint64) {
	return coreHash, uint64(slices.Index(coreOrder, c.Type))
}
func (Core) Model() world.BlockModel     { return model.Solid{} }
func (c Core) Name() string              { return c.ctype().Name }
func (Core) Category() category.Category { return category.Items() }
func (Core) Geometry() []byte            { return coreModel }
func (c Core) Texture() image.Image      { return itemTexture("core_" + c.ctype().ID) }

// Textures: each tier has its own face textures (textures/blocks/
// core_t<n>_<face>.png).
func (c Core) Textures() map[string]image.Image { return cubeTextures("core_" + c.ctype().ID) }
func (c Core) Properties() customblock.Properties {
	return cubeProperties("geometry.dragonfactory_core", "core_"+c.ctype().ID)
}

// LightEmissionLevel makes cores glow faintly.
func (Core) LightEmissionLevel() uint8 { return 10 }

// BreakInfo: a core is mined back as its item.
func (c Core) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 1.5, BlastResistance: 3600,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   customBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			return []item.Stack{coreItem(c.ctype().ID)}
		},
	}
}

func (c Core) EncodeNBT() map[string]any {
	return map[string]any{"id": "DragonFactoryCore", "Progress": int32(c.Progress), "Recipe": c.Recipe}
}

func (c Core) DecodeNBT(data map[string]any) any {
	p, _ := data["Progress"].(int32)
	c.Progress = int(p)
	c.Recipe, _ = data["Recipe"].(string)
	return c
}

// Tick makes the recipe's block above the core once per interval while that
// space is empty, so a block appears one full interval after the previous
// one was taken. Without a recipe for its base the core waits.
func (c Core) Tick(_ int64, pos cube.Pos, tx *world.Tx) {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	if now, ok := legacyCores[c.Type]; ok {
		convertLegacyCoreBlock(tx, pos, c.Type)
		c.Type = now
		tx.SetBlock(pos, c, nil)
		return
	}
	before := c
	if _, air := tx.Block(pos.Side(cube.FaceUp)).(block.Air); !air {
		c.Progress = 0
	} else {
		r, _ := matchRecipe(tx, pos, c.ctype().Tier)
		id := ""
		if r != nil {
			id = r.ID
		}
		if id != c.Recipe {
			c.Recipe, c.Progress = id, 0
		}
		if r != nil {
			if c.Progress++; c.Progress >= s.coreInterval(c.ctype().Tier) {
				tx.SetBlock(pos.Side(cube.FaceUp), r.Output, nil)
				c.Progress = 0
				s.discover(tx, r)
			}
		}
	}
	if c != before {
		tx.SetBlockEntity(pos, c)
	}
}

// convertLegacyCoreBlock keeps an old save's start core working: the old
// wood core sat on the island floor, which is no base, so it gets the oak
// log base the new start core has.
func convertLegacyCoreBlock(tx *world.Tx, pos cube.Pos, old string) {
	if pos != startCorePos || (old != "wood" && old != "earth") {
		return
	}
	if r, _ := matchRecipe(tx, pos, 4); r == nil {
		tx.SetBlock(pos.Side(cube.FaceDown), oakLog, nil)
	}
}

// coreItem returns the item of a core type.
func coreItem(typ string) item.Stack {
	t := coreTypes[typ]
	return item.NewStack(Core{Type: t.ID}, 1).
		WithLore("§7아래 코어 기반 블록에 따라 만드는 것이 달라집니다.", "§7레시피: 메뉴 → 코어 관리 → 코어 도감")
}

// convertLegacyCores turns old core items in an inventory into the cores
// that replace them.
func convertLegacyCores(inv *inventory.Inventory) {
	for slot, st := range inv.Slots() {
		// Only cores: comparing items of other types may panic (a chest
		// item holds an uncomparable inventory).
		if c, ok := st.Item().(Core); ok {
			if _, legacy := legacyCores[c.Type]; legacy {
				_ = inv.SetItem(slot, modernCore(st))
			}
		}
	}
}

// modernCore returns the core that replaces an old core stack, or st.
func modernCore(st item.Stack) item.Stack {
	if c, ok := st.Item().(Core); ok {
		if now, ok := legacyCores[c.Type]; ok {
			return coreItem(now).Grow(st.Count() - 1)
		}
	}
	return st
}

// Tier speed upgrades (core points): each level shortens the interval of
// every core of the tier. Drafts.
const (
	maxTierSpeed     = 3
	tierSpeedPercent = 12
)

// upgradeCost is the core points the next level of an upgrade costs.
func upgradeCost(level int) int { return level + 1 }

// coreInterval is the generation interval of a core tier in ticks.
func (s *session) coreInterval(tier int) int {
	return coreTiers[tier] * (100 - tierSpeedPercent*s.state.TierSpeed[tier]) / 100
}

// spendPoints takes n core points, telling p if there are not enough.
func (s *session) spendPoints(p *player.Player, n int) bool {
	if s.state.CorePoints < n {
		ui.Notify(p, "§c코어 강화 포인트가 부족합니다. (필요 %d, 보유 %d)", n, s.state.CorePoints)
		return false
	}
	s.state.CorePoints -= n
	return true
}

// showCores shows the core upgrades: tier speeds and crop growth.
func (s *session) showCores(p *player.Player, _ *world.Tx) {
	body, opts := s.coreMenu()
	ui.Menu(p, "코어 관리", body, opts...)
}

func (s *session) coreMenu() (string, []ui.Option) {
	var b strings.Builder
	fmt.Fprintf(&b, "§f코어 강화 포인트 §b%d§r\n§7레벨이 오르면 포인트를 받습니다. 티어 속도를 올리면 그 티어의 모든 코어가 빨라집니다. 코어는 상점에서 사고, 무엇을 만들지는 아래 코어 기반 블록이 정합니다(코어 도감).\n", s.state.CorePoints)
	opts := []ui.Option{s.opt("§d코어 도감", iconMission, func(p *player.Player, _ *world.Tx) { s.showCoreBook(p) })}
	for tier := 1; tier <= len(coreTiers); tier++ {
		lv := s.state.TierSpeed[tier]
		fmt.Fprintf(&b, "\n§f%d티어 §7%.1f초 · 속도 Lv.%d/%d", tier, float64(s.coreInterval(tier))/20, lv, maxTierSpeed)
		if lv >= maxTierSpeed {
			continue
		}
		cost := upgradeCost(lv)
		opts = append(opts, s.opt(fmt.Sprintf("§b%d티어 코어 속도 Lv.%d → %d §7(%d포인트)", tier, lv, lv+1, cost), iconCores, func(p *player.Player, tx *world.Tx) {
			// The form may be stale: another player may have bought it.
			if s.state.TierSpeed[tier] == lv && s.spendPoints(p, cost) {
				if s.state.TierSpeed == nil {
					s.state.TierSpeed = map[int]int{}
				}
				s.state.TierSpeed[tier]++
				s.storeState()
				ui.Flash(p, "§a%d티어 코어 속도 Lv.%d", tier, s.state.TierSpeed[tier])
			}
			s.showCores(p, tx)
		}))
	}
	if o, ok := s.cropUpgradeOption(); ok {
		opts = append(opts, o)
	}
	return b.String(), append(opts, s.backOption())
}

// Activate shows what the core makes and how fast.
func (c Core) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 {
		return false
	}
	if !s.debounced(p) {
		s.showCoreInfo(p, tx, pos, c)
	}
	return true
}

// showCoreInfo describes a core: its tier, base and what it makes.
func (s *session) showCoreInfo(p *player.Player, tx *world.Tx, pos cube.Pos, c Core) {
	t := c.ctype()
	var b strings.Builder
	fmt.Fprintf(&b, "§f%s §7· %d티어 속도 Lv.%d\n§7코어 기반: §f%s\n", t.Name, t.Tier, s.state.TierSpeed[t.Tier], blockName(tx.Block(pos.Side(cube.FaceDown))))
	r, need := matchRecipe(tx, pos, t.Tier)
	switch {
	case r != nil && s.known(r):
		interval := time.Duration(s.coreInterval(t.Tier)) * time.Second / 20
		fmt.Fprintf(&b, "§a%s §7(%.1f초마다, 위 칸이 비어 있을 때)", r.text(), interval.Seconds())
	case r != nil:
		b.WriteString("§d무언가에 반응하고 있습니다... §7(위 칸을 비워 두세요)")
	case need > 0:
		fmt.Fprintf(&b, "§c%d티어 이상 코어가 필요한 코어 기반입니다.", need)
	default:
		b.WriteString("§c반응 없음 §7— 이 코어 기반(과 둘레)에 맞는 레시피가 없습니다. 코어 도감을 확인하세요.")
	}
	b.WriteString("\n\n§7코어는 여러 개 둘 수 있고, 캐면 아이템으로 돌아옵니다. 코어 기반은 닳지 않습니다.")
	ui.Menu(p, t.Name, b.String(),
		s.opt("§d코어 도감", iconMission, func(p *player.Player, _ *world.Tx) { s.showCoreBook(p) }),
		s.opt("§b코어 관리", iconCores, func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) }))
}

// placeCoreAchievement grants the tier achievement for a placed core.
func (s *session) placeCoreAchievement(p *player.Player, c Core) {
	if c.ctype().Tier >= 2 {
		account.Grant(p, achCoreTier)
	}
}
