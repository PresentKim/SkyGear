// Command termuigen writes assets/resourcepack/skygear/ui/sg_terminal.json,
// the storage network terminal's window (content/skygear/netstore): every
// slot of termlayout at its fixed collection index, which JSON UI cannot loop
// over. Run it from the repository root: go run ./tools/termuigen
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/presentkim/skygear/common/invui"
	"github.com/presentkim/skygear/content/skygear/netstore/termlayout"
)

const out = "assets/resourcepack/skygear/ui/sg_terminal.json"

type obj = map[string]any

const columns = 9

func slots(typ, orientation string, size []any, children []any) obj {
	o := obj{"type": typ, "size": size, "collection_name": "container_items", "controls": children}
	if orientation != "" {
		o["orientation"] = orientation
	}
	return o
}

func at(name, def string, index int, props obj) obj {
	if props == nil {
		props = obj{}
	}
	props["collection_index"] = index
	return obj{name + "@sg_terminal." + def: props}
}

func main() {
	text := "(#hover_text - '§o' - '§f')"
	count := fmt.Sprintf("('%%.%ds' * %s)", 3*invui.CountChars, text)
	d := obj{"namespace": "sg_terminal"}
	d["none"] = obj{"type": "panel", "size": []any{0, 0}}

	// A kind's picture: the slot's frame, the item and its count.
	d["show"] = obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
		obj{"bg": obj{"type": "image", "texture": "textures/ui/cell_image", "size": []any{18, 18}}},
		obj{"item": obj{
			"type": "custom", "renderer": "inventory_item_renderer", "size": []any{16, 16}, "layer": 2,
			"property_bag": obj{"#item_id_aux": 0},
			"bindings":     []any{obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": "#item_id_aux", "binding_condition": "always"}},
		}},
		obj{"count@sg_machine.net_count": obj{}},
		obj{"count_long@sg_machine.net_count_long": obj{}},
	}}

	// The tooltip of a kind: its show slot's name after the count text, and
	// its lore, wrapped.
	d["tip"] = obj{
		"type": "image", "texture": "textures/ui/purpleBorder", "size": []any{158, "100%c + 8px"}, "layer": 30,
		"allow_clipping": false, "anchor_from": "top_left", "anchor_to": "top_left",
		"controls": []any{obj{"l": obj{
			"type": "label", "localize": false, "color": []any{1, 1, 1}, "shadow": true, "size": []any{150, "default"},
			"offset": []any{4, 4}, "anchor_from": "top_left", "anchor_to": "top_left", "text": "#text",
			"bindings": []any{
				obj{"binding_type": "collection", "binding_collection_name": "container_items", "binding_name": "#hover_text", "binding_condition": "always"},
				obj{"binding_type": "view", "source_property_name": "(" + text + " - " + count + ")", "target_property_name": "#text"},
			},
		}}},
	}
	d["take_hover"] = obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
		obj{"highlight@common.highlight_slot": obj{}},
		obj{"tip": obj{
			"type": "stack_panel", "orientation": "vertical", "size": []any{158, "100%c"}, "collection_name": "container_items",
			"layer": 30, "anchor_from": "top_right", "anchor_to": "top_left", "offset": []any{4, 0},
			"controls": []any{obj{"t@sg_terminal.tip": obj{"collection_index": "$show_index"}}},
		}},
	}}

	// A take slot: the vanilla slot, clicked as any chest's, drawn only
	// as the hover over its kind's picture.
	d["take@chest.chest_grid_item"] = obj{
		"$item_renderer": "sg_terminal.none", "$background_images": "sg_machine.no_bg",
		"$stack_count_required": false, "$durability_bar_required": false, "$storage_bar_required": false,
		"$highlight_control": "sg_terminal.take_hover",
	}

	rows := []any{}
	for r := 0; r*columns < termlayout.Kinds; r++ {
		cells := []any{}
		for c := 0; c < columns; c++ {
			k := r*columns + c
			cells = append(cells, obj{fmt.Sprintf("c%d", c): obj{"type": "panel", "size": []any{18, 18}, "controls": []any{
				obj{"s": slots("stack_panel", "vertical", []any{18, 18}, []any{at("show", "show", termlayout.Show0+k, nil)})},
				obj{"t": slots("stack_panel", "vertical", []any{18, 18}, []any{at("take", "take", termlayout.Take0+k, obj{"$show_index": termlayout.Show0 + k})})},
			}}})
		}
		rows = append(rows, obj{fmt.Sprintf("r%d", r): obj{"type": "stack_panel", "orientation": "horizontal", "size": []any{columns * 18, 18}, "controls": cells}})
	}
	d["list"] = obj{"type": "stack_panel", "orientation": "vertical", "size": []any{columns * 18, "100%c"}, "anchor_from": "top_left", "anchor_to": "top_left", "controls": rows}

	// The insert slots, all at the middle of the list, where what is put in
	// flies.
	inserts := []any{}
	for i := 0; i < termlayout.Inserts; i++ {
		inserts = append(inserts, at(fmt.Sprintf("i%d", i), "insert", termlayout.Insert0+i, obj{"anchor_from": "center", "anchor_to": "center"}))
	}
	d["insert@chest.chest_grid_item"] = obj{
		"size": []any{1, 1}, "$item_renderer": "sg_terminal.none", "$background_images": "sg_machine.no_bg",
		"$stack_count_required": false, "$durability_bar_required": false, "$storage_bar_required": false,
		"$highlight_control": "sg_terminal.none",
	}

	buttons := []any{}
	for i := 0; i < termlayout.Buttons; i++ {
		if termlayout.Button0+i == termlayout.Info {
			continue
		}
		buttons = append(buttons, obj{fmt.Sprintf("b%d@sg_machine.net_button", i): obj{"collection_index": termlayout.Button0 + i}})
	}

	// The bar above the window's top right: the info button left of the
	// close button.
	d["bar"] = obj{
		"type": "image", "texture": "textures/ui/toolbar_background", "size": []any{"100%c + 14px", 25},
		"offset": []any{0, -22}, "anchor_from": "top_right", "anchor_to": "top_right", "layer": 3,
		"controls": []any{obj{"row": obj{
			"type": "stack_panel", "orientation": "horizontal", "size": []any{"100%c", 22}, "offset": []any{7, 1},
			"anchor_from": "top_left", "anchor_to": "top_left",
			"controls": []any{
				obj{"info": slots("stack_panel", "vertical", []any{18, 22}, []any{
					obj{"b@sg_machine.net_button": obj{"collection_index": termlayout.Info}},
				})},
				obj{"close": obj{"type": "panel", "size": []any{22, 22}, "controls": []any{
					obj{"x@common.light_close_button": obj{"$close_button_offset": []any{0, 0}, "size": []any{15, 15}, "anchor_from": "center", "anchor_to": "center"}},
				}}},
			},
		}}},
	}

	d["slots"] = obj{"type": "panel", "size": []any{168, 90}, "anchor_from": "top_left", "anchor_to": "top_left", "controls": []any{
		obj{"list@common.container_scroll_panel": obj{
			"$touch": false, "$scrolling_panel_size": []any{168, 90}, "size": []any{168, 90},
			"$scroll_view_port_size": []any{168, 90}, "$scroll_view_port_size_touch": []any{168, 90},
			"$scroll_view_port_offset": []any{0, 0}, "$scroll_view_port_offset_touch": []any{0, 0},
			"$scrolling_content": "sg_terminal.list", "anchor_from": "top_left", "anchor_to": "top_left",
		}},
		obj{"inserts": with(slots("collection_panel", "", []any{1, 1}, inserts), obj{"anchor_from": "center", "anchor_to": "center"})},
		obj{"buttons": obj{
			"type": "image", "texture": "textures/ui/dialog_background_opaque", "size": []any{26, "100%c + 8px"},
			"anchor_from": "top_left", "anchor_to": "top_right", "offset": []any{-10, -21},
			"controls": []any{obj{"column": with(slots("stack_panel", "vertical", []any{18, "100%c"}, buttons), obj{"offset": []any{4, 4}, "anchor_from": "top_left", "anchor_to": "top_left"})}},
		}},
	}}

	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		panic(err)
	}
	text = strings.NewReplacer(`>`, ">", `<`, "<", `&`, "&").Replace(string(b))
	if err := os.WriteFile(out, []byte(text+"\n"), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", out)
}

func with(o obj, props obj) obj {
	for k, v := range props {
		o[k] = v
	}
	return o
}
