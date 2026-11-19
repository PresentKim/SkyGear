// Command invuigen writes assets/resourcepack/inventoryui/ui/sg_menu.json, the
// menu window of common/invui: the slots of its layout read at fixed
// collection indexes, which JSON UI cannot loop over. Run it from the
// repository root: go run ./tools/invuigen
//
// The window follows the bedrock-research entity inventory examples
// (docs/design/entity-inventory-ui.mdx, inventory-ui-ddui): pages are panels
// shown by slot 0's key, and the bindings of a control that does not decide
// its own visibility only run while it is visible.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/presentkim/skygear/common/invui"
)

const out = "assets/resourcepack/inventoryui/ui/sg_menu.json"

type obj = map[string]any

var (
	light = []any{0.95, 0.95, 0.97}
	ink   = []any{0.2, 0.2, 0.2}
	head  = []any{0.3, 0.3, 0.3}
	title = []any{0.25, 0.25, 0.25}
	dim   = []any{0.8, 0.8, 0.82}
	top   = obj{"anchor_from": "top_left", "anchor_to": "top_left"}
)

// clean is the text of a slot: its name without the italic code the client
// puts on custom names and without the kind markers.
func clean() string {
	s := "(#hover_text - '§o'"
	for _, k := range invui.Kinds {
		s += " - '" + k + "'"
	}
	return s + ")"
}

func texturePath() string { return "(" + clean() + " - '§r' - '§f')" }

func slotText() obj {
	return obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": "#hover_text", "binding_condition": "always"}
}

func slotValue(name string) obj {
	return obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": name, "binding_condition": "always"}
}

func view(src, target string) obj {
	return obj{"binding_type": "view", "binding_condition": "always", "source_property_name": src, "target_property_name": target}
}

// view2 copies property of the control named from into the same property.
func view2(from, property string) obj {
	return obj{"binding_type": "view", "source_control_name": from, "source_property_name": property, "target_property_name": property}
}

func has(marker string) string { return "(not ((#hover_text - '" + marker + "') = #hover_text))" }

const filled = "(not (#hover_text = ''))"

func shown(cond string) []any { return []any{slotText(), view(cond, "#visible")} }

// shownButton is the button b shown while cond holds. Its own bindings stay:
// without its collection_details binding a press reaches no slot.
func shownButton(b obj, cond string) obj {
	b["bindings"] = append(b["bindings"].([]any), shown(cond)...)
	return b
}

func with(o obj, props obj) obj {
	for k, v := range props {
		o[k] = v
	}
	return o
}

func ref(name, def string, props obj) obj {
	if props == nil {
		props = obj{}
	}
	return obj{name + "@sg_menu." + def: props}
}

func at(name, def string, index int, props obj) obj {
	if props == nil {
		props = obj{}
	}
	props["collection_index"] = index
	return ref(name, def, props)
}

// slots is a stack panel of container slots, which the indexes of its
// direct children read.
func slots(orientation string, size any, children []any) obj {
	return obj{"type": "stack_panel", "orientation": orientation, "size": size, "collection_name": "container_items", "controls": children}
}

func panel(size any, controls ...any) obj {
	return obj{"type": "panel", "size": size, "controls": controls}
}

func label(color []any, scale float64, size []any, props obj) obj {
	return with(obj{
		"type": "label", "layer": 3, "localize": false, "color": color, "font_scale_factor": scale, "size": size, "text": "#text",
		"bindings": []any{slotText(), view(clean(), "#text")},
	}, props)
}

func image(texture string, size []any, layer int, props obj) obj {
	return with(obj{"type": "image", "texture": texture, "size": size, "layer": layer}, props)
}

// button is a slot pressed by moving its item to the player's inventory (a
// quick move), which the server cancels and answers: the press that reaches
// the server on touch screens too (entity-inventory-ui 2.5). Its states
// draw only backgrounds and the tooltip; what it shows sits over it.
func button(size any, def, hover, pressed string, tip bool) obj {
	mappings := []any{}
	for _, from := range []string{"button.menu_select", "button.menu_ok"} {
		mappings = append(mappings, obj{"from_button_id": from, "to_button_id": "button.container_auto_place", "mapping_type": "pressed", "consume_event": false})
	}
	mappings = append(mappings, obj{"to_button_id": "button.container_slot_hovered", "mapping_type": "pressed"})
	state := func(texture string, withTip bool) obj {
		c := []any{}
		if texture != "" {
			c = append(c, obj{"bg": image(texture, []any{"100%", "100%"}, 1, nil)})
		}
		if withTip {
			c = append(c, obj{"tip": obj{"type": "custom", "renderer": "hover_text_renderer", "allow_clipping": false, "layer": 30}})
		}
		return panel([]any{"100%", "100%"}, c...)
	}
	return obj{
		"type": "button", "size": size, "layer": 1, "focus_enabled": true, "focus_magnet_enabled": true,
		"default_control": "default", "hover_control": "hover", "pressed_control": "pressed",
		"sound_name": "random.click", "sound_volume": 1.0, "sound_pitch": 1.0,
		"button_mappings": mappings,
		"bindings":        []any{obj{"binding_type": "collection_details", "binding_collection_name": "container_items"}},
		"controls": []any{
			obj{"default": state(def, false)},
			obj{"hover": state(hover, tip)},
			obj{"pressed": state(pressed, false)},
		},
	}
}

// flatButton is a button of the lobby: flat colours, a dark edge and a band
// under it (oui-mc.css .mc-lobby .mc-btn). Its states draw only the faces.
func flatButton(size any, fill, hover, band []any, tip bool) obj {
	b := button(size, "", "", "", tip)
	face := func(c []any, pressed bool) []any {
		top, bottom := 1, 3
		if pressed {
			top, bottom = 2, 2
		}
		return []any{
			obj{"edge": rect([]any{0.118, 0.118, 0.122}, []any{"100%", "100%"}, nil, 1)},
			obj{"band": rect(band, []any{"100% - 2px", "100% - 2px"}, []any{1, 1}, 1)},
			obj{"fill": rect(c, []any{"100% - 2px", fmt.Sprintf("100%% - %dpx", top+bottom)}, []any{1, top}, 1)},
		}
	}
	states := b["controls"].([]any)
	for i, s := range []struct {
		c       []any
		pressed bool
	}{{fill, false}, {hover, false}, {hover, true}} {
		for name, st := range states[i].(obj) {
			p := st.(obj)
			p["controls"] = append(face(s.c, s.pressed), p["controls"].([]any)...)
			states[i] = obj{name: p}
		}
	}
	return b
}

