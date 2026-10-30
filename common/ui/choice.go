package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// ChoiceBoard is a screen of settings to pick from: a strip of figures over
// groups of choices, the current choice of each group marked. More groups
// scroll the window down.
type ChoiceBoard struct {
	// Stats are the figures in the strip at the top.
	Stats []Stat
	// Note is a small line under the strip ("": none).
	Note string
	// Groups are the groups of choices, each under its title.
	Groups []ChoiceGroup
	// Actions are more buttons: tabs, back and help go where they go in
	// every window, the rest after the groups.
	Actions []Option
}

// Stat is a figure of a ChoiceBoard: an icon, a small label and its value.
type Stat struct {
	Icon, Label, Value string
}

// ChoiceGroup is a titled row of choices.
type ChoiceGroup struct {
	Title   string
	Choices []Choice
}

// Choice is a tile of a ChoiceBoard: its icon and name, with a line of what
// it does in the tooltip. On marks the current one (Badge in the tooltip)
// and does not press; Dim greys out one that cannot be had now (it still presses).
type Choice struct {
	Label, Info, Icon, Badge string
	On, Dim                  bool
	Run                      func(p *player.Player, tx *world.Tx)
}

// ChoiceMenu shows a ChoiceBoard. A press that shows nothing closes it, so a
// Run shows the board again.
func ChoiceMenu(p *player.Player, title string, b ChoiceBoard) {
	oreChoices(p, title, b)
}

// choiceData is the data of the choice screen.
type choiceData struct {
	Chrome dChrome        `json:"chrome"`
	Stats  []dStat        `json:"stats"`
	Note   string         `json:"note"`
	Groups []dChoiceGroup `json:"groups"`
	Items  []dItem        `json:"items"`
}

// dStat is a figure of the strip at the top.
type dStat struct {
	Icon  string `json:"icon"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// dChoiceGroup is a titled row of choice tiles.
type dChoiceGroup struct {
	Title   string    `json:"title"`
	Choices []dChoice `json:"choices"`
}

// dChoice is a choice tile; the current one has no id.
type dChoice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
	Tip   string `json:"tip"`
	On    bool   `json:"on"`
	Dim   bool   `json:"dim"`
}

// oreChoices is the DDUI ChoiceBoard: the figures, a row per group of
// choices, then the other buttons, sent as the choice screen.
func oreChoices(p *player.Player, title string, b ChoiceBoard) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, b.Actions)
	d := choiceData{Chrome: chrome, Stats: make([]dStat, 0, len(b.Stats)), Note: b.Note,
		Groups: make([]dChoiceGroup, 0, len(b.Groups)), Items: l.itemsOf(rest)}
	for _, s := range b.Stats {
		d.Stats = append(d.Stats, dStat{Icon: oreIcon(s.Icon), Label: s.Label, Value: s.Value})
	}
	for _, g := range b.Groups {
		dg := dChoiceGroup{Title: g.Title, Choices: make([]dChoice, 0, len(g.Choices))}
		for _, c := range g.Choices {
			dg.Choices = append(dg.Choices, l.choiceOf(c))
		}
		d.Groups = append(d.Groups, dg)
	}
	showScreen(p, l, "choice", d)
}

// choiceOf is the data of a tile of a ChoiceBoard: what it does in the
// tooltip; the current one does not press.
func (l *oreLayout) choiceOf(c Choice) dChoice {
	tip := c.Label
	if c.On && c.Badge != "" {
		tip += " §a" + c.Badge
	}
	if c.Info != "" {
		tip += "\n§7" + c.Info
	}
	d := dChoice{Label: c.Label, Icon: oreIcon(c.Icon), Tip: tip, On: c.On, Dim: c.Dim && !c.On}
	if !c.On {
		d.ID = l.id(c.Run)
	}
	return d
}
