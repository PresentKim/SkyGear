package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/waila"
	"github.com/presentkim/dragonfactory/internal/fluidnet"
	"github.com/presentkim/dragonfactory/internal/itemio"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
)

// Barrels (docs/design/02-dragonfactory.md §5) are the family of wooden
// tubs of the game; the first is the oak barrel. Plant items go in like into
// the composter, 32 of them, and the barrel shows six fill stages (stage =
// count * 6 / 32). A full barrel takes barrelWaterDelay to turn the
// contents into water (stage 7); a bucket takes the water, and the barrel
// is empty again. The contents of each barrel are in the save
// (State.Barrels, by position).

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
const barrelStageState = "dragonfactory:barrel_stage"

// barrelWoods are the materials of the barrel family, in registration order:
// the oak barrel makes water from plants, the stone barrel lava from
// cobblestone (stoneWood).
var barrelWoods = []string{"oak", stoneWood}

// The stone barrel (docs/design/02-dragonfactory.md §5): eight cobblestone
// go in, and with a heat source right under it they turn into lava, one
// bucket's worth (a lava stage like the water one). A torch is the least
// heat (speed 1×) and takes stoneHeatNeeded; lava (source or flowing) is 4×,
// a magma block 8×.
const (
	stoneWood           = "stone"
	stoneBarrelCapacity = 8
	stoneHeatNeeded     = time.Minute
)

// heatMultiplier is the speed the block under a stone barrel heats it with
// (0: no heat source).
func heatMultiplier(b world.Block) int {
	switch b.(type) {
	case block.Torch:
		return 1
	case block.Lava:
		return 4
	case block.Magma:
		return 8
	}
	return 0
}

// barrelCapacity is how many items a barrel of wood takes; barrelAccepts
// whether it takes st.
func barrelCapacity(wood string) int {
	if wood == stoneWood {
		return stoneBarrelCapacity
	}
	return matterCapacity
}

func barrelAccepts(wood string, st item.Stack) bool {
	if wood == stoneWood {
		return !st.Empty() && itemKey(st.Item()) == itemKey(block.Cobblestone{}) && plain(st)
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
	// Lava is set once the cobblestone of a stone barrel turned into lava;
	// Heat is the heat it has had so far (instance time × the speed of the
	// heat source) and HeatAt when it was last counted.
	Lava   bool          `json:"lava,omitempty"`
	Heat   time.Duration `json:"heat,omitempty"`
	HeatAt time.Duration `json:"heat_at,omitempty"`
	// Witch is set once the water of an oak barrel on mycelium turned into
	// witch water; WitchAt is the instance time it turns (0: not on mycelium).
	Witch   bool          `json:"witch,omitempty"`
	WitchAt time.Duration `json:"witch_at,omitempty"`
}

// barrelStage is the stage a record of an oak barrel shows.
func barrelStage(r *BarrelRecord) int { return barrelStageOf("oak", r) }

// barrelStageOf is the stage a record of a barrel of wood shows.
func barrelStageOf(wood string, r *BarrelRecord) int {
	switch {
	case r == nil:
		return 0
	case r.Witch:
		return barrelWitch
	case r.Water || r.Lava:
		return barrelWater
	}
	return r.Count * barrelFillStages / barrelCapacity(wood)
}

func (b Barrel) EncodeItem() (string, int16) { return "dragonfactory:barrel_" + b.Wood, 0 }
func (b Barrel) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:barrel_" + b.Wood, map[string]any{barrelStageState: fmt.Sprint(b.Stage)}
}
func (b Barrel) Hash() (uint64, uint64) { return barrelHashes[b.Wood], uint64(b.Stage) }
func (Barrel) Model() world.BlockModel  { return tubModel{} }
func (b Barrel) Name() string           { return woodNames[b.Wood] + i18n.T(i18n.Default, "name.Barrel.suffix") }
func (Barrel) Category() category.Category {
	return category.Construction()
}

var woodNames = map[string]string{"oak": i18n.T(i18n.Default, "dragonfactory.barrel.init.1"), stoneWood: i18n.T(i18n.Default, "barrel.stone.name")}

