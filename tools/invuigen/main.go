// Command invuigen writes assets/resourcepack/inventoryui/ui/sg_menu.json, the
// menu window of common/invui: the slots of its layout read at fixed
// collection indexes, which JSON UI cannot loop over. Run it from the
// repository root: go run ./tools/invuigen
//
// The window follows the bedrock-research entity inventory examples
// (docs/design/entity-inventory-ui.mdx, inventory-ui-ddui): a layout's
// controls are built only in a window titled for it (invui.Title), parts of a
// layout are shown by slot 0's key, and the bindings of a control that does
// not decide its own visibility only run while it is visible.
package main

import (
	"encoding/json"
	"fmt"
	"math"
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
func clean() string { return cleanOf("#hover_text") }

func cleanOf(text string) string {
	s := "(" + text + " - '§o'"
	for _, k := range invui.Kinds {
		s += " - '" + k + "'"
	}
	return s + ")"
}

// faceHead is the front of a face (invui.Face): the count text and the
// padded texture path.
func faceHead() string {
	return fmt.Sprintf("('%%.%ds' * (#hover_text - '§o' - '§f'))", 3*invui.CountChars+invui.FacePathBytes)
}

func facePath() string { return "((" + faceHead() + " - " + countBytes() + ") - ' ')" }

// faceText is the label of a face: what follows its front, without the
// kinds.
func faceText() string { return cleanOf("((#hover_text - '§o' - '§f') - " + faceHead() + ")") }

// faceLabelText is the label of an element's face, read only up to the
// face's fixed length (invui.FaceName): an item's own lines come after it.
func faceLabelText() string {
	whole := fmt.Sprintf("('%%.%ds' * (#hover_text - '§o' - '§f'))", invui.FaceBytes+invui.FaceTextBytes)
	return cleanOf("(" + whole + " - " + faceHead() + ")")
}

// faceLabel is a label of the element's face.
func faceLabel(color []any, scale float64, size []any, props obj) obj {
	l := label(color, scale, size, props)
	l["bindings"] = []any{slotText(), view(faceLabelText(), "#text")}
	return l
}

// press puts the button b at the element's button slot ($press_index),
// shown while cond holds on the element's face ("" for always): the face
// draws the element, and the press empties only the button slot.
func press(b obj, cond string) obj {
	outer := panel([]any{"100%", "100%"})
	for _, k := range []string{"size", "offset", "anchor_from", "anchor_to", "layer"} {
		if v, ok := b[k]; ok {
			outer[k] = v
			delete(b, k)
		}
	}
	b["size"] = []any{"100%", "100%"}
	b["collection_index"] = "$press_index"
	if cond != "" {
		outer["bindings"] = shown(cond)
	}
	outer["controls"] = []any{obj{"s": slots("vertical", []any{"100%", "100%"}, []any{obj{"b": b}})}}
	return outer
}

// countBytes is the count text at the front of a cell's name, after the
// client's own format codes.
func countBytes() string {
	return fmt.Sprintf("('%%.%ds' * (#hover_text - '§o' - '§f'))", 3*invui.CountChars)
}

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

// unless is the "ignored" of a control built only in a window opened for one
// of layouts (invui.Title): an ignored control is never created, where a
// hidden one is.
func unless(layouts ...string) obj {
	s := "$sg_title"
	for _, l := range layouts {
		s += " - '" + invui.TitleMark(l) + "'"
	}
	return obj{"ignored": "((" + s + ") = $sg_title)"}
}

// hasAny holds while the slot's name has one of keys, as one comparison: a
// parenthesized "not" as an operand of "and" or "or" yields no value
// (bedrock-research EXP-29, Windows 1.26.65.1).
func hasAny(keys []string) string {
	s := "#hover_text"
	for _, k := range keys {
		s += " - '" + k + "'"
	}
	return "(not ((" + s + ") = #hover_text))"
}

// numbered is the layouts key followed by each digit from first to last.
func numbered(key string, first, last int) []string {
	out := []string{}
	for n := first; n <= last; n++ {
		out = append(out, key+fmt.Sprint(n))
	}
	return out
}

// layoutGate draws def only in a window opened for one of layouts, and in
// that window while slot 0's key is one of them: a window can draw more
// than one layout (invui.Shared).
func layoutGate(d obj, name, def string, size []any, layouts ...string) obj {
	d[def+"_gate"] = with(panel(size, ref("p", def, nil)), obj{"bindings": shown(hasAny(layouts))})
	return obj{name: with(slots("vertical", size, []any{at("g", def+"_gate", invui.SlotKey, nil)}), unless(layouts...))}
}

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

// face is the props of an element drawn from its face at icon and pressed
// at its button slot (invui.Faces).
func face(icon, button int) obj { return obj{"$icon_index": icon, "$press_index": button} }

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
// the server on touch screens too (entity-inventory-ui 2.5). After a drop
// (button.drop_one) the client drew no change of the window until its next
// input, so drops are not used (measured on Windows 1.26.65.1). Its states
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
		"type": "button", "size": size, "layer": 1, "focus_enabled": false, "focus_magnet_enabled": false,
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

// flatButton is a button of the lobby in the colours of scheme: flat, a dark
// edge and a band under it (oui-mc.css .mc-lobby .mc-btn), one nine-sliced
// picture a state (tools/uitextures flat_*).
func flatButton(size any, scheme string, tip bool) obj {
	tex := "textures/ui/sg/flat_" + scheme
	return button(size, tex, tex+"_hover", tex+"_pressed", tip)
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
	lobbyPanel  = rgb(0x31, 0x32, 0x33)
	lobbyHead   = rgb(0xb1, 0xb2, 0xb5)
	lobbyDark   = rgb(0x1e, 0x1e, 0x1f)
	white       = []any{1.0, 1.0, 1.0}
	chosenGreen = rgb(0x4c, 0xc9, 0x3c)
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

// cellTip is the tooltip of a hovered cell, right of it: the text after the
// cell's face, wrapped to cellTipWidth. The client's own tooltip draws the
// whole name unbroken, the padded path with it, and runs off the screen. It
// sets no visibility of its own, so its text is read only while the cell is
// hovered (onlyWhenVisible).
func cellTip() obj {
	return image("textures/ui/purpleBorder", []any{cellTipWidth + 8, "100%c + 8px"}, 30, obj{
		"allow_clipping": false, "anchor_from": "top_right", "anchor_to": "top_left", "offset": []any{4, 0},
		"controls": []any{obj{"l": label(white, 1.0, []any{cellTipWidth, "default"}, obj{
			"layer": 1, "shadow": true, "anchor_from": "top_left", "anchor_to": "top_left", "offset": []any{4, 4},
			"bindings": []any{slotText(), view(faceText(), "#text")},
		})}},
	})
}

const cellTipWidth = 150

// faceIcon draws the icon of a face (invui.Face): its texture, its item, or
// nothing for a blank one.
func faceIcon(size []any) obj {
	return iconOf(size, facePath(), "((#hover_text - 'textures/' - '"+invui.KindBlank+"') = #hover_text)")
}

func iconOf(size []any, path, isItem string) obj {
	return panel(size,
		obj{"image": obj{
			"type": "image", "size": []any{"100%", "100%"}, "layer": 2, "texture": "#texture",
			"bindings": []any{slotText(), view(path, "#texture"), view(has("textures/"), "#visible")},
		}},
		obj{"item": obj{
			"type": "custom", "renderer": "inventory_item_renderer", "size": []any{"100%", "100%"}, "layer": 2,
			"property_bag": obj{"#item_id_aux": 0},
			"bindings":     []any{slotText(), slotValue("#item_id_aux"), view(isItem, "#visible")},
		}},
	)
}

// countText is the count text of a cell (invui.CountMark) without the
// format signs, after an "r": taking the signs off a text that reads as a
// number gives an empty text (measured on Windows 1.26.65.1), so the reset
// code goes in before and its letter stays. countLabel clips it off.
func countText() string { return "(('§r' + " + countBytes() + ") - '§')" }

// countLabel is the count text at the bottom right of a cell, in a box that
// cuts off the blanks in front of it and the letter before them. A count of
// four or more characters is drawn smaller to fit the cell.
func countLabel(long bool) obj {
	scale, width, cond := 0.86, 20, "(#count_head = '§ § § § ')"
	if long {
		scale, width, cond = 0.62, 19, "(not "+cond+")"
	}
	return obj{
		"type": "panel", "size": []any{width, 10}, "layer": 5, "clips_children": true,
		"anchor_from": "bottom_right", "anchor_to": "bottom_right", "offset": []any{0, 1},
		"bindings": []any{slotText(), view("('%.12s' * (#hover_text - '§o' - '§f'))", "#count_head"), view(cond, "#visible")},
		"controls": []any{obj{"l": obj{
			"type": "label", "localize": false, "color": []any{1, 1, 1}, "shadow": true, "font_scale_factor": scale,
			"anchor_from": "bottom_right", "anchor_to": "bottom_right", "text": "#text",
			"bindings": []any{slotText(), view(countText(), "#text")},
		}}},
	}
}

func defs() obj {
	d := obj{"namespace": "sg_menu"}
	d["icon_slot"] = faceIcon([]any{"100%", "100%"})
	d["empty"] = panel([]any{0, 0})

	textLabel := label(ink, 1.0, []any{"100% - 2px", "default"}, with(obj{"offset": []any{2, 0}, "font_scale_factor": "$text_scale"}, top))
	d["text_slot"] = with(panel([]any{"100%", "100%c"}, obj{"l": textLabel}), obj{"$text_scale|default": 1.0, "bindings": shown(filled)})
	d["title_slot"] = panel([]any{"100%", 10}, obj{"l": label(title, 1.0, []any{"100%", 10}, obj{"text_alignment": "center"})})

	// Tabs over the window.
	tabCells := []any{}
	for i := 0; i < invui.Tabs; i++ {
		tabCells = append(tabCells, at(fmt.Sprintf("t%d", i), "tab_cell", invui.TabIcon0+i, face(invui.TabIcon0+i, invui.TabBtn0+i)))
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
			obj{"b": press(with(button([]any{24, 28}, "textures/ui/TabTopBack", "textures/ui/TabTopBackHover", "textures/ui/TabTopBackHover", true), top), "")},
			obj{"icon": tabIcon(3)},
		), obj{"bindings": shown(has(invui.KindTab))})},
	), obj{"bindings": shown(filled)})

	// Control bar buttons.
	barCells := []any{}
	for i := 0; i < invui.Bars; i++ {
		barCells = append(barCells, at(fmt.Sprintf("b%d", i), "bar_cell", invui.BarIcon0+i, face(invui.BarIcon0+i, invui.BarBtn0+i)))
	}
	d["bar_cell"] = with(panel([]any{15, 22},
		obj{"b": press(button([]any{15, 15}, "", "textures/ui/sg/hover_flat", "textures/ui/sg/press_flat", true), "")},
		obj{"i": with(icon(15), obj{"layer": 3})},
		obj{"q": obj{"type": "label", "layer": 3, "text": "?", "localize": false, "color": title, "bindings": shown(has(invui.KindHelp))}},
	), obj{"bindings": shown(filled)})

	// List rows: rows, rows that cannot be pressed and toggles share one
	// shape; headings and dividers are drawn apart.
	rowCells := []any{}
	for i := 0; i < invui.Rows; i++ {
		rowCells = append(rowCells, at(fmt.Sprintf("r%d", i), "row_cell", invui.RowIcon0+i, face(invui.RowIcon0+i, invui.RowBtn0+i)))
	}
	switchImage := func(tex, marker string) obj {
		return image(tex, []any{30, 16}, 3, obj{"offset": []any{-3, 0}, "anchor_from": "right_middle", "anchor_to": "right_middle", "bindings": shown(has(marker))})
	}
	d["row_cell"] = with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{"100%", "100%c"}, "controls": []any{
		obj{"row": with(panel([]any{"100%", 32},
			obj{"b": press(with(button([]any{"100%", 30}, btnTex, btnHover, btnPress, false), top), "")},
			obj{"dim": image(btnTex, []any{"100%", 30}, 2, with(obj{"alpha": 0.4, "color": []any{0.6, 0.6, 0.6}, "bindings": shown(has(invui.KindStatic))}, top))},
			obj{"i": with(panel([]any{20, 20}, obj{"i": icon(20)}), obj{"layer": 3, "offset": []any{5, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
			obj{"l": faceLabel(light, 1.0, []any{"100% - 34px", "default"}, obj{"layer": 4, "shadow": true, "offset": []any{29, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
			obj{"on": switchImage("textures/ui/toggle_on", invui.KindToggleOn)},
			obj{"off": switchImage("textures/ui/toggle_off", invui.KindToggleOff)},
		), obj{"bindings": shown(has(invui.FamRow))})},
		obj{"heading": with(panel([]any{"100%", 10},
			obj{"l": faceLabel(head, 0.8, []any{"100% - 2px", "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
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
		obj{"icon": with(faceIcon([]any{16, 16}), obj{"layer": 2})},
		obj{"count": countLabel(false)},
		obj{"count_long": countLabel(true)},
		obj{"chosen": image("textures/ui/sg/slot_chosen", []any{18, 18}, 1, obj{"bindings": shown(has(invui.KindChosen))})},
		obj{"done": image("textures/ui/sg/frame1", []any{18, 18}, 3, obj{"color": chosenGreen, "bindings": shown(has(invui.KindDone))})},
		obj{"locked": image("textures/ui/White", []any{16, 16}, 4, obj{"color": []any{0.1, 0.1, 0.12}, "alpha": 0.55, "bindings": shown(has(invui.KindLocked))})},
	)
	d["mini_face"] = panel([]any{18, 18}, obj{"icon": with(faceIcon([]any{12, 12}), obj{"layer": 2})})
	cellButton := button([]any{18, 18}, "", "", "", false)
	cellButton["controls"].([]any)[1].(obj)["hover"].(obj)["controls"] = append(
		cellButton["controls"].([]any)[1].(obj)["hover"].(obj)["controls"].([]any),
		obj{"lit": image("textures/ui/White", []any{16, 16}, 4, obj{"alpha": 0.3})},
		obj{"tip": cellTip()},
	)
	d["cell_button"] = cellButton
	cellText := func(width int, color []any, scale float64) obj {
		return obj{"l": label(color, scale, []any{width, "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})}
	}
	cellKind := func(name, marker string, controls ...any) obj {
		return obj{name: with(panel([]any{18, 18}, controls...), obj{"bindings": shown(has(marker))})}
	}
	d["cell"] = panel([]any{18, 18},
		cellKind("slot", invui.FamSlot, ref("face", "slot_face", nil)),
		cellKind("mini", invui.FamMini, ref("face", "mini_face", nil)),
		cellKind("frame", invui.KindFrame, obj{"f": image("textures/ui/cell_image", []any{18, 18}, 1, nil)}),
		cellKind("arrow", invui.KindArrow, obj{"a": image("textures/ui/arrow_large", []any{15, 10}, 2, nil)}),
		cellKind("text", invui.KindText, cellText(194, ink, 0.7)),
		cellKind("heading", invui.KindHeading, cellText(300, head, 0.8)),
		obj{"press": with(panel([]any{18, 18}, ref("b", "cell_button", nil)), obj{"bindings": shown(hasAny([]string{invui.FamSlot, invui.FamMini}))})},
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
			cards = append(cards, at(fmt.Sprintf("c%d", c), "card_cell", invui.RowIcon0+i, face(invui.RowIcon0+i, invui.RowBtn0+i)))
		}
		cardRows = append(cardRows, obj{fmt.Sprintf("r%d", r): slots("horizontal", []any{304, "100%cm"}, cards)})
	}
	// A card that is a heading starts a row of its own; list rows in the same
	// slots are drawn by the row list instead.
	isCard := "(((#hover_text - '" + invui.KindText + "') = #hover_text) and ((#hover_text - '" + invui.FamRow + "') = #hover_text))"
	d["card_cell"] = with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{76, "100%c"}, "controls": []any{
		obj{"card": with(panel([]any{76, 50},
			obj{"chosen": image("textures/ui/sg/frame2", []any{76, 50}, 5, obj{"color": chosenGreen, "bindings": shown(has(invui.KindChosen))})},
			obj{"b": press(button([]any{72, 46}, btnTex, btnHover, btnPress, false), "")},
			obj{"dim": image(btnTex, []any{72, 46}, 2, obj{"alpha": 0.5, "color": []any{0.6, 0.6, 0.6}, "bindings": shown(has(invui.KindStatic))})},
			obj{"i": with(panel([]any{22, 22}, obj{"i": icon(22)}), obj{"layer": 3, "offset": []any{2, 7}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
			obj{"l": faceLabel(light, 0.8, []any{66, "default"}, obj{"layer": 4, "shadow": true, "offset": []any{0, 31}, "anchor_from": "top_middle", "anchor_to": "top_middle", "text_alignment": "center"})},
		), obj{"bindings": shown(isCard)})},
		obj{"heading": with(panel([]any{76, 13},
			obj{"l": faceLabel(head, 0.8, []any{300, "default"}, obj{"offset": []any{2, 1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
		), obj{"bindings": shown(has(invui.KindText))})},
	}}, obj{"bindings": shown(filled)})

	// Pick cards, three to a row: a dark card edged in its rarity's colour
	// with the tag, the icon, the title and the text down its middle.
	pickRows := []any{}
	for r := 0; r*invui.PickColumns < invui.PickCards; r++ {
		cards := []any{}
		for c := 0; c < invui.PickColumns; c++ {
			i := r*invui.PickColumns + c
			cards = append(cards, at(fmt.Sprintf("c%d", c), "pick_cell", invui.RowIcon0+i, with(face(invui.RowIcon0+i, invui.RowBtn0+i), obj{
				"$tag_index": invui.Cell0 + 2*i, "$text_index": invui.Cell0 + 2*i + 1,
			})))
		}
		name := fmt.Sprintf("pick_row_%d", r)
		d[name] = with(panel([]any{"100%", 106}, obj{"s": with(slots("horizontal", []any{"100%c", 106}, cards), obj{"anchor_from": "top_middle", "anchor_to": "top_middle"})}), obj{"bindings": shown(filled)})
		pickRows = append(pickRows, obj{fmt.Sprintf("r%d", r): slots("vertical", []any{"100%", "100%c"}, []any{at("g", name, invui.RowIcon0+r*invui.PickColumns, nil)})})
	}
	d["pick_line"] = with(panel([]any{88, "100%c"}, obj{"l": label(dim, 1.0, []any{"100%", "default"}, obj{"font_scale_factor": "$line_scale", "text_alignment": "center", "shadow": true})}), obj{"$line_scale|default": 1.0})
	pickLine := func(index string, scale float64, y int) obj {
		return with(slots("vertical", []any{88, "100%c"}, []any{ref("t", "pick_line", obj{"collection_index": index, "$line_scale": scale})}), obj{"layer": 4, "offset": []any{-1, y}, "anchor_from": "top_middle", "anchor_to": "top_middle"})
	}
	edge := func(c []any, marker string) obj {
		r := rect(c, []any{100, 104}, nil, 1)
		if marker != "" {
			r["bindings"] = shown(has(marker))
		}
		return r
	}
	d["pick_cell"] = with(panel([]any{102, 106},
		obj{"common": edge(rgb(0x8b, 0x8b, 0x8e), "")},
		obj{"rare": edge(rgb(0x4a, 0xa3, 0xff), invui.KindRare)},
		obj{"epic": edge(rgb(0xb0, 0x5c, 0xff), invui.KindEpic)},
		obj{"b": press(with(flatButton([]any{94, 98}, "pick", false), obj{"layer": 2, "offset": []any{3, 3}, "anchor_from": "top_left", "anchor_to": "top_left"}), "")},
		obj{"tag": pickLine("$tag_index", 0.8, 7)},
		obj{"i": with(panel([]any{24, 24}, obj{"i": icon(24)}), obj{"layer": 4, "offset": []any{-1, 18}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
		obj{"title": faceLabel(light, 1.0, []any{92, "default"}, obj{"layer": 4, "shadow": true, "text_alignment": "center", "offset": []any{-1, 47}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
		obj{"text": pickLine("$text_index", 0.7, 60)},
	), obj{"bindings": shown(filled)})

	gap := func(h int) obj { return obj{"gap": panel([]any{"100%", h})} }

	// The result: the headline, the stars and the rewards centred, and the
	// lines as a table of labels and values.
	centred := func(first, n int) obj {
		cells := []any{}
		for k := 0; k < n; k++ {
			cells = append(cells, at(fmt.Sprintf("c%d", k), "filled_cell", invui.Cell0+first+k, nil))
		}
		return panel([]any{"100%", 20}, obj{"s": with(slots("horizontal", []any{"100%c", 18}, cells), obj{"anchor_from": "center", "anchor_to": "center"})})
	}
	d["filled_cell"] = with(panel([]any{18, 18}, ref("c", "cell", nil)), obj{"bindings": shown(filled)})
	d["result_head"] = panel([]any{"100%", 22}, obj{"l": label(light, 1.6, []any{"100%", "default"}, obj{"shadow": true, "text_alignment": "center", "anchor_from": "center", "anchor_to": "center"})})
	d["result_value"] = panel([]any{"100%", 11}, obj{"l": label(ink, 0.8, []any{"100%", "default"}, obj{"text_alignment": "right", "anchor_from": "left_middle", "anchor_to": "left_middle"})})
	d["result_line"] = with(panel([]any{200, 12},
		obj{"l": label(head, 0.8, []any{"100%", "default"}, obj{"anchor_from": "left_middle", "anchor_to": "left_middle"})},
		obj{"v": slots("vertical", []any{"100%", 11}, []any{ref("v", "result_value", obj{"collection_index": "$value_index"})})},
		obj{"line": rect([]any{0.55, 0.55, 0.56}, []any{"100%", 1}, []any{0, 11}, 1)},
	), obj{"bindings": shown(filled)})
	resultLines := []any{}
	for k := 0; k < invui.ResultLines; k++ {
		i := invui.Cell0 + invui.ResultLine0 + 2*k
		resultLines = append(resultLines, at(fmt.Sprintf("l%d", k), "result_line", i, obj{"$value_index": i + 1}))
	}
	resultTable := with(slots("vertical", []any{200, "100%c"}, resultLines), obj{"anchor_from": "top_middle", "anchor_to": "top_middle"})
	resultBody := []any{
		obj{"head": slots("vertical", []any{"100%", 22}, []any{at("h", "result_head", invui.SlotInfo, nil)})},
		obj{"stars": centred(invui.ResultStar0, invui.ResultStars)},
		gap(2),
		obj{"table": panel([]any{"100%", "100%c"}, obj{"t": resultTable})},
		gap(3),
		obj{"rewards": centred(invui.ResultReward0, invui.ResultRewards)},
		gap(3),
	}

	// Action buttons (split and list-split screens).
	actionCells := func(first, last int) []any {
		out := []any{}
		for i := first; i < last; i++ {
			out = append(out, at(fmt.Sprintf("a%d", i), "action_cell", invui.RowIcon0+i, face(invui.RowIcon0+i, invui.RowBtn0+i)))
		}
		return out
	}
	d["action_cell"] = with(panel([]any{"100%", 19},
		obj{"b": press(with(button([]any{"100%", 17}, btnTex, btnHover, btnPress, false), top), "")},
		obj{"i": with(panel([]any{12, 12}, obj{"i": icon(12)}), obj{"layer": 3, "offset": []any{2, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
		obj{"l": faceLabel(light, 0.8, []any{"100% - 17px", "default"}, obj{"layer": 4, "shadow": true, "offset": []any{15, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	), obj{"bindings": shown(has(invui.KindRow))})

	column := func(width string, controls ...any) obj {
		return with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{width, "100%c"}, "controls": controls}, top)
	}
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
	named := func(controls []any) []any {
		out := make([]any, len(controls))
		for i, c := range controls {
			if o, ok := c.(obj); ok && o["type"] != nil {
				c = obj{fmt.Sprintf("c%d", i): o}
			}
			out[i] = c
		}
		return out
	}
	// gate shows controls while slot 0's key contains key.
	gate := func(name, key string, controls ...any) obj {
		d["gate_"+name] = with(panel([]any{"100%", "100%c"}, named(controls)...), obj{"bindings": shown(has(key))})
		return obj{name: slots("vertical", []any{"100%", "100%c"}, []any{at("g", "gate_"+name, invui.SlotKey, nil)})}
	}
	// part builds controls only in a window opened for layout.
	part := func(name, layout string, controls ...any) obj {
		d["part_"+name] = panel([]any{"100%", "100%c"}, named(controls)...)
		return layoutGate(d, name, "part_"+name, []any{"100%", "100%c"}, layout)
	}

	// The list content: the body, the content of the layout and the rows.
	board := cellGrid(0, invui.Cells/invui.BoardColumns, invui.BoardColumns)
	board["anchor_from"], board["anchor_to"] = "top_middle", "top_middle"
	d["field_row"] = with(panel([]any{"100%", 24},
		obj{"edge": rect(rgb(0xa0, 0xa0, 0xa6), []any{"100%", 20}, []any{0, 0}, 1)},
		obj{"fill": rect(rgb(0x31, 0x32, 0x33), []any{"100% - 2px", 18}, []any{1, 1}, 1)},
		obj{"b": press(with(button([]any{"100%", 20}, "", "textures/ui/sg/hover_flat", "textures/ui/sg/press_flat", false), obj{"alpha": 0.25, "anchor_from": "top_left", "anchor_to": "top_left"}), "")},
		obj{"i": with(panel([]any{12, 12}, obj{"i": icon(12)}), obj{"layer": 3, "offset": []any{4, 4}, "anchor_from": "top_left", "anchor_to": "top_left"})},
		obj{"l": faceLabel(white, 0.9, []any{"100% - 24px", "default"}, obj{"layer": 4, "offset": []any{20, 6}, "anchor_from": "top_left", "anchor_to": "top_left"})},
	), obj{"bindings": shown(has(invui.KindField))})
	d["list_page_body"] = column("100% - 8px",
		obj{"field": slots("vertical", []any{"100%", "100%c"}, []any{at("f", "field_row", invui.RowIcon0+invui.RowField, face(invui.RowIcon0+invui.RowField, invui.RowBtn0+invui.RowField))})},
		body(1.0), gap(6),
		part("cards", invui.KeyCards, column("100%", cardRows...)),
		part("pick", invui.KeyPick, column("100%", pickRows...), gap(3)),
		part("result", invui.KeyResult, column("100%", resultBody...)),
		part("grid", invui.KeyGrid, cellGrid(0, invui.Cells/invui.GridColumns, invui.GridColumns), gap(3)),
		part("board", invui.KeyBoard, panel([]any{"100%", "100%c"}, obj{"g": board}), gap(3)),
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
	d["detail_name"] = panel([]any{"fill", 28}, obj{"l": faceLabel(ink, 0.8, []any{"100% - 4px", "default"}, obj{"offset": []any{4, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})})
	d["split_right_body"] = column("100% - 8px",
		obj{"head": slots("horizontal", []any{"100%", 30}, []any{
			at("big", "detail_big", invui.RowBtn0, face(invui.RowIcon0, invui.RowBtn0)),
			at("name", "detail_name", invui.RowIcon0, nil),
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
		listCells = append(listCells, at(fmt.Sprintf("r%d", i), "list_cell", invui.RowIcon0+i, with(face(invui.RowIcon0+i, invui.RowBtn0+i), obj{"$info_index": invui.Cell0 + i})))
	}
	infoLine := slots("vertical", []any{"100% - 30px", 8}, []any{ref("t", "info_slot", obj{"collection_index": "$info_index"})})
	d["info_slot"] = panel([]any{"100%", 8}, obj{"l": label(dim, 0.7, []any{"100%", "default"}, top)})
	d["list_cell"] = with(panel([]any{"100%", 32},
		obj{"b": press(button([]any{"100%", 30}, btnTex, btnHover, btnPress, false), has(invui.KindRow))},
		obj{"chosen": image(btnHover, []any{"100%", 30}, 1, with(obj{"bindings": shown(has(invui.KindChosen))}, top))},
		obj{"i": with(panel([]any{20, 20}, obj{"i": icon(20)}), obj{"layer": 3, "offset": []any{5, -1}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
		obj{"name": faceLabel(light, 1.0, []any{"100% - 30px", 9}, with(obj{"layer": 4, "shadow": true, "offset": []any{27, 5}}, top))},
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
	facePager := func(arrow string) obj {
		return with(panel([]any{18, 18},
			obj{"b": press(button([]any{18, 18}, btnTex, btnHover, btnPress, false), has(invui.KindRow))},
			obj{"off": image(btnTex, []any{18, 18}, 1, obj{"alpha": 0.4, "bindings": shown(has(invui.KindStatic))})},
			obj{"a": image("textures/ui/sg/arrow_"+arrow, []any{12, 12}, 3, nil)},
		), obj{"bindings": shown(filled)})
	}
	d["range_minus"] = facePager("left")
	d["range_plus"] = facePager("right")
	d["range_value"] = with(panel([]any{50, 18},
		obj{"l": label(ink, 1.0, []any{"100% - 4px", "default"}, obj{"offset": []any{4, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	), obj{"bindings": shown(filled)})
	d["range_heading"] = with(panel([]any{"100%", 13},
		obj{"l": label(head, 0.8, []any{"100% - 2px", "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	), obj{"bindings": shown(filled)})
	axisLine := func(i int) obj {
		return obj{fmt.Sprintf("a%d", i): slots("horizontal", []any{"100%", "100%cm"}, []any{
			at("m", "range_minus", invui.RowIcon0+2*i, face(invui.RowIcon0+2*i, invui.RowBtn0+2*i)),
			at("v", "range_value", invui.Cell0+i, nil),
			at("p", "range_plus", invui.RowIcon0+2*i+1, face(invui.RowIcon0+2*i+1, invui.RowBtn0+2*i+1)),
		})}
	}
	heading := func(name string, cell int) obj {
		return obj{name: slots("vertical", []any{"100%", "100%c"}, []any{at("h", "range_heading", invui.Cell0+cell, nil)})}
	}
	rangeRows := []any{}
	for i := invui.RangeRow0; i < invui.RangeRow0+4; i++ {
		rangeRows = append(rangeRows, at(fmt.Sprintf("r%d", i), "action_cell", invui.RowIcon0+i, face(invui.RowIcon0+i, invui.RowBtn0+i)))
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
				"button_mappings":         []any{obj{"from_button_id": "button.menu_select", "to_button_id": "button.turn_structure", "mapping_type": "pressed", "button_up_right_of_first_refusal": true}},
				"controls":                views,
			}}), obj{"clips_children": true, "anchor_from": "center", "anchor_to": "center"})},
		}}, top))},
		obj{"right": with(panel([]any{170, 142}, ref("s", "range_right_scroll", nil)), obj{"anchor_from": "top_right", "anchor_to": "top_right"})},
	)

	homeDefs(d)

	pages := []any{}
	for _, p := range []struct {
		layouts []string
		def     string
	}{
		{invui.ListLayouts, "list_page"}, {[]string{invui.KeySplit}, "split_page"}, {[]string{invui.KeyListSplit}, "listsplit_page"}, {[]string{invui.KeyRange}, "range_page"},
	} {
		pages = append(pages, layoutGate(d, strings.ReplaceAll(p.def, "_page", ""), p.def, []any{"100%", "100%"}, p.layouts...))
	}

	window := func(w, h int, pages []any) obj {
		tabs := with(slots("horizontal", []any{"100%c", 24}, tabCells), obj{"offset": []any{8, -22}, "layer": 0})
		return panel([]any{w, h},
			obj{"frame": image("textures/ui/dialog_background_opaque", []any{"100%", "100%"}, 1, nil)},
			obj{"title": with(panel([]any{w - 16, 10}, obj{"s": slots("vertical", []any{w - 16, 10}, []any{at("t", "title_slot", invui.SlotTitle, nil)})}), with(obj{"offset": []any{8, 6}, "layer": 3}, top))},
			obj{"tabs": with(tabs, top)},
			ref("tab_keys", "tab_keys", nil),
			obj{"toolbar": image("textures/ui/toolbar_background", []any{"100%c + 14px", 25}, 4, obj{
				"offset": []any{0, -22}, "anchor_from": "top_right", "anchor_to": "top_right",
				"controls": []any{obj{"row": with(obj{
					"type": "stack_panel", "orientation": "horizontal", "size": []any{"100%c", 22}, "offset": []any{7, 1},
					"controls": []any{
						obj{"bar": slots("horizontal", []any{"100%c", 22}, barCells)},
						obj{"close": panel([]any{17, 22}, obj{"x": with(slots("vertical", []any{15, 15}, []any{
							obj{"b": with(button([]any{15, 15}, "textures/ui/close_button_default_light", "textures/ui/close_button_hover_light", "textures/ui/close_button_pressed_light", false), obj{"collection_index": invui.SlotEsc})},
						}), obj{"anchor_from": "center", "anchor_to": "center"})})},
					},
				}, top)}},
			})},
			obj{"content": with(panel([]any{w - 12, h - 24}, obj{"pages": panel([]any{"100%", "100%"}, pages...)}), with(obj{"layer": 2, "offset": []any{6, 18}}, top))},
		)
	}
	d["window"] = window(326, 166, pages)
	// The tab keys press the tab left or right of the one shown: the key
	// of tab i is drawn while tab i+1 (or i-1) is the one shown.
	keyButton := func(from string) obj {
		return obj{
			"type": "button", "size": []any{1, 1}, "layer": 0, "collection_index": "$press_index",
			"default_control": "", "hover_control": "", "pressed_control": "",
			"button_mappings": []any{obj{"from_button_id": from, "to_button_id": "button.container_auto_place", "mapping_type": "global"}},
			"bindings":        []any{obj{"binding_type": "collection_details", "binding_collection_name": "container_items"}},
		}
	}
	for _, k := range []string{"left", "right"} {
		d["tab_key_"+k] = with(panel([]any{1, 1}, obj{"s": slots("vertical", []any{1, 1}, []any{obj{"b": keyButton("button.menu_tab_" + k)}})}), obj{"bindings": shown(has(invui.KindTabOn))})
	}
	tabKeys := []any{}
	for i := 0; i+1 < invui.Tabs; i++ {
		tabKeys = append(tabKeys,
			at(fmt.Sprintf("l%d", i), "tab_key_left", invui.TabIcon0+i+1, obj{"$press_index": invui.TabBtn0 + i}),
			at(fmt.Sprintf("r%d", i), "tab_key_right", invui.TabIcon0+i, obj{"$press_index": invui.TabBtn0 + i + 1}),
		)
	}
	// From the last tab the right key presses the first, and from the first
	// the left key presses the last: a chain of slots that must hold.
	type need struct {
		slot int
		cond string
	}
	chain := func(key string, press int, needs ...need) obj {
		var c any = obj{"b": with(keyButton("button.menu_tab_"+key), obj{"$press_index": press})}
		for k := len(needs) - 1; k >= 0; k-- {
			c = obj{"s": slots("vertical", []any{1, 1}, []any{obj{"g": with(panel([]any{1, 1}, c), obj{
				"collection_index": needs[k].slot, "bindings": shown(needs[k].cond),
			})}})}
		}
		return panel([]any{1, 1}, c)
	}
	empty := "(#hover_text = '')"
	wraps := []any{}
	for j := 1; j < invui.Tabs; j++ {
		last := []need{{invui.TabIcon0 + j, filled}}
		if j+1 < invui.Tabs {
			last = append(last, need{invui.TabIcon0 + j + 1, empty})
		}
		wraps = append(wraps,
			obj{fmt.Sprintf("wr%d", j): chain("right", invui.TabBtn0, append([]need{{invui.TabIcon0 + j, has(invui.KindTabOn)}}, last[1:]...)...)},
			obj{fmt.Sprintf("wl%d", j): chain("left", invui.TabBtn0+j, append([]need{{invui.TabIcon0, has(invui.KindTabOn)}}, last...)...)},
		)
	}
	d["tab_keys"] = panel([]any{1, 1}, append([]any{obj{"steps": slots("vertical", []any{1, 1}, tabKeys)}}, wraps...)...)
	d["esc_button"] = obj{
		"type": "button", "size": []any{1, 1}, "layer": 0,
		"default_control": "", "hover_control": "", "pressed_control": "",
		"button_mappings": []any{
			obj{"from_button_id": "button.menu_cancel", "to_button_id": "button.container_auto_place", "mapping_type": "global"},
		},
		"bindings": []any{obj{"binding_type": "collection_details", "binding_collection_name": "container_items"}},
	}
	questDefs(d, gap)
	d["quest_window"] = window(questW, questH, []any{ref("quest", "quest_page", nil)})

	full := []any{"100%", "100%"}
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
					"button_mappings": []any{
						obj{"from_button_id": "button.menu_cancel", "to_button_id": "button.menu_exit", "mapping_type": "global"},
						obj{"from_button_id": "button.menu_inventory_cancel", "to_button_id": "button.menu_exit", "mapping_type": "global"},
					},
					"controls": []any{
						obj{"esc": slots("vertical", []any{1, 1}, []any{at("b", "esc_button", invui.SlotEsc, nil)})},
						obj{"sync": obj{"type": "grid", "size": []any{0, 0}, "maximum_grid_items": 27, "collection_name": "container_items", "grid_item_template": "sg_menu.empty"}},
						layoutGate(d, "quest", "quest_window", full, numbered(invui.KeyQuest, 0, len(invui.QuestGrids)-1)...),
						layoutGate(d, "window", "window", full, invui.WindowLayouts...),
						layoutGate(d, "home", "home_page", full, invui.KeyHome),
						obj{"lobby": slots("vertical", full, []any{at("g", "lobby_gate", invui.SlotKey, nil)})},
						layoutGate(d, "loading", "loading_page", full, invui.KeyLoading),
						layoutGate(d, "radial", "radial_page", full, numbered(invui.KeyRadial, 1, invui.RadialItems)...),
					},
				}},
			},
		}}},
	}
	return d
}

// The quest book's window, its chapter list and the cells of its canvas: a
// quest is an 18 pixel shape at a cell's top left, the rest of the cell the
// gaps a line runs through, its lane at laneX.
const (
	questW, questH         = 410, 196
	questSideW             = 116
	questCellW, questCellH = 24, 24
	laneX                  = 21
)

func questDefs(d obj, gap func(int) obj) {
	// The pictures of a cell are named from the digits of its code
	// (invui.QuestCode) after its face: no picture is drawn hidden.
	text := "(#hover_text - '§o' - '§f')"
	digit := func(k int) string {
		at := invui.FaceBytes + 3*k
		return fmt.Sprintf("(('%%.%ds' * %s) - ('%%.%ds' * %s))", at+3, text, at+2, text)
	}
	pictured := func(name, texture string, size, offset []any, layer int) obj {
		return obj{name: image("#texture", size, layer, with(obj{"offset": offset,
			"bindings": []any{slotText(), view("('textures/ui/sg/"+texture, "#texture")}}, top))}
	}
	parts := []any{obj{"canvas": rect(rgb(0x1c, 0x1c, 0x22), []any{questCellW, questCellH}, nil, 0)}}
	for p, ln := range []struct {
		name         string
		size, offset []any
	}{
		{"n", []any{2, 10}, []any{8, 0}},
		{"s", []any{2, questCellH - 8}, []any{8, 8}},
		{"w", []any{10, 2}, []any{0, 8}},
		{"e", []any{10, 2}, []any{8, 8}},
		{"gw", []any{laneX + 2 - 18, 2}, []any{18, 8}},
		{"ge", []any{questCellW - laneX, 2}, []any{laneX, 8}},
		{"gn", []any{2, 10}, []any{laneX, 0}},
		{"gs", []any{2, questCellH - 8}, []any{laneX, 8}},
	} {
		parts = append(parts, pictured(ln.name, "ql' + "+digit(1+p)+" + "+digit(0)+")", ln.size, ln.offset, 1))
	}
	shapeDigit, stateDigit := digit(1+len(invui.QuestPieces)), digit(2+len(invui.QuestPieces))
	fill := pictured("fill", "qf' + "+shapeDigit+")", []any{18, 18}, []any{0, 0}, 2)
	fill["fill"].(obj)["color"] = rgb(0x2c, 0x2c, 0x34)
	node := []any{
		fill,
		pictured("edge", "qe' + "+shapeDigit+" + "+stateDigit+")", []any{18, 18}, []any{0, 0}, 3),
	}
	node = append(node,
		obj{"icon": with(faceIcon([]any{12, 12}), obj{"layer": 4, "offset": []any{3, 3}, "anchor_from": "top_left", "anchor_to": "top_left"})},
		obj{"locked": image("textures/ui/sg/quest_locked", []any{14, 14}, 6, with(obj{"offset": []any{2, 2}, "bindings": shown(has(invui.KindLocked))}, top))},
		obj{"chosen": image("textures/ui/sg/slot_chosen", []any{18, 18}, 5, with(obj{"bindings": shown(has(invui.KindChosen))}, top))},
		obj{"new": rect(rgb(0xff, 0xd7, 0x5e), []any{4, 4}, []any{14, 0}, 7)},
		ref("b", "cell_button", obj{"anchor_from": "top_left", "anchor_to": "top_left"}),
	)
	node[len(node)-2].(obj)["new"].(obj)["bindings"] = shown(has(invui.KindNew))
	d["quest_cell"] = panel([]any{questCellW, questCellH},
		obj{"map": with(panel([]any{questCellW, questCellH}, parts...), top)},
		obj{"node": with(panel([]any{18, 18}, node...), with(obj{"bindings": shown(has(invui.FamSlot))}, top))},
	)

	canvasW := questW - 12 - questSideW - 4
	for g, size := range invui.QuestGrids {
		cols, rows := size[0], size[1]
		lines := []any{}
		for r := 0; r < rows; r++ {
			start := invui.Cell0 + r*cols
			cells := []any{}
			for c := 0; c < cols; c++ {
				cells = append(cells, at(fmt.Sprintf("c%d", c), "quest_cell", start+c, nil))
			}
			name := fmt.Sprintf("quest_row_%d_%d", g, r)
			d[name] = with(panel([]any{cols * questCellW, questCellH}, obj{"cells": slots("horizontal", []any{cols * questCellW, questCellH}, cells)}), obj{"bindings": shown(filled)})
			lines = append(lines, at(fmt.Sprintf("r%d", r), name, start, nil))
		}
		body := fmt.Sprintf("quest_canvas_%d", g)
		d[body] = with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{canvasW - 6, "100%c"}, "controls": []any{
			gap(4),
			obj{"grid": panel([]any{"100%", "100%c"}, obj{"g": with(slots("vertical", []any{cols * questCellW, "100%c"}, lines), obj{"anchor_from": "top_middle", "anchor_to": "top_middle"})})},
			gap(4),
		}}, top)
		d[fmt.Sprintf("quest_scroll_%d@common.scrolling_panel", g)] = obj{
			"anchor_to": "top_left", "anchor_from": "top_left", "$show_background": false, "size": []any{"100%", "100%"},
			"$scroll_size": []any{4, "100%"}, "$scrolling_pane_size": []any{"100%", "100%"}, "$scrolling_pane_offset": []any{0, 0},
			"$scroll_bar_right_padding_size": []any{0, 0}, "$scrolling_content": "sg_menu." + body,
		}
		d[fmt.Sprintf("quest_grid_%d", g)] = panel([]any{"100%", "100%"}, ref("s", fmt.Sprintf("quest_scroll_%d", g), nil))
	}
	grids := []any{}
	for g := range invui.QuestGrids {
		grids = append(grids, layoutGate(d, fmt.Sprintf("g%d", g), fmt.Sprintf("quest_grid_%d", g), []any{"100%", "100%"}, invui.KeyQuest+fmt.Sprint(g)))
	}

	chapters := []any{}
	for i := 0; i < invui.QuestChapters; i++ {
		chapters = append(chapters, at(fmt.Sprintf("r%d", i), "quest_chapter", invui.RowIcon0+i, with(face(invui.RowIcon0+i, invui.RowBtn0+i), obj{"$count_index": invui.RowBtn0 + invui.QuestChapters + i})))
	}
	d["quest_count"] = panel([]any{30, 10}, obj{"l": label(dim, 0.7, []any{"100%", "default"}, obj{"text_alignment": "right", "anchor_from": "right_middle", "anchor_to": "right_middle"})})
	d["quest_chapter"] = with(panel([]any{"100%", 22},
		obj{"b": press(with(button([]any{"100%", 20}, btnTex, btnHover, btnPress, false), top), has(invui.KindRow))},
		obj{"on": image(btnHover, []any{"100%", 20}, 1, with(obj{"bindings": shown(has(invui.KindChosen))}, top))},
		obj{"i": with(panel([]any{14, 14}, obj{"i": icon(14)}), obj{"layer": 3, "offset": []any{3, 3}, "anchor_from": "top_left", "anchor_to": "top_left"})},
		obj{"l": faceLabel(light, 0.75, []any{"100% - 46px", "default"}, obj{"layer": 4, "shadow": true, "offset": []any{20, 6}, "anchor_from": "top_left", "anchor_to": "top_left"})},
		obj{"lon": faceLabel(rgb(0xff, 0xd7, 0x5e), 0.75, []any{"100% - 46px", "default"}, obj{"layer": 5, "shadow": true, "offset": []any{20, 6}, "anchor_from": "top_left", "anchor_to": "top_left"})},
		obj{"n": with(slots("vertical", []any{30, 10}, []any{ref("c", "quest_count", obj{"collection_index": "$count_index"})}), obj{"layer": 4, "offset": []any{-4, 5}, "anchor_from": "top_right", "anchor_to": "top_right"})},
		obj{"new": rect(rgb(0xff, 0xd7, 0x5e), []any{4, 4}, []any{2, 2}, 6)},
	), obj{"bindings": shown(filled)})
	ch := d["quest_chapter"].(obj)["controls"].([]any)
	ch[3].(obj)["l"].(obj)["bindings"] = append(ch[3].(obj)["l"].(obj)["bindings"].([]any), view("((#hover_text - '"+invui.KindChosen+"') = #hover_text)", "#visible"))
	ch[4].(obj)["lon"].(obj)["bindings"] = append(ch[4].(obj)["lon"].(obj)["bindings"].([]any), view(has(invui.KindChosen), "#visible"))
	ch[6].(obj)["new"].(obj)["bindings"] = shown(has(invui.KindNew))
	d["quest_side_body"] = with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{"100% - 6px", "100%c"}, "controls": []any{
		obj{"rows": slots("vertical", []any{"100%", "100%c"}, chapters)},
	}}, top)
	d["quest_side_scroll@common.scrolling_panel"] = obj{
		"anchor_to": "top_left", "anchor_from": "top_left", "$show_background": false, "size": []any{"100%", "100%"},
		"$scroll_size": []any{4, "100%"}, "$scrolling_pane_size": []any{"100%", "100%"}, "$scrolling_pane_offset": []any{0, 0},
		"$scroll_bar_right_padding_size": []any{0, 0}, "$scrolling_content": "sg_menu.quest_side_body",
	}
	d["quest_heading"] = panel([]any{canvasW, 12}, obj{"l": label(ink, 0.9, []any{"100%", "default"}, obj{"anchor_from": "left_middle", "anchor_to": "left_middle"})})
	d["quest_page"] = with(panel([]any{"100%", "100%"},
		obj{"side": with(panel([]any{questSideW, "100%"}, ref("s", "quest_side_scroll", nil)), top)},
		obj{"heading": with(slots("vertical", []any{canvasW, 12}, []any{at("h", "quest_heading", invui.SlotInfo, nil)}), with(obj{"offset": []any{questSideW + 4, 0}}, top))},
		obj{"canvas": with(panel([]any{canvasW, questH - 24 - 14},
			obj{"bg": rect(rgb(0x1c, 0x1c, 0x22), []any{"100%", "100%"}, nil, 0)},
			obj{"edge": with(rect(rgb(0x55, 0x55, 0x5c), []any{"100% + 2px", "100% + 2px"}, []any{-1, -1}, -1), top)},
			obj{"grids": with(panel([]any{"100%", "100%"}, grids...), obj{"layer": 1})},
		), with(obj{"offset": []any{questSideW + 4, 14}}, top))},
	), obj{"bindings": shown(has(invui.KeyQuest))})
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
		obj{"b": press(flatButton([]any{135, 37}, "lobby", false), "")},
		obj{"head": with(icon(23), with(obj{"layer": 3, "offset": []any{5, 6}}, top))},
		obj{"name": faceLabel(white, 1.14, []any{96, "default"}, with(obj{"layer": 4, "offset": []any{33, 4}}, top))},
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
		obj{"card": with(slots("vertical", []any{135, 37}, []any{at("c", "home_profile", pi, face(pi, pb))}), top)},
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
		corners = append(corners, at(fmt.Sprintf("c%d", i), "home_corner", ci, with(face(ci, cb), obj{"$badge_index": cellSlot(invui.HomeBadgeCell0 + i)})))
	}
	d["home_badge"] = with(panel([]any{11, 9},
		obj{"edge": rect(lobbyDark, []any{11, 9}, nil, 1)},
		obj{"fill": rect(rgb(0xd8, 0x3a, 0x3a), []any{9, 7}, []any{1, 1}, 2)},
		obj{"l": label(white, 0.79, []any{11, "default"}, obj{"layer": 3, "text_alignment": "center"})},
	), obj{"bindings": shown(filled)})
	d["home_corner"] = with(panel([]any{26, 22},
		obj{"b": press(with(flatButton([]any{22, 22}, "lobby", true), obj{"anchor_from": "right_middle", "anchor_to": "right_middle"}), "")},
		obj{"i": with(icon(10), obj{"layer": 3, "offset": []any{-6, -1}, "anchor_from": "right_middle", "anchor_to": "right_middle"})},
		obj{"badge": with(slots("vertical", []any{11, 9}, []any{ref("b", "home_badge", obj{"collection_index": "$badge_index"})}), obj{"layer": 5, "offset": []any{6, -6}, "anchor_from": "top_right", "anchor_to": "top_right"})},
	), obj{"bindings": shown(filled)})
	corner := with(slots("horizontal", []any{"100%c", 22}, corners), pos("top_right", -8, 9))

	// The tiles down the left.
	sides := []any{}
	for i := 0; i < invui.HomeSides; i++ {
		si, sb := row(invui.HomeSideRow0 + i)
		sides = append(sides, at(fmt.Sprintf("s%d", i), "home_tile", si, with(face(si, sb), obj{"$badge_index": cellSlot(invui.HomeSideBadge0 + i)})))
	}
	d["home_tile"] = with(panel([]any{38, 32},
		obj{"b": press(with(flatButton([]any{38, 28}, "lobby", false), top), "")},
		obj{"i": with(icon(14), obj{"layer": 3, "offset": []any{0, 4}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
		obj{"l": faceLabel(white, 0.79, []any{36, "default"}, obj{"layer": 4, "offset": []any{0, 19}, "anchor_from": "top_middle", "anchor_to": "top_middle", "text_alignment": "center"})},
		obj{"badge": with(slots("vertical", []any{11, 9}, []any{ref("b", "home_badge", obj{"collection_index": "$badge_index"})}), obj{"layer": 5, "offset": []any{4, -3}, "anchor_from": "top_right", "anchor_to": "top_right"})},
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
			obj{"b": shownButton(flatButton([]any{14, 14}, "lobby", false), has(invui.KindRow))},
			obj{"a": image("textures/ui/sg/chevron_"+dir, []any{8, 8}, 3, obj{"bindings": shown(filled)})},
		)
	}
	d["home_prev"] = arrow("left")
	d["home_next"] = arrow("right")
	d["home_save_name"] = panel([]any{92, 10}, obj{"l": faceLabel(white, 1.14, []any{"100%", "default"}, top)})
	save := panel([]any{156, 46},
		obj{"edge": rect(lobbyDark, []any{156, 46}, nil, 1)},
		obj{"fill": rect(lobbyPanel, []any{154, 44}, []any{1, 1}, 1)},
		obj{"pic": with(icon(30), with(obj{"layer": 3, "offset": []any{5, 8}, "$icon_index": si, "$press_index": sb}, top))},
		obj{"text": with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{92, "100%c"}, "layer": 3, "offset": []any{40, 0}, "controls": []any{
			textAt("head", cellSlot(invui.HomeHeadCell), lobbyHead, 0.71, []any{92, 7}, nil),
			obj{"name": slots("vertical", []any{92, 10}, []any{at("n", "home_save_name", si, nil)})},
			obj{"lines": slots("vertical", []any{92, "100%c"}, saveLines)},
		}}, obj{"anchor_from": "left_middle", "anchor_to": "left_middle"})},
		obj{"arrows": with(slots("vertical", []any{14, 30}, []any{
			at("prev", "home_prev", invui.SlotPrev, nil),
			obj{"gap": panel([]any{14, 2})},
			at("next", "home_next", invui.SlotNext, nil),
		}), obj{"layer": 3, "offset": []any{-4, 0}, "anchor_from": "right_middle", "anchor_to": "right_middle"})},
	)
	playI, playB := row(invui.HomePlayRow)
	d["home_play"] = panel([]any{156, 30},
		obj{"b": press(flatButton([]any{156, 30}, "play", false), "")},
		obj{"top": rect(rgb(0x6c, 0xc4, 0x4a), []any{154, 1}, []any{1, 1}, 2)},
		obj{"l": faceLabel(white, 1.43, []any{150, "default"}, obj{"layer": 4, "offset": []any{0, -3}, "text_alignment": "center", "shadow": true})},
	)
	d["home_play_sub"] = panel([]any{150, 6}, obj{"l": label(rgb(0xe3, 0xf5, 0xda), 0.79, []any{"100%", "default"}, obj{"text_alignment": "center"})})
	play := panel([]any{156, 30},
		obj{"b": slots("vertical", []any{156, 30}, []any{at("p", "home_play", playI, face(playI, playB))})},
		obj{"sub": with(slots("vertical", []any{150, 6}, []any{at("s", "home_play_sub", cellSlot(invui.HomeSubCell), nil)}), obj{"layer": 5, "offset": []any{0, -5}, "anchor_from": "bottom_middle", "anchor_to": "bottom_middle"})},
	)
	manageI, manageB := row(invui.HomeManageRow)
	lightButton := func(w, h int) obj {
		return panel([]any{w, h},
			obj{"b": press(flatButton([]any{w, h}, "light", false), "")},
			obj{"l": faceLabel(lobbyDark, 0.93, []any{w - 6, "default"}, obj{"layer": 4, "offset": []any{0, -1}, "text_alignment": "center"})},
		)
	}
	d["home_manage"] = with(panel([]any{156, 22}, obj{"b": with(lightButton(156, 18), obj{"anchor_from": "bottom_middle", "anchor_to": "bottom_middle"})}), obj{"bindings": shown(filled)})
	start := with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{156, "100%c"}, "controls": []any{
		obj{"save": save},
		obj{"gap": panel([]any{156, 4})},
		obj{"play": play},
		obj{"manage": slots("vertical", []any{156, "100%c"}, []any{at("m", "home_manage", manageI, face(manageI, manageB))})},
	}}, pos("bottom_right", -8, -8))

	// The overlay panel over the dimmed screen.
	overButtons := []any{}
	for i := 0; i < invui.HomeOvers; i++ {
		oi, ob := row(invui.HomeOverRow0 + i)
		overButtons = append(overButtons, at(fmt.Sprintf("b%d", i), "home_over_button", oi, face(oi, ob)))
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

	d["home_page"] = panel([]any{"100%", "100%"},
		obj{"profile": profile}, obj{"corner": corner}, obj{"side": side}, obj{"feed": feed}, obj{"start": start},
		obj{"overlay": with(slots("vertical", []any{"100%", "100%"}, []any{at("g", "home_overlay_gate", invui.SlotKey, nil)}), obj{"layer": 20})},
	)

	// The loading screen over the whole screen.
	line := func(name string, slot int, color []any, scale float64, h int) obj {
		def := "loading_" + name
		d[def] = panel([]any{"100%", h}, obj{"l": label(color, scale, []any{"100%", "default"}, obj{"text_alignment": "center", "shadow": true})})
		return obj{name: slots("vertical", []any{"100%", h}, []any{at("t", def, slot, nil)})}
	}
	d["loading_page"] = panel([]any{"100%", "100%"},
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
	)
	d["loading_tip"] = panel([]any{300, 10}, obj{"l": label(lobbyHead, 0.8, []any{"100%", "default"}, obj{"text_alignment": "center", "shadow": true})})

	// The radial menu over the whole screen: the title and line over a ring
	// of the items, spread evenly from the top, the info under it.
	const discSize, ringRadius = 164, 56
	d["radial_item"] = with(panel([]any{60, 38},
		obj{"b": press(button([]any{"100%", "100%"}, "", "textures/ui/sg/hover_flat", "textures/ui/sg/press_flat", false), "")},
		obj{"i": with(panel([]any{16, 16}, obj{"i": icon(16)}), obj{"layer": 3, "offset": []any{0, 2}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
		obj{"l": faceLabel(white, 0.75, []any{58, "default"}, obj{"layer": 4, "shadow": true, "text_alignment": "center", "offset": []any{0, 20}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
		obj{"v": with(slots("vertical", []any{58, 8}, []any{ref("v", "radial_value", obj{"collection_index": "$value_index"})}), obj{"layer": 4, "offset": []any{0, 29}, "anchor_from": "top_middle", "anchor_to": "top_middle"})},
	), obj{"bindings": shown(filled)})
	d["radial_value"] = panel([]any{58, 8}, obj{"l": label(rgb(0xff, 0xd7, 0x5e), 0.65, []any{"100%", "default"}, obj{"text_alignment": "center", "shadow": true})})
	d["radial_center"] = with(panel([]any{30, 30},
		obj{"b": press(button([]any{"100%", "100%"}, "", "textures/ui/sg/hover_flat", "textures/ui/sg/press_flat", true), "")},
		obj{"i": with(panel([]any{16, 16}, obj{"i": icon(16)}), obj{"layer": 3, "anchor_from": "center", "anchor_to": "center"})},
	), obj{"bindings": shown(filled)})
	rings := []any{}
	for n := 1; n <= invui.RadialItems; n++ {
		items := []any{}
		for k := 0; k < n; k++ {
			a := -math.Pi/2 + 2*math.Pi*float64(k)/float64(n)
			x, y := math.Round(ringRadius*math.Cos(a)), math.Round(ringRadius*math.Sin(a))
			items = append(items, obj{fmt.Sprintf("i%d", k): with(slots("vertical", []any{60, 38}, []any{at("i", "radial_item", invui.RowIcon0+k, with(face(invui.RowIcon0+k, invui.RowBtn0+k), obj{"$value_index": invui.Cell0 + k}))}), obj{"layer": 3, "offset": []any{x, y}, "anchor_from": "center", "anchor_to": "center"})})
		}
		name := fmt.Sprintf("radial_ring_%d", n)
		d[name] = panel([]any{"100%", "100%"}, items...)
		rings = append(rings, layoutGate(d, fmt.Sprintf("r%d", n), name, []any{"100%", "100%"}, invui.KeyRadial+fmt.Sprint(n)))
	}
	rings = append(rings,
		obj{"disc": image("textures/ui/sg/radial_disc", []any{discSize, discSize}, 1, obj{"anchor_from": "center", "anchor_to": "center"})},
		obj{"center": with(slots("vertical", []any{30, 30}, []any{at("c", "radial_center", invui.RowIcon0+invui.RadialItems, face(invui.RowIcon0+invui.RadialItems, invui.RowBtn0+invui.RadialItems))}), obj{"layer": 3, "anchor_from": "center", "anchor_to": "center"})},
	)
	d["radial_page"] = panel([]any{"100%", "100%"},
		obj{"bg": with(rect([]any{0, 0, 0}, []any{"100%", "100%"}, nil, 1), obj{"alpha": 0.35})},
		obj{"middle": with(obj{"type": "stack_panel", "orientation": "vertical", "size": []any{300, "100%c"}, "layer": 2, "controls": []any{
			line("rtitle", invui.SlotTitle, rgb(0xff, 0xd7, 0x5e), 1.3, 14),
			line("rline", invui.SlotBody, lobbyHead, 0.75, 10),
			obj{"ring": panel([]any{"100%", discSize + 4}, rings...)},
			line("rinfo", invui.SlotInfo, white, 0.8, 10),
		}}, obj{"anchor_from": "center", "anchor_to": "center"})},
	)

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
