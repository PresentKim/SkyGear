// Package dragonfactory is the factory building game (docs/design/02-dragonfactory.md).
// M1 provides only the starting island; cores, missions and the shop follow in M2.
package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Game is the dragonfactory game type.
type Game struct{}

// IslandY is the Y level of the island surface.
const IslandY = 64

func (Game) ID() string   { return "dragonfactory" }
func (Game) Name() string { return "드래곤팩토리" }

// Init builds the starting island: a small platform around the origin.
func (Game) Init(tx *world.Tx, _ *instance.Instance) {
	for x := -2; x <= 2; x++ {
		for z := -2; z <= 2; z++ {
			tx.SetBlock(cube.Pos{x, IslandY - 1, z}, block.Grass{}, nil)
			tx.SetBlock(cube.Pos{x, IslandY - 2, z}, block.Dirt{}, nil)
		}
	}
}

// Start begins a session for an activated save.
func (Game) Start(inst *instance.Instance) instance.Session { return &session{inst: inst} }

type session struct{ inst *instance.Instance }

func (*session) Spawn() mgl64.Vec3 { return mgl64.Vec3{0.5, IslandY, 0.5} }

func (s *session) Join(p *player.Player, first bool) {
	_, day, _ := s.inst.Clock.Now()
	p.Messagef("§a'%s'에 입장했습니다. §7%d일차", s.inst.Save.Name, day)
	if first {
		_, _ = p.Inventory().AddItem(item.NewStack(item.Pickaxe{Tier: item.ToolTierWood}, 1))
		_, _ = p.Inventory().AddItem(item.NewStack(item.Axe{Tier: item.ToolTierWood}, 1))
	}
}

func (*session) Leave(*player.Player)                  {}
func (*session) Handler(*player.Player) player.Handler { return nil }
func (*session) Close(*world.Tx)                       {}

func (*session) NewDay(tx *world.Tx, day int) {
	for e := range tx.Players() {
		e.(*player.Player).Message(fmt.Sprintf("§e%d일차가 시작되었습니다.", day))
	}
}
