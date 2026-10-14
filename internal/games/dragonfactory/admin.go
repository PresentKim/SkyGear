package dragonfactory

import (
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
		o.Error("세이브 안에서 사용하세요.")
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
		o.Errorf("알 수 없는 아이템: %s", name)
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
		o.Printf("인스턴스 시간 %s · %d일차", formatPlayTime(total), day)
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
			o.Printf("보는 블록: %s (%d, %d, %d)", blockName(b), at[0], at[1], at[2])
		} else {
			o.Printf("보는 블록 없음")
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
		o.Error("손에 든 아이템이 없습니다.")
		return
	}
	if c.Seconds <= 0 {
		p.SetHeldItems(timed.Clear(main), off)
		o.Printf("기간제를 해제했습니다.")
		return
	}
	p.SetHeldItems(timed.For(main, time.Duration(c.Seconds)*time.Second), off)
	o.Printf("기간제로 바꿨습니다: %s 남음", timed.Describe(time.Duration(c.Seconds)*time.Second))
}

// adminFairy sends the gift fairy to the operator now.
type adminFairy struct {
	adminBase
	Sub cmd.SubCommand `cmd:"fairy"`
}

func (c adminFairy) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		s.spawnFairy(tx, p)
		o.Printf("요정을 불렀습니다.")
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
		o.Errorf("알 수 없는 아이템: %s", c.Item)
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

func registerAdmin(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("dfadmin", "드래곤팩토리 운영자 도구", nil,
		adminItem{adminBase: b}, adminSkip{adminBase: b}, adminGive{adminBase: b}, adminStatus{adminBase: b}, adminTimed{adminBase: b}, adminFairy{adminBase: b}, adminLook{adminBase: b}, adminFactory{adminBase: b}, adminDense{adminBase: b}, adminExpandView{adminBase: b}, adminTPS{adminBase: b}, adminResearch{adminBase: b}, adminAdvance{adminBase: b}))
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
		o.Printf("섬 확장 보기를 %s(으)로 바꿨습니다.", c.View)
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
		o.Printf("서버 부하 표시를 켰습니다(왼쪽 위).")
	} else {
		o.Printf("서버 부하 표시를 껐습니다.")
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
		o.Printf("모든 연구를 완료했습니다.")
	case "reset":
		s.state.Researched, s.state.ResearchPaid = nil, nil
		s.storeState()
		s.sendRecipes(tx)
		o.Printf("연구를 모두 지웠습니다.")
	case "free":
		s.recipesFree = !s.recipesFree
		s.sendRecipes(tx)
		if s.recipesFree {
			o.Printf("모든 조합법을 열었습니다(인스턴스가 닫힐 때까지).")
		} else {
			o.Printf("조합법을 다시 연구대로 잠갔습니다.")
		}
	default:
		r, ok := researchByID(string(c.Research))
		if !ok {
			o.Errorf("알 수 없는 연구: %s", c.Research)
			return
		}
		if !s.researched(r.ID) {
			s.completeResearch(tx, r)
		}
		o.Printf("%s 연구를 완료했습니다.", r.Title(i18n.Default))
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
		o.Printf("모든 발전과제를 달성했습니다.")
	case "reset":
		s.state.Advancements = nil
		s.storeState()
		s.updateBoards(tx)
		o.Printf("발전과제를 모두 지웠습니다.")
	default:
		a, ok := advancementByID(string(c.Advancement))
		if !ok {
			o.Errorf("알 수 없는 발전과제: %s", c.Advancement)
			return
		}
		s.advance(tx, a.ID)
		o.Printf("%s 발전과제를 달성했습니다.", a.Name)
	}
}
