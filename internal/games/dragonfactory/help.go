package dragonfactory

import (
	"fmt"
	"strings"

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
	// text, if set, writes the text instead of the catalog, for pages that
	// list what the game's tables hold.
	text func(lang string) string
}{
	{"start", func() string { return iconMission }, nil},
	{"goal", func() string { return iconOf(item.NewStack(block.DragonEgg{}, 1)) }, nil},
	{"handwork", func() string { return iconOf(item.NewStack(Pebble{Kind: PebbleStone}, 1)) }, nil},
	{"sieve", func() string { return iconOf(machineItem(KindSieve)) }, nil},
	{"water", func() string { return iconOf(item.NewStack(item.Bucket{}, 1)) }, nil},
	{"research", func() string { return iconOf(item.NewStack(ResearchTable{}, 1)) }, nil},
	{"essence", func() string { return iconOf(item.NewStack(ResearchEssence{Kind: EssenceEngineering}, 1)) }, helpEssenceText},
	{"quest", func() string { return iconMission }, nil},
	{"power", func() string { return iconOf(machineItem(KindGenerator)) }, nil},
	{"conduit", func() string { return iconOf(item.NewStack(item.RedstoneWire{}, 1)) }, nil},
	{"drawer", func() string { return iconOf(item.NewStack(Drawer{}, 1)) }, nil},
	{"metal", func() string { return iconOf(machineItem(KindAlloy)) }, nil},
	{"wave", func() string { return iconOf(machineItem(KindGate)) }, nil},
	{"gear", func() string { return iconOf(machineItem(KindTinker)) }, nil},
	{"magic", func() string { return iconOf(item.NewStack(Petal{Colour: "red"}, 1)) }, nil},
	{"lines", func() string { return iconOf(machineItem(KindMiner)) }, nil},
	{"monitor", func() string { return iconOf(machineItem(KindCrusher)) }, nil},
	{"tier", func() string { return iconOf(item.NewStack(TierKit{Level: 1}, 1)) }, nil},
	{"copier", func() string { return iconOf(item.NewStack(ConfigCopier{}, 1)) }, nil},
	{"magicsieve", func() string { return iconOf(machineItem(KindMagicSieve)) }, nil},
	{"island", func() string { return iconIsland }, nil},
}

// showHelp lists the help topics.
func (s *session) showHelp(p *player.Player) {
	lang := i18n.Of(p)
	opts := []ui.Option{s.opt("§e"+i18n.T(lang, "codex.title"), iconOf(item.NewStack(item.Book{}, 1)), func(p *player.Player, _ *world.Tx) { s.showCodex(p) })}
	for _, t := range helpTopics {
		title := i18n.T(lang, "help."+t.id+".title")
		text := i18n.T(lang, "help."+t.id+".text")
		if t.text != nil {
			text = t.text(lang)
		}
		opts = append(opts, s.opt("§f"+title, t.icon(), func(p *player.Player, _ *world.Tx) {
			ui.Menu(p, i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.1")+title, text, s.opt(i18n.T(i18n.Of(p), "dragonfactory.help.showHelp.2"), iconBack, func(p *player.Player, _ *world.Tx) { s.showHelp(p) }))
		}))
	}
	ui.CardMenu(p, i18n.T(lang, "dragonfactory.help.showHelp.3"), i18n.T(lang, "dragonfactory.help.showHelp.4"), append(opts, s.backOption())...)
}

// helpEssenceText is the help page of the research essences: what they are
// for and the recipes of the laboratory, written from its table.
func helpEssenceText(lang string) string {
	lines := []string{i18n.T(lang, "help.essence.text"), ""}
	for i := range labRecipes {
		r := &labRecipes[i]
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "help.essence.recipe"), itemName(item.NewStack(r.Out, 1)), labIngredientsText(r), r.OutCount))
	}
	return strings.Join(append(lines, "", i18n.T(lang, "help.essence.foot")), "\n")
}
