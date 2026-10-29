package skygear

import (
	"fmt"
	"time"

	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/common/waila"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Barrels: an oak barrel takes 32 plant items like the composter and turns
// them into water after barrelWaterDelay, which a bucket takes. Contents are
// saved in State.Barrels.

const (
	// barrelFillStages are the stages of filling; the next stage is water.
	barrelFillStages = 6
	// barrelWater is the stage of a barrel full of water (lava), barrelWitch
	// of one full of witch water (witchwater.go).
	barrelWater = 7
	barrelWitch = 8
	// barrelWaterDelay is how long (instance time) a full barrel needs to
	// make water. Draft.
	barrelWaterDelay = time.Minute
	// barrelClayBuffer is how many clay blocks a barrel of water holds for
	// a hopper or conduit to take.
	barrelClayBuffer = 16
)

// barrelStageState is the one block state of a barrel: its stage 0 to 8.
const barrelStageState = "skygear:barrel_stage"

// barrelWoods are the materials of the barrel family, in registration order:
// the oak barrel makes water from plants, the stone barrel lava from
// cobblestone (stoneWood).
var barrelWoods = []string{"oak", stoneWood}

// The stone barrel melts blocks into lava like Ex Deorum's crucible: each
// block adds stoneMeltMB of solids, and every stoneMeltStep the heat source
// under it melts its heatValue in mB of them into lava.
const (
	stoneWood      = "stone"
	stoneMeltMB    = 250
	stoneSolidsMax = 1000
	stoneLavaMax   = 4000
	// stoneMeltStep is Ex Deorum's crucible step: 10 ticks.
	stoneMeltStep = 500 * time.Millisecond
	// legacyStoneCount is the cobblestone that made a bucket of lava in the
	// stone barrel of older saves (stoneRecord).
	legacyStoneCount = 8
)

// stoneMelts are the blocks a stone barrel melts.
var stoneMelts = []world.Item{block.Cobblestone{}, block.Stone{}, block.Gravel{}, block.Netherrack{}}

// heatValue is the heat of the block under a stone barrel: the mB of solids
// it melts every stoneMeltStep (0: no heat source). The values are Ex
// Deorum's crucible heat sources.
func heatValue(b world.Block) int {
	switch b := b.(type) {
	case block.Torch:
		if b.Type == block.SoulFire() {
			return 2
		}
		return 1
	case block.Lantern:
		if b.Type == block.SoulFire() {
			return 2
		}
		return 1
	case block.Campfire:
		if !b.Extinguished {
			return 2
		}
	case block.Lava:
		return 3
	case block.Fire:
		return 5
	}
	return 0
}

// barrelCapacity is how many plant items an oak barrel takes.
func barrelCapacity(string) int { return matterCapacity }

// barrelAccepts reports whether a barrel of wood takes st: plant matter, or
// a block to melt in a stone barrel.
func barrelAccepts(wood string, st item.Stack) bool {
	if wood == stoneWood {
		if st.Empty() || !plain(st) {
			return false
		}
		for _, m := range stoneMelts {
			if naming.ItemKey(st.Item()) == naming.ItemKey(m) {
				return true
			}
		}
		return false
	}
	return plantMatter(st)
}

var barrelHashes = func() map[string]uint64 {
	m := map[string]uint64{}
	for _, w := range barrelWoods {
		m[w] = block.NextHash()
	}
	return m
}()

// Barrel is a wooden barrel block; Stage is what it shows (0 empty, 1 to 6
// filling, 7 water).
type Barrel struct {
	Wood  string
	Stage int
}

// BarrelRecord is what a barrel holds.
type BarrelRecord struct {
	// Count is the plant items in the barrel.
	Count int `json:"count,omitempty"`
	// Water is set once the contents turned into water.
	Water bool `json:"water,omitempty"`
	// Clay is the clay blocks dust made in the water that nobody has taken
	// yet (dust put in by a hopper or conduit).
	Clay int `json:"clay,omitempty"`
	// ReadyAt is the instance time the contents turn into water (a full
	// barrel only).
	ReadyAt time.Duration `json:"ready_at,omitempty"`
	// Solids are the mB of blocks a stone barrel has yet to melt and LavaMB
	// the lava it holds; MeltAt is the instance time the melting was counted
	// to and Made the lava made since the last bucket's worth was counted
	// (barrel_lava).
	Solids int           `json:"solids,omitempty"`
	LavaMB int           `json:"lava_mb,omitempty"`
	MeltAt time.Duration `json:"melt_at,omitempty"`
	Made   int           `json:"made,omitempty"`
	// Lava, Heat and HeatAt are a stone barrel of older saves (a bucket of
	// lava; the heat of its cobblestone); stoneRecord turns them into the
	// fields above.
	Lava   bool          `json:"lava,omitempty"`
	Heat   time.Duration `json:"heat,omitempty"`
	HeatAt time.Duration `json:"heat_at,omitempty"`
	// Witch is set once the water of an oak barrel on mycelium turned into
	// witch water; WitchAt is the instance time it turns (0: not on mycelium).
	Witch   bool          `json:"witch,omitempty"`
	WitchAt time.Duration `json:"witch_at,omitempty"`
}

// barrelStageOf is the stage a record of a barrel of wood shows: a stone
// barrel shows its lava, or its solids as the fill stages.
func barrelStageOf(wood string, r *BarrelRecord) int {
	switch {
	case r == nil:
		return 0
	case wood == stoneWood && r.LavaMB > 0:
		return barrelWater
	case wood == stoneWood:
		return (r.Solids*barrelFillStages + stoneSolidsMax - 1) / stoneSolidsMax
	case r.Witch:
		return barrelWitch
	case r.Water:
		return barrelWater
	}
	return r.Count * barrelFillStages / barrelCapacity(wood)
}

// stoneRecord turns the fields of a stone barrel of an older save into the
// solids and lava it holds.
func stoneRecord(r *BarrelRecord) {
	if r == nil {
		return
	}
	if r.Lava {
		// Its cobblestone had turned into the lava.
		r.LavaMB, r.Lava, r.Count = min(r.LavaMB+bucketMB, stoneLavaMax), false, 0
	}
	if r.Count > 0 {
		r.Solids, r.Count = min(r.Solids+r.Count*bucketMB/legacyStoneCount, stoneSolidsMax), 0
	}
	r.Heat, r.HeatAt = 0, 0
}

func (b Barrel) EncodeItem() (string, int16) { return "skygear:barrel_" + b.Wood, 0 }
func (b Barrel) EncodeBlock() (string, map[string]any) {
	return "skygear:barrel_" + b.Wood, map[string]any{barrelStageState: fmt.Sprint(b.Stage)}
}
func (b Barrel) Hash() (uint64, uint64) { return barrelHashes[b.Wood], uint64(b.Stage) }
func (Barrel) Model() world.BlockModel  { return tubModel{} }
func (b Barrel) Name() string           { return woodNames[b.Wood] + i18n.T(i18n.Default, "name.Barrel.suffix") }
func (Barrel) Category() category.Category {
	return category.Construction()
}

var woodNames = map[string]string{"oak": i18n.T(i18n.Default, "barrel.oak.name"), stoneWood: i18n.T(i18n.Default, "barrel.stone.name")}

// barrelMaterials are the textures of a barrel: the wood ("*") and what
// fills it ("content": compost, or water at the last stage; cobblestone and
// lava in a stone barrel).
func barrelMaterials(wood string, stage int) map[string]customblock.Material {
	m := map[string]customblock.Material{"*": blockmodel.AlphaMaterial("skygear_barrel_" + wood)}
	liquid, filling := "water", "compost"
	if wood == stoneWood {
		liquid, filling = "lava", "cobble"
	}
	switch {
	case stage == barrelWitch:
		m["content"] = blockmodel.AlphaMaterial("skygear_barrel_witch_water")
	case stage == barrelWater:
		m["content"] = blockmodel.AlphaMaterial("skygear_barrel_" + liquid)
	case stage > 0:
		m["content"] = blockmodel.AlphaMaterial("skygear_barrel_" + filling)
	}
	return m
}

func (b Barrel) Properties() customblock.Properties {
	return customblock.Properties{Geometry: "geometry.skygear_barrel_0", Textures: barrelMaterials(b.Wood, 0), CollisionBoxes: tubBoxes}
}

// States implements block.Permutable.
func (Barrel) States() map[string][]any {
	var values []any
	for i := 0; i <= barrelWitch; i++ {
		values = append(values, fmt.Sprint(i))
	}
	return map[string][]any{barrelStageState: values}
}

// Permutations implements block.Permutable: one geometry per stage.
func (b Barrel) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for stage := 1; stage <= barrelWitch; stage++ {
		out = append(out, customblock.Permutation{
			Condition: fmt.Sprintf("q.block_state('%s') == '%d'", barrelStageState, stage),
			Properties: customblock.Properties{
				Geometry: fmt.Sprintf("geometry.skygear_barrel_%d", min(stage, barrelWater)),
				Textures: barrelMaterials(b.Wood, stage),
			},
		})
	}
	return out
}

