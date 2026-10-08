package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
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

type adminLevel struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"level"`
	Level int            `cmd:"level"`
}

func (c adminLevel) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		s.state.Level = max(1, min(c.Level, len(Levels)+1))
		s.state.Delivered = map[string]int{}
		s.updateBoards(tx)
		o.Printf("레벨을 %d(으)로 바꿨습니다.", s.state.Level)
	}
}

type adminMoney struct {
	adminBase
	Sub    cmd.SubCommand `cmd:"money"`
	Amount int            `cmd:"amount"`
}

func (c adminMoney) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		s.AddMoney(p, int64(c.Amount))
		o.Printf("소지금: %s", formatMoney(s.Money(p)))
	}
}

type machineEnum string

func (machineEnum) Type() string { return "machine" }
func (machineEnum) Options(cmd.Source) []string {
	return machineOrder
}

type adminGive struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"give"`
	Kind  machineEnum       `cmd:"machine"`
	Count cmd.Optional[int] `cmd:"count"`
}

func (c adminGive) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, _, ok := c.session(src, o, tx); ok {
		n := c.Count.LoadOr(1)
		_, _ = p.Inventory().AddItem(machineItem(string(c.Kind)).Grow(n - 1))
		o.Printf("%s x%d", machineKinds[string(c.Kind)].name, n)
	}
}

// adminLine builds the sample hands-free line of docs/tech/machines.md on
// the start core: auto miner above the core's output, crusher east of it
// with a hopper into a chest below, sell box west of it.
type adminLine struct {
	adminBase
	Sub cmd.SubCommand `cmd:"line"`
}

func (c adminLine) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	minerPos := startCorePos.Add(cube.Pos{0, 2, 0})
	crusherPos, sellPos := minerPos.Side(cube.FaceEast), minerPos.Side(cube.FaceWest)
	h := block.NewHopper()
	h.Facing = cube.FaceDown
	tx.SetBlock(crusherPos.Side(cube.FaceDown), h, nil)
	tx.SetBlock(crusherPos.Add(cube.Pos{0, -2, 0}), block.NewChest(), nil)
	tx.SetBlock(minerPos, newMachine(KindMiner, "", ""), nil)
	tx.SetBlock(crusherPos, newMachine(KindCrusher, "", ""), nil)
	tx.SetBlock(sellPos, newMachine(KindSellBox, instance.AccountID(p), p.Name()), nil)
	o.Printf("생산 라인을 만들었습니다.")
}

type adminStatus struct {
	adminBase
	Sub cmd.SubCommand `cmd:"status"`
}

func (c adminStatus) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		bb := s.buyback()
		sold := 0
		for _, n := range bb.Sold {
			sold += n
		}
		o.Printf("level=%d money=%d budget=%d sold=%d", s.state.Level, s.Money(p), bb.Budget, sold)
	}
}

func registerAdmin(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("dfadmin", "드래곤팩토리 운영자 도구", nil,
		adminLevel{adminBase: b}, adminMoney{adminBase: b}, adminGive{adminBase: b}, adminLine{adminBase: b}, adminStatus{adminBase: b}))
}
