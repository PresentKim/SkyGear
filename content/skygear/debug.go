package skygear

import (
	"fmt"
	"github.com/presentkim/skygear/content/skygear/look"
	"strings"
	"time"

	"github.com/presentkim/skygear/content/skygear/wave"

	"github.com/presentkim/skygear/content/skygear/fairy"

	"github.com/presentkim/skygear/content/skygear/climate"

	"github.com/presentkim/skygear/content/skygear/storage"

	"github.com/presentkim/skygear/content/skygear/machine"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/timed"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Debugging tools (/sgdebug): helpers for scenarios and manual tests that
// make things happen now or report state. Like /sgadmin, only operators see
// and run them (adminBase).

func registerDebug(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("sgdebug", i18n.T(i18n.Default, "skygear.debug.register"), nil,
		debugUI{adminBase: b}, debugLook{adminBase: b}, debugStatus{adminBase: b}, debugSkip{adminBase: b}, debugTimed{adminBase: b}, debugHolderPose{adminBase: b}, debugFairy{adminBase: b}, debugWeather{adminBase: b}, debugSun{adminBase: b}, debugWave{adminBase: b}, debugResearchRandom{adminBase: b}, debugResearchLoad{adminBase: b}, debugFactory{adminBase: b}, debugDense{adminBase: b}, debugPlace{adminBase: b}, debugFloor{adminBase: b},
		debugResearchScreen{adminBase: b}, debugOreInput{adminBase: b}, debugOreRes{adminBase: b}))
}

type debugResearchScreen struct {
	adminBase
	Sub cmd.SubCommand `cmd:"research"`
}

// Run opens the research screen, as a research table does (the game menu has
// no research card).
func (c debugResearchScreen) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		s.showResearch(p)
	}
}

type debugSkip struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"skip"`
	Minutes int            `cmd:"minutes"`
}

// Run advances the instance time (days, crops, lost-and-found).
func (c debugSkip) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		for _, day := range s.Clock.Advance(time.Duration(max(0, c.Minutes)) * time.Minute) {
			s.NewDay(tx, day)
		}
		s.Second(tx)
		total, day, _ := s.Clock.Now()
		o.Printf(i18n.T(i18n.Default, "skygear.admin.time"), textfmt.PlayTime(i18n.Default, total), day)
	}
}

type debugLook struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"look"`
	Yaw   float64        `cmd:"yaw"`
	Pitch float64        `cmd:"pitch"`
}

// Run turns the operator's view (yaw 0 = south, pitch 90 = down) and
// updates the WAILA panel at once (tests the panel without aiming).
func (c debugLook) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		r := p.Rotation()
		p.Move(mgl64.Vec3{}, c.Yaw-r.Yaw(), c.Pitch-r.Pitch())
		// Move does not turn the client's own camera; a teleport to the same
		// spot sends it the rotation.
		p.Teleport(p.Position())
		look.Update(s.Session, p, tx, p.Position(), p.Rotation(), true)
		if at, b, ok := look.LookedAt(p, tx, p.Position(), p.Rotation()); ok {
			o.Printf(i18n.T(i18n.Default, "skygear.admin.lookingAt"), naming.BlockName(i18n.Default, b), at[0], at[1], at[2])
		} else {
			o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.lookingAtNone"))
		}
	}
}

type debugTimed struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"timed"`
	Seconds int            `cmd:"seconds"`
}

func (c debugTimed) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	main, off := p.HeldItems()
	if main.Empty() {
		o.Error(i18n.T(i18n.Default, "skygear.admin.noHeldItem"))
		return
	}
	if c.Seconds <= 0 {
		p.SetHeldItems(timed.Clear(main), off)
		o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.timed.off"))
		return
	}
	p.SetHeldItems(timed.For(main, time.Duration(c.Seconds)*time.Second), off)
	o.Printf(i18n.T(i18n.Default, "skygear.admin.timed.on"), timed.Describe(time.Duration(c.Seconds)*time.Second))
}

// debugHolderPose tries a pose for hand-equipped items (sticks, bones) on
// drawer faces without reloading the pack (holderPose, docs/tech/drawer.md).
type debugHolderPose struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"holderpose"`
	X     int               `cmd:"x"`
	Y     int               `cmd:"y"`
	Z     int               `cmd:"z"`
	Scale int               `cmd:"scale"`
	DX    cmd.Optional[int] `cmd:"dx"`
	DY    cmd.Optional[int] `cmd:"dy"`
	DZ    cmd.Optional[int] `cmd:"dz"`
}