func (b Barrel) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 2, BlastResistance: 2,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   blockmodel.CustomBlockEffective,
		Drops: func(item.Tool, []item.Enchantment) []item.Stack {
			return []item.Stack{item.NewStack(Barrel{Wood: b.Wood}, 1)}
		},
	}
}

// Activate puts a plant item (cobblestone in a stone barrel) in or takes the
// water (lava) out with a bucket.
func (b Barrel) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, ctx *item.UseContext) bool {
	p, ok := u.(*player.Player)
	if !ok || p.Sneaking() {
		return false
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return false
	}
	held, _ := p.HeldItems()
	return s.useBarrel(p, tx, pos, b, held, ctx)
}

// barrelAt is the record of the barrel at pos (nil when it holds nothing).
func (s *session) barrelAt(pos cube.Pos) *BarrelRecord { return s.state.Barrels[cropKey(pos)] }

// showBarrel sets the barrel block to the stage of its record.
func (s *session) showBarrel(tx *world.Tx, pos cube.Pos, b Barrel) {
	if stage := barrelStageOf(b.Wood, s.barrelAt(pos)); stage != b.Stage {
		b.Stage = stage
		tx.SetBlock(pos, b, nil)
	}
}

// useBarrel handles a player using the barrel at pos. It returns true if
// the click was used.
func (s *session) useBarrel(p *player.Player, tx *world.Tx, pos cube.Pos, b Barrel, held item.Stack, ctx *item.UseContext) bool {
	key := cropKey(pos)
	rec := s.state.Barrels[key]
	capacity := barrelCapacity(b.Wood)
	stone := b.Wood == stoneWood
	if stone {
		stoneRecord(rec)
	}
	if out, bucket, ok := barrelMix(rec, held); ok {
		if !bucket.Empty() && held.Count() == 1 {
			_, left := p.HeldItems()
			p.SetHeldItems(bucket, left)
		} else {
			ctx.SubtractFromCount(1)
			if !bucket.Empty() {
				s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), bucket)
			}
		}
		s.mixBarrel(tx, pos, b, out)
		s.storeState()
		return true
	}
	if stone {
		return s.useStoneBarrel(p, tx, pos, b, held, ctx)
	}
	poured, empty, liq, pourOK := pourBucket(b.Wood, held)
	switch {
	case rec != nil && rec.Witch:
		if bk, ok := held.Item().(item.Bucket); ok && bk.Empty() {
			filled := item.NewStack(sgitem.WitchWaterBucket{}, 1)
			if held.Count() == 1 {
				_, left := p.HeldItems()
				p.SetHeldItems(filled, left)
			} else {
				ctx.SubtractFromCount(1)
				s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), filled)
			}
			delete(s.state.Barrels, key)
			s.showBarrel(tx, pos, b)
			tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Water{}})
			s.storeState()
		}
		return true
	case rec != nil && rec.Water:
		if _, ok := held.Item().(sgitem.Dust); ok {
			// Dust soaks into the water and makes a clay block.
			ctx.SubtractFromCount(1)
			s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), item.NewStack(block.Clay{}, 1))
			tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Water{}})
			return true
		}
		if filled, fill, ok := emptyBucketFill(held, false); ok {
			if held.Count() == 1 {
				// A single bucket turns into the full one in the hand.
				_, left := p.HeldItems()
				p.SetHeldItems(filled, left)
			} else {
				ctx.SubtractFromCount(1)
				s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), filled)
			}
			delete(s.state.Barrels, key)
			s.showBarrel(tx, pos, b)
			tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: fill})
			s.storeState()
			return true
		}
		p.SendTip(i18n.T(i18n.Of(p), "barrel.full.water"))
		return true
	case pourOK && (rec == nil || (rec.Count == 0 && rec.Clay == 0)):
		_, left := p.HeldItems()
		p.SetHeldItems(empty, left)
		if s.state.Barrels == nil {
			s.state.Barrels = map[string]*BarrelRecord{}
		}
		s.state.Barrels[key] = &poured
		s.showBarrel(tx, pos, b)
		tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: liq})
		s.storeState()
		return true
	case rec != nil && rec.Count >= capacity:
		p.SendTip(i18n.T(i18n.Of(p), "barrel.busy"))
		return true
	case barrelAccepts(b.Wood, held):
		if s.fillBarrel(tx, pos, b, held, 1) == 0 {
			return false
		}
		ctx.SubtractFromCount(1)
		if s.barrelAt(pos).Count >= capacity {
			p.SendTip(i18n.T(i18n.Of(p), "barrel.full.plants"))
		}
		return true
	}
	return false
}

