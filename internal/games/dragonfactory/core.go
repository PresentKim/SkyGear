package dragonfactory

import (
	"fmt"
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
	"github.com/presentkim/dragonfactory/internal/ui"
)

// The dimension core (docs/design/02-dragonfactory.md §3): the block under
// it (its base) chooses what it makes (corerecipe.go): oak logs or
// cobblestone. It makes its block above itself whenever that space is
// empty, every interval. Any number of cores can be placed. The speed
// upgrade (core points) speeds up all cores.

// CoreType is a kind of core item. Only "t1" is left (13th playtest); the
// other types are old ones kept so old worlds load.
type CoreType struct {
	ID, Name string
}

// coreBaseInterval is the generation interval in ticks. Draft.
const coreBaseInterval = 60

// coreTypes are all core types by ID (old IDs map to the core);
// coreOrder is the registration order. New types go at the end: the index
// is part of the block hash. Every type but "t1" is an old core (one
// resource each, then tiers 2 to 4), kept so old worlds load; they turn
// into the core (legacyCores).
var coreTypes, coreOrder = buildCoreTypes()

// legacyCores maps old core types to the core that replaces them.
var legacyCores = map[string]string{
	"earth": "t1", "life": "t1",
	"wood": "t1", "stone": "t1", "dirt": "t1", "sand": "t1", "gravel": "t1",
	"coal": "t1", "clay": "t1", "pumpkin": "t1", "melon": "t1", "hay": "t1",
	"iron": "t1", "copper": "t1", "kelp": "t1", "mud": "t1", "gold": "t1",
	"t2": "t1", "t3": "t1", "t4": "t1",
}

func buildCoreTypes() (map[string]*CoreType, []string) {
	core := &CoreType{"t1", "차원의 코어"}
	types := map[string]*CoreType{core.ID: core}
	order := []string{"earth", "life", "wood", "stone", "dirt", "sand", "gravel", "coal", "clay", "pumpkin", "melon", "hay", "iron", "copper", "kelp", "mud", "gold", "t1", "t2", "t3", "t4"}
	for old := range legacyCores {
		types[old] = core
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

// Properties: a cube with its own side, top and bottom textures (old types
// look like the core until they turn into it).
func (c Core) Properties() customblock.Properties { return cubeProperties("core_" + c.ctype().ID) }

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
		r := matchRecipe(tx, pos)
		id := ""
		if r != nil {
			id = r.ID
		}
		if id != c.Recipe {
			c.Recipe, c.Progress = id, 0
		}
		if r != nil {
			if c.Progress++; c.Progress >= s.coreInterval() {
				tx.SetBlock(pos.Side(cube.FaceUp), r.Output, nil)
				c.Progress = 0
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
	if matchRecipe(tx, pos) == nil {
		tx.SetBlock(pos.Side(cube.FaceDown), oakLog, nil)
	}
}

// coreItem returns the item of a core type.
func coreItem(typ string) item.Stack {
	t := coreTypes[typ]
	return item.NewStack(Core{Type: t.ID}, 1).
		WithLore("§7아래 코어 기반과 같은 블록을 만듭니다.", "§7코어 기반: 참나무 원목 또는 조약돌")
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

// The speed upgrade (core points): each level shortens the interval of
// every core. Drafts. The level is kept as TierSpeed[1] (older saves had a
// level per tier).
const (
	maxTierSpeed     = 3
	tierSpeedPercent = 12
)

// upgradeCost is the core points the next level of an upgrade costs.
func upgradeCost(level int) int { return level + 1 }

// coreSpeed is the core speed upgrade level.
func (s *session) coreSpeed() int { return s.state.TierSpeed[1] }

// coreInterval is the generation interval of the cores in ticks.
func (s *session) coreInterval() int {
	return coreBaseInterval * (100 - tierSpeedPercent*s.coreSpeed()) / 100
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

// showCores shows the core upgrades: core speed and crop growth.
func (s *session) showCores(p *player.Player, _ *world.Tx) {
	body, opts := s.coreMenu()
	ui.Menu(p, "코어 관리", body, opts...)
}

func (s *session) coreMenu() (string, []ui.Option) {
	var b strings.Builder
	lv := s.coreSpeed()
	fmt.Fprintf(&b, "§f코어 강화 포인트 §b%d§r\n§7레벨이 오르면 포인트를 받습니다. 코어 속도를 올리면 모든 코어가 빨라집니다. 코어는 아래 코어 기반과 같은 블록을 만듭니다:\n%s\n", s.state.CorePoints, coreRecipeTexts())
	fmt.Fprintf(&b, "\n§f코어 §7%.1f초 · 속도 Lv.%d/%d", float64(s.coreInterval())/20, lv, maxTierSpeed)
	var opts []ui.Option
	if lv < maxTierSpeed {
		cost := upgradeCost(lv)
		opts = append(opts, s.opt(fmt.Sprintf("§b코어 속도 Lv.%d → %d §7(%d포인트)", lv, lv+1, cost), iconCores, func(p *player.Player, tx *world.Tx) {
			// The form may be stale: another player may have bought it.
			if s.coreSpeed() == lv && s.spendPoints(p, cost) {
				if s.state.TierSpeed == nil {
					s.state.TierSpeed = map[int]int{}
				}
				s.state.TierSpeed[1]++
				s.storeState()
				ui.Flash(p, "§a코어 속도 Lv.%d", s.coreSpeed())
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

// showCoreInfo describes a core: its base and what it makes.
func (s *session) showCoreInfo(p *player.Player, tx *world.Tx, pos cube.Pos, c Core) {
	t := c.ctype()
	var b strings.Builder
	fmt.Fprintf(&b, "§f%s §7· 속도 Lv.%d\n§7코어 기반: §f%s\n", t.Name, s.coreSpeed(), blockName(tx.Block(pos.Side(cube.FaceDown))))
	if r := matchRecipe(tx, pos); r != nil {
		interval := time.Duration(s.coreInterval()) * time.Second / 20
		fmt.Fprintf(&b, "§a%s §7(%.1f초마다, 위 칸이 비어 있을 때)", r.text(), interval.Seconds())
	} else {
		fmt.Fprintf(&b, "§c반응 없음 §7— 코어 기반은 다음 중 하나여야 합니다:\n%s", coreRecipeTexts())
	}
	b.WriteString("\n\n§7코어는 여러 개 둘 수 있고, 캐면 아이템으로 돌아옵니다. 코어 기반은 닳지 않습니다. 다른 자원은 부수기·체로 얻습니다.")
	ui.Menu(p, t.Name, b.String(),
		s.opt("§b코어 관리", iconCores, func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) }))
}
