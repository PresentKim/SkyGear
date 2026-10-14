package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
)

// Barrels (docs/design/02-dragonfactory.md §3.6) are the family of wooden
// tubs of the game; the first is the oak barrel. Plant items go in like into
// the composter, 32 of them, and the barrel shows six fill stages (stage =
// count * 6 / 32). A full barrel takes barrelWaterDelay to turn the
// contents into water (stage 7); a bucket takes the water, and the barrel
// is empty again. The contents of each barrel are in the save
// (State.Barrels, by position).

const (
	// barrelFillStages are the stages of filling; the next stage is water.
	barrelFillStages = 6
	// barrelWater is the stage of a barrel full of water.
	barrelWater = 7
	// barrelWaterDelay is how long (instance time) a full barrel needs to
	// make water. Draft.
	barrelWaterDelay = 2 * time.Minute
)

// barrelStageState is the one block state of a barrel: its stage 0 to 7.
const barrelStageState = "dragonfactory:barrel_stage"

// barrelWoods are the materials of the barrel family, in registration order:
// the oak barrel makes water from plants, the stone barrel lava from
// cobblestone (stoneWood).
var barrelWoods = []string{"oak", stoneWood}

// The stone barrel (docs/design/02-dragonfactory.md §3.7): eight cobblestone
// go in, and with a heat source right under it they turn into lava, one
// bucket's worth (a lava stage like the water one). A lit... any candle is the
// least heat (speed 1×) and takes stoneHeatNeeded; lava (source or flowing)
// is 4×, a magma block 8×.
const (
	stoneWood           = "stone"
	stoneBarrelCapacity = 8
	stoneHeatNeeded     = 2 * time.Minute
)