// useStoneBarrel handles a player using a stone barrel: an empty bucket takes
// a bucket's worth of its lava, a lava bucket pours one in, and a block it
// melts goes in. A click on a barrel that cannot take or give it is used up
// all the same, so the block in the hand is not placed.
func (s *session) useStoneBarrel(p *player.Player, tx *world.Tx, pos cube.Pos, b Barrel, held item.Stack, ctx *item.UseContext) bool {
	key := cropKey(pos)
	rec := s.state.Barrels[key]
	if filled, fill, ok := emptyBucketFill(held, true); ok {
		if rec == nil || rec.LavaMB < bucketMB {
			return true
		}
		if held.Count() == 1 {
			_, left := p.HeldItems()
			p.SetHeldItems(filled, left)
		} else {
			ctx.SubtractFromCount(1)
			s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), filled)
		}
		rec.LavaMB -= bucketMB
		s.forgetEmptyStone(pos, rec)
		s.showBarrel(tx, pos, b)
		tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: fill})
		s.storeState()
		return true
	}
	if wb, ok := held.Item().(sgitem.WoodenBucket); ok && !wb.Water {
		if rec != nil && rec.LavaMB > 0 {
			p.SendTip(i18n.T(i18n.Of(p), "barrel.woodenNoLava"))
		}
		return true
	}
	if bk, ok := held.Item().(item.Bucket); ok && fluidOfBucket(bk) == fluidnet.Lava {
		if rec != nil && rec.LavaMB+bucketMB > stoneLavaMax {
			return true
		}
		_, left := p.HeldItems()
		p.SetHeldItems(item.NewStack(item.Bucket{}, 1), left)
		rec = s.stoneBarrel(pos)
		rec.LavaMB += bucketMB
		s.showBarrel(tx, pos, b)
		tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Lava{}})
		s.storeState()
		return true
	}
	if barrelAccepts(b.Wood, held) {
		if s.fillBarrel(tx, pos, b, held, 1) > 0 {
			ctx.SubtractFromCount(1)
		}
		return true
	}
	return false
}

