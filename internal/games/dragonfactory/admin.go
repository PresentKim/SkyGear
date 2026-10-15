package dragonfactory

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/perf"
	"github.com/presentkim/dragonfactory/internal/timed"
)

// Operator tools (/dfadmin) for testing. Only players for whom the game's
// operator check returns true see and run them.

type adminBase struct{ g *Game }

func (a adminBase) Allow(src cmd.Source) bool {
	p, ok := src.(*player.Player)
	return ok && a.g.isOperator != nil && a.g.isOperator(p)
}

// session returns the session of the operator's world.
func (a adminBase) session(src cmd.Source, o *cmd.Output, tx *world.Tx) (*player.Player, *session, bool) {
	p := src.(*player.Player)
	s, ok := a.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.admin.session.1"))
	}
	return p, s, ok
}

type adminItem struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"item"`
	Name  string            `cmd:"name"`
	Count cmd.Optional[int] `cmd:"count"`
}

// Run gives any registered item by name, e.g. minecraft:iron_ore.
func (c adminItem) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	name := c.Name
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	it, ok := world.ItemByName(name, 0)
	if !ok {
		o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.Run.1"), name)
		return
	}
	st := item.NewStack(it, max(1, c.Count.LoadOr(1)))
	if added, _ := p.Inventory().AddItem(st); added < st.Count() {
		s.dropAt(tx, p.Position(), st.Grow(added-st.Count()))
	}
	o.Printf("%s x%d", itemName(st), st.Count())
}

type adminSkip struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"skip"`
	Minutes int            `cmd:"minutes"`
}

// Run advances the instance time (days, crops, lost-and-found).
func (c adminSkip) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		for _, day := range s.clock.Advance(time.Duration(max(0, c.Minutes)) * time.Minute) {
			s.NewDay(tx, day)
		}
		s.Second(tx)
		total, day, _ := s.clock.Now()
		o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.2"), formatPlayTime(total), day)
	}
}

type adminLook struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"look"`
	Yaw   float64        `cmd:"yaw"`
	Pitch float64        `cmd:"pitch"`
}

// Run turns the operator's view (yaw 0 = south, pitch 90 = down) and
// updates the WAILA panel at once (tests the panel without aiming).
func (c adminLook) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		r := p.Rotation()
		p.Move(mgl64.Vec3{}, c.Yaw-r.Yaw(), c.Pitch-r.Pitch())
		s.updateLook(p, tx, p.Position(), p.Rotation(), true)
		if at, b, ok := lookedAt(p, tx, p.Position(), p.Rotation()); ok {
			o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.3"), blockName(b), at[0], at[1], at[2])
		} else {
			o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.4"))
		}
	}
}

type adminTimed struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"timed"`
	Seconds int            `cmd:"seconds"`
}

func (c adminTimed) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	main, off := p.HeldItems()
	if main.Empty() {
		o.Error(i18n.T(i18n.Default, "dragonfactory.admin.Run.5"))
		return
	}
	if c.Seconds <= 0 {
		p.SetHeldItems(timed.Clear(main), off)
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.6"))
		return
	}
	p.SetHeldItems(timed.For(main, time.Duration(c.Seconds)*time.Second), off)
	o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.7"), timed.Describe(time.Duration(c.Seconds)*time.Second))
}

// adminFairy sends the gift fairy to the operator now.
type adminFairy struct {
	adminBase
	Sub cmd.SubCommand `cmd:"fairy"`
}

func (c adminFairy) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		s.spawnFairy(tx, p)
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.8"))
	}
}

