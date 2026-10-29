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

// oreChoices is the DDUI ChoiceBoard: the figures in a panel, an icon and a
// value each (the label is the tooltip), then a row per group, its title
// beside its tiles (oui-mc.css .mc-pick*).
func oreChoices(p *player.Player, title string, b ChoiceBoard) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(b.Actions)
	var content []any
	if len(b.Stats) > 0 {
		var stats []any
		for _, s := range b.Stats {
			ch := []any{}
			if s.Icon != "" {
				ch = append(ch, img("mc-stat-icon", s.Icon))
			}
			ch = append(ch, txt("mc-stat-value", s.Value))
			// the tooltip goes right of the node that has it: the icon and
			// the value, not the third of the strip around them
			stats = append(stats, box("mc-stat", "children", []any{box("mc-stat-body", "tip", s.Label, "children", ch)}))
		}
		content = append(content, box("mc-panel mc-stats", "children", stats))
	}
	if b.Note != "" {
		content = append(content, txt("mc-pick-note", b.Note))
	}
	for _, g := range b.Groups {
		var tiles []any
		for _, c := range g.Choices {
			tiles = append(tiles, l.choice(c))
		}
		content = append(content, box("mc-pick-group", "children", []any{txt("mc-pick-title", g.Title), box("mc-picks", "children", tiles)}))
	}
	if len(rest) > 0 {
		content = append(content, box("mc-gap-6"))
		for _, o := range rest {
			content = append(content, l.listButton(o))
		}
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(content...)))
}

// choice is a tile of a ChoiceBoard: the icon over the name, what it does in
// the tooltip; the current one is ringed.
func (l *oreLayout) choice(c Choice) map[string]any {
	ch := []any{}
	if c.Icon != "" {
		ch = append(ch, img("", c.Icon))
	}
	ch = append(ch, txt("mc-pick-name", c.Label))
	if c.On {
		ch = append(ch, box("mc-pick-ring"))
	}
	tip := c.Label
	if c.On && c.Badge != "" {
		tip += " §a" + c.Badge
	}
	if c.Info != "" {
		tip += "\n§7" + c.Info
	}
	class := "mc-btn mc-pick"
	switch {
	case c.On:
		class += " on"
	case c.Dim:
		class += " dim"
	}
	n := box(class, "ink", "dark", "tip", tip, "children", ch)
	if c.On {
		return n
	}
	return l.press(n, c.Run)
}