// stoneBarrel is the record of the stone barrel at pos, made when it has none.
func (s *session) stoneBarrel(pos cube.Pos) *BarrelRecord {
	rec := s.state.Barrels[cropKey(pos)]
	if rec == nil {
		rec = &BarrelRecord{}
		if s.state.Barrels == nil {
			s.state.Barrels = map[string]*BarrelRecord{}
		}
		s.state.Barrels[cropKey(pos)] = rec
	}
	stoneRecord(rec)
	return rec
}

// forgetEmptyStone forgets the record of a stone barrel that holds nothing.
func (s *session) forgetEmptyStone(pos cube.Pos, rec *BarrelRecord) {
	if rec.LavaMB == 0 && rec.Solids == 0 {
		delete(s.state.Barrels, cropKey(pos))
	}
}

// emptyBucketFill is what an empty bucket in the hand of a player becomes
// when it takes the water (lava) of a barrel, and the liquid sound; ok is
// false for anything else. A wooden bucket holds no lava.
func emptyBucketFill(held item.Stack, lava bool) (filled item.Stack, fill world.Liquid, ok bool) {
	switch b := held.Item().(type) {
	case item.Bucket:
		if !b.Empty() {
			return item.Stack{}, nil, false
		}
		if lava {
			return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Lava{Depth: 8, Still: true})}, 1), block.Lava{}, true
		}
		return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1), block.Water{}, true
	case sgitem.WoodenBucket:
		if b.Water || lava {
			return item.Stack{}, nil, false
		}
		return item.NewStack(sgitem.WoodenBucket{Water: true}, 1), block.Water{}, true
	}
	return item.Stack{}, nil, false
}

