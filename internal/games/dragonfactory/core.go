package dragonfactory

import (
	"fmt"
	"image"
	"math/rand/v2"

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

// CoreType is a theme of dimension core (docs/design/02-dragonfactory.md §3).
type CoreType struct {
	ID   string
	Name string
	// Interval is the generation interval in ticks at tier 1.
	Interval int
	// Table lists weighted blocks generated at tier 1.
	Table   []WeightedBlock
	texture image.Image
}

// WeightedBlock is a generation table entry.
type WeightedBlock struct {
	Block  world.Block
	Weight int
}

func (t *CoreType) pick(r *rand.Rand) world.Block {
	total := 0
	for _, e := range t.Table {
		total += e.Weight
	}
	n := r.IntN(total)
	for _, e := range t.Table {
		if n < e.Weight {
			return e.Block
		}
		n -= e.Weight
	}
	return t.Table[0].Block
}

// coreTypes are all core themes by ID. Only the earth core exists in M2.
var coreTypes = map[string]*CoreType{
	"earth": {
		ID: "earth", Name: "대지 코어", Interval: 60,
		Table: []WeightedBlock{
			{block.Cobblestone{}, 30}, {block.Dirt{}, 20}, {block.Log{Wood: block.OakWood()}, 18},
			{block.Stone{}, 10}, {block.Sand{}, 10}, {block.Gravel{}, 8}, {block.CoalOre{}, 4},
		},
		texture: frameTexture(rgb(60, 40, 25), rgb(110, 80, 50), rgb(90, 200, 90), 6),
	},
}

// coreOrder is the display order of core types.
var coreOrder = []string{"earth"}

// CoreRecord tracks the single core of a type in a save (§3.3).
type CoreRecord struct {
	Type string `json:"type"`
	Tier int    `json:"tier"`
	// Generation increases on every reissue; older copies are invalid.
	Generation int32  `json:"generation"`
	Placed     bool   `json:"placed"`
	Pos        [3]int `json:"pos"`
}

func (r *CoreRecord) pos() cube.Pos { return cube.Pos{r.Pos[0], r.Pos[1], r.Pos[2]} }

// startCorePos is where the earth core of a new save stands.
var startCorePos = cube.Pos{0, IslandY, 0}

const coreGenerationKey = "core_generation"

// Core is a dimension core block. It generates a block from its type's
// table above itself whenever that space is empty.
type Core struct {
	Type       string
	Generation int32
	// Progress counts ticks towards the next generation.
	Progress int
}

var coreHash = block.NextHash()

func (c Core) ctype() *CoreType { return coreTypes[c.Type] }

func (c Core) EncodeItem() (string, int16)           { return "dragonfactory:core_" + c.Type, 0 }
func (c Core) EncodeBlock() (string, map[string]any) { return "dragonfactory:core_" + c.Type, nil }
func (c Core) Hash() (uint64, uint64) {
	for i, id := range coreOrder {
		if id == c.Type {
			return coreHash, uint64(i)
		}
	}
	return coreHash, 0
}
func (Core) Model() world.BlockModel     { return model.Solid{} }
func (c Core) Name() string              { return c.ctype().Name }
func (Core) Category() category.Category { return category.Items() }
func (Core) Geometry() []byte            { return nil }
func (c Core) Texture() image.Image      { return c.ctype().texture }
func (c Core) Textures() map[string]image.Image {
	return map[string]image.Image{"dragonfactory_core_" + c.Type: c.ctype().texture}
}
func (c Core) Properties() customblock.Properties {
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*": customblock.NewMaterial("dragonfactory_core_"+c.Type, customblock.OpaqueRenderMethod()),
	}}
}

// LightEmissionLevel makes cores glow faintly.
func (Core) LightEmissionLevel() uint8 { return 7 }

// BreakInfo: mined with a pickaxe, dropping the core item of the same generation.
func (c Core) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 3, BlastResistance: 3600,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(t item.Tool) bool { return t.ToolType() == item.TypePickaxe },
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			return []item.Stack{coreItem(c.Type, c.Generation)}
		},
	}
}

func (c Core) EncodeNBT() map[string]any {
	return map[string]any{"id": "DragonFactoryCore", "Generation": c.Generation, "Progress": int32(c.Progress)}
}

func (c Core) DecodeNBT(data map[string]any) any {
	c.Generation, _ = data["Generation"].(int32)
	p, _ := data["Progress"].(int32)
	c.Progress = int(p)
	return c
}

// Tick generates blocks and removes stale copies (§3.1, §3.3).
func (c Core) Tick(_ int64, pos cube.Pos, tx *world.Tx) {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	rec := s.state.Cores[c.Type]
	if rec == nil || !rec.Placed || rec.pos() != pos || rec.Generation != c.Generation {
		// An outdated copy (reissued) or a core the record does not know.
		tx.SetBlock(pos, nil, nil)
		return
	}
	// The interval counts only while the space above is free, so a block
	// appears one full interval after the previous one was taken.
	before := c.Progress
	if _, air := tx.Block(pos.Side(cube.FaceUp)).(block.Air); !air {
		c.Progress = 0
	} else if c.Progress++; c.Progress >= s.coreInterval(rec) {
		tx.SetBlock(pos.Side(cube.FaceUp), c.ctype().pick(s.rand), nil)
		c.Progress = 0
	}
	if c.Progress != before {
		tx.SetBlockEntity(pos, c)
	}
}