// rect is a rectangle of one colour.
func rect(c []any, size []any, offset []any, layer int) obj {
	r := obj{"type": "image", "texture": "textures/ui/White", "color": c, "size": size, "layer": layer, "anchor_from": "top_left", "anchor_to": "top_left"}
	if offset != nil {
		r["offset"] = offset
	}
	return r
}

func rgb(r, g, b int) []any { return []any{float64(r) / 255, float64(g) / 255, float64(b) / 255} }

var (
	lobbyFill  = rgb(0x48, 0x49, 0x4a)
	lobbyHover = rgb(0x5a, 0x5b, 0x5c)
	lobbyBand  = rgb(0x24, 0x25, 0x26)
	lobbyPanel = rgb(0x31, 0x32, 0x33)
	lobbyHead  = rgb(0xb1, 0xb2, 0xb5)
	lobbyDark  = rgb(0x1e, 0x1e, 0x1f)
	lightFill  = rgb(0xd0, 0xd1, 0xd4)
	lightHover = rgb(0xe6, 0xe8, 0xeb)
	lightBand  = rgb(0x58, 0x58, 0x5a)
	playFill   = rgb(0x3c, 0x85, 0x27)
	playHover  = rgb(0x4a, 0x9b, 0x31)
	playBand   = rgb(0x1d, 0x4d, 0x13)
	white      = []any{1.0, 1.0, 1.0}
)

const (
	btnTex   = "textures/ui/sg/button"
	btnHover = "textures/ui/sg/button_hover"
	btnPress = "textures/ui/sg/button_pressed"
)

// icon is the icon of the slot $icon_index: a texture path named on a paper,
// or the item in the slot.
func icon(size int) obj {
	s := []any{size, size}
	return slots("vertical", s, []any{ref("i", "icon_slot", obj{"size": s, "collection_index": "$icon_index"})})
}

// itemIcon draws the slot it reads: a texture path named on a paper, or the
// item itself.
func itemIcon(size []any) obj {
	return panel(size,
		obj{"image": obj{
			"type": "image", "size": []any{"100%", "100%"}, "layer": 2, "texture": "#texture",
			"bindings": []any{slotText(), view(texturePath(), "#texture"), view(has("textures/"), "#visible")},
		}},
		obj{"item": obj{
			"type": "custom", "renderer": "inventory_item_renderer", "size": []any{"100%", "100%"}, "layer": 2,
			"property_bag": obj{"#item_id_aux": 0},
			"bindings":     []any{slotText(), slotValue("#item_id_aux"), view("((#hover_text - 'textures/') = #hover_text)", "#visible")},
		}},
	)
}