// pourBucket is what a full bucket in the hand pours into an empty barrel of
// wood, the bucket left and the liquid sound; ok is false when the bucket
// holds nothing that barrel takes. An oak barrel takes water and witch water;
// a stone barrel takes lava into what it holds (useStoneBarrel).
func pourBucket(wood string, held item.Stack) (rec BarrelRecord, empty item.Stack, liq world.Liquid, ok bool) {
	stone := wood == stoneWood
	switch b := held.Item().(type) {
	case item.Bucket:
		switch fluidOfBucket(b) {
		case fluidnet.Water:
			if !stone {
				return BarrelRecord{Water: true}, item.NewStack(item.Bucket{}, 1), block.Water{}, true
			}
		}
	case sgitem.WoodenBucket:
		if b.Water && !stone {
			return BarrelRecord{Water: true}, item.NewStack(sgitem.WoodenBucket{}, 1), block.Water{}, true
		}
	case sgitem.WitchWaterBucket:
		if !stone {
			return BarrelRecord{Witch: true}, item.NewStack(item.Bucket{}, 1), block.Water{}, true
		}
	}
	return BarrelRecord{}, item.Stack{}, nil, false
}

// fillBarrel puts up to n plant items of st into the barrel at pos (whose
// block is b) and returns how many it took; a barrel that is full or full of
// water takes none. A barrel that just filled starts making water.
func (s *session) fillBarrel(tx *world.Tx, pos cube.Pos, b Barrel, st item.Stack, n int) int {
	rec := s.state.Barrels[cropKey(pos)]
	if b.Wood == stoneWood {
		return s.fillStoneBarrel(tx, pos, b, st, n)
	}
	if _, dust := st.Item().(sgitem.Dust); dust {
		return s.soakDust(tx, pos, rec, st, n)
	}
	capacity := barrelCapacity(b.Wood)
	if (rec != nil && (rec.Water || rec.Count >= capacity)) || !barrelAccepts(b.Wood, st) {
		return 0
	}
	if rec == nil {
		rec = &BarrelRecord{}
		if s.state.Barrels == nil {
			s.state.Barrels = map[string]*BarrelRecord{}
		}
		s.state.Barrels[cropKey(pos)] = rec
	}
	n = min(n, capacity-rec.Count)
	before := barrelStageOf(b.Wood, rec)
	rec.Count += n
	tx.AddParticle(pos.Vec3Centre().Add([3]float64{0, 0.4, 0}), particle.BoneMeal{})
	switch {
	case rec.Count >= capacity:
		now, _, _ := s.clock.Now()
		rec.ReadyAt = now + barrelWaterDelay
		tx.PlaySound(pos.Vec3Centre(), sound.ComposterReady{})
	case barrelStageOf(b.Wood, rec) != before:
		tx.PlaySound(pos.Vec3Centre(), sound.ComposterFillLayer{})
	default:
		tx.PlaySound(pos.Vec3Centre(), sound.ComposterFill{})
	}
	s.showBarrel(tx, pos, b)
	s.storeState()
	return n
}