// coreInterval is the generation interval of a core in ticks.
func (s *session) coreInterval(rec *CoreRecord) int { return coreTypes[rec.Type].Interval }

// coreItem returns the item of a core type for a generation.
func coreItem(typ string, gen int32) item.Stack {
	return item.NewStack(Core{Type: typ}, 1).
		WithValue(coreGenerationKey, gen).
		WithLore(fmt.Sprintf("§7세대 %d · 종류별로 섬에 하나만 설치할 수 있습니다.", gen))
}

func coreItemGeneration(s item.Stack) (string, int32, bool) {
	c, ok := s.Item().(Core)
	if !ok {
		return "", 0, false
	}
	g, _ := s.Value(coreGenerationKey)
	gen, _ := g.(int32)
	return c.Type, gen, true
}

// ensureCores creates the starting core record for a new state.
func (s *session) ensureCores() {
	if s.state.Cores == nil {
		s.state.Cores = map[string]*CoreRecord{}
	}
	if _, ok := s.state.Cores["earth"]; !ok {
		p := startCorePos
		s.state.Cores["earth"] = &CoreRecord{Type: "earth", Tier: 1, Generation: 1, Placed: true, Pos: [3]int{p.X(), p.Y(), p.Z()}}
	}
}

// placeCore validates placing a core item at pos. It returns false if the
// placement must be cancelled.
func (s *session) placeCore(p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) bool {
	typ, gen, _ := coreItemGeneration(held)
	rec := s.state.Cores[typ]
	switch {
	case rec == nil:
		p.Message("§c이 세이브에서 얻지 않은 코어입니다.")
		return false
	case rec.Generation != gen:
		p.Message("§c재발급으로 무효가 된 코어입니다. 인벤토리에서 제거합니다.")
		removeStaleCores(p.Inventory(), s)
		return false
	case rec.Placed:
		p.Messagef("§c%s는 이미 섬에 설치되어 있습니다.", coreTypes[typ].Name)
		return false
	}
	rec.Placed, rec.Pos = true, [3]int{pos.X(), pos.Y(), pos.Z()}
	tx.Defer(func(tx *world.Tx) { tx.SetBlock(pos, Core{Type: typ, Generation: gen}, nil) })
	s.storeState()
	return true
}

// breakCore records that a core was mined.
func (s *session) breakCore(c Core, pos cube.Pos) {
	if rec := s.state.Cores[c.Type]; rec != nil && rec.Placed && rec.pos() == pos {
		rec.Placed = false
		s.storeState()
	}
}

// removeStaleCores deletes core items of outdated generations.
func removeStaleCores(inv *inventory.Inventory, s *session) {
	for slot, st := range inv.Slots() {
		typ, gen, ok := coreItemGeneration(st)
		if !ok {
			continue
		}
		if rec := s.state.Cores[typ]; rec == nil || rec.Generation != gen {
			_ = inv.SetItem(slot, item.Stack{})
		}
	}
}

// verifyCores marks records whose placed block is gone as not placed.
func (s *session) verifyCores(tx *world.Tx) {
	for _, rec := range s.state.Cores {
		if !rec.Placed {
			continue
		}
		if c, ok := tx.Block(rec.pos()).(Core); !ok || c.Type != rec.Type || c.Generation != rec.Generation {
			rec.Placed = false
		}
	}
}

func (s *session) showCores(p *player.Player, tx *world.Tx) {
	s.verifyCores(tx)
	var opts []ui.Option
	body := "코어는 종류별로 섬에 하나만 설치할 수 있습니다.\n설치되어 있지 않은 코어는 무상으로 재발급할 수 있고, 재발급하면 이전 코어 아이템은 무효가 됩니다.\n"
	for _, id := range coreOrder {
		rec := s.state.Cores[id]
		if rec == nil {
			continue
		}
		t := coreTypes[id]
		if rec.Placed {
			body += fmt.Sprintf("\n§f%s §7티어 %d · 설치됨 (%d, %d, %d)", t.Name, rec.Tier, rec.Pos[0], rec.Pos[1], rec.Pos[2])
			continue
		}
		body += fmt.Sprintf("\n§f%s §7티어 %d · §e설치 안 됨", t.Name, rec.Tier)
		opts = append(opts, ui.Option{Label: t.Name + " 재발급", Run: func(p *player.Player, tx *world.Tx) { s.reissueCore(p, tx, id) }})
	}
	ui.Menu(p, "코어 관리", body, opts...)
}

func (s *session) reissueCore(p *player.Player, tx *world.Tx, typ string) {
	s.verifyCores(tx)
	rec := s.state.Cores[typ]
	if rec == nil || rec.Placed {
		p.Message("§c설치되어 있는 코어는 재발급할 수 없습니다.")
		return
	}
	rec.Generation++
	s.storeState()
	removeStaleCores(p.Inventory(), s)
	it := coreItem(typ, rec.Generation)
	if _, err := p.Inventory().AddItem(it); err != nil {
		s.dropAt(tx, p.Position(), it)
	}
	p.Messagef("§a%s를 재발급했습니다. (세대 %d)", coreTypes[typ].Name, rec.Generation)
}

func init() {
	for _, id := range coreOrder {
		world.RegisterBlock(Core{Type: id})
		world.RegisterItem(Core{Type: id})
	}
}
