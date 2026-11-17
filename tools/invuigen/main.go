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

	d["text_slot"] = obj{"type": "panel", "size": []any{"100%", "100%c"}, "controls": []any{obj{"l": label(ink, 1.0, []any{"100% - 2px", "default"}, obj{"offset": []any{2, 0}, "anchor_from": "top_left", "anchor_to": "top_left"})}}}

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

	d["menu_body"] = obj{
		"type": "stack_panel", "orientation": "vertical", "size": []any{"100% - 8px", "100%c"}, "anchor_from": "top_left", "anchor_to": "top_left",
		"controls": []any{
			obj{"body": slots("vertical", []any{"100%", "100%c"}, []any{at("t", "text_slot", invui.SlotBody, nil)})},
			obj{"gap": obj{"type": "panel", "size": []any{"100%", 6}}},
			obj{"rows": slots("vertical", []any{"100%", "100%c"}, rowCells)},
		},
	}
	d["menu_content@common.scrolling_panel"] = obj{
		"anchor_to": "top_left", "anchor_from": "top_left", "$show_background": false, "size": []any{"100%", "100%"},
		"$scroll_size": []any{4, "100%"}, "$scrolling_pane_size": []any{"100%", "100%"}, "$scrolling_pane_offset": []any{0, 0},
		"$scroll_bar_right_padding_size": []any{0, 0}, "$scrolling_content": "sg_menu.menu_body",
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
				ref("menu", "menu_content", nil),
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