// fillStoneBarrel puts up to n blocks of st to melt into the stone barrel at
// pos, as many as its solids have room for, and returns how many it took.
func (s *session) fillStoneBarrel(tx *world.Tx, pos cube.Pos, b Barrel, st item.Stack, n int) int {
	if !barrelAccepts(b.Wood, st) {
		return 0
	}
	if rec := s.state.Barrels[cropKey(pos)]; rec != nil {
		stoneRecord(rec)
		n = min(n, (stoneSolidsMax-rec.Solids)/stoneMeltMB)
	} else {
		n = min(n, stoneSolidsMax/stoneMeltMB)
	}
	if n <= 0 {
		return 0
	}
	rec := s.stoneBarrel(pos)
	if rec.Solids == 0 {
		// The melting is counted from now: solids that were not there did not melt.
		rec.MeltAt, _, _ = s.clock.Now()
	}
	rec.Solids += n * stoneMeltMB
	tx.PlaySound(pos.Vec3Centre(), sound.ComposterFill{})
	s.showBarrel(tx, pos, b)
	s.storeState()
	return n
}

// soakDust soaks up to n dust of st in the water of the barrel at pos: each
// makes a clay block, kept in the barrel until something takes it. It
// returns how many dust it took (none when the barrel holds no water or its
// clay is full).
func (s *session) soakDust(tx *world.Tx, pos cube.Pos, rec *BarrelRecord, st item.Stack, n int) int {
	if rec == nil || !rec.Water {
		return 0
	}
	n = min(n, barrelClayBuffer-rec.Clay)
	if n <= 0 {
		return 0
	}
	rec.Clay += n
	tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Water{}})
	s.storeState()
	return n
}

// PeekItems implements itemio.Extractor: the clay blocks dust made, through
// any face.
func (b Barrel) PeekItems(tx *world.Tx, pos cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
	s := activeGameSession(tx)
	if s == nil {
		return item.Stack{}
	}
	rec := s.barrelAt(pos)
	if rec == nil || rec.Clay == 0 {
		return item.Stack{}
	}
	clay := item.NewStack(block.Clay{}, rec.Clay)
	if want != nil && !want(clay) {
		return item.Stack{}
	}
	return clay
}

// ExtractItems implements itemio.Extractor.
func (b Barrel) ExtractItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
	s := activeGameSession(tx)
	if s == nil || n < 1 || !st.Comparable(item.NewStack(block.Clay{}, 1)) {
		return item.Stack{}
	}
	rec := s.barrelAt(pos)
	if rec == nil || rec.Clay == 0 {
		return item.Stack{}
	}
	n = min(n, rec.Clay)
	rec.Clay -= n
	s.storeState()
	return item.NewStack(block.Clay{}, n)
}

// InsertItems implements itemio.Inserter: any face takes plant items (cobblestone
// in a stone barrel) and, in a barrel of water, dust.
func (b Barrel) InsertItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack) int {
	if s := activeGameSession(tx); s != nil {
		return s.fillBarrel(tx, pos, b, st, st.Count())
	}
	return 0
}

// tickBarrels turns the contents of the full barrels whose time is up into
// water, and forgets barrels that are gone.
func (s *session) tickBarrels(tx *world.Tx) {
	if len(s.state.Barrels) == 0 {
		return
	}
	now, _, _ := s.clock.Now()
	changed := false
	for key, rec := range s.state.Barrels {
		pos, ok := parseCropKey(key)
		if !ok {
			delete(s.state.Barrels, key)
			changed = true
			continue
		}
		b, ok := tx.Block(pos).(Barrel)
		if !ok {
			delete(s.state.Barrels, key)
			changed = true
			continue
		}
		if b.Wood == stoneWood {
			if s.meltBarrel(tx, pos, b, rec, now) {
				changed = true
			}
			continue
		}
		if !rec.Water && !rec.Witch && rec.Count >= matterCapacity && now >= rec.ReadyAt {
			rec.Water = true
			countAll(tx, "barrel_water", 1)
			s.showBarrel(tx, pos, b)
			tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Water{}})
			changed = true
		}
		if s.tickWitchWater(tx, pos, b, rec, now) {
			changed = true
		}
	}
	if changed {
		s.storeState()
	}
}

