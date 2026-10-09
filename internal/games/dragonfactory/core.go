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

// Dimension cores (docs/design/02-dragonfactory.md §3): small cores that
// each make one resource. A core makes its block above itself whenever that
// space is empty, every interval of its tier. Any number of cores of any
// type can be placed. New saves start with a wood core; the others open in
// the shop with the levels. Tier speed upgrades (core points) speed up all
// cores of a tier.

// CoreType is a kind of core: the block it makes and its tier.
type CoreType struct {
	ID, Name string
	Tier     int
	Block    world.Block
	// Texture is the vanilla terrain texture shown in the core's crystal.
	Texture string
}

// coreTiers are the base generation intervals (ticks) by tier. Drafts.
var coreTiers = map[int]int{1: 60, 2: 100, 3: 160, 4: 240}

// coreTypes are all core types by ID (old IDs map to their replacements);
// coreOrder is the display and registration order. New types go at the
// end: the index is part of the block hash. "earth" and "life" are the
// cores of the first core design, kept so old worlds load; they turn into
// wood and melon cores (legacyCores).
var coreTypes, coreOrder = buildCoreTypes()

// legacyCores maps old core types to the types that replace them.
var legacyCores = map[string]string{"earth": "wood", "life": "melon"}

func buildCoreTypes() (map[string]*CoreType, []string) {
	types := map[string]*CoreType{}
	order := []string{"earth", "life"}
	for _, t := range []*CoreType{
		{"wood", "나무 코어", 1, block.Log{Wood: block.OakWood()}, "log_side"},
		{"stone", "돌 코어", 1, block.Stone{}, "stone"},
		{"dirt", "흙 코어", 1, block.Dirt{}, "dirt"},
		{"sand", "모래 코어", 1, block.Sand{}, "sand"},
		{"gravel", "자갈 코어", 1, block.Gravel{}, "gravel"},
		{"coal", "석탄 코어", 2, block.CoalOre{}, "coal_ore"},
		{"clay", "점토 코어", 2, block.Clay{}, "clay"},
		{"pumpkin", "호박 코어", 2, block.Pumpkin{}, "pumpkin_side"},
		{"melon", "수박 코어", 2, block.Melon{}, "melon_side"},
		{"hay", "건초 코어", 2, block.HayBale{Axis: cube.Y}, "hayblock_side"},
		{"iron", "철 코어", 3, block.IronOre{}, "iron_ore"},
		{"copper", "구리 코어", 3, block.CopperOre{}, "copper_ore"},
		{"kelp", "말린 켈프 코어", 3, block.DriedKelp{}, "dried_kelp_block_side_a"},
		{"mud", "진흙 코어", 3, block.Mud{}, "mud"},
		{"gold", "금 코어", 4, block.GoldOre{}, "gold_ore"},
	} {
		types[t.ID] = t
		order = append(order, t.ID)
	}
	types["earth"], types["life"] = types["wood"], types["melon"]
	return types, order
}

func init() {
	for _, id := range coreOrder {
		world.RegisterBlock(Core{Type: id})
		world.RegisterItem(Core{Type: id})
	}
}

// startCore is the core type and position of a new save.
var (
	startCore    = "wood"
	startCorePos = cube.Pos{0, IslandY, 0}
)

// Core is a core block. Progress counts ticks towards the next block.
type Core struct {
	Type     string
	Progress int
}

var coreHash = block.NextHash()

func (c Core) ctype() *CoreType { return coreTypes[c.Type] }

func (c Core) EncodeItem() (string, int16)           { return "dragonfactory:core_" + c.Type, 0 }
func (c Core) EncodeBlock() (string, map[string]any) { return "dragonfactory:core_" + c.Type, nil }
func (c Core) Hash() (uint64, uint64) {
	return coreHash, uint64(slices.Index(coreOrder, c.Type))
}
func (Core) Model() world.BlockModel     { return model.Solid{} }
func (c Core) Name() string              { return fmt.Sprintf("[%d티어] %s", c.ctype().Tier, c.ctype().Name) }
func (Core) Category() category.Category { return category.Items() }
func (Core) Geometry() []byte            { return coreModel }
func (c Core) Texture() image.Image      { return itemTexture("core_" + c.ctype().ID) }
func (c Core) Textures() map[string]image.Image {
	return map[string]image.Image{"dragonfactory_core_frame": blockTexture("core_frame")}
}
func (c Core) Properties() customblock.Properties {
	return customblock.Properties{Geometry: "geometry.dragonfactory_core", Textures: map[string]customblock.Material{
		"*": alphaMaterial("dragonfactory_core_frame"),
		// The crystal shows the vanilla texture of the block the core makes.
		"crystal": alphaMaterial(c.ctype().Texture),
	}}
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
	return map[string]any{"id": "DragonFactoryCore", "Progress": int32(c.Progress)}
}