// giveItems are the items /dfadmin give hands out: the game's own items
// and every machine, plus a water bucket.
func giveItems() map[string]func() item.Stack {
	out := map[string]func() item.Stack{
		"research_table":   func() item.Stack { return item.NewStack(ResearchTable{}, 1) },
		"drawer":           func() item.Stack { return item.NewStack(Drawer{}, 1) },
		"drawer_key":       func() item.Stack { return item.NewStack(DrawerKey{}, 1) },
		"upgrade_capacity": func() item.Stack { return item.NewStack(DrawerUpgrade{Kind: UpgradeCapacity}, 1) },
		"upgrade_void":     func() item.Stack { return item.NewStack(DrawerUpgrade{Kind: UpgradeVoid}, 1) },
		"hammer":           func() item.Stack { return item.NewStack(Hammer{}, 1) },
		"string_mesh":      func() item.Stack { return item.NewStack(Mesh{Tier: "string"}, 1) },
		"barrel":           func() item.Stack { return item.NewStack(Barrel{Wood: "oak"}, 1) },
		"stone_barrel":     func() item.Stack { return item.NewStack(Barrel{Wood: stoneWood}, 1) },
		"wood_hopper":      func() item.Stack { return item.NewStack(newFlowHopper("wood_hopper", cube.FaceDown), 1) },
		"wood_hopper_reverse": func() item.Stack {
			return item.NewStack(newFlowHopper("wood_hopper_reverse", cube.FaceUp), 1)
		},
		"iron_hopper_reverse": func() item.Stack {
			return item.NewStack(newFlowHopper("iron_hopper_reverse", cube.FaceUp), 1)
		},
		"water": func() item.Stack {
			return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
		},
	}
	for _, k := range machineOrder {
		out[k] = func() item.Stack { return machineItem(k) }
	}
	for t := 1; t <= conduitTiers; t++ {
		out[fmt.Sprintf("energy_conduit_%d", t)] = func() item.Stack { return item.NewStack(EnergyConduit{Tier: t}, 1) }
		out[fmt.Sprintf("item_conduit_%d", t)] = func() item.Stack { return item.NewStack(ItemConduit{Tier: t}, 1) }
		out[fmt.Sprintf("fluid_conduit_%d", t)] = func() item.Stack { return item.NewStack(FluidConduit{Tier: t}, 1) }
	}
	for _, k := range oreKinds {
		out["ore_chunk_"+k.ore] = func() item.Stack { return item.NewStack(OreChunk{Ore: k.ore}, 1) }
	}
	for _, k := range toolKinds {
		for _, h := range headMaterials {
			for _, hd := range handleMaterials {
				out[fmt.Sprintf("tool_%s_%s_%s", k, h.id, hd.id)] = func() item.Stack {
					return describeTool(item.NewStack(TinkerTool{Kind: k, Head: h.id, Handle: hd.id}, 1))
				}
			}
		}
	}
	for _, c := range petalColours {
		out["petal_"+c] = func() item.Stack { return item.NewStack(Petal{Colour: c}, 1) }
	}
	for _, sd := range plantSeeds {
		out["seed_"+sd] = func() item.Stack { return item.NewStack(PlantSeed{Seed: sd}, 1) }
	}
	for _, m := range mobKinds {
		out["essence_"+m] = func() item.Stack { return item.NewStack(Essence{Mob: m}, 1) }
	}
	for _, k := range researchEssenceKinds {
		out["research_essence_"+k] = func() item.Stack { return item.NewStack(ResearchEssence{Kind: k}, 1) }
	}
	for _, m := range dustMetals {
		out["metal_dust_"+m] = func() item.Stack { return item.NewStack(MetalDust{Metal: m}, 1) }
	}
	for _, m := range ingotMetals {
		out["ingot_"+m] = func() item.Stack { return item.NewStack(Ingot{Metal: m}, 1) }
	}
	return out
}

type giveEnum string

func (giveEnum) Type() string { return "df_item" }
func (giveEnum) Options(cmd.Source) []string {
	ids := slices.Sorted(maps.Keys(giveItems()))
	return ids
}

type adminGive struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"give"`
	Item  giveEnum          `cmd:"item"`
	Count cmd.Optional[int] `cmd:"count"`
}

func (c adminGive) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	f, ok := giveItems()[string(c.Item)]
	if !ok {
		o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.Run.9"), c.Item)
		return
	}
	n := max(1, c.Count.LoadOr(1))
	for range n {
		it := f()
		if added, _ := p.Inventory().AddItem(it); added < it.Count() {
			s.dropAt(tx, p.Position(), it.Grow(added-it.Count()))
		}
	}
	o.Printf("%s x%d", itemName(f()), n)
}

type adminStatus struct {
	adminBase
	Sub cmd.SubCommand `cmd:"status"`
}

func (c adminStatus) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		o.Printf("researched=%d chunks=%d advanced=%d", len(s.state.Researched), len(s.state.Chunks), len(s.state.Advancements))
	}
}

// adminWave places a dimension gate beside the operator and starts a level of
// it, opening the levels below (for testing the waves).
type adminWave struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"wave"`
	Level int            `cmd:"level"`
}

func (c adminWave) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	if c.Level < 1 || c.Level > len(waveLevels) {
		o.Errorf("level 1-%d", len(waveLevels))
		return
	}
	pos := cube.PosFromVec3(p.Position()).Add(cube.Pos{2, 0, 0})
	if _, isGate := tx.Block(pos).(Machine); !isGate {
		tx.SetBlock(pos, newMachine(KindGate, "", ""), nil)
	}
	s.state.GateTier = max(s.state.GateTier, c.Level-1)
	if !s.startWave(tx, pos, c.Level) {
		o.Error("a wave is already going on")
		return
	}
	o.Printf("wave %d started with %d monsters", c.Level, len(s.wave.left))
}

func registerAdmin(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("dfadmin", i18n.T(i18n.Default, "dragonfactory.admin.registerAdmin.1"), nil,
		adminItem{adminBase: b}, adminSkip{adminBase: b}, adminGive{adminBase: b}, adminStatus{adminBase: b}, adminTimed{adminBase: b}, adminFairy{adminBase: b}, adminLook{adminBase: b}, adminFactory{adminBase: b}, adminDense{adminBase: b}, adminExpandView{adminBase: b}, adminTPS{adminBase: b}, adminResearch{adminBase: b}, adminAdvance{adminBase: b}, adminWave{adminBase: b}))
}