func defs() obj {
	d := obj{"namespace": "sg_menu"}
	d["icon_slot"] = itemIcon([]any{"100%", "100%"})
	d["empty"] = panel([]any{0, 0})

	textLabel := label(ink, 1.0, []any{"100% - 2px", "default"}, with(obj{"offset": []any{2, 0}, "font_scale_factor": "$text_scale"}, top))
	d["text_slot"] = with(panel([]any{"100%", "100%c"}, obj{"l": textLabel}), obj{"$text_scale|default": 1.0})
	d["title_slot"] = panel([]any{310, 10}, obj{"l": label(title, 1.0, []any{"100%", 10}, obj{"text_alignment": "center"})})

	// Tabs over the window.
	tabCells := []any{}
	for i := 0; i < invui.Tabs; i++ {
		tabCells = append(tabCells, at(fmt.Sprintf("t%d", i), "tab_cell", invui.TabBtn0+i, obj{"$icon_index": invui.TabIcon0 + i}))
	}
	tabIcon := func(layer int) obj {
		return with(panel([]any{24, 22}, obj{"i": icon(14)}), with(obj{"layer": layer}, top))
	}
	d["tab_cell"] = with(panel([]any{24, 24},
		obj{"on": with(panel([]any{24, 24},
			obj{"bg": image("textures/ui/TabTopFront", []any{24, 26}, 10, top)},
			obj{"icon": tabIcon(11)},
		), obj{"bindings": shown(has(invui.KindTabOn))})},
		obj{"off": with(panel([]any{24, 24},
			obj{"b": with(button([]any{24, 28}, "textures/ui/TabTopBack", "textures/ui/TabTopBackHover", "textures/ui/TabTopBackHover", true), top)},
			obj{"icon": tabIcon(3)},
		), obj{"bindings": shown(has(invui.KindTab))})},
	), obj{"bindings": shown(filled)})

	// Control bar buttons.
	barCells := []any{}
	for i := 0; i < invui.Bars; i++ {
		barCells = append(barCells, at(fmt.Sprintf("b%d", i), "bar_cell", invui.BarBtn0+i, obj{"$icon_index": invui.BarIcon0 + i}))
	}
	d["bar_cell"] = with(panel([]any{15, 22},
		obj{"b": button([]any{15, 15}, "", "textures/ui/sg/hover_flat", "textures/ui/sg/press_flat", true)},
		obj{"i": with(icon(15), obj{"layer": 3})},
		obj{"q": obj{"type": "label", "layer": 3, "text": "?", "localize": false, "color": title, "bindings": shown(has(invui.KindHelp))}},
	), obj{"bindings": shown(filled)})

	// List rows: rows, rows that cannot be pressed and toggles share one
	// shape; headings and dividers are drawn apart.
	rowCells := []any{}
	for i := 0; i < invui.Rows; i++ {
		rowCells = append(rowCells, at(fmt.Sprintf("r%d", i), "row_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i}))
	}
	switchImage := func(tex, marker string) obj {
		return image(tex, []any{30, 16}, 3, obj{"offset": []any{-3, 0}, "anchor_from": "right_middle", "anchor_to": "right_middle", "bindings": shown(has(marker))})
	}
	d["row_cell"] = with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{"100%", "100%c"}, "controls": []any{
		obj{"row": with(panel([]any{"100%", 32},
			obj{"b": with(button([]any{"100%", 30}, btnTex, btnHover, btnPress, false), top)},
			obj{"dim": image(btnTex, []any{"100%", 30}, 2, with(obj{"alpha": 0.4, "color": []any{0.6, 0.6, 0.6}, "bindings": shown(has(invui.KindStatic))}, top))},
			obj{"i": with(panel([]any{20, 20}, obj{"i": icon(20)}), obj{"layer": 3, "offset": []any{5, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
			obj{"l": label(light, 1.0, []any{"100% - 34px", "default"}, obj{"layer": 4, "offset": []any{29, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
			obj{"on": switchImage("textures/ui/toggle_on", invui.KindToggleOn)},
			obj{"off": switchImage("textures/ui/toggle_off", invui.KindToggleOff)},
		), obj{"bindings": shown(has(invui.FamRow))})},
		obj{"heading": with(panel([]any{"100%", 10},
			obj{"l": label(head, 0.8, []any{"100% - 2px", "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
		), obj{"bindings": shown(has(invui.KindHeading))})},
		obj{"divider": with(panel([]any{"100%", 9},
			obj{"line": image("textures/ui/White", []any{"100%", 1}, 1, obj{"color": []any{0.545, 0.545, 0.545}})},
		), obj{"bindings": shown(has(invui.KindDivider))})},
	}}, obj{"bindings": shown(filled)})
	rowList := func() obj { return slots("vertical", []any{"100%", "100%c"}, rowCells) }

	// Item cells: an item slot (or a small icon) with its frame, count,
	// chosen frame and tooltip; frames, arrows and lines of text.
	d["slot_face"] = panel([]any{18, 18},
		obj{"frame": image("textures/ui/cell_image", []any{18, 18}, 1, nil)},
		obj{"icon": with(itemIcon([]any{16, 16}), obj{"layer": 2})},
		obj{"count": obj{
			"type": "label", "layer": 5, "localize": false, "color": []any{1, 1, 1}, "shadow": true, "font_scale_factor": 0.86,
			"anchor_from": "bottom_right", "anchor_to": "bottom_right", "offset": []any{0, 1}, "text": "#text",
			"bindings": []any{slotValue("#inventory_stack_count"), view("('§r' + #inventory_stack_count)", "#text")},
		}},
		obj{"chosen": image("textures/ui/sg/slot_chosen", []any{18, 18}, 1, obj{"bindings": shown(has(invui.KindChosen))})},
	)
	d["mini_face"] = panel([]any{18, 18}, obj{"icon": with(itemIcon([]any{12, 12}), obj{"layer": 2})})
	cellButton := button([]any{18, 18}, "", "", "", true)
	cellButton["controls"].([]any)[1].(obj)["hover"].(obj)["controls"] = append(
		cellButton["controls"].([]any)[1].(obj)["hover"].(obj)["controls"].([]any),
		obj{"lit": image("textures/ui/White", []any{16, 16}, 4, obj{"alpha": 0.3})},
	)
	d["cell_button"] = cellButton
	cellText := func(width int, color []any, scale float64) obj {
		return obj{"l": label(color, scale, []any{width, "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})}
	}
	cellKind := func(name, marker string, controls ...any) obj {
		return obj{name: with(panel([]any{18, 18}, controls...), obj{"bindings": shown(has(marker))})}
	}
	d["cell"] = panel([]any{18, 18},
		cellKind("slot", invui.FamSlot, ref("face", "slot_face", nil), ref("b", "cell_button", nil)),
		cellKind("mini", invui.FamMini, ref("face", "mini_face", nil), ref("b", "cell_button", nil)),
		cellKind("frame", invui.KindFrame, obj{"f": image("textures/ui/cell_image", []any{18, 18}, 1, nil)}),
		cellKind("arrow", invui.KindArrow, obj{"a": image("textures/ui/arrow_large", []any{15, 10}, 2, nil)}),
		cellKind("text", invui.KindText, cellText(194, ink, 0.7)),
		cellKind("heading", invui.KindHeading, cellText(300, head, 0.8)),
	)
	// A row of a cell grid shows while its first cell holds something, so
	// the rows after the last one used take no room.
	cellGrid := func(first, rows, columns int) obj {
		out := []any{}
		for r := 0; r < rows; r++ {
			start := invui.Cell0 + first + r*columns
			cells := []any{}
			for c := 0; c < columns; c++ {
				cells = append(cells, at(fmt.Sprintf("c%d", c), "cell", start+c, nil))
			}
			name := fmt.Sprintf("cell_row_%d_%d", start, columns)
			d[name] = with(panel([]any{columns * 18, 18}, obj{"cells": slots("horizontal", []any{columns * 18, 18}, cells)}), obj{"bindings": shown(filled)})
			out = append(out, at(fmt.Sprintf("r%d", r), name, start, nil))
		}
		return slots("vertical", []any{columns * 18, "100%c"}, out)
	}

	// Cards, four to a row.
	cardRows := []any{}
	for r := 0; r*invui.CardColumns < invui.Rows; r++ {
		cards := []any{}
		for c := 0; c < invui.CardColumns; c++ {
			i := r*invui.CardColumns + c
			cards = append(cards, at(fmt.Sprintf("c%d", c), "card_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i}))
		}
		cardRows = append(cardRows, obj{fmt.Sprintf("r%d", r): slots("horizontal", []any{304, "100%cm"}, cards)})
	}
	d["card_cell"] = with(panel([]any{76, 50},
		obj{"b": button([]any{72, 46}, btnTex, btnHover, btnPress, false)},
		obj{"i": with(panel([]any{22, 22}, obj{"i": icon(22)}), obj{"layer": 3, "offset": []any{2, 10}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
		obj{"l": label(light, 0.8, []any{66, "default"}, obj{"layer": 4, "offset": []any{0, -6}, "anchor_from": "bottom_middle", "anchor_to": "bottom_middle", "text_alignment": "center"})},
	), obj{"bindings": shown(filled)})

	// Action buttons (split and list-split screens).
	actionCells := func(first, last int) []any {
		out := []any{}
		for i := first; i < last; i++ {
			out = append(out, at(fmt.Sprintf("a%d", i), "action_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i}))
		}
		return out
	}
	d["action_cell"] = with(panel([]any{"100%", 19},
		obj{"b": with(button([]any{"100%", 17}, btnTex, btnHover, btnPress, false), top)},
		obj{"i": with(panel([]any{12, 12}, obj{"i": icon(12)}), obj{"layer": 3, "offset": []any{2, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
		obj{"l": label(light, 0.8, []any{"100% - 17px", "default"}, obj{"layer": 4, "offset": []any{15, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	), obj{"bindings": shown(has(invui.KindRow))})

	column := func(width string, controls ...any) obj {
		return with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{width, "100%c"}, "controls": controls}, top)
	}
	gap := func(h int) obj { return obj{"gap": panel([]any{"100%", h})} }
	body := func(scale float64) obj {
		return obj{"body": slots("vertical", []any{"100%", "100%c"}, []any{at("t", "text_slot", invui.SlotBody, obj{"$text_scale": scale})})}
	}
	scroll := func(name, content string) {
		d[name+"@common.scrolling_panel"] = obj{
			"anchor_to": "top_left", "anchor_from": "top_left", "$show_background": false, "size": []any{"100%", "100%"},
			"$scroll_size": []any{4, "100%"}, "$scrolling_pane_size": []any{"100%", "100%"}, "$scrolling_pane_offset": []any{0, 0},
			"$scroll_bar_right_padding_size": []any{0, 0}, "$scrolling_content": "sg_menu." + content,
		}
	}
	// gate shows controls while slot 0's key contains key.
	gate := func(name, key string, controls ...any) obj {
		named := make([]any, len(controls))
		for i, c := range controls {
			if o, ok := c.(obj); ok && o["type"] != nil {
				c = obj{fmt.Sprintf("c%d", i): o}
			}
			named[i] = c
		}
		d["gate_"+name] = with(panel([]any{"100%", "100%c"}, named...), obj{"bindings": shown(has(key))})
		return obj{name: slots("vertical", []any{"100%", "100%c"}, []any{at("g", "gate_"+name, invui.SlotKey, nil)})}
	}

	// The list content: the body, the content of the layout and the rows.
	board := cellGrid(0, invui.Cells/invui.BoardColumns, invui.BoardColumns)
	board["anchor_from"], board["anchor_to"] = "top_middle", "top_middle"
	d["list_page_body"] = column("100% - 8px",
		body(1.0), gap(6),
		gate("cards", invui.KeyCards, column("100%", cardRows...)),
		gate("grid", invui.KeyGrid, cellGrid(0, invui.Cells/invui.GridColumns, invui.GridColumns), gap(3)),
		gate("board", invui.KeyBoard, panel([]any{"100%", "100%c"}, obj{"g": board}), gap(3)),
		gate("rows", invui.KeyRows, rowList()),
	)
	scroll("list_page", "list_page_body")

	// The split screen: the grid of slots with the page arrows on the left,
	// the chosen slot's detail on the right.
	pager := func(arrow string) obj {
		return panel([]any{18, 18},
			obj{"b": shownButton(button([]any{18, 18}, btnTex, btnHover, btnPress, false), has(invui.KindRow))},
			obj{"off": image(btnTex, []any{18, 18}, 1, obj{"alpha": 0.4, "bindings": shown(has(invui.KindStatic))})},
			obj{"a": image("textures/ui/sg/arrow_"+arrow, []any{12, 12}, 3, nil)},
		)
	}
	d["prev_cell"] = pager("left")
	d["next_cell"] = pager("right")
	d["detail_big"] = panel([]any{28, 28},
		obj{"f": image("textures/ui/cell_image", []any{28, 28}, 1, nil)},
		obj{"i": with(icon(22), obj{"layer": 2})},
	)
	d["detail_name"] = panel([]any{"fill", 28}, obj{"l": label(ink, 0.8, []any{"100% - 4px", "default"}, obj{"offset": []any{4, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})})
	d["split_right_body"] = column("100% - 8px",
		obj{"head": slots("horizontal", []any{"100%", 30}, []any{
			at("big", "detail_big", invui.RowBtn0, obj{"$icon_index": invui.RowIcon0}),
			at("name", "detail_name", invui.RowBtn0, nil),
		})},
		obj{"status": slots("vertical", []any{"100%", "100%c"}, []any{at("t", "text_slot", invui.SlotInfo, obj{"$text_scale": 0.8})})},
		gap(3),
		obj{"actions": slots("vertical", []any{"100%", "100%c"}, actionCells(1, invui.Rows))},
		body(0.8),
		gap(3),
		obj{"rows": cellGrid(invui.SplitCells, (invui.Cells-invui.SplitCells)/invui.RightColumns, invui.RightColumns)},
	)
	scroll("split_right_scroll", "split_right_body")
	splitGrid := cellGrid(0, invui.SplitCells/invui.SplitColumns, invui.SplitColumns)
	splitGrid["offset"], splitGrid["layer"] = []any{4, 4}, 2
	splitPager := slots("horizontal", []any{144, 18}, []any{
		at("prev", "prev_cell", invui.SlotPrev, nil),
		obj{"fill": panel([]any{"fill", 18})},
		at("next", "next_cell", invui.SlotNext, nil),
	})
	splitPager["offset"], splitPager["layer"] = []any{4, 114}, 2
	d["split_page"] = panel([]any{"100%", "100%"},
		obj{"left": image("textures/ui/sg/panel", []any{152, 142}, 1, with(obj{"controls": []any{obj{"grid": with(splitGrid, top)}, obj{"pager": with(splitPager, top)}}}, top))},
		obj{"right": image("textures/ui/sg/panel", []any{156, 142}, 1, obj{"anchor_from": "top_right", "anchor_to": "top_right", "controls": []any{
			obj{"inner": panel([]any{148, 134}, ref("s", "split_right_scroll", nil))},
		}})},
	)

	// The list-split screen: a list of rows on the left, the body and the
	// actions of the chosen row on the right.
	listCells := []any{}
	for i := 0; i < invui.ListRows; i++ {
		listCells = append(listCells, at(fmt.Sprintf("r%d", i), "list_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i, "$info_index": invui.Cell0 + i}))
	}
	infoLine := slots("vertical", []any{"100% - 30px", 8}, []any{ref("t", "info_slot", obj{"collection_index": "$info_index"})})
	d["info_slot"] = panel([]any{"100%", 8}, obj{"l": label(dim, 0.7, []any{"100%", "default"}, top)})
	d["list_cell"] = with(panel([]any{"100%", 32},
		obj{"b": shownButton(button([]any{"100%", 30}, btnTex, btnHover, btnPress, false), has(invui.KindRow))},
		obj{"chosen": image(btnHover, []any{"100%", 30}, 1, with(obj{"bindings": shown(has(invui.KindChosen))}, top))},
		obj{"i": with(panel([]any{20, 20}, obj{"i": icon(20)}), obj{"layer": 3, "offset": []any{5, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
		obj{"name": label(light, 1.0, []any{"100% - 30px", 9}, with(obj{"layer": 4, "offset": []any{27, 5}}, top))},
		obj{"info": with(infoLine, with(obj{"offset": []any{27, 16}, "layer": 4}, top))},
	), obj{"bindings": shown(filled)})
	d["list_body"] = column("100% - 8px", obj{"rows": slots("vertical", []any{"100%", "100%c"}, listCells)})
	scroll("list_scroll", "list_body")
	d["list_right_body"] = column("100% - 8px", body(0.8), gap(6), obj{"actions": slots("vertical", []any{"100%", "100%c"}, actionCells(invui.ListRows, invui.Rows))})
	scroll("list_right_scroll", "list_right_body")
	d["listsplit_page"] = panel([]any{"100%", "100%"},
		obj{"left": image("textures/ui/sg/panel", []any{156, 142}, 1, with(obj{"controls": []any{obj{"inner": panel([]any{148, 134}, ref("s", "list_scroll", nil))}}}, top))},
		obj{"right": with(panel([]any{154, 142}, ref("s", "list_right_scroll", nil)), obj{"anchor_from": "top_right", "anchor_to": "top_right"})},
	)

	// The range screen: the 3D preview of the area on the left, the offset
	// and reach axes with their minus and plus buttons on the right.
	d["range_minus"] = with(pager("left"), obj{"bindings": shown(filled)})
	d["range_plus"] = with(pager("right"), obj{"bindings": shown(filled)})
	d["range_value"] = with(panel([]any{50, 18},
		obj{"l": label(ink, 1.0, []any{"100% - 4px", "default"}, obj{"offset": []any{4, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	), obj{"bindings": shown(filled)})
	d["range_heading"] = with(panel([]any{"100%", 13},
		obj{"l": label(head, 0.8, []any{"100% - 2px", "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	), obj{"bindings": shown(filled)})
	axisLine := func(i int) obj {
		return obj{fmt.Sprintf("a%d", i): slots("horizontal", []any{"100%", "100%cm"}, []any{
			at("m", "range_minus", invui.RowBtn0+2*i, nil),
			at("v", "range_value", invui.Cell0+i, nil),
			at("p", "range_plus", invui.RowBtn0+2*i+1, nil),
		})}
	}
	heading := func(name string, cell int) obj {
		return obj{name: slots("vertical", []any{"100%", "100%c"}, []any{at("h", "range_heading", invui.Cell0+cell, nil)})}
	}
	rangeRows := []any{}
	for i := invui.RangeRow0; i < invui.RangeRow0+4; i++ {
		rangeRows = append(rangeRows, at(fmt.Sprintf("r%d", i), "action_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i}))
	}
	right := []any{heading("h0", invui.RangeHead0)}
	for i := 0; i < invui.RangeAxes; i++ {
		if i == invui.RangeAxes/2 {
			right = append(right, gap(4), heading("h1", invui.RangeHead1))
		}
		right = append(right, axisLine(i))
	}
	right = append(right, gap(4), obj{"rows": slots("vertical", []any{"100%", "100%c"}, rangeRows)})
	d["range_right_body"] = column("100% - 8px", right...)
	scroll("range_right_scroll", "range_right_body")
	turn := "sg_range_turn"
	views := []any{}
	for k, size := range invui.RangePreviews {
		def := fmt.Sprintf("range_view_%d", k)
		d[def] = with(panel([]any{"100%", "100%"}, obj{"r": obj{
			"type": "custom", "renderer": "3d_structure_renderer", "size": []any{200, 200}, "layer": 3,
			"anchor_from": "center", "anchor_to": "center",
			"property_bag": obj{
				"#has_focus": false, "#block_position": []any{0, 0, 0},
				"#bottom_left_block": []any{-size / 2, invui.PreviewY, -size / 2},
				"#top_right_block":   []any{size / 2, invui.PreviewY + size, size / 2},
				"#include_entities":  false, "#remove_blocks": false, "#include_players": false,
			},
			"bindings": []any{
				view2(turn, "#gesture_delta_source"),
				view2(turn, "#gesture_mouse_delta_x"),
				view2(turn, "#gesture_mouse_delta_y"),
			},
		}}), obj{"bindings": shown(has(fmt.Sprintf("v%d", k+1)))})
		views = append(views, obj{fmt.Sprintf("v%d", k): slots("vertical", []any{"100%", "100%"}, []any{at("g", def, invui.Cell0+invui.RangeView, nil)})})
	}
	d["range_page"] = panel([]any{"100%", "100%"},
		obj{"left": image("textures/ui/sg/panel", []any{138, 138}, 1, with(obj{"offset": []any{0, 2}, "controls": []any{
			obj{"clip": with(panel([]any{130, 130}, obj{turn: obj{
				"type": "input_panel", "size": []any{"100%", "100%"}, "focus_enabled": true,
				"gesture_tracking_button": "button.turn_structure",
				"button_mappings": []any{obj{"from_button_id": "button.menu_select", "to_button_id": "button.turn_structure", "mapping_type": "pressed", "button_up_right_of_first_refusal": true}},
				"controls": views,
			}}), obj{"clips_children": true, "anchor_from": "center", "anchor_to": "center"})},
		}}, top))},
		obj{"right": with(panel([]any{170, 142}, ref("s", "range_right_scroll", nil)), obj{"anchor_from": "top_right", "anchor_to": "top_right"})},
	)

	homeDefs(d)

	pages := []any{}
	for _, p := range []struct{ key, def string }{
		{invui.KeyList, "list_page"}, {invui.KeySplit, "split_page"}, {invui.KeyListSplit, "listsplit_page"}, {invui.KeyRange, "range_page"},
	} {
		name := strings.ReplaceAll(p.def, "_page", "")
		d["page_"+name] = with(panel([]any{"100%", "100%"}, ref("page", p.def, nil)), obj{"bindings": shown(has(p.key))})
		pages = append(pages, obj{name: slots("vertical", []any{"100%", "100%"}, []any{at("g", "page_"+name, invui.SlotKey, nil)})})
	}

	tabs := with(slots("horizontal", []any{"100%c", 24}, tabCells), obj{"offset": []any{8, -22}, "layer": 0})
	d["window"] = panel([]any{326, 166},
		obj{"frame": image("textures/ui/dialog_background_opaque", []any{"100%", "100%"}, 1, nil)},
		obj{"title": with(panel([]any{310, 10}, obj{"s": slots("vertical", []any{310, 10}, []any{at("t", "title_slot", invui.SlotTitle, nil)})}), with(obj{"offset": []any{8, 6}, "layer": 3}, top))},
		obj{"tabs": with(tabs, top)},
		obj{"toolbar": image("textures/ui/toolbar_background", []any{"100%c + 14px", 25}, 4, obj{
			"offset": []any{0, -22}, "anchor_from": "top_right", "anchor_to": "top_right",
			"controls": []any{obj{"row": with(obj{
				"type": "stack_panel", "orientation": "horizontal", "size": []any{"100%c", 22}, "offset": []any{7, 1},
				"controls": []any{
					obj{"bar": slots("horizontal", []any{"100%c", 22}, barCells)},
					obj{"close": panel([]any{17, 22}, obj{"x@common.light_close_button": obj{"$close_button_offset": []any{0, 0}, "size": []any{15, 15}, "anchor_from": "center", "anchor_to": "center"}})},
				},
			}, top)}},
		})},
		obj{"content": with(panel([]any{314, 142}, obj{"pages": panel([]any{"100%", "100%"}, pages...)}), with(obj{"layer": 2, "offset": []any{6, 18}}, top))},
	)

	d["overlay"] = obj{
		"type": "panel", "size": []any{"100%", "100%"}, "layer": 1,
		"$sg_title": "$container_title",
		"controls": []any{obj{"host": obj{
			"type": "panel", "size": []any{"100%", "100%"},
			"ignored": "(($sg_title - '" + invui.Marker + "') = $sg_title)",
			"controls": []any{
				obj{"entity_ui_hide_vanilla": obj{"type": "panel", "size": []any{0, 0}, "property_bag": obj{"#entity_ui_vanilla_visible": false}}},
				obj{"cover": obj{
					"type": "input_panel", "modal": true, "size": []any{"100%", "100%"}, "layer": 1,
					"controls": []any{
						obj{"sync": obj{"type": "grid", "size": []any{0, 0}, "maximum_grid_items": 27, "collection_name": "container_items", "grid_item_template": "sg_menu.empty"}},
						obj{"window": slots("vertical", []any{"100%", "100%"}, []any{at("g", "window_gate", invui.SlotKey, nil)})},
						obj{"home": slots("vertical", []any{"100%", "100%"}, []any{at("g", "home_gate", invui.SlotKey, nil)})},
						obj{"lobby": slots("vertical", []any{"100%", "100%"}, []any{at("g", "lobby_gate", invui.SlotKey, nil)})},
						obj{"loading": slots("vertical", []any{"100%", "100%"}, []any{at("g", "loading_gate", invui.SlotKey, nil)})},
					},
				}},
			},
		}}},
	}
	return d
}

// homeDefs adds the lobby's main screen (oui-mc.css .mc-home, in GUI pixels:
// 1rem is 5): no window, the profile card at the top left, square buttons at
// the top right, tiles down the left, the feed at the bottom left and the
// chosen save with the play and manage buttons at the bottom right; and the
// lobby's logo and name at the top middle of every lobby screen.
func homeDefs(d obj) {
	row := func(r int) (int, int) { return invui.RowIcon0 + r, invui.RowBtn0 + r }
	cellSlot := func(c int) int { return invui.Cell0 + c }
	textAt := func(name string, slot int, color []any, scale float64, size []any, props obj) obj {
		def := "home_text_" + name
		d[def] = panel(size, obj{"l": label(color, scale, []any{"100%", "default"}, with(obj{"shadow": false}, top))})
		return obj{name: with(slots("vertical", size, []any{at("t", def, slot, nil)}), props)}
	}
	iconAt := func(name string, slot, size int, props obj) obj {
		return obj{name: with(slots("vertical", []any{size, size}, []any{ref("i", "icon_slot", obj{"size": []any{size, size}, "collection_index": slot})}), props)}
	}
	pos := func(from string, x, y int) obj {
		return obj{"anchor_from": from, "anchor_to": from, "offset": []any{x, y}}
	}

	// The profile card.
	pi, pb := row(invui.HomeProfileRow)
	d["home_profile"] = panel([]any{135, 37},
		obj{"b": flatButton([]any{135, 37}, lobbyFill, lobbyHover, lobbyBand, false)},
		obj{"head": with(icon(23), with(obj{"layer": 3, "offset": []any{5, 6}}, top))},
		obj{"name": label(white, 1.14, []any{96, "default"}, with(obj{"layer": 4, "offset": []any{33, 4}}, top))},
	)
	// The bar's fill is one of invui.HomeFillSteps widths, chosen by the
	// slot's name "w00" to "w20" (bedrock-research basic components 3.16
	// draws bars this way).
	fills := []any{}
	for k := 1; k <= invui.HomeFillSteps; k++ {
		fills = append(fills, obj{fmt.Sprintf("w%02d", k): with(rect(rgb(0x62, 0xc2, 0x40), []any{94 * k / invui.HomeFillSteps, 2}, nil, 2), obj{"bindings": shown(has(fmt.Sprintf("w%02d", k)))})})
	}
	d["home_fill"] = panel([]any{94, 2}, fills...)
	profile := with(panel([]any{135, 37},
		obj{"card": with(slots("vertical", []any{135, 37}, []any{at("c", "home_profile", pb, obj{"$icon_index": pi})}), top)},
		textAt("title", cellSlot(invui.HomeTitleCell), lobbyHead, 0.8, []any{96, 7}, with(obj{"layer": 4, "offset": []any{33, 13}}, top)),
		obj{"bar": with(panel([]any{96, 4},
			obj{"bg": rect(lobbyDark, []any{96, 4}, nil, 1)},
			obj{"fill": with(slots("vertical", []any{94, 2}, []any{at("f", "home_fill", cellSlot(invui.HomeFillCell), nil)}), with(obj{"offset": []any{1, 1}, "layer": 2}, top))},
		), with(obj{"layer": 4, "offset": []any{33, 21}}, top))},
		textAt("line", cellSlot(invui.HomeLineCell), lobbyHead, 0.71, []any{96, 6}, with(obj{"layer": 4, "offset": []any{33, 27}}, top)),
	), pos("top_left", 8, 7))

	// The square buttons at the top right, the label as the tooltip, the
	// badge at the corner.
	corners := []any{}
	for i := 0; i < invui.HomeCorners; i++ {
		ci, cb := row(invui.HomeCornerRow0 + i)
		corners = append(corners, at(fmt.Sprintf("c%d", i), "home_corner", cb, obj{"$icon_index": ci, "$badge_index": cellSlot(invui.HomeBadgeCell0 + i)}))
	}
	d["home_badge"] = with(panel([]any{11, 9},
		obj{"edge": rect(lobbyDark, []any{11, 9}, nil, 1)},
		obj{"fill": rect(rgb(0xd8, 0x3a, 0x3a), []any{9, 7}, []any{1, 1}, 2)},
		obj{"l": label(white, 0.79, []any{11, "default"}, obj{"layer": 3, "text_alignment": "center"})},
	), obj{"bindings": shown(filled)})
	d["home_corner"] = with(panel([]any{26, 22},
		obj{"b": with(flatButton([]any{22, 22}, lobbyFill, lobbyHover, lobbyBand, true), obj{"anchor_from": "right_middle", "anchor_to": "right_middle"})},
		obj{"i": with(icon(10), obj{"layer": 3, "offset": []any{-6, -1}, "anchor_from": "right_middle", "anchor_to": "right_middle"})},
		obj{"badge": with(slots("vertical", []any{11, 9}, []any{ref("b", "home_badge", obj{"collection_index": "$badge_index"})}), obj{"layer": 5, "offset": []any{6, -6}, "anchor_from": "top_right", "anchor_to": "top_right"})},
	), obj{"bindings": shown(filled)})
	corner := with(slots("horizontal", []any{"100%c", 22}, corners), pos("top_right", -8, 9))

	// The tiles down the left.
	sides := []any{}
	for i := 0; i < invui.HomeSides; i++ {
		si, sb := row(invui.HomeSideRow0 + i)
		sides = append(sides, at(fmt.Sprintf("s%d", i), "home_tile", sb, obj{"$icon_index": si}))
	}
	d["home_tile"] = with(panel([]any{38, 32},
		obj{"b": with(flatButton([]any{38, 28}, lobbyFill, lobbyHover, lobbyBand, false), top)},
		obj{"i": with(icon(14), obj{"layer": 3, "offset": []any{0, 4}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
		obj{"l": label(white, 0.79, []any{36, "default"}, obj{"layer": 4, "offset": []any{0, 19}, "anchor_from": "top_middle", "anchor_to": "top_middle", "text_alignment": "center"})},
	), obj{"bindings": shown(filled)})
	side := with(slots("vertical", []any{38, "100%c"}, sides), pos("top_left", 8, 53))

	// The feed at the bottom left.
	feeds := []any{}
	for i := 0; i < invui.HomeFeeds; i++ {
		feeds = append(feeds, at(fmt.Sprintf("f%d", i), "home_feed_line", cellSlot(invui.HomeFeedCell0+i), nil))
	}
	d["home_feed_line"] = with(panel([]any{170, 13},
		obj{"bg": with(rect(rgb(0x1e, 0x1e, 0x1f), []any{170, 11}, []any{0, 2}, 1), obj{"alpha": 0.7})},
		obj{"band": rect(rgb(0x52, 0xa5, 0x35), []any{2, 11}, []any{0, 2}, 2)},
		obj{"l": label(white, 0.86, []any{162, "default"}, obj{"layer": 3, "offset": []any{6, 1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	), obj{"bindings": shown(filled)})
	feed := with(slots("vertical", []any{170, "100%c"}, feeds), pos("bottom_left", 8, -9))

	// The chosen save, the play and manage buttons at the bottom right.
	si, sb := row(invui.HomeSaveRow)
	saveLines := []any{}
	for i := 0; i < invui.HomeLines; i++ {
		saveLines = append(saveLines, at(fmt.Sprintf("l%d", i), "home_save_line", cellSlot(invui.HomeLineCell0+i), nil))
	}
	d["home_save_line"] = with(panel([]any{92, "100%c"}, obj{"l": label(lobbyHead, 0.79, []any{"100%", "default"}, top)}), obj{"bindings": shown(filled)})
	arrow := func(dir string) obj {
		return panel([]any{14, 14},
			obj{"b": shownButton(flatButton([]any{14, 14}, lobbyFill, lobbyHover, lobbyBand, false), has(invui.KindRow))},
			obj{"a": image("textures/ui/sg/chevron_"+dir, []any{8, 8}, 3, obj{"bindings": shown(filled)})},
		)
	}
	d["home_prev"] = arrow("left")
	d["home_next"] = arrow("right")
	d["home_save_name"] = panel([]any{92, 10}, obj{"l": label(white, 1.14, []any{"100%", "default"}, top)})
	save := panel([]any{156, 46},
		obj{"edge": rect(lobbyDark, []any{156, 46}, nil, 1)},
		obj{"fill": rect(lobbyPanel, []any{154, 44}, []any{1, 1}, 1)},
		iconAt("pic", si, 30, with(obj{"layer": 3, "offset": []any{5, 8}}, top)),
		obj{"text": with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{92, "100%c"}, "layer": 3, "offset": []any{40, 0}, "controls": []any{
			textAt("head", cellSlot(invui.HomeHeadCell), lobbyHead, 0.71, []any{92, 7}, nil),
			obj{"name": slots("vertical", []any{92, 10}, []any{at("n", "home_save_name", sb, nil)})},
			obj{"lines": slots("vertical", []any{92, "100%c"}, saveLines)},
		}}, obj{"anchor_from": "left_middle", "anchor_to": "left_middle"})},
		obj{"arrows": with(slots("vertical", []any{14, 30}, []any{
			at("prev", "home_prev", invui.SlotPrev, nil),
			obj{"gap": panel([]any{14, 2})},
			at("next", "home_next", invui.SlotNext, nil),
		}), obj{"layer": 3, "offset": []any{-4, 0}, "anchor_from": "right_middle", "anchor_to": "right_middle"})},
	)
	_, playB := row(invui.HomePlayRow)
	d["home_play"] = panel([]any{156, 30},
		obj{"b": flatButton([]any{156, 30}, playFill, playHover, playBand, false)},
		obj{"top": rect(rgb(0x6c, 0xc4, 0x4a), []any{154, 1}, []any{1, 1}, 2)},
		obj{"l": label(white, 1.43, []any{150, "default"}, obj{"layer": 4, "offset": []any{0, -3}, "text_alignment": "center", "shadow": true})},
	)
	d["home_play_sub"] = panel([]any{150, 6}, obj{"l": label(rgb(0xe3, 0xf5, 0xda), 0.79, []any{"100%", "default"}, obj{"text_alignment": "center"})})
	play := panel([]any{156, 30},
		obj{"b": slots("vertical", []any{156, 30}, []any{at("p", "home_play", playB, nil)})},
		obj{"sub": with(slots("vertical", []any{150, 6}, []any{at("s", "home_play_sub", cellSlot(invui.HomeSubCell), nil)}), obj{"layer": 5, "offset": []any{0, -5}, "anchor_from": "bottom_middle", "anchor_to": "bottom_middle"})},
	)
	_, manageB := row(invui.HomeManageRow)
	lightButton := func(w, h int) obj {
		return panel([]any{w, h},
			obj{"b": flatButton([]any{w, h}, lightFill, lightHover, lightBand, false)},
			obj{"l": label(lobbyDark, 0.93, []any{w - 6, "default"}, obj{"layer": 4, "offset": []any{0, -1}, "text_alignment": "center"})},
		)
	}
	d["home_manage"] = with(panel([]any{156, 22}, obj{"b": with(lightButton(156, 18), obj{"anchor_from": "bottom_middle", "anchor_to": "bottom_middle"})}), obj{"bindings": shown(filled)})
	start := with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{156, "100%c"}, "controls": []any{
		obj{"save": save},
		obj{"gap": panel([]any{156, 4})},
		obj{"play": play},
		obj{"manage": slots("vertical", []any{156, "100%c"}, []any{at("m", "home_manage", manageB, nil)})},
	}}, pos("bottom_right", -8, -8))

	// The overlay panel over the dimmed screen.
	overButtons := []any{}
	for i := 0; i < invui.HomeOvers; i++ {
		_, ob := row(invui.HomeOverRow0 + i)
		overButtons = append(overButtons, at(fmt.Sprintf("b%d", i), "home_over_button", ob, nil))
	}
	stripe := func(h int, controls ...any) obj {
		return panel([]any{150, h}, append([]any{obj{"bg": rect(lobbyPanel, []any{150, h}, nil, 1)}}, controls...)...)
	}
	d["home_over_button"] = with(stripe(24, obj{"b": with(lightButton(130, 20), obj{"layer": 2, "anchor_from": "bottom_middle", "anchor_to": "bottom_middle"})}), obj{"bindings": shown(filled)})
	d["home_over_title"] = stripe(12, obj{"l": label(white, 1.14, []any{130, "default"}, obj{"layer": 3, "text_alignment": "center"})})
	d["home_over_body"] = with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{150, "100%c"}, "controls": []any{
		obj{"gap": stripe(4)},
		obj{"text": panel([]any{150, "100%c"}, obj{"l": label(white, 0.93, []any{130, "default"}, obj{"layer": 3, "text_alignment": "center", "anchor_from": "top_middle", "anchor_to": "top_middle", "controls": []any{
			obj{"bg": with(rect(lobbyPanel, []any{"100% + 20px", "100% + 4px"}, nil, -2), obj{"anchor_from": "center", "anchor_to": "center"})},
		}})})},
	}}, obj{"bindings": shown(filled)})
	overlay := panel([]any{"100%", "100%"},
		obj{"dim": with(rect([]any{0.0, 0.0, 0.0}, []any{"100%", "100%"}, nil, 1), obj{"alpha": 0.5})},
		obj{"panel": with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{150, "100%c"}, "layer": 2, "controls": []any{
			obj{"pad": stripe(10)},
			obj{"title": slots("vertical", []any{150, 12}, []any{at("t", "home_over_title", cellSlot(invui.HomeOverTitle), nil)})},
			obj{"body": slots("vertical", []any{150, "100%c"}, []any{at("b", "home_over_body", cellSlot(invui.HomeOverBody), nil)})},
			obj{"buttons": slots("vertical", []any{150, "100%c"}, overButtons)},
			obj{"pad2": stripe(10)},
		}}, obj{"anchor_from": "center", "anchor_to": "center"})},
	)
	d["home_overlay_gate"] = with(panel([]any{"100%", "100%"}, obj{"o": overlay}), obj{"bindings": shown(has(invui.KeyOverlay))})

	d["home_gate"] = with(panel([]any{"100%", "100%"},
		obj{"profile": profile}, obj{"corner": corner}, obj{"side": side}, obj{"feed": feed}, obj{"start": start},
		obj{"overlay": with(slots("vertical", []any{"100%", "100%"}, []any{at("g", "home_overlay_gate", invui.SlotKey, nil)}), obj{"layer": 20})},
	), obj{"bindings": shown(has(invui.KeyHome))})
	d["window_gate"] = with(panel([]any{"100%", "100%"}, ref("w", "window", nil)), obj{"bindings": shown("(((#hover_text - '" + invui.KeyHome + "') = #hover_text) and ((#hover_text - '" + invui.KeyLoading + "') = #hover_text))")})

	// The loading screen over the whole screen.
	line := func(name string, slot int, color []any, scale float64, h int) obj {
		def := "loading_" + name
		d[def] = panel([]any{"100%", h}, obj{"l": label(color, scale, []any{"100%", "default"}, obj{"text_alignment": "center", "shadow": true})})
		return obj{name: slots("vertical", []any{"100%", h}, []any{at("t", def, slot, nil)})}
	}
	d["loading_gate"] = with(panel([]any{"100%", "100%"},
		obj{"bg": rect(rgb(0x2b, 0x4a, 0x7a), []any{"100%", "100%"}, nil, 1)},
		obj{"middle": with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{300, "100%c"}, "layer": 2, "controls": []any{
			obj{"pic": panel([]any{"100%", 32}, iconAt("i", invui.RowIcon0, 32, nil))},
			obj{"gap": panel([]any{"100%", 8})},
			line("title", invui.SlotTitle, white, 1.5, 16),
			line("sub", invui.SlotInfo, lobbyHead, 0.9, 10),
			obj{"gap2": panel([]any{"100%", 10})},
			line("line", invui.SlotBody, white, 1.0, 10),
		}}, obj{"anchor_from": "center", "anchor_to": "center"})},
		obj{"tip": with(slots("vertical", []any{300, 10}, []any{at("t", "loading_tip", invui.Cell0, nil)}), obj{"layer": 2, "anchor_from": "bottom_middle", "anchor_to": "bottom_middle", "offset": []any{0, -12}})},
	), obj{"bindings": shown(has(invui.KeyLoading))})
	d["loading_tip"] = panel([]any{300, 10}, obj{"l": label(lobbyHead, 0.8, []any{"100%", "default"}, obj{"text_alignment": "center", "shadow": true})})

	d["lobby_name"] = panel([]any{"100%c", 18}, obj{"l": label(white, 1.86, []any{"default", "default"}, obj{"shadow": true})})
	d["lobby_gate"] = with(panel([]any{"100%", "100%"},
		obj{"head": with(obj{"type": "stack_panel", "orientation": "horizontal", "size": []any{"100%c", 18}, "controls": []any{
			iconAt("logo", invui.SlotLogo, 16, nil),
			obj{"gap": panel([]any{5, 18})},
			obj{"name": slots("vertical", []any{"100%c", 18}, []any{at("n", "lobby_name", invui.SlotGame, nil)})},
		}}, pos("top_middle", 0, 8))},
	), obj{"bindings": shown(has(invui.KeyLobby))})
}

// onlyWhenVisible makes the bindings of every control that does not set its
// own visibility run only while it is visible, so hidden pages do not read
// their slots (inventory-ui-ddui basic components 1.2). Bindings are copied,
// never changed in place.
func onlyWhenVisible(v any) any {
	switch v := v.(type) {
	case obj:
		out := obj{}
		for k, x := range v {
			out[k] = onlyWhenVisible(x)
		}
		if bs, ok := v["bindings"].([]any); ok && !setsVisible(bs) {
			copied := make([]any, len(bs))
			for i, b := range bs {
				bm, ok := b.(obj)
				if !ok {
					copied[i] = b
					continue
				}
				c := obj{}
				for k, x := range bm {
					c[k] = x
				}
				if c["binding_condition"] == "always" {
					c["binding_condition"] = "visible"
				}
				copied[i] = c
			}
			out["bindings"] = copied
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = onlyWhenVisible(x)
		}
		return out
	}
	return v
}

func setsVisible(bs []any) bool {
	for _, b := range bs {
		if bm, ok := b.(obj); ok && bm["target_property_name"] == "#visible" {
			return true
		}
	}
	return false
}

func main() {
	b, err := json.MarshalIndent(onlyWhenVisible(defs()), "", "  ")
	if err != nil {
		panic(err)
	}
	text := strings.ReplaceAll(string(b), `>`, ">")
	text = strings.ReplaceAll(text, `<`, "<")
	text = strings.ReplaceAll(text, `&`, "&")
	if err := os.WriteFile(out, []byte(text+"\n"), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", out)
}