func (c debugHolderPose) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	_, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	storage.LocalOf(s.Session).HolderPose = storage.NewHolderPose(c.X, c.Y, c.Z, c.Scale, c.DX.LoadOr(0), c.DY.LoadOr(0), c.DZ.LoadOr(0))
	for _, disp := range storage.LocalOf(s.Session).Displays {
		e, ok := disp.Item.Entity(tx)
		if !ok {
			continue
		}
		e.(*storage.ItemHolder).Data.Data.(*storage.HolderData).Pose = storage.LocalOf(s.Session).HolderPose
		for _, v := range tx.Viewers(e.Position()) {
			v.ViewEntityState(e)
		}
	}
	o.Printf(i18n.T(i18n.Default, "skygear.admin.holderPose"), c.X, c.Y, c.Z, c.Scale, c.DX.LoadOr(0), c.DY.LoadOr(0), c.DZ.LoadOr(0))
}

// debugFairy sends the gift fairy to the operator now.
type debugFairy struct {
	adminBase
	Sub cmd.SubCommand `cmd:"fairy"`
}

func (c debugFairy) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		fairy.Spawn(s.Session, tx, p)
		o.Printf("%s", i18n.T(i18n.Default, "skygear.admin.fairy"))
	}
}

// weatherEnum and sunEnum name the weathers and sun modes of the fairy
// board for /sgdebug weather and sun.
type weatherEnum string

func (weatherEnum) Type() string                { return "sg_weather" }
func (weatherEnum) Options(cmd.Source) []string { return []string{"clear", climate.Rain, climate.Snow} }

type sunEnum string

func (sunEnum) Type() string { return "sg_sun" }
func (sunEnum) Options(cmd.Source) []string {
	return []string{"flow", climate.SunDay, climate.SunDusk, climate.SunNight}
}

// debugWeather casts a weather without a fairy emerald.
type debugWeather struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"weather"`
	Weather weatherEnum    `cmd:"weather"`
}

func (c debugWeather) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		w := string(c.Weather)
		if w == "clear" {
			w = ""
		}
		climate.CastWeather(s.Session, tx, w)
		o.Printf(i18n.T(i18n.Default, "skygear.admin.weather"), i18n.T(i18n.Default, "fairy.weather."+climate.WeatherKey(w)))
	}
}

// debugSun sets the sun without a fairy emerald.
type debugSun struct {
	adminBase
	Sub cmd.SubCommand `cmd:"sun"`
	Sun sunEnum        `cmd:"sun"`
}

func (c debugSun) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		m := string(c.Sun)
		if m == "flow" {
			m = ""
		}
		climate.CastSun(s.Session, tx, m)
		o.Printf(i18n.T(i18n.Default, "skygear.admin.sun"), i18n.T(i18n.Default, "fairy.sun."+climate.SunKey(m)))
	}
}

type debugStatus struct {
	adminBase
	Sub cmd.SubCommand `cmd:"status"`
}

func (c debugStatus) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		o.Printf("researched=%d chunks=%d advanced=%d", len(progress.StateOf(s.Session).Researched), len(island.StateOf(s.Session).Chunks), len(progress.StateOf(s.Session).Advancements))
	}
}

// debugWave places a dimension gate beside the operator and starts a level of
// it, opening the levels below (for testing the waves).
type debugWave struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"wave"`
	Level int            `cmd:"level"`
}

func (c debugWave) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	if c.Level < 1 || c.Level > len(wave.Levels) {
		o.Errorf("level 1-%d", len(wave.Levels))
		return
	}
	pos := cube.PosFromVec3(p.Position()).Add(cube.Pos{2, 0, 0})
	if _, isGate := tx.Block(pos).(machine.Machine); !isGate {
		tx.SetBlock(pos, machine.New(machine.KindGate, "", ""), nil)
	}
	wave.StateOf(s.Session).GateTier = max(wave.StateOf(s.Session).GateTier, c.Level-1)
	if !wave.Start(s.Session, tx, pos, p.Position(), c.Level) {
		o.Error("a wave is already going on")
		return
	}
	o.Printf("wave %d started with %d monsters", c.Level, len(wave.LocalOf(s.Session).Wave.Left)+len(wave.LocalOf(s.Session).Wave.Pending))
}

// debugResearchRandom marks a share of the researches done, picked at random
// (percent, 50 by default; the order of the tree is ignored), to fill both
// tabs of the research screen with a mix.
type debugResearchRandom struct {
	adminBase
	Sub     cmd.SubCommand    `cmd:"research"`
	Random  cmd.SubCommand    `cmd:"random"`
	Percent cmd.Optional[int] `cmd:"percent"`
}

