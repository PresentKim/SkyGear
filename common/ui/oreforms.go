package ui

import (
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
)

// DDUI screens for ListSplitMenu, Input, ItemGrid(Sections) and SlotMenu.
// They share the common window size, so moving between them updates the open
// screen in place.

// oreListSplit is a ListSplitMenu: icon tabs over the window, the rows in a
// scrolling panel on the left, and on the right the body and the actions,
// the first full width and the rest two to a row.
func oreListSplit(p *player.Player, title string, sp ListSplit) {
	l := newOreLayout()
	tabs, bar, actions := l.chrome(sp.Actions)
	var rows []any
	for _, r := range sp.Rows {
		name, info := listRowParts(r.Label)
		text := []any{txt("mc-lrow-name", name)}
		if info != "" {
			text = append(text, txt("mc-lrow-info", info))
		}
		ch := []any{}
		if r.Icon != "" {
			ch = append(ch, img("", r.Icon))
		}
		ch = append(ch, box("mc-lrow-text", "children", text))
		class := "mc-btn mc-lrow"
		if r.Selected {
			class += " on"
		}
		rows = append(rows, l.press(box(class, "ink", "dark", "children", ch), r.Run))
	}
	right := []any{}
	if sp.Body != "" {
		right = append(right, txt("mc-small-text", sp.Body), box("mc-gap-6"))
	}
	for i := 0; i < len(actions); {
		if i == 0 {
			right = append(right, l.actionButton(actions[0]))
			i++
			continue
		}
		pair := []any{l.actionButton(actions[i])}
		if i+1 < len(actions) {
			pair = append(pair, l.actionButton(actions[i+1]))
		}
		right = append(right, box("mc-lsplit-pair", "children", pair))
		i += 2
	}
	content := box("mc-row", "style", map[string]any{"height": "100%"}, "children", []any{
		box("mc-panel mc-lsplit-left", "children", []any{scroll(rows...)}),
		box("mc-lsplit-right", "children", []any{scroll(right...)}),
	})
	showOre(p, l, l.window(title, tabs, bar, content))
}

// listRowParts cuts a ListRowText label into the name (the padding taken
// off) and the information line. A label without the name field is all name.
func listRowParts(label string) (name, info string) {
	if len(label) <= listNameBytes {
		return label, ""
	}
	name, info = label[:listNameBytes], label[listNameBytes:]
	name = strings.TrimRight(name, " ")
	for strings.HasSuffix(name, "§r") {
		name = strings.TrimSuffix(name, "§r")
	}
	return name, info
}

// oreInput is an Input: the label over an input box with the text def, and a
// button that runs f with the text (Enter presses it too). With back, a back
// button in the control bar shows the screen it came from.
func oreInput(p *player.Player, title, label, def string, f func(p *player.Player, tx *world.Tx, value string), back func(p *player.Player, tx *world.Tx)) {
	lang := i18n.Of(p)
	l := newOreLayout()
	var opts []Option
	if back != nil {
		opts = append(opts, Back(i18n.T(lang, "ui.input.back"), back))
	}
	_, bar, _ := l.chrome(opts)
	submit := i18n.T(lang, "ui.input.ok")
	if strings.Contains(title, CreateToken) {
		submit = i18n.T(lang, "ui.input.create")
	}
	const id = "text"
	ok := l.listButton(Option{Label: submit, Icon: "textures/ui/confirm", Run: func(p *player.Player, tx *world.Tx) {
		f(p, tx, l.value(id))
	}})
	content := []any{}
	if label != "" {
		content = append(content, txt("mc-body-text", label), box("mc-gap-3"))
	}
	enter, _ := ok["id"].(string)
	content = append(content, box("mc-search", "children", []any{l.field(id, def, "", enter)}), box("mc-gap-6"), ok)
	showOre(p, l, l.window(title, nil, bar, scroll(box("mc-board", "children", content))))
}

// oreItemGrid is an ItemGrid or ItemGridSections: the body text, the headed
// sections of slots wrapping in the window's width, and the actions (tabs
// and the control bar's buttons in their places, the rest as list buttons).
func oreItemGrid(p *player.Player, title, body string, sections []GridSection, actions []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions)
	content := []any{}
	if body != "" {
		content = append(content, txt("mc-body-text", body), box("mc-gap-3"))
	}
	for _, sec := range sections {
		var slots []any
		for _, sl := range sec.Slots {
			slots = append(slots, l.slotNode(sl))
		}
		if sec.Heading != "" {
			content = append(content, txt("mc-head", sec.Heading))
		}
		content = append(content, box("mc-grid", "children", slots), box("mc-gap-3"))
	}
	for _, o := range rest {
		content = append(content, l.listButton(o))
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(content...)))
}

// oreSlotMenu is a SlotMenu (sg_form slot_form): the storage slot with the
// info text beside it, under them the left slots at the left and the right
// slots at the right, and the actions as list buttons. A slot's text is its
// tooltip; the storage slot's is its count.
func oreSlotMenu(p *player.Player, title string, store Slot, info string, left, right []Slot, actions []Option) {
	l := newOreLayout()
	tabs, bar, rest := l.chrome(actions)
	slots := func(list []Slot) []any {
		var out []any
		for _, sl := range list {
			out = append(out, l.slotNode(ItemSlot{Icon: sl.Icon, Tip: sl.Text, Run: sl.Run}))
		}
		return out
	}
	content := []any{
		box("mc-row", "children", []any{
			l.slotNode(ItemSlot{Icon: store.Icon, Count: store.Text}, "mc-big"),
			box("mc-slotmenu-info", "children", []any{txt("mc-body-text", info)}),
		}),
		box("mc-gap-6"),
		box("mc-slotmenu-slots", "children", []any{
			box("mc-row", "children", slots(left)),
			box("mc-row", "children", slots(right)),
		}),
	}
	for _, o := range rest {
		content = append(content, box("mc-gap-6"), l.listButton(o))
	}
	showOre(p, l, l.window(title, tabs, bar, scroll(box("mc-board", "children", content))))
}