// meltBarrel melts the solids of a stone barrel for the melting steps since
// they were last counted: each step the heat value of the block under it in
// mB, as long as there are solids and the lava has room. Every bucket's worth
// made counts as a lava made (barrel_lava). It reports whether the barrel
// changed.
func (s *session) meltBarrel(tx *world.Tx, pos cube.Pos, b Barrel, rec *BarrelRecord, now time.Duration) bool {
	changed := rec.Lava || rec.Count > 0
	stoneRecord(rec)
	if rec.Solids == 0 || now < rec.MeltAt {
		rec.MeltAt = now
		if changed {
			s.showBarrel(tx, pos, b)
		}
		return changed
	}
	steps := int((now - rec.MeltAt) / stoneMeltStep)
	if steps == 0 {
		return changed
	}
	rec.MeltAt += time.Duration(steps) * stoneMeltStep
	n := min(steps*heatValue(tx.Block(pos.Side(cube.FaceDown))), rec.Solids, stoneLavaMax-rec.LavaMB)
	if n <= 0 {
		return changed
	}
	first := rec.LavaMB == 0
	rec.Solids -= n
	rec.LavaMB += n
	rec.Made += n
	for ; rec.Made >= bucketMB; rec.Made -= bucketMB {
		countAll(tx, "barrel_lava", 1)
	}
	s.showBarrel(tx, pos, b)
	if first {
		tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Lava{}})
	}
	return true
}

// breakBarrel forgets what a broken barrel held; the clay it had made drops.
func (s *session) breakBarrel(pos cube.Pos, drops *[]item.Stack) {
	if rec := s.barrelAt(pos); rec != nil && rec.Clay > 0 {
		*drops = append(*drops, item.NewStack(block.Clay{}, rec.Clay))
	}
	delete(s.state.Barrels, cropKey(pos))
}

// barrelLiquid is the panel line of the liquid a barrel holds (water, witch
// water, lava), nil when it holds none.
func (s *session) barrelLiquid(lang string, pos cube.Pos) *waila.ItemLine {
	rec := s.barrelAt(pos)
	bucket := fmt.Sprintf(i18n.T(lang, "barrel.lines.mb"), bucketMB)
	switch {
	case rec == nil:
		return nil
	case rec.Witch:
		return sgitem.LiquidLine(lang, sgitem.LiquidWitchWater, bucket)
	case rec.Water:
		return sgitem.LiquidLine(lang, sgitem.LiquidWater, bucket)
	case rec.LavaMB > 0:
		return sgitem.LiquidLine(lang, sgitem.LiquidLava, fmt.Sprintf(i18n.T(lang, "barrel.lines.tank"), rec.LavaMB, stoneLavaMax))
	}
	return nil
}

// barrelLines are the lines of a barrel's WAILA panel; its liquid shows on
// the icon line (barrelLiquid).
func (s *session) barrelLines(lang string, tx *world.Tx, pos cube.Pos) []string {
	if b, ok := tx.Block(pos).(Barrel); ok && b.Wood == stoneWood {
		return s.stoneBarrelLines(lang, tx, pos)
	}
	rec := s.barrelAt(pos)
	switch {
	case rec != nil && rec.Witch:
		return nil
	case rec != nil && rec.Water && rec.WitchAt > 0:
		now, _, _ := s.clock.Now()
		left := max(rec.WitchAt-now, 0).Round(time.Second)
		return []string{fmt.Sprintf(i18n.T(lang, "barrel.lines.toWitch"), int(left.Minutes()), int(left.Seconds())%60)}
	case rec == nil || rec.Count == 0:
		return []string{i18n.T(lang, "barrel.lines.empty")}
	case rec.Water && rec.Clay > 0:
		return []string{fmt.Sprintf(i18n.T(lang, "barrel.lines.clay"), rec.Clay)}
	case rec.Water:
		return nil
	case rec.Count >= matterCapacity:
		now, _, _ := s.clock.Now()
		left := max(rec.ReadyAt-now, 0).Round(time.Second)
		return []string{fmt.Sprintf(i18n.T(lang, "barrel.lines.water"), int(left.Minutes()), int(left.Seconds())%60)}
	}
	return []string{fmt.Sprintf(i18n.T(lang, "barrel.lines.plants"), rec.Count, matterCapacity)}
}

