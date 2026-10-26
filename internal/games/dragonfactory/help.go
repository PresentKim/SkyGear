package dragonfactory

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/formicon"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/machinerecipe"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
)

// helpTab is a tab of the help window: the topics of one stage of the game,
// in the order a new player meets them (docs/design/10-user-flow.md).
type helpTab int

const (
	helpStartTab helpTab = iota
	helpHandTab
	helpAutoTab
	helpMetalTab
	helpFightTab
	helpTabCount
)

// helpTabKeys are the label keys of the tabs.
var helpTabKeys = [helpTabCount]string{"help.tab.start", "help.tab.hand", "help.tab.auto", "help.tab.metal", "help.tab.fight"}

// helpTabIcon is the icon of a tab.
func helpTabIcon(t helpTab) string {
	switch t {
	case helpHandTab:
		return formicon.Of(item.NewStack(dfitem.Pebble{Kind: dfitem.PebbleStone}, 1))
	case helpAutoTab:
		return formicon.Of(machineItem(KindGenerator))
	case helpMetalTab:
		return formicon.Of(machineItem(KindAlloy))
	case helpFightTab:
		return formicon.Of(machineItem(KindGate))
	}
	return formicon.Mission
}

// helpTopics are the in-game help pages (menu → 도움말, /도움말), by tab and
// in display order within it. Their texts are "help.<id>.title" and
// "help.<id>.text" in the catalog. The icon is made when the list opens, once
// the items are registered.
var helpTopics = []struct {
	id   string
	tab  helpTab
	icon func() string
	// text, if set, writes the text instead of the catalog, for pages that
	// list what the game's tables hold.
	text func(lang string) string
}{
	{"start", helpStartTab, func() string { return formicon.Mission }, nil},
	{"goal", helpStartTab, func() string { return formicon.Of(item.NewStack(block.DragonEgg{}, 1)) }, nil},
	{"quest", helpStartTab, func() string { return formicon.Mission }, nil},
	{"research", helpStartTab, func() string { return formicon.Of(item.NewStack(ResearchTable{}, 1)) }, nil},
	{"island", helpStartTab, func() string { return formicon.Island }, nil},
	{"handwork", helpHandTab, func() string { return formicon.Of(item.NewStack(dfitem.Pebble{Kind: dfitem.PebbleStone}, 1)) }, nil},
	{"sieve", helpHandTab, func() string { return formicon.Of(machineItem(KindSieve)) }, nil},
	{"water", helpHandTab, func() string { return formicon.Of(item.NewStack(item.Bucket{}, 1)) }, nil},
	{"fairy", helpHandTab, func() string { return formicon.Of(item.NewStack(FairyBoard{}, 1)) }, nil},
	{"power", helpAutoTab, func() string { return formicon.Of(machineItem(KindGenerator)) }, nil},
	{"conduit", helpAutoTab, func() string { return formicon.Of(item.NewStack(item.RedstoneWire{}, 1)) }, nil},
	{"drawer", helpAutoTab, func() string { return formicon.Of(item.NewStack(Drawer{}, 1)) }, nil},
	{"lines", helpAutoTab, func() string { return formicon.Of(machineItem(KindMiner)) }, nil},
	{"monitor", helpAutoTab, func() string { return formicon.Of(machineItem(KindCrusher)) }, nil},
	{"copier", helpAutoTab, func() string { return formicon.Of(item.NewStack(dfitem.ConfigCopier{}, 1)) }, nil},
	{"essence", helpAutoTab, func() string {
		return formicon.Of(item.NewStack(dfitem.ResearchEssence{Kind: dfitem.EssenceEngineering}, 1))
	}, helpEssenceText},
	{"fluid", helpAutoTab, func() string { return formicon.Of(machineItem(KindTank)) }, nil},
	{"metal", helpMetalTab, func() string { return formicon.Of(machineItem(KindAlloy)) }, nil},
	{"tier", helpMetalTab, func() string { return formicon.Of(item.NewStack(dfitem.TierKit{Level: 1}, 1)) }, nil},
	{"wave", helpFightTab, func() string { return formicon.Of(machineItem(KindGate)) }, nil},
	{"gear", helpFightTab, func() string { return formicon.Of(machineItem(KindTinker)) }, nil},
	{"magic", helpFightTab, func() string { return formicon.Of(item.NewStack(dfitem.Petal{Colour: "red"}, 1)) }, nil},
	{"magicsieve", helpFightTab, func() string { return formicon.Of(machineItem(KindMagicSieve)) }, nil},
}

