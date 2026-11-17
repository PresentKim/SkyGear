// Command invuigen writes assets/resourcepack/inventoryui/ui/sg_menu.json, the
// menu window of common/invui: the slots of its layout read at fixed
// collection indexes, which JSON UI cannot loop over. Run it from the
// repository root: go run ./tools/invuigen
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

func texturePath() string {
	return "(" + clean() + " - '§r' - '§f')"
}

func slotText() obj {
	return obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": "#hover_text", "binding_condition": "always"}
}

func view(src, target string) obj {
	return obj{"binding_type": "view", "binding_condition": "always", "source_property_name": src, "target_property_name": target}
}

func has(marker string) string {
	return "(not ((#hover_text - '" + marker + "') = #hover_text))"
}

func shown(cond string) []any {
	return []any{slotText(), view(cond, "#visible")}
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
func slots(orientation string, size []any, children []any) obj {
	return obj{"type": "stack_panel", "orientation": orientation, "size": size, "collection_name": "container_items", "controls": children}
}

func label(color []any, scale float64, size []any, props obj) obj {
	l := obj{
		"type": "label", "layer": 3, "localize": false, "color": color, "font_scale_factor": scale, "size": size, "text": "#text",
		"bindings": []any{slotText(), view(clean(), "#text")},
	}
	for k, v := range props {
		l[k] = v
	}
	return l
}

func image(texture string, size []any, layer int, props obj) obj {
	i := obj{"type": "image", "texture": texture, "size": size, "layer": layer}
	for k, v := range props {
		i[k] = v
	}
	return i
}

// button is a slot pressed by dropping one of its item, which the server
// cancels and answers (guide INP-17).
func button(size []any, def, hover, pressed string) obj {
	mappings := []any{}
	for _, from := range []string{"button.menu_select", "button.menu_ok", "button.menu_secondary_select", "button.menu_auto_place"} {
		mappings = append(mappings, obj{"from_button_id": from, "to_button_id": "button.drop_one", "mapping_type": "pressed"})
	}
	mappings = append(mappings, obj{"to_button_id": "button.container_slot_hovered", "mapping_type": "pressed"})
	return obj{
		"type": "button", "size": size, "focus_enabled": true, "focus_wrap_enabled": false, "focus_magnet_enabled": true, "default_focus_precedence": 1,
		"default_control": "default", "hover_control": "hover", "pressed_control": "pressed",
		"sound_name": "random.click", "sound_volume": 1.0, "sound_pitch": 1.0,
		"button_mappings": mappings,
		"bindings":        []any{obj{"binding_type": "collection_details", "binding_collection_name": "container_items"}},
		"controls": []any{
			obj{"default": obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": face(def)}},
			obj{"hover": obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": append(face(hover), obj{"tip": tip()})}},
			obj{"pressed": obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": face(pressed)}},
		},
	}
}

func face(texture string) []any {
	c := []any{obj{"content@$button_content": obj{}}}
	if texture != "" {
		c = append([]any{obj{"bg": image(texture, []any{"100%", "100%"}, 1, nil)}}, c...)
	}
	return c
}

func tip() obj {
	return obj{"type": "custom", "renderer": "hover_text_renderer", "allow_clipping": false, "layer": 30, "$tip_visible|default": false, "visible": "$tip_visible"}
}

func defs() obj {
	d := obj{"namespace": "sg_menu"}

	d["icon_slot"] = obj{
		"type": "panel",
		"controls": []any{
			obj{"image": obj{
				"type": "image", "size": []any{"100%", "100%"}, "layer": 2, "texture": "#texture",
				"bindings": []any{slotText(), view(texturePath(), "#texture"), view(has("textures/"), "#visible")},
			}},
			obj{"item": obj{
				"type": "custom", "renderer": "inventory_item_renderer", "size": []any{"100%", "100%"}, "layer": 2,
				"property_bag": obj{"#item_id_aux": 0},
				"bindings": []any{
					slotText(),
					obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": "#item_id_aux", "binding_condition": "always"},
					view("((#hover_text - 'textures/') = #hover_text)", "#visible"),
				},
			}},
		},
	}

	textLabel := label(ink, 1.0, []any{"100% - 2px", "default"}, obj{"offset": []any{2, 0}, "anchor_from": "top_left", "anchor_to": "top_left"})
	textLabel["font_scale_factor"] = "$text_scale"
	d["text_slot"] = obj{"type": "panel", "size": []any{"100%", "100%c"}, "$text_scale|default": 1.0, "controls": []any{obj{"l": textLabel}}}

	d["title_slot"] = obj{"type": "panel", "size": []any{310, 10}, "controls": []any{obj{"l": label(title, 1.0, []any{"100%", 10}, obj{"text_alignment": "center"})}}}

	tabCells := []any{}
	for i := 0; i < invui.Tabs; i++ {
		tabCells = append(tabCells, at(fmt.Sprintf("t%d", i), "tab_cell", invui.TabBtn0+i, obj{"$icon_index": invui.TabIcon0 + i}))
	}
	d["tab_cell"] = obj{
		"type": "stack_panel", "orientation": "vertical", "size": []any{24, 24},
		"bindings": shown("(not (#hover_text = ''))"),
		"controls": []any{
			obj{"on": obj{
				"type": "panel", "size": []any{24, 24}, "bindings": shown(has(invui.KindTabOn)),
				"controls": []any{
					obj{"bg": image("textures/ui/TabTopFront", []any{24, 26}, 10, obj{"anchor_from": "top_left", "anchor_to": "top_left"})},
					obj{"icon": obj{"type": "panel", "layer": 11, "size": []any{24, 22}, "anchor_from": "top_left", "anchor_to": "top_left", "controls": []any{obj{"i": iconVar(14)}}}},
				},
			}},
			obj{"off": obj{
				"type": "panel", "size": []any{24, 24}, "bindings": shown(has(invui.KindTab)),
				"controls": []any{ref("b", "tab_button", obj{"$button_content": "sg_menu.tab_icon"})},
			}},
		},
	}
	tb := button([]any{24, 28}, "textures/ui/TabTopBack", "textures/ui/TabTopBackHover", "textures/ui/TabTopBackHover")
	tb["anchor_from"], tb["anchor_to"], tb["$tip_visible"] = "top_left", "top_left", true
	d["tab_button"] = tb
	d["tab_icon"] = obj{"type": "panel", "size": []any{24, 22}, "anchor_from": "top_left", "anchor_to": "top_left", "controls": []any{obj{"i": iconVar(14)}}}

	barCells := []any{}
	for i := 0; i < invui.Bars; i++ {
		barCells = append(barCells, at(fmt.Sprintf("b%d", i), "bar_cell", invui.BarBtn0+i, obj{"$icon_index": invui.BarIcon0 + i}))
	}
	d["bar_cell"] = obj{
		"type": "panel", "size": []any{15, 22},
		"bindings": shown("(not (#hover_text = ''))"),
		"controls": []any{ref("b", "bar_button", obj{"$button_content": "sg_menu.bar_face"})},
	}
	bb := button([]any{15, 15}, "", "textures/ui/sg/hover_flat", "textures/ui/sg/press_flat")
	bb["$tip_visible"] = true
	d["bar_button"] = bb
	d["bar_face"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{
		obj{"i": iconVar(15)},
		obj{"q": obj{"type": "label", "layer": 3, "text": "?", "localize": false, "color": title, "bindings": shown(has(invui.KindHelp))}},
	}}

	rowCells := []any{}
	for i := 0; i < invui.Rows; i++ {
		rowCells = append(rowCells, at(fmt.Sprintf("r%d", i), "row_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i}))
	}
	rowContent := func(textRight int) []any {
		return []any{
			obj{"i": obj{"type": "panel", "size": []any{20, 20}, "offset": []any{5, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle", "controls": []any{obj{"i": iconVar(20)}}}},
			obj{"l": label(light, 1.0, []any{fmt.Sprintf("100%% - %dpx", 29+textRight), "default"}, obj{"offset": []any{29, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
		}
	}
	d["row_content"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": rowContent(5)}
	toggle := func(tex string) obj {
		return obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{
			obj{"l": label(light, 1.0, []any{"100% - 48px", "default"}, obj{"offset": []any{6, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
			obj{"s": image(tex, []any{30, 16}, 2, obj{"offset": []any{-3, 0}, "anchor_from": "right_middle", "anchor_to": "right_middle"})},
		}}
	}
	d["toggle_off_content"] = toggle("textures/ui/toggle_off")
	d["toggle_on_content"] = toggle("textures/ui/toggle_on")
	d["row_button"] = button([]any{"100%", 30}, "textures/ui/sg/button", "textures/ui/sg/button_hover", "textures/ui/sg/button_pressed")
	variant := func(name, marker string, height int, controls []any) obj {
		return obj{name: obj{"type": "panel", "size": []any{"100%", height}, "bindings": shown(has(marker)), "controls": controls}}
	}
	top := obj{"anchor_from": "top_left", "anchor_to": "top_left"}
	withTop := func(o obj) obj {
		for k, v := range top {
			o[k] = v
		}
		return o
	}
	d["row_cell"] = obj{
		"type": "stack_panel", "orientation": "vertical", "size": []any{"100%", "100%c"},
		"bindings": shown("(not (#hover_text = ''))"),
		"controls": []any{
			variant("row", invui.KindRow, 32, []any{ref("b", "row_button", withTop(obj{"$button_content": "sg_menu.row_content"}))}),
			variant("static", invui.KindStatic, 32, []any{
				obj{"bg": image("textures/ui/sg/button", []any{"100%", 30}, 1, withTop(obj{"alpha": 0.6}))},
				ref("c", "row_content", withTop(obj{"size": []any{"100%", 30}})),
			}),
			variant("off", invui.KindToggleOff, 32, []any{ref("b", "row_button", withTop(obj{"$button_content": "sg_menu.toggle_off_content"}))}),
			variant("on", invui.KindToggleOn, 32, []any{ref("b", "row_button", withTop(obj{"$button_content": "sg_menu.toggle_on_content"}))}),
			variant("heading", invui.KindHeading, 10, []any{obj{"l": label(head, 0.8, []any{"100% - 2px", "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})}}),
			variant("divider", invui.KindDivider, 9, []any{obj{"line": image("textures/ui/White", []any{"100%", 1}, 1, obj{"color": []any{0.545, 0.545, 0.545}})}}),
		},
	}

	scroll := func(name, content string) {
		d[name+"@common.scrolling_panel"] = obj{
			"anchor_to": "top_left", "anchor_from": "top_left", "$show_background": false, "size": []any{"100%", "100%"},
			"$scroll_size": []any{4, "100%"}, "$scrolling_pane_size": []any{"100%", "100%"}, "$scrolling_pane_offset": []any{0, 0},
			"$scroll_bar_right_padding_size": []any{0, 0}, "$scrolling_content": "sg_menu." + content,
		}
	}
	column := func(width string, controls ...any) obj {
		return obj{"type": "stack_panel", "orientation": "vertical", "size": []any{width, "100%c"}, "anchor_from": "top_left", "anchor_to": "top_left", "controls": controls}
	}
	gap := func(h int) obj { return obj{"gap": obj{"type": "panel", "size": []any{"100%", h}}} }
	body := func(scale float64) obj {
		return obj{"body": slots("vertical", []any{"100%", "100%c"}, []any{at("t", "text_slot", invui.SlotBody, obj{"$text_scale": scale})})}
	}

	d["cell_frame"] = image("textures/ui/cell_image", []any{18, 18}, 1, nil)
	d["cell_face"] = obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
		ref("f", "cell_frame", nil),
		ref("i", "cell_item", nil),
		obj{"n": obj{
			"type": "label", "layer": 5, "localize": false, "color": []any{1, 1, 1}, "shadow": true, "font_scale_factor": 0.86,
			"anchor_from": "bottom_right", "anchor_to": "bottom_right", "offset": []any{0, 1}, "text": "#text",
			"bindings": []any{
				obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": "#inventory_stack_count", "binding_condition": "always"},
				view("('§r' + #inventory_stack_count)", "#text"),
			},
		}},
	}}
	d["cell_item"] = obj{"type": "panel", "size": []any{16, 16}, "layer": 2, "controls": []any{
		obj{"image": obj{
			"type": "image", "size": []any{"100%", "100%"}, "texture": "#texture",
			"bindings": []any{slotText(), view(texturePath(), "#texture"), view(has("textures/"), "#visible")},
		}},
		obj{"item": obj{
			"type": "custom", "renderer": "inventory_item_renderer", "size": []any{"100%", "100%"},
			"property_bag": obj{"#item_id_aux": 0},
			"bindings": []any{
				slotText(),
				obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": "#item_id_aux", "binding_condition": "always"},
				view("((#hover_text - 'textures/') = #hover_text)", "#visible"),
			},
		}},
	}}
	d["cell_mini"] = obj{"type": "panel", "size": []any{18, 18}, "controls": []any{ref("i", "cell_item", obj{"size": []any{12, 12}})}}
	d["cell_lit"] = obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
		ref("f", "cell_face", nil),
		obj{"lit": image("textures/ui/White", []any{16, 16}, 4, obj{"alpha": 0.3})},
	}}
	d["cell_lit_mini"] = obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
		ref("f", "cell_mini", nil),
		obj{"lit": image("textures/ui/White", []any{16, 16}, 4, obj{"alpha": 0.3})},
	}}
	cellButton := func(face, lit string) obj {
		b := button([]any{18, 18}, "", "", "")
		b["$tip_visible"] = true
		b["controls"] = []any{
			obj{"default": obj{"type": "panel", "size": []any{18, 18}, "controls": []any{ref("f", face, nil)}}},
			obj{"hover": obj{"type": "panel", "size": []any{18, 18}, "controls": []any{ref("f", lit, nil), obj{"tip": tip()}}}},
			obj{"pressed": obj{"type": "panel", "size": []any{18, 18}, "controls": []any{ref("f", lit, nil)}}},
		}
		return b
	}
	d["cell_button"] = cellButton("cell_face", "cell_lit")
	d["mini_button"] = cellButton("cell_mini", "cell_lit_mini")
	viewButton := func(face string, extra ...any) obj {
		b := button([]any{18, 18}, "", "", "")
		b["$tip_visible"] = true
		b["button_mappings"] = []any{obj{"to_button_id": "button.container_slot_hovered", "mapping_type": "pressed"}}
		hover := append([]any{ref("f", face, nil), obj{"tip": tip()}}, extra...)
		b["controls"] = []any{
			obj{"default": obj{"type": "panel", "size": []any{18, 18}, "controls": append([]any{ref("f", face, nil)}, extra...)}},
			obj{"hover": obj{"type": "panel", "size": []any{18, 18}, "controls": hover}},
			obj{"pressed": obj{"type": "panel", "size": []any{18, 18}, "controls": hover}},
		}
		return b
	}
	d["cell_view"] = viewButton("cell_face")
	d["cell_chosen"] = viewButton("cell_face", obj{"on": image("textures/ui/sg/slot_chosen", []any{18, 18}, 6, nil)})
	d["mini_view"] = viewButton("cell_mini")
	cellVariant := func(name, marker string, controls ...any) obj {
		return obj{name: obj{"type": "panel", "size": []any{18, 18}, "bindings": shown(has(marker)), "controls": controls}}
	}
	cellText := func(width int, color []any, scale float64) obj {
		return obj{"l": label(color, scale, []any{width, "default"}, obj{"offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})}
	}
	d["cell"] = obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
		cellVariant("press", invui.KindRow, ref("b", "cell_button", nil)),
		cellVariant("view", invui.KindStatic, ref("b", "cell_view", nil)),
		cellVariant("chosen", invui.KindChosen, ref("b", "cell_chosen", nil)),
		cellVariant("frame", invui.KindFrame, ref("f", "cell_frame", nil)),
		cellVariant("mini", invui.KindMini, ref("b", "mini_button", nil)),
		cellVariant("mini_view", invui.KindMiniView, ref("b", "mini_view", nil)),
		cellVariant("arrow", invui.KindArrow, obj{"a": image("textures/ui/arrow_large", []any{15, 10}, 2, nil)}),
		cellVariant("text", invui.KindText, cellText(194, ink, 0.7)),
		cellVariant("heading", invui.KindHeading, cellText(300, head, 0.8)),
	}}
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
			d[name] = obj{
				"type": "panel", "size": []any{columns * 18, 18},
				"bindings": shown("(not (#hover_text = ''))"),
				"controls": []any{obj{"cells": slots("horizontal", []any{columns * 18, 18}, cells)}},
			}
			out = append(out, at(fmt.Sprintf("r%d", r), name, start, nil))
		}
		return slots("vertical", []any{columns * 18, "100%c"}, out)
	}

	d["menu_page_body"] = column("100% - 8px", body(1.0), gap(6), obj{"rows": slots("vertical", []any{"100%", "100%c"}, rowCells)})
	scroll("menu_page", "menu_page_body")

	cardCells := []any{}
	for r := 0; r*invui.CardColumns < invui.Rows; r++ {
		cards := []any{}
		for c := 0; c < invui.CardColumns; c++ {
			i := r*invui.CardColumns + c
			cards = append(cards, at(fmt.Sprintf("c%d", c), "card_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i}))
		}
		cardCells = append(cardCells, obj{fmt.Sprintf("r%d", r): slots("horizontal", []any{304, "100%cm"}, cards)})
	}
	d["card_content"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{
		obj{"i": obj{"type": "panel", "size": []any{22, 22}, "offset": []any{0, 8}, "anchor_from": "top_middle", "anchor_to": "top_middle", "controls": []any{obj{"i": iconVar(22)}}}},
		obj{"l": label(light, 0.8, []any{"100% - 6px", "default"}, obj{"offset": []any{0, -4}, "anchor_from": "bottom_middle", "anchor_to": "bottom_middle", "text_alignment": "center"})},
	}}
	d["card_button"] = button([]any{72, 46}, "textures/ui/sg/button", "textures/ui/sg/button_hover", "textures/ui/sg/button_pressed")
	d["card_cell"] = obj{
		"type": "panel", "size": []any{76, 50},
		"bindings": shown("(not (#hover_text = ''))"),
		"controls": []any{ref("b", "card_button", obj{"$button_content": "sg_menu.card_content"})},
	}
	d["cards_page_body"] = column("100% - 8px", body(1.0), gap(6), obj{"cards": column("100%", cardCells...)})
	scroll("cards_page", "cards_page_body")

	d["grid_page_body"] = column("100% - 8px", body(1.0), gap(3), obj{"grid": cellGrid(0, (invui.Cells+invui.GridColumns-1)/invui.GridColumns-1, invui.GridColumns)}, gap(3),
		obj{"rows": slots("vertical", []any{"100%", "100%c"}, rowCells)})
	scroll("grid_page", "grid_page_body")

	d["board_page_body"] = column("100% - 8px", body(1.0), gap(3),
		obj{"board": obj{"type": "panel", "size": []any{"100%", "100%c"}, "controls": []any{
			obj{"g": func() obj {
				g := cellGrid(0, invui.Cells/invui.BoardColumns, invui.BoardColumns)
				g["anchor_from"], g["anchor_to"] = "top_middle", "top_middle"
				return g
			}()},
		}}},
		gap(3), obj{"rows": slots("vertical", []any{"100%", "100%c"}, rowCells)})
	scroll("board_page", "board_page_body")

	actionCells := func(first, last int) []any {
		out := []any{}
		for i := first; i < last; i++ {
			out = append(out, at(fmt.Sprintf("a%d", i), "action_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i}))
		}
		return out
	}
	d["action_content"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{
		obj{"i": obj{"type": "panel", "size": []any{12, 12}, "offset": []any{2, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle", "controls": []any{obj{"i": iconVar(12)}}}},
		obj{"l": label(light, 0.8, []any{"100% - 17px", "default"}, obj{"offset": []any{15, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	}}
	d["action_button"] = button([]any{"100%", 17}, "textures/ui/sg/button", "textures/ui/sg/button_hover", "textures/ui/sg/button_pressed")
	d["action_cell"] = obj{
		"type": "panel", "size": []any{"100%", 19},
		"bindings": shown(has(invui.KindRow)),
		"controls": []any{ref("b", "action_button", withTop(obj{"$button_content": "sg_menu.action_content"}))},
	}

	pagerCell := func(arrow string) obj {
		return obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
			obj{"on": obj{"type": "panel", "size": []any{18, 18}, "bindings": shown(has(invui.KindRow)), "controls": []any{
				ref("b", "row_button", obj{"size": []any{18, 18}, "$button_content": "sg_menu.pager_" + arrow}),
			}}},
			obj{"off": obj{"type": "panel", "size": []any{18, 18}, "bindings": shown(has(invui.KindStatic)), "controls": []any{
				obj{"bg": image("textures/ui/sg/button", []any{18, 18}, 1, obj{"alpha": 0.4})},
				obj{"a": image("textures/ui/sg/arrow_"+arrow, []any{12, 12}, 2, obj{"alpha": 0.4})},
			}}},
		}}
	}
	d["pager_left"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{obj{"a": image("textures/ui/sg/arrow_left", []any{12, 12}, 2, nil)}}}
	d["pager_right"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{obj{"a": image("textures/ui/sg/arrow_right", []any{12, 12}, 2, nil)}}}
	d["prev_cell"] = pagerCell("left")
	d["next_cell"] = pagerCell("right")
	d["detail_name"] = obj{"type": "panel", "size": []any{"fill", 28}, "controls": []any{
		obj{"l": label(ink, 0.8, []any{"100% - 4px", "default"}, obj{"offset": []any{4, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle"})},
	}}
	d["detail_big"] = obj{"type": "panel", "size": []any{28, 28}, "controls": []any{
		obj{"f": image("textures/ui/cell_image", []any{28, 28}, 1, nil)},
		obj{"i": iconVar(22)},
	}}
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
	d["split_page"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{
		obj{"left": image("textures/ui/sg/panel", []any{152, 142}, 1, obj{"anchor_from": "top_left", "anchor_to": "top_left", "controls": []any{
			obj{"grid": withTop(func() obj {
				g := cellGrid(0, invui.SplitCells/invui.SplitColumns, invui.SplitColumns)
				g["offset"], g["layer"] = []any{4, 4}, 2
				return g
			}())},
			obj{"pager": withTop(func() obj {
				p := slots("horizontal", []any{144, 18}, []any{
					at("prev", "prev_cell", invui.SlotPrev, nil),
					obj{"fill": obj{"type": "panel", "size": []any{"fill", 18}}},
					at("next", "next_cell", invui.SlotNext, nil),
				})
				p["offset"], p["layer"] = []any{4, 114}, 2
				return p
			}())},
		}})},
		obj{"right": image("textures/ui/sg/panel", []any{156, 142}, 1, obj{"anchor_from": "top_right", "anchor_to": "top_right", "controls": []any{
			obj{"inner": obj{"type": "panel", "layer": 2, "size": []any{148, 134}, "controls": []any{ref("s", "split_right_scroll", nil)}}},
		}})},
	}}

	listCells := []any{}
	for i := 0; i < invui.ListRows; i++ {
		listCells = append(listCells, at(fmt.Sprintf("r%d", i), "list_cell", invui.RowBtn0+i, obj{"$icon_index": invui.RowIcon0 + i, "$info_index": invui.Cell0 + i}))
	}
	d["info_slot"] = obj{"type": "panel", "size": []any{"100%", 8}, "controls": []any{
		obj{"l": label([]any{0.8, 0.8, 0.82}, 0.7, []any{"100%", "default"}, obj{"anchor_from": "top_left", "anchor_to": "top_left"})},
	}}
	d["list_content"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{
		obj{"i": obj{"type": "panel", "size": []any{20, 20}, "offset": []any{5, 0}, "anchor_from": "left_middle", "anchor_to": "left_middle", "controls": []any{obj{"i": iconVar(20)}}}},
		obj{"name": label(light, 1.0, []any{"100% - 30px", 9}, obj{"offset": []any{27, 5}, "anchor_from": "top_left", "anchor_to": "top_left"})},
		obj{"info": withTop(func() obj {
			s := slots("vertical", []any{"100% - 30px", 8}, []any{ref("t", "info_slot", obj{"collection_index": "$info_index"})})
			s["offset"] = []any{27, 16}
			s["layer"] = 3
			return s
		}())},
	}}
	d["list_cell"] = obj{
		"type": "stack_panel", "orientation": "vertical", "size": []any{"100%", "100%c"},
		"bindings": shown("(not (#hover_text = ''))"),
		"controls": []any{
			variant("row", invui.KindRow, 32, []any{ref("b", "row_button", withTop(obj{"$button_content": "sg_menu.list_content"}))}),
			variant("chosen", invui.KindChosen, 32, []any{
				obj{"bg": image("textures/ui/sg/button_hover", []any{"100%", 30}, 1, withTop(obj{}))},
				ref("c", "list_content", withTop(obj{"size": []any{"100%", 30}})),
			}),
		},
	}
	d["list_body"] = column("100% - 8px", obj{"rows": slots("vertical", []any{"100%", "100%c"}, listCells)})
	scroll("list_scroll", "list_body")
	d["list_right_body"] = column("100% - 8px", body(0.8), gap(6), obj{"actions": slots("vertical", []any{"100%", "100%c"}, actionCells(invui.ListRows, invui.Rows))})
	scroll("list_right_scroll", "list_right_body")
	d["listsplit_page"] = obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": []any{
		obj{"left": image("textures/ui/sg/panel", []any{156, 142}, 1, obj{"anchor_from": "top_left", "anchor_to": "top_left", "controls": []any{
			obj{"inner": obj{"type": "panel", "layer": 2, "size": []any{148, 134}, "controls": []any{ref("s", "list_scroll", nil)}}},
		}})},
		obj{"right": obj{"type": "panel", "size": []any{154, 142}, "anchor_from": "top_right", "anchor_to": "top_right", "controls": []any{ref("s", "list_right_scroll", nil)}}},
	}}

	pages := []any{}
	for _, p := range []struct{ key, def string }{
		{invui.KeyMenu, "menu_page"}, {invui.KeyCards, "cards_page"}, {invui.KeyGrid, "grid_page"},
		{invui.KeyBoard, "board_page"}, {invui.KeySplit, "split_page"}, {invui.KeyListSplit, "listsplit_page"},
	} {
		d["gate_"+p.key] = obj{
			"type": "panel", "size": []any{"100%", "100%"},
			"bindings": shown(has(p.key)),
			"controls": []any{ref("page", p.def, nil)},
		}
		pages = append(pages, obj{p.key: slots("vertical", []any{"100%", "100%"}, []any{at("g", "gate_"+p.key, invui.SlotKey, nil)})})
	}

	tabs := slots("horizontal", []any{"100%c", 24}, tabCells)
	tabs["offset"], tabs["layer"] = []any{8, -22}, 0
	d["window"] = obj{
		"type": "panel", "size": []any{326, 166},
		"controls": []any{
			obj{"frame": image("textures/ui/dialog_background_opaque", []any{"100%", "100%"}, 1, nil)},
			obj{"title": withTop(obj{"type": "panel", "size": []any{310, 10}, "offset": []any{8, 6}, "layer": 3, "controls": []any{
				obj{"s": slots("vertical", []any{310, 10}, []any{at("t", "title_slot", invui.SlotTitle, nil)})},
			}})},
			obj{"tabs": withTop(tabs)},
			obj{"toolbar": image("textures/ui/toolbar_background", []any{"100%c + 14px", 25}, 4, obj{
				"offset": []any{0, -22}, "anchor_from": "top_right", "anchor_to": "top_right",
				"controls": []any{obj{"row": obj{
					"type": "stack_panel", "orientation": "horizontal", "size": []any{"100%c", 22}, "offset": []any{7, 1}, "anchor_from": "top_left", "anchor_to": "top_left",
					"controls": []any{
						obj{"bar": slots("horizontal", []any{"100%c", 22}, barCells)},
						obj{"close": obj{"type": "panel", "size": []any{17, 22}, "controls": []any{
							obj{"x@common.light_close_button": obj{"$close_button_offset": []any{0, 0}, "size": []any{15, 15}, "anchor_from": "center", "anchor_to": "center"}},
						}}},
					},
				}}},
			})},
			obj{"content": withTop(obj{"type": "panel", "layer": 2, "size": []any{314, 142}, "offset": []any{6, 18}, "controls": []any{
				obj{"pages": obj{"type": "panel", "size": []any{"100%", "100%"}, "controls": pages}},
			}})},
		},
	}

	d["overlay"] = obj{
		"type": "panel", "size": []any{"100%", "100%"}, "layer": 50,
		"$sg_title": "$container_title",
		"controls": []any{obj{"host": obj{
			"type": "panel", "size": []any{"100%", "100%"},
			"ignored": "(($sg_title - '" + invui.Marker + "') = $sg_title)",
			"controls": []any{
				obj{"sg_menu_hide_vanilla": obj{"type": "panel", "size": []any{0, 0}, "property_bag": obj{"#sg_menu_vanilla_visible": false}}},
				obj{"cover": obj{
					"type": "input_panel", "modal": true, "size": []any{"100%", "100%"},
					"controls": []any{
						obj{"sync": obj{"type": "grid", "size": []any{0, 0}, "maximum_grid_items": 27, "collection_name": "container_items", "grid_item_template": "sg_menu.empty"}},
						ref("window", "window", nil),
					},
				}},
			},
		}}},
	}
	d["empty"] = obj{"type": "panel", "size": []any{0, 0}}
	return d
}

// iconVar is the icon of the slot $icon_index, the icon slot of the element
// around it.
func iconVar(size int) obj {
	s := []any{size, size}
	return slots("vertical", s, []any{ref("i", "icon_slot", obj{"size": s, "collection_index": "$icon_index"})})
}

func main() {
	b, err := json.MarshalIndent(defs(), "", "  ")
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