func (c debugResearchRandom) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	_, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	pct := max(0, min(100, c.Percent.LoadOr(50)))
	progress.StateOf(s.Session).Researched, progress.StateOf(s.Session).ResearchPaid = nil, nil
	n := 0
	for i := range progress.Tree {
		if s.Rand.IntN(100) < pct {
			progress.Complete(s.Session, tx, &progress.Tree[i])
			n++
		}
	}
	for _, r := range progress.LocalOf(s.Session).Copies {
		progress.LocalOf(s.Session).CopiesDone[r.ID] = s.Rand.IntN(100) < pct
	}
	s.MarkDirty()
	progress.SendRecipes(s.Session, tx)
	s.updateBoards(tx)
	o.Printf(i18n.T(i18n.Default, "skygear.admin.researchRandom"), n, len(progress.Tree), pct)
}

// debugResearchLoad makes count test copies of the researches (0 removes them)
// for trying the research screen with a long list; each is done or not at
// random. The copies are not saved and cost nothing.
type debugResearchLoad struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"research"`
	Load  cmd.SubCommand `cmd:"load"`
	Count int            `cmd:"count"`
}

func (c debugResearchLoad) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	_, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	progress.LocalOf(s.Session).Copies, progress.LocalOf(s.Session).CopiesDone = nil, map[string]bool{}
	for i := range max(0, min(c.Count, 2000)) {
		r := progress.Tree[i%len(progress.Tree)]
		r.ID = fmt.Sprintf("%s#%d", r.ID, i+1)
		r.Needs = nil
		progress.LocalOf(s.Session).Copies = append(progress.LocalOf(s.Session).Copies, &r)
		progress.LocalOf(s.Session).CopiesDone[r.ID] = s.Rand.IntN(2) == 0
	}
	o.Printf(i18n.T(i18n.Default, "skygear.admin.researchLoad"), len(progress.LocalOf(s.Session).Copies))
}

// debugPlace places a block dx, dy, dz from the operator's feet as the
// operator would by hand: holding the item (a /sgadmin give id, or any
// vanilla item name), it clicks the top of the block below that spot, so
// the game's own placing runs (machine state, facing, handlers), so tests
// build without aiming the camera.
type debugPlace struct {
	adminBase
	Sub  cmd.SubCommand `cmd:"place"`
	Item string         `cmd:"item"`
	DX   int            `cmd:"dx"`
	DY   int            `cmd:"dy"`
	DZ   int            `cmd:"dz"`
}

func (c debugPlace) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	var st item.Stack
	if f, ok := giveItems()[c.Item]; ok {
		st = f()
	} else {
		name := c.Item
		if !strings.Contains(name, ":") {
			name = "minecraft:" + name
		}
		it, ok := world.ItemByName(name, 0)
		if !ok {
			o.Errorf("unknown item %s", c.Item)
			return
		}
		st = item.NewStack(it, 1)
	}
	pos := cube.PosFromVec3(p.Position()).Add(cube.Pos{c.DX, c.DY, c.DZ})
	below := pos.Side(cube.FaceDown)
	if _, air := tx.Block(below).(block.Air); air {
		o.Errorf("nothing to place on at %v", below)
		return
	}
	held, off := p.HeldItems()
	p.SetHeldItems(st, off)
	p.UseItemOnBlock(below, cube.FaceUp, mgl64.Vec3{0.5, 1, 0.5})
	p.SetHeldItems(held, off)
	o.Printf("%s at %d %d %d", naming.BlockName(i18n.Default, tx.Block(pos)), pos[0], pos[1], pos[2])
}

// debugFloor lays stone around the operator at the height of the block under
// its feet, radius blocks out, without replacing what is there: a place to
// build on in a test (the island is small and the rest is void).
type debugFloor struct {
	adminBase
	Sub    cmd.SubCommand `cmd:"floor"`
	Radius int            `cmd:"radius"`
}

func (c debugFloor) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	r := max(1, min(c.Radius, 16))
	under := cube.PosFromVec3(p.Position()).Side(cube.FaceDown)
	n := 0
	for x := -r; x <= r; x++ {
		for z := -r; z <= r; z++ {
			pos := under.Add(cube.Pos{x, 0, z})
			if _, air := tx.Block(pos).(block.Air); air {
				tx.SetBlock(pos, block.Stone{}, nil)
				n++
			}
		}
	}
	o.Printf("stone x%d at y %d", n, under[1])
}