// heatMultiplier is the speed the block under a stone barrel heats it with
// (0: no heat source).
func heatMultiplier(b world.Block) int {
	switch b.(type) {
	case block.Candle:
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
	// ReadyAt is the instance time the contents turn into water (a full
	// barrel only).
	ReadyAt time.Duration `json:"ready_at,omitempty"`
	// Lava is set once the cobblestone of a stone barrel turned into lava;
	// Heat is the heat it has had so far (instance time × the speed of the
	// heat source) and HeatAt when it was last counted.
	Lava   bool          `json:"lava,omitempty"`
	Heat   time.Duration `json:"heat,omitempty"`
	HeatAt time.Duration `json:"heat_at,omitempty"`
}

// barrelStage is the stage a record of an oak barrel shows.
func barrelStage(r *BarrelRecord) int { return barrelStageOf("oak", r) }

// barrelStageOf is the stage a record of a barrel of wood shows.
func barrelStageOf(wood string, r *BarrelRecord) int {
	switch {
	case r == nil:
		return 0
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
func (Barrel) Model() world.BlockModel  { return model.Solid{} }
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
	for i := 0; i <= barrelWater; i++ {
		values = append(values, fmt.Sprint(i))
	}
	return map[string][]any{barrelStageState: values}
}

// Permutations implements block.Permutable: one geometry per stage.
func (b Barrel) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for stage := 1; stage <= barrelWater; stage++ {
		out = append(out, customblock.Permutation{
			Condition: fmt.Sprintf("q.block_state('%s') == '%d'", barrelStageState, stage),
			Properties: customblock.Properties{
				Geometry: fmt.Sprintf("geometry.dragonfactory_barrel_%d", stage),
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
	switch {
	case rec != nil && (rec.Water || rec.Lava):
		if bucket, ok := held.Item().(item.Bucket); ok && bucket.Empty() {
			var liquid world.Liquid = block.Water{Depth: 8, Still: true}
			var fill world.Liquid = block.Water{}
			if rec.Lava {
				liquid, fill = block.Lava{Depth: 8, Still: true}, block.Lava{}
			}
			filled := item.NewStack(item.Bucket{Content: item.LiquidBucketContent(liquid)}, 1)
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

// fillBarrel puts up to n plant items of st into the barrel at pos (whose
// block is b) and returns how many it took; a barrel that is full or full of
// water takes none. A barrel that just filled starts making water.
func (s *session) fillBarrel(tx *world.Tx, pos cube.Pos, b Barrel, st item.Stack, n int) int {
	rec := s.state.Barrels[cropKey(pos)]
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

// InsertItems implements itemio.Inserter: any face takes plant items (cobblestone
// in a stone barrel).
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
		if !rec.Water && rec.Count >= matterCapacity && now >= rec.ReadyAt {
			rec.Water = true
			s.showBarrel(tx, pos, b)
			tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Water{}})
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
	s.showBarrel(tx, pos, b)
	tx.PlaySound(pos.Vec3Centre(), sound.BucketFill{Liquid: block.Lava{}})
	return true
}

// breakBarrel forgets what a broken barrel held.
func (s *session) breakBarrel(pos cube.Pos) {
	delete(s.state.Barrels, cropKey(pos))
}

// barrelLines are the lines of a barrel's WAILA panel.
func (s *session) barrelLines(tx *world.Tx, pos cube.Pos) []string {
	if b, ok := tx.Block(pos).(Barrel); ok && b.Wood == stoneWood {
		return s.stoneBarrelLines(tx, pos)
	}
	rec := s.barrelAt(pos)
	switch {
	case rec == nil || rec.Count == 0:
		return []string{i18n.T(i18n.Default, "dragonfactory.barrel.barrelLines.1"), i18n.T(i18n.Default, "dragonfactory.barrel.barrelLines.2")}
	case rec.Water:
		return []string{i18n.T(i18n.Default, "dragonfactory.barrel.barrelLines.3"), i18n.T(i18n.Default, "dragonfactory.barrel.barrelLines.4")}
	case rec.Count >= matterCapacity:
		now, _, _ := s.clock.Now()
		left := max(rec.ReadyAt-now, 0).Round(time.Second)
		return []string{fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.barrel.barrelLines.5"), int(left.Minutes()), int(left.Seconds())%60)}
	}
	return []string{fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.barrel.barrelLines.6"), rec.Count, matterCapacity)}
}

// stoneBarrelLines are the lines of a stone barrel's WAILA panel.
func (s *session) stoneBarrelLines(tx *world.Tx, pos cube.Pos) []string {
	rec := s.barrelAt(pos)
	switch {
	case rec != nil && rec.Lava:
		return []string{i18n.T(i18n.Default, "barrel.stone.lines.lava")}
	case rec == nil || rec.Count == 0:
		return []string{i18n.T(i18n.Default, "barrel.stone.lines.empty"), i18n.T(i18n.Default, "barrel.stone.lines.heatHint")}
	case rec.Count < stoneBarrelCapacity:
		return []string{fmt.Sprintf(i18n.T(i18n.Default, "barrel.stone.lines.count"), rec.Count, stoneBarrelCapacity)}
	}
	mult := heatMultiplier(tx.Block(pos.Side(cube.FaceDown)))
	if mult == 0 {
		return []string{i18n.T(i18n.Default, "barrel.stone.lines.noHeat")}
	}
	left := max(stoneHeatNeeded-rec.Heat, 0) / time.Duration(mult)
	left = left.Round(time.Second)
	return []string{fmt.Sprintf(i18n.T(i18n.Default, "barrel.stone.lines.heating"), int(left.Minutes()), int(left.Seconds())%60, mult)}
}

func init() {
	for _, w := range barrelWoods {
		world.RegisterBlock(Barrel{Wood: w})
		world.RegisterItem(Barrel{Wood: w})
		// Every stage is a block state of its own.
		for stage := 1; stage <= barrelWater; stage++ {
			world.RegisterBlock(Barrel{Wood: w, Stage: stage})
		}
	}
}