// helpTopicsOf are the indexes in helpTopics of the topics of a tab.
func helpTopicsOf(t helpTab) []int {
	var out []int
	for i, h := range helpTopics {
		if h.tab == t {
			out = append(out, i)
		}
	}
	return out
}

// showHelp opens the help window on the tab p looked at last in this save,
// the first one the first time.
func (s *session) showHelp(p *player.Player) {
	s.showHelpTab(p, s.helpTab[p.H()])
}

// helpTabs is the tab strip with cur selected.
func (s *session) helpTabs(p *player.Player, cur helpTab) []ui.Option {
	lang := i18n.Of(p)
	var out []ui.Option
	for t := helpTab(0); t < helpTabCount; t++ {
		t := t
		o := ui.Tab(i18n.T(lang, helpTabKeys[t]), helpTabIcon(t), t == cur, func(p *player.Player, _ *world.Tx) { s.showHelpTab(p, t) })
		out = append(out, s.opt(o.Label, o.Icon, o.Run))
	}
	return out
}

// showHelpTab lists the topics of a tab and remembers it as p's tab.
func (s *session) showHelpTab(p *player.Player, t helpTab) {
	if s.helpTab == nil {
		s.helpTab = map[*world.EntityHandle]helpTab{}
	}
	s.helpTab[p.H()] = t
	lang := i18n.Of(p)
	var opts []ui.Option
	for _, i := range helpTopicsOf(t) {
		h := helpTopics[i]
		opts = append(opts, s.opt("§f"+i18n.T(lang, "help."+h.id+".title"), h.icon(), func(p *player.Player, _ *world.Tx) { s.showHelpTopic(p, i) }))
	}
	opts = append(opts, s.helpTabs(p, t)...)
	ui.CardMenu(p, i18n.T(lang, "help.title"), i18n.T(lang, "help.body"), append(opts, s.backOption())...)
}

// showHelpTopic shows a help page with the topics before and after it in its
// tab, so a new player can read a tab through, and back to the tab.
func (s *session) showHelpTopic(p *player.Player, i int) {
	lang := i18n.Of(p)
	h := helpTopics[i]
	text := i18n.T(lang, "help."+h.id+".text")
	if h.text != nil {
		text = h.text(lang)
	}
	var opts []ui.Option
	in := helpTopicsOf(h.tab)
	at := slices.Index(in, i)
	if at > 0 {
		prev := helpTopics[in[at-1]]
		opts = append(opts, s.opt(fmt.Sprintf(i18n.T(lang, "help.prev"), i18n.T(lang, "help."+prev.id+".title")), prev.icon(), func(p *player.Player, _ *world.Tx) { s.showHelpTopic(p, in[at-1]) }))
	}
	if at < len(in)-1 {
		next := helpTopics[in[at+1]]
		opts = append(opts, s.opt(fmt.Sprintf(i18n.T(lang, "help.next"), i18n.T(lang, "help."+next.id+".title")), next.icon(), func(p *player.Player, _ *world.Tx) { s.showHelpTopic(p, in[at+1]) }))
	}
	back := ui.Back(i18n.T(lang, "help.title"), func(p *player.Player, _ *world.Tx) { s.showHelpTab(p, h.tab) })
	opts = append(opts, s.opt(back.Label, back.Icon, back.Run))
	title := ui.SubTitle(i18n.T(lang, "help.title"), i18n.T(lang, "help."+h.id+".title"))
	ui.Menu(p, title, text, opts...)
}

// helpEssenceText is the help page of the research essences: what they are
// for and the recipes of the laboratory, written from its table.
func helpEssenceText(lang string) string {
	lines := []string{i18n.T(lang, "help.essence.text"), ""}
	for i := range machinerecipe.Labs {
		r := &machinerecipe.Labs[i]
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "help.essence.recipe"), naming.ItemName(lang, item.NewStack(r.Out, 1)), machinerecipe.LabIngredientsText(lang, r), r.OutCount))
	}
	return strings.Join(append(lines, "", i18n.T(lang, "help.essence.foot")), "\n")
}