func (c Core) DecodeNBT(data map[string]any) any {
	p, _ := data["Progress"].(int32)
	c.Progress = int(p)
	return c
}

// Tick makes the core's block above it once per interval while that space
// is empty, so a block appears one full interval after the previous one
// was taken.
func (c Core) Tick(_ int64, pos cube.Pos, tx *world.Tx) {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	if now, ok := legacyCores[c.Type]; ok {
		c.Type = now
		tx.SetBlock(pos, c, nil)
		return
	}
	before := c.Progress
	if _, air := tx.Block(pos.Side(cube.FaceUp)).(block.Air); !air {
		c.Progress = 0
	} else if c.Progress++; c.Progress >= s.coreInterval(c.ctype()) {
		tx.SetBlock(pos.Side(cube.FaceUp), c.ctype().Block, nil)
		c.Progress = 0
	}
	if c.Progress != before {
		tx.SetBlockEntity(pos, c)
	}
}

// coreItem returns the item of a core type.
func coreItem(typ string) item.Stack {
	t := coreTypes[typ]
	return item.NewStack(Core{Type: t.ID}, 1).
		WithLore(fmt.Sprintf("§7위 칸이 비면 %s을(를) 만듭니다.", itemName(item.NewStack(t.Block.(world.Item), 1))))
}

// convertLegacyCores turns old core items in an inventory into the cores
// that replace them.
func convertLegacyCores(inv *inventory.Inventory) {
	for slot, st := range inv.Slots() {
		if now := modernCore(st); now.Item() != st.Item() {
			_ = inv.SetItem(slot, now)
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

// coreInterval is the generation interval of a core type in ticks.
func (s *session) coreInterval(t *CoreType) int {
	return coreTiers[t.Tier] * (100 - tierSpeedPercent*s.state.TierSpeed[t.Tier]) / 100
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

// showCores shows the core upgrades: tier speeds and crop growth.
func (s *session) showCores(p *player.Player, _ *world.Tx) {
	body, opts := s.coreMenu()
	ui.Menu(p, "코어 관리", body, opts...)
}

func (s *session) coreMenu() (string, []ui.Option) {
	var b strings.Builder
	fmt.Fprintf(&b, "§f코어 강화 포인트 §b%d§r\n§7레벨이 오르면 포인트를 받습니다. 티어 속도를 올리면 그 티어의 모든 코어가 빨라집니다. 코어는 상점에서 삽니다.\n", s.state.CorePoints)
	var opts []ui.Option
	for tier := 1; tier <= len(coreTiers); tier++ {
		lv := s.state.TierSpeed[tier]
		var names []string
		for _, id := range coreOrder {
			if t := coreTypes[id]; t.Tier == tier && t.ID == id {
				names = append(names, strings.TrimSuffix(t.Name, " 코어"))
			}
		}
		fmt.Fprintf(&b, "\n§f%d티어 §7%.1f초 · 속도 Lv.%d/%d §8(%s)", tier, float64(s.coreInterval(&CoreType{Tier: tier}))/20, lv, maxTierSpeed, strings.Join(names, "·"))
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
				p.Messagef("§a%d티어 코어 속도 Lv.%d", tier, s.state.TierSpeed[tier])
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
		s.showCoreInfo(p, c)
	}
	return true
}

// showCoreInfo describes a core: its tier, block and interval.
func (s *session) showCoreInfo(p *player.Player, c Core) {
	t := c.ctype()
	interval := time.Duration(s.coreInterval(t)) * time.Second / 20
	body := fmt.Sprintf("§f%s §7· %d티어\n§7위 칸이 비어 있으면 §f%.1f초§7마다 §f%s§7을(를) 만듭니다.\n§7%d티어 코어 속도 Lv.%d\n\n§7코어는 여러 개 둘 수 있고, 캐면 아이템으로 돌아옵니다. 더 빠르게 하려면 코어 관리에서 티어 속도를 올리세요.",
		t.Name, t.Tier, interval.Seconds(), itemName(item.NewStack(t.Block.(world.Item), 1)), t.Tier, s.state.TierSpeed[t.Tier])
	ui.Menu(p, t.Name, body, s.opt("§b코어 관리", iconCores, func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) }))
}

// placeCoreAchievement grants the tier achievement for a placed core.
func (s *session) placeCoreAchievement(p *player.Player, c Core) {
	if c.ctype().Tier >= 2 {
		account.Grant(p, achCoreTier)
	}
}
