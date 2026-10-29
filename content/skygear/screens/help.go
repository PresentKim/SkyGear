package screens

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// HelpTab is a tab of the help window: the topics of one stage of the game,
// in the order a new player meets them.
type HelpTab int

const (
	helpStartTab HelpTab = iota
	helpHandTab
	helpAutoTab
	helpMetalTab
	helpFightTab
	HelpTabCount
)

var HelpTabKeys = [HelpTabCount]string{"help.tab.start", "help.tab.hand", "help.tab.auto", "help.tab.metal", "help.tab.fight"}

func helpTabIcon(t HelpTab) string {
	switch t {
	case helpHandTab:
		return formicon.Of(item.NewStack(sgitem.Pebble{Kind: sgitem.PebbleStone}, 1))
	case helpAutoTab:
		return formicon.Of(machine.Item(machine.KindGenerator))
	case helpMetalTab:
		return formicon.Of(machine.Item(machine.KindAlloy))
	case helpFightTab:
		return formicon.Of(machine.Item(machine.KindGate))
	}
	return formicon.Mission
}

// HelpTopics are the in-game help pages (menu → 도움말, /도움말), by tab and
// in display order within it. Their texts are "help.<id>.title" and
// "help.<id>.text" in the catalog. The icon is made when the list opens, once
// the items are registered.
var HelpTopics = []struct {
	id   string
	tab  HelpTab
	icon func() string
	// text, if set, writes the text instead of the catalog, for pages that
	// list what the game's tables hold.
	text func(lang string) string
}{
	{"start", helpStartTab, func() string { return formicon.Mission }, nil},
	{"goal", helpStartTab, func() string { return formicon.Of(item.NewStack(block.DragonEgg{}, 1)) }, nil},
	{"quest", helpStartTab, func() string { return formicon.Mission }, nil},
	{"research", helpStartTab, func() string { return formicon.Of(item.NewStack(progress.ResearchTable{}, 1)) }, nil},
	{"island", helpStartTab, func() string { return formicon.Island }, nil},
	{"handwork", helpHandTab, func() string { return formicon.Of(item.NewStack(sgitem.Pebble{Kind: sgitem.PebbleStone}, 1)) }, nil},
	{"sieve", helpHandTab, func() string { return formicon.Of(machine.Item(machine.KindSieve)) }, nil},
	{"water", helpHandTab, func() string { return formicon.Of(item.NewStack(item.Bucket{}, 1)) }, nil},
	{"fairy", helpHandTab, func() string { return formicon.Of(item.NewStack(fairy.Board{}, 1)) }, nil},
	{"power", helpAutoTab, func() string { return formicon.Of(machine.Item(machine.KindGenerator)) }, nil},
	{"conduit", helpAutoTab, func() string { return formicon.Of(item.NewStack(item.RedstoneWire{}, 1)) }, nil},
	{"drawer", helpAutoTab, func() string { return formicon.Of(item.NewStack(storage.Drawer{}, 1)) }, nil},
	{"lines", helpAutoTab, func() string { return formicon.Of(machine.Item(machine.KindMiner)) }, nil},
	{"monitor", helpAutoTab, func() string { return formicon.Of(machine.Item(machine.KindCrusher)) }, nil},
	{"copier", helpAutoTab, func() string { return formicon.Of(item.NewStack(sgitem.ConfigCopier{}, 1)) }, nil},
	{"essence", helpAutoTab, func() string {
		return formicon.Of(item.NewStack(sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}, 1))
	}, helpEssenceText},
	{"fluid", helpAutoTab, func() string { return formicon.Of(machine.Item(machine.KindTank)) }, nil},
	{"metal", helpMetalTab, func() string { return formicon.Of(machine.Item(machine.KindAlloy)) }, nil},
	{"tier", helpMetalTab, func() string { return formicon.Of(item.NewStack(sgitem.TierKit{Level: 1}, 1)) }, nil},
	{"wave", helpFightTab, func() string { return formicon.Of(machine.Item(machine.KindGate)) }, nil},
	{"gear", helpFightTab, func() string { return formicon.Of(machine.Item(machine.KindTinker)) }, nil},
	{"magic", helpFightTab, func() string { return formicon.Of(item.NewStack(sgitem.Petal{Colour: "red"}, 1)) }, nil},
	{"magicsieve", helpFightTab, func() string { return formicon.Of(machine.Item(machine.KindMagicSieve)) }, nil},
}