// expandViewEnum names an expansion view (expand.go expandViews).
type expandViewEnum string

func (expandViewEnum) Type() string                { return "df_expand_view" }
func (expandViewEnum) Options(cmd.Source) []string { return expandViews }

// adminExpandView picks how expansion mode shows the chunks in this save,
// to compare the views in game.
type adminExpandView struct {
	adminBase
	Sub  cmd.SubCommand `cmd:"expandview"`
	View expandViewEnum `cmd:"view"`
}

func (c adminExpandView) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		s.expandView = string(c.View)
		for e := range tx.Players() {
			p := e.(*player.Player)
			if ex, ok := s.expansionOf(p); ok {
				// Only the map lets the cursor go far from the body.
				if reach := int32(expandReach); abs32(ex.target[0]-ex.body[0]) > reach || abs32(ex.target[1]-ex.body[1]) > reach {
					ex.target = ex.body
				}
				s.drawFloor(p, ex)
				s.moveExpandCamera(p, ex, 0)
			}
		}
		o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.10"), c.View)
	}
}

// adminTPS turns the server load line at the top left of the operator's
// HUD on or off (perf.Toggle), in the lobby too. It is kept until the
// server restarts.
type adminTPS struct {
	adminBase
	Sub cmd.SubCommand `cmd:"tps"`
}

func (c adminTPS) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	on := perf.Toggle(p.Name())
	// In the lobby, platform.ShowLoad shows or clears it within a second.
	if s, ok := c.g.sessionOf(tx.World()); ok {
		s.updateBoard(p)
	}
	if on {
		o.Printf("%s", i18n.T(i18n.Of(p), "dragonfactory.admin.Run.11"))
	} else {
		o.Printf("%s", i18n.T(i18n.Of(p), "dragonfactory.admin.Run.12"))
	}
}

// researchEnum names a research of the tree or all, reset, free.
type researchEnum string

func (researchEnum) Type() string { return "df_research" }
func (researchEnum) Options(cmd.Source) []string {
	opts := []string{"all", "reset", "free"}
	for _, r := range researchTree {
		opts = append(opts, r.ID)
	}
	return opts
}

// adminResearch completes a research (or all), forgets every research
// (reset), or opens every recipe until the instance closes (free; again to
// lock).
type adminResearch struct {
	adminBase
	Sub      cmd.SubCommand `cmd:"research"`
	Research researchEnum   `cmd:"research"`
}

func (c adminResearch) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	_, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	switch c.Research {
	case "all":
		for i := range researchTree {
			if !s.researched(researchTree[i].ID) {
				s.completeResearch(tx, &researchTree[i])
			}
		}
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.13"))
	case "reset":
		s.state.Researched, s.state.ResearchPaid = nil, nil
		s.storeState()
		s.sendRecipes(tx)
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.14"))
	case "free":
		s.recipesFree = !s.recipesFree
		s.sendRecipes(tx)
		if s.recipesFree {
			o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.15"))
		} else {
			o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.16"))
		}
	default:
		r, ok := researchByID(string(c.Research))
		if !ok {
			o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.Run.17"), c.Research)
			return
		}
		if !s.researched(r.ID) {
			s.completeResearch(tx, r)
		}
		o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.18"), r.Title(i18n.Default))
	}
}

// advanceEnum names an advancement of the tree or all, reset.
type advanceEnum string

func (advanceEnum) Type() string { return "df_advancement" }
func (advanceEnum) Options(cmd.Source) []string {
	opts := []string{"all", "reset"}
	for _, a := range advancementTree {
		opts = append(opts, a.ID)
	}
	return opts
}

// adminAdvance completes an advancement (with the ones before it, or all)
// or forgets every advancement (reset).
type adminAdvance struct {
	adminBase
	Sub         cmd.SubCommand `cmd:"advance"`
	Advancement advanceEnum    `cmd:"advancement"`
}

func (c adminAdvance) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	_, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	switch c.Advancement {
	case "all":
		var as []*Advancement
		for i := range advancementTree {
			if !s.advanced(advancementTree[i].ID) {
				as = append(as, &advancementTree[i])
			}
		}
		if len(as) > 0 {
			s.completeAdvancements(tx, as)
		}
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.19"))
	case "reset":
		s.state.Advancements = nil
		s.storeState()
		s.updateBoards(tx)
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.20"))
	default:
		a, ok := advancementByID(string(c.Advancement))
		if !ok {
			o.Errorf(i18n.T(i18n.Default, "dragonfactory.admin.Run.21"), c.Advancement)
			return
		}
		s.advance(tx, a.ID)
		o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.22"), a.Name)
	}
}