// stoneBarrelLines are the lines of a stone barrel's WAILA panel (its lava
// shows on the icon line): the solids and how they melt.
func (s *session) stoneBarrelLines(lang string, tx *world.Tx, pos cube.Pos) []string {
	rec := s.barrelAt(pos)
	switch {
	case rec == nil || (rec.Solids == 0 && rec.LavaMB == 0):
		return []string{i18n.T(lang, "barrel.stone.lines.empty")}
	case rec.Solids == 0:
		return nil
	}
	lines := []string{fmt.Sprintf(i18n.T(lang, "barrel.stone.lines.solids"), rec.Solids, stoneSolidsMax)}
	heat := heatValue(tx.Block(pos.Side(cube.FaceDown)))
	switch {
	case heat == 0:
		return append(lines, i18n.T(lang, "barrel.stone.lines.noHeat"))
	case rec.LavaMB >= stoneLavaMax:
		return append(lines, i18n.T(lang, "barrel.stone.lines.lavaFull"))
	}
	steps := (min(rec.Solids, stoneLavaMax-rec.LavaMB) + heat - 1) / heat
	left := (time.Duration(steps) * stoneMeltStep).Round(time.Second)
	return append(lines, fmt.Sprintf(i18n.T(lang, "barrel.stone.lines.melting"), int(left.Minutes()), int(left.Seconds())%60, heat))
}

func init() {
	for _, w := range barrelWoods {
		world.RegisterBlock(Barrel{Wood: w})
		world.RegisterItem(Barrel{Wood: w})
		// Every stage is a block state of its own.
		for stage := 1; stage <= barrelWitch; stage++ {
			world.RegisterBlock(Barrel{Wood: w, Stage: stage})
		}
	}
}

// FluidRoles implements fluidnet.Holder: a barrel of water gives its bucket's
// worth to a fluid conduit, a stone barrel any part of its lava; it takes
// none.
func (Barrel) FluidRoles() (source, sink bool) { return true, false }

// FillFluid implements fluidnet.Holder.
func (Barrel) FillFluid(fluidnet.World, cube.Pos, fluidnet.Kind, int, bool) int { return 0 }

// DrainFluid implements fluidnet.Holder: the water of a barrel all of the
// bucket at once, and only when max allows it; the lava of a stone barrel up
// to max.
func (b Barrel) DrainFluid(w fluidnet.World, pos cube.Pos, max int, simulate bool) (fluidnet.Kind, int) {
	tx, ok := w.(*world.Tx)
	if !ok || max <= 0 {
		return fluidnet.None, 0
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return fluidnet.None, 0
	}
	rec := s.state.Barrels[cropKey(pos)]
	if b.Wood == stoneWood {
		stoneRecord(rec)
		if rec == nil || rec.LavaMB == 0 {
			return fluidnet.None, 0
		}
		n := min(max, rec.LavaMB)
		if !simulate {
			rec.LavaMB -= n
			s.forgetEmptyStone(pos, rec)
			s.showBarrel(tx, pos, b)
			s.storeState()
		}
		return fluidnet.Lava, n
	}
	if rec == nil || !rec.Water || max < bucketMB {
		return fluidnet.None, 0
	}
	k := fluidnet.Water
	if !simulate {
		delete(s.state.Barrels, cropKey(pos))
		s.showBarrel(tx, pos, b)
		s.storeState()
	}
	return k, bucketMB
}

var _ fluidnet.Holder = Barrel{}
var _ itemio.Extractor = Barrel{}
