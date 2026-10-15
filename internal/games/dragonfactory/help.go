package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// helpTopics are the in-game help pages (menu → 도움말), in display order.
// Their texts are "help.<id>.title" and "help.<id>.text" in the catalog. The
// icon is made when the list opens, once the items are registered.
var helpTopics = []struct {
	id   string
	icon func() string
}{
	{"start", func() string { return iconMission }},
	{"goal", func() string { return iconOf(item.NewStack(block.DragonEgg{}, 1)) }},
	{"handwork", func() string { return iconOf(item.NewStack(Pebble{Kind: PebbleStone}, 1)) }},
	{"sieve", func() string { return iconOf(machineItem(KindSieve)) }},
	{"water", func() string { return iconOf(item.NewStack(item.Bucket{}, 1)) }},
	{"research", func() string { return iconOf(item.NewStack(ResearchTable{}, 1)) }},
	{"quest", func() string { return iconMission }},
	{"power", func() string { return iconOf(machineItem(KindGenerator)) }},
	{"conduit", func() string { return iconOf(item.NewStack(item.RedstoneWire{}, 1)) }},
	{"drawer", func() string { return iconOf(item.NewStack(Drawer{}, 1)) }},
	{"metal", func() string { return iconOf(machineItem(KindAlloy)) }},
	{"wave", func() string { return iconOf(machineItem(KindGate)) }},
	{"gear", func() string { return iconOf(machineItem(KindTinker)) }},
	{"magic", func() string { return iconOf(item.NewStack(Petal{Colour: "red"}, 1)) }},
	{"lines", func() string { return iconOf(machineItem(KindMiner)) }},
	{"island", func() string { return iconIsland }},
}

// showHelp lists the help topics.
func (s *session) showHelp(p *player.Player) {
	lang := i18n.Of(p)
	opts := []ui.Option{s.opt("§e"+i18n.T(lang, "codex.title"), iconOf(item.NewStack(item.Book{}, 1)), func(p *player.Player, _ *world.Tx) { s.showCodex(p) })}
	for _, t := range helpTopics {
		title := i18n.T(lang, "help."+t.id+".title")
		text := i18n.T(lang, "help."+t.id+".text")
		opts = append(opts, s.opt("§f"+title, t.icon(), func(p *player.Player, _ *world.Tx) {
			ui.Menu(p, i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.1")+title, text, s.opt(i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.2"), iconBack, func(p *player.Player, _ *world.Tx) { s.showHelp(p) }))
		}))
	}
	ui.CardMenu(p, i18n.T(lang, "dragonfactory.help.showHelp.3"), i18n.T(lang, "dragonfactory.help.showHelp.4"), append(opts, s.backOption())...)
}
