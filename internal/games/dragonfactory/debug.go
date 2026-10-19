package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/timed"
)

// Debugging tools (/dfdebug): helpers for scenarios and playtests that make
// things happen now or report state. Like /dfadmin, only operators see and
// run them (adminBase). docs/tech/admin.md.

func registerDebug(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("dfdebug", i18n.T(i18n.Default, "dragonfactory.debug.register"), nil,
		debugUI{adminBase: b}, debugLook{adminBase: b}, debugStatus{adminBase: b}, debugSkip{adminBase: b}, debugTimed{adminBase: b}, debugFairy{adminBase: b}, debugWave{adminBase: b}, debugResearchRandom{adminBase: b}, debugResearchLoad{adminBase: b}, debugFactory{adminBase: b}, debugDense{adminBase: b}))
}

type debugSkip struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"skip"`
	Minutes int            `cmd:"minutes"`
}

// Run advances the instance time (days, crops, lost-and-found).
func (c debugSkip) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		for _, day := range s.clock.Advance(time.Duration(max(0, c.Minutes)) * time.Minute) {
			s.NewDay(tx, day)
		}
		s.Second(tx)
		total, day, _ := s.clock.Now()
		o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.2"), formatPlayTime(total), day)
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
		s.updateLook(p, tx, p.Position(), p.Rotation(), true)
		if at, b, ok := lookedAt(p, tx, p.Position(), p.Rotation()); ok {
			o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.Run.3"), blockName(b), at[0], at[1], at[2])
		} else {
			o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.4"))
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

// debugFairy sends the gift fairy to the operator now.
type debugFairy struct {
	adminBase
	Sub cmd.SubCommand `cmd:"fairy"`
}

func (c debugFairy) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		s.spawnFairy(tx, p)
		o.Printf("%s", i18n.T(i18n.Default, "dragonfactory.admin.Run.8"))
	}
}

type debugStatus struct {
	adminBase
	Sub cmd.SubCommand `cmd:"status"`
}

func (c debugStatus) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		o.Printf("researched=%d chunks=%d advanced=%d", len(s.state.Researched), len(s.state.Chunks), len(s.state.Advancements))
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
	s.state.Researched, s.state.ResearchPaid = nil, nil
	n := 0
	for i := range researchTree {
		if s.rand.IntN(100) < pct {
			s.completeResearch(tx, &researchTree[i])
			n++
		}
	}
	for _, r := range s.copies {
		s.copiesDone[r.ID] = s.rand.IntN(100) < pct
	}
	s.storeState()
	s.sendRecipes(tx)
	s.updateBoards(tx)
	o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.researchRandom"), n, len(researchTree), pct)
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
	s.copies, s.copiesDone = nil, map[string]bool{}
	for i := range max(0, min(c.Count, 2000)) {
		r := researchTree[i%len(researchTree)]
		r.ID = fmt.Sprintf("%s#%d", r.ID, i+1)
		r.Needs = nil
		s.copies = append(s.copies, &r)
		s.copiesDone[r.ID] = s.rand.IntN(2) == 0
	}
	o.Printf(i18n.T(i18n.Default, "dragonfactory.admin.researchLoad"), len(s.copies))
}
