package ui

import (
	"strconv"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Console is the content of ConsoleScreen.
type Console struct {
	// Lines are the log, oldest first: the commands run and what came back.
	Lines []string
	// Hint is the text of the input box while it is empty, Submit the label
	// of the run button.
	Hint, Submit string
	// Run runs the command typed (Enter or the run button).
	Run func(p *player.Player, tx *world.Tx, command string)
	// Back is the control bar's back button (nil: none), labelled BackLabel.
	Back      func(p *player.Player, tx *world.Tx)
	BackLabel string
}

// ConsoleScreen shows p a command console: the log scrolled to its end and,
// under it, an input box with the run button. Each time it is shown with
// more lines the input box is new and empty, so a command run clears it.
func ConsoleScreen(p *player.Player, title string, c Console) {
	l := newOreLayout()
	var opts []Option
	if c.Back != nil {
		opts = append(opts, Back(c.BackLabel, c.Back))
	}
	_, bar, _ := l.chrome(opts)
	var log []any
	for _, line := range c.Lines {
		log = append(log, txt("mc-console-line", line))
	}
	id := "cmd" + strconv.Itoa(len(c.Lines))
	run := l.actionButton(Option{Label: c.Submit, Run: func(p *player.Player, tx *world.Tx) {
		if v := l.value(id); v != "" && c.Run != nil {
			c.Run(p, tx, v)
		}
	}})
	enter, _ := run["id"].(string)
	field := l.field(id, "", c.Hint, enter)
	content := box("mc-console", "children", []any{
		box("mc-scroll mc-col mc-console-log", "end", true, "children", log),
		box("mc-console-input", "children", []any{box("mc-search", "children", []any{field}), run}),
	})
	showOre(p, l, l.window(title, nil, bar, content))
}