// HelpTopicsOf are the indexes in helpTopics of the topics of a tab.
func HelpTopicsOf(t HelpTab) []int {
	var out []int
	for i, h := range HelpTopics {
		if h.tab == t {
			out = append(out, i)
		}
	}
	return out
}

// ShowHelp opens the help window on the tab p looked at last in this save,
// the first one the first time.
func ShowHelp(s *core.Session, p *player.Player) {
	showHelpTab(s, p, lastHelpTab.Of(s)[p.H()])
}

// helpTabs is the tab strip with cur selected.
func helpTabs(s *core.Session, p *player.Player, cur HelpTab) []ui.Option {
	lang := i18n.Of(p)
	var out []ui.Option
	for t := HelpTab(0); t < HelpTabCount; t++ {
		t := t
		o := ui.Tab(i18n.T(lang, HelpTabKeys[t]), helpTabIcon(t), t == cur, func(p *player.Player, _ *world.Tx) { showHelpTab(s, p, t) })
		out = append(out, s.Opt(o.Label, o.Icon, o.Run))
	}
	return out
}

// showHelpTab lists the topics of a tab and remembers it as p's tab.
func showHelpTab(s *core.Session, p *player.Player, t HelpTab) {
	lastHelpTab.Of(s)[p.H()] = t
	lang := i18n.Of(p)
	var opts []ui.Option
	for _, i := range HelpTopicsOf(t) {
		h := HelpTopics[i]
		opts = append(opts, s.Opt("§f"+i18n.T(lang, "help."+h.id+".title"), h.icon(), func(p *player.Player, _ *world.Tx) { showHelpTopic(s, p, i) }))
	}
	opts = append(opts, helpTabs(s, p, t)...)
	ui.CardMenu(p, i18n.T(lang, "help.title"), i18n.T(lang, "help.body"), append(opts, s.BackOption(lang))...)
}

// showHelpTopic shows a help page with the topics before and after it in its
// tab, so a new player can read a tab through, and back to the tab.
func showHelpTopic(s *core.Session, p *player.Player, i int) {
	lang := i18n.Of(p)
	h := HelpTopics[i]
	text := i18n.T(lang, "help."+h.id+".text")
	if h.text != nil {
		text = h.text(lang)
	}
	var opts []ui.Option
	in := HelpTopicsOf(h.tab)
	at := slices.Index(in, i)
	if at > 0 {
		prev := HelpTopics[in[at-1]]
		opts = append(opts, s.Opt(fmt.Sprintf(i18n.T(lang, "help.prev"), i18n.T(lang, "help."+prev.id+".title")), prev.icon(), func(p *player.Player, _ *world.Tx) { showHelpTopic(s, p, in[at-1]) }))
	}
	if at < len(in)-1 {
		next := HelpTopics[in[at+1]]
		opts = append(opts, s.Opt(fmt.Sprintf(i18n.T(lang, "help.next"), i18n.T(lang, "help."+next.id+".title")), next.icon(), func(p *player.Player, _ *world.Tx) { showHelpTopic(s, p, in[at+1]) }))
	}
	back := ui.Back(i18n.T(lang, "help.title"), func(p *player.Player, _ *world.Tx) { showHelpTab(s, p, h.tab) })
	opts = append(opts, s.Opt(back.Label, back.Icon, back.Run))
	title := ui.SubTitle(i18n.T(lang, "help.title"), i18n.T(lang, "help."+h.id+".title"))
	ui.Menu(p, title, text, opts...)
}

// helpEssenceText is the help page of the research essences: what they are
// for and the recipes of the laboratory, written from its table.
func helpEssenceText(lang string) string {
	lines := []string{i18n.T(lang, "help.essence.text"), ""}
	for i := range machinerecipe.Labs {
		r := &machinerecipe.Labs[i]
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "help.essence.recipe"), naming.UIItemName(lang, item.NewStack(r.Out, 1)), machinerecipe.LabIngredientsText(lang, r), r.OutCount))
	}
	return strings.Join(append(lines, "", i18n.T(lang, "help.essence.foot")), "\n")
}

// lastHelpTab is the help tab each player looked at last.
var lastHelpTab = core.NewPlayers[HelpTab]()
