package dragonfactory

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

const menuItemKey = "dragonfactory_menu"

// menuItem opens the game menu when used.
func menuItem() item.Stack {
	return item.NewStack(item.NetherStar{}, 1).WithCustomName("§r§6드래곤팩토리 메뉴").WithValue(menuItemKey, true)
}

func isMenuItem(s item.Stack) bool {
	_, ok := s.Value(menuItemKey)
	return ok
}

func registerCommands(g *Game) {
	cmd.Register(cmd.New("menu", "드래곤팩토리 메뉴를 엽니다", nil, menuCmd{g: g}))
	// A separate command rather than an alias: clients check typed commands
	// against the command list, where aliases may not be accepted.
	cmd.Register(cmd.New("df", "드래곤팩토리 메뉴를 엽니다", nil, menuCmd{g: g}))
}

type menuCmd struct{ g *Game }

func (c menuCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error("세이브에 입장한 뒤 사용할 수 있습니다.")
		return
	}
	s.showMenu(p)
}

// itemName returns a readable name of an item stack.
func itemName(s item.Stack) string {
	if n := s.CustomName(); n != "" {
		return n
	}
	if c, ok := s.Item().(world.CustomItem); ok {
		return c.Name()
	}
	name, _ := s.Item().EncodeItem()
	return strings.TrimPrefix(name, "minecraft:")
}