// barrelMaterials are the textures of a barrel: the wood ("*") and what
// fills it ("content": compost, or water at the last stage; cobblestone and
// lava in a stone barrel).
func barrelMaterials(wood string, stage int) map[string]customblock.Material {
	m := map[string]customblock.Material{"*": alphaMaterial("dragonfactory_barrel_" + wood)}
	liquid, filling := "water", "compost"
	if wood == stoneWood {
		liquid, filling = "lava", "cobble"
	}
	switch {
	case stage == barrelWitch:
		m["content"] = alphaMaterial("dragonfactory_barrel_witch_water")
	case stage == barrelWater:
		m["content"] = alphaMaterial("dragonfactory_barrel_" + liquid)
	case stage > 0:
		m["content"] = alphaMaterial("dragonfactory_barrel_" + filling)
	}
	return m
}

func (b Barrel) Properties() customblock.Properties {
	return customblock.Properties{Geometry: "geometry.dragonfactory_barrel_0", Textures: barrelMaterials(b.Wood, 0)}
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
				Geometry: fmt.Sprintf("geometry.dragonfactory_barrel_%d", min(stage, barrelWater)),
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
		Effective:   customBlockEffective,
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
	switch {
	case rec != nil && rec.Witch:
		if bk, ok := held.Item().(item.Bucket); ok && bk.Empty() {
			filled := item.NewStack(WitchWaterBucket{}, 1)
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
	case rec != nil && (rec.Water || rec.Lava):
		if _, ok := held.Item().(Dust); ok && rec.Water {
			// Dust soaks into the water and makes a clay block (docs/design/02-dragonfactory.md).
			ctx.SubtractFromCount(1)
			s.giveItem(p, tx, pos.Side(cube.FaceUp).Vec3Middle(), item.NewStack(block.Clay{}, 1))
			tx.PlaySound(pos.Vec3Centre(), sound.BucketEmpty{Liquid: block.Water{}})
			return true
		}
		if filled, fill, ok := emptyBucketFill(held, rec.Lava); ok {
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
		if wb, ok := held.Item().(WoodenBucket); ok && !wb.Water && rec.Lava {
			p.SendTip(i18n.T(i18n.Of(p), "barrel.woodenNoLava"))
			return true
		}
		if rec.Lava {
			p.SendTip(i18n.T(i18n.Of(p), "barrel.stone.lavaFull"))
		} else {
			p.SendTip(i18n.T(i18n.Of(p), "dragonfactory.barrel.useBarrel.1"))
		}
		return true
	case rec != nil && rec.Count >= capacity:
		if stone {
			p.SendTip(i18n.T(i18n.Of(p), "barrel.stone.heating"))
		} else {
			p.SendTip(i18n.T(i18n.Of(p), "dragonfactory.barrel.useBarrel.2"))
		}
		return true
	case barrelAccepts(b.Wood, held):
		if s.fillBarrel(tx, pos, b, held, 1) == 0 {
			return false
		}
		ctx.SubtractFromCount(1)
		if s.barrelAt(pos).Count >= capacity {
			if stone {
				p.SendTip(i18n.T(i18n.Of(p), "barrel.stone.full"))
			} else {
				p.SendTip(i18n.T(i18n.Of(p), "dragonfactory.barrel.useBarrel.3"))
			}
		}
		return true
	}
	return false
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
	case WoodenBucket:
		if b.Water || lava {
			return item.Stack{}, nil, false
		}
		return item.NewStack(WoodenBucket{Water: true}, 1), block.Water{}, true
	}
	return item.Stack{}, nil, false
}

// fillBarrel puts up to n plant items of st into the barrel at pos (whose
// block is b) and returns how many it took; a barrel that is full or full of
// water takes none. A barrel that just filled starts making water.
func (s *session) fillBarrel(tx *world.Tx, pos cube.Pos, b Barrel, st item.Stack, n int) int {
	rec := s.state.Barrels[cropKey(pos)]
	if _, dust := st.Item().(Dust); dust && b.Wood != stoneWood {
		return s.soakDust(tx, pos, rec, st, n)
	}
	capacity := barrelCapacity(b.Wood)
	if (rec != nil && (rec.Water || rec.Lava || rec.Count >= capacity)) || !barrelAccepts(b.Wood, st) {
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
		rec.Heat, rec.HeatAt = 0, now
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
		var pos cube.Pos
		if _, err := fmt.Sscanf(key, "%d,%d,%d", &pos[0], &pos[1], &pos[2]); err != nil {
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
			if s.heatBarrel(tx, pos, b, rec, now) {
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

// heatBarrel counts the heat a full stone barrel got since it was last
// counted (the time × the speed of the heat source under it) and turns its
// cobblestone into lava once it has had stoneHeatNeeded. It reports whether
// the barrel turned into lava.
func (s *session) heatBarrel(tx *world.Tx, pos cube.Pos, b Barrel, rec *BarrelRecord, now time.Duration) bool {
	if rec.Lava || rec.Count < stoneBarrelCapacity {
		return false
	}
	dt := max(now-rec.HeatAt, 0)
	rec.HeatAt = now
	rec.Heat += dt * time.Duration(heatMultiplier(tx.Block(pos.Side(cube.FaceDown))))
	if rec.Heat < stoneHeatNeeded {
		return false
	}
	rec.Lava = true
	countAll(tx, "barrel_lava", 1)
	s.showBarrel(tx, pos, b)
	tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Lava{}})
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
	full := i18n.T(lang, "barrel.lines.full")
	switch {
	case rec == nil:
		return nil
	case rec.Witch:
		return liquidLine(lang, liquidWitchWater, full)
	case rec.Water:
		return liquidLine(lang, liquidWater, full)
	case rec.Lava:
		return liquidLine(lang, liquidLava, full)
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
		return []string{i18n.T(lang, "dragonfactory.barrel.barrelLines.1")}
	case rec.Water && rec.Clay > 0:
		return []string{fmt.Sprintf(i18n.T(lang, "barrel.lines.clay"), rec.Clay)}
	case rec.Water:
		return nil
	case rec.Count >= matterCapacity:
		now, _, _ := s.clock.Now()
		left := max(rec.ReadyAt-now, 0).Round(time.Second)
		return []string{fmt.Sprintf(i18n.T(lang, "dragonfactory.barrel.barrelLines.5"), int(left.Minutes()), int(left.Seconds())%60)}
	}
	return []string{fmt.Sprintf(i18n.T(lang, "dragonfactory.barrel.barrelLines.6"), rec.Count, matterCapacity)}
}

// stoneBarrelLines are the lines of a stone barrel's WAILA panel.
func (s *session) stoneBarrelLines(lang string, tx *world.Tx, pos cube.Pos) []string {
	rec := s.barrelAt(pos)
	switch {
	case rec != nil && rec.Lava:
		return nil
	case rec == nil || rec.Count == 0:
		return []string{i18n.T(lang, "barrel.stone.lines.empty")}
	case rec.Count < stoneBarrelCapacity:
		return []string{fmt.Sprintf(i18n.T(lang, "barrel.stone.lines.count"), rec.Count, stoneBarrelCapacity)}
	}
	mult := heatMultiplier(tx.Block(pos.Side(cube.FaceDown)))
	if mult == 0 {
		return []string{i18n.T(lang, "barrel.stone.lines.noHeat")}
	}
	left := max(stoneHeatNeeded-rec.Heat, 0) / time.Duration(mult)
	left = left.Round(time.Second)
	return []string{fmt.Sprintf(i18n.T(lang, "barrel.stone.lines.heating"), int(left.Minutes()), int(left.Seconds())%60, mult)}
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

// FluidRoles implements fluidnet.Holder: a barrel full of water (lava) gives a
// bucket's worth to a fluid conduit; it takes none.
func (Barrel) FluidRoles() (source, sink bool) { return true, false }

// FillFluid implements fluidnet.Holder.
func (Barrel) FillFluid(fluidnet.World, cube.Pos, fluidnet.Kind, int, bool) int { return 0 }

// DrainFluid implements fluidnet.Holder: all of the bucket at once, and only
// when max allows it.
func (b Barrel) DrainFluid(w fluidnet.World, pos cube.Pos, max int, simulate bool) (fluidnet.Kind, int) {
	tx, ok := w.(*world.Tx)
	if !ok || max < bucketMB {
		return fluidnet.None, 0
	}
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return fluidnet.None, 0
	}
	rec := s.state.Barrels[cropKey(pos)]
	if rec == nil || !(rec.Water || rec.Lava) {
		return fluidnet.None, 0
	}
	k := fluidnet.Water
	if rec.Lava {
		k = fluidnet.Lava
	}
	if !simulate {
		delete(s.state.Barrels, cropKey(pos))
		s.showBarrel(tx, pos, b)
		s.storeState()
	}
	return k, bucketMB
}

var _ fluidnet.Holder = Barrel{}
var _ itemio.Extractor = Barrel{}
