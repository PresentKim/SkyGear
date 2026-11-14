// Command quickmatch writes assets/resourcepack/inventoryui/ui/sg_quick_match.json, the
// part of the take all and put all bar (ui/sg_quick.json) that puts only
// what the container already holds. JSON UI has no loops, so the controls
// that read every container slot are written out here. Run it from the
// repository root: go run ./tools/quickmatch
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const out = "assets/resourcepack/inventoryui/ui/sg_quick_match.json"

// slots is how many container slots an inventory slot is compared with: a
// large chest.
const slots = 54

// group is how many container slots one join binding puts together, so no
// expression grows long (a small chest and a large chest fill whole groups).
const group = 9

// ids is the name of the control that joins the items of the container
// slots; the slot moves read the joined string from it (source_control_name
// searches the screen from its root, so the name is used by no other control).
const ids = "sg_quick_ids"

func main() {
	b, err := build()
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// build returns the contents of sg_quick_match.json.
func build() ([]byte, error) {
	var cells []any
	var bindings []any
	// Until a binding writes them, a slot is x (no item id) and a group
	// or the joined string an empty field (never '', which is no string
	// to join onto).
	bag := map[string]any{"#sg_ids": "|"}
	var parts []string
	for k := range slots {
		name := fmt.Sprintf("sg_quick_c%d", k)
		// collection_index is taken only where a definition is used, not
		// in a control defined in place.
		cells = append(cells, map[string]any{
			name + "@sg_quick_match.container_id": map[string]any{"collection_index": k},
		})
		bindings = append(bindings, map[string]any{
			"binding_type":         "view",
			"source_control_name":  name,
			"source_property_name": "#item_id_aux",
			"target_property_name": fmt.Sprintf("#c%d", k),
		})
		bag[fmt.Sprintf("#c%d", k)] = "x"
	}
	// '|' + id for every slot of a group: a string from the first '|' on,
	// so the numbers are joined as text, not added.
	for g := 0; g < slots; g += group {
		var terms []string
		for k := g; k < min(g+group, slots); k++ {
			terms = append(terms, fmt.Sprintf("'|' + #c%d", k))
		}
		target := fmt.Sprintf("#sg_ids%d", g/group)
		bindings = append(bindings, map[string]any{
			"binding_type":         "view",
			"source_property_name": "(" + strings.Join(terms, " + ") + ")",
			"target_property_name": target,
		})
		bag[target] = "|"
		parts = append(parts, target)
	}
	// |id|id|...|id| with a '|' on both sides of every id, so one id is
	// never found inside a longer one.
	bindings = append(bindings, map[string]any{
		"binding_type":         "view",
		"source_property_name": "(" + strings.Join(parts, " + ") + " + '|')",
		"target_property_name": "#sg_ids",
	})
	doc := map[string]any{
		"namespace": "sg_quick_match",
		// The item of each container slot (named so the control can read
		// it), and the items of all of them joined in one string once for
		// every slot move.
		"container_ids": map[string]any{
			"type":            "stack_panel",
			"size":            []int{0, 0},
			"collection_name": "container_items",
			"property_bag":    bag,
			"bindings":        bindings,
			"controls":        cells,
		},
		"container_id": map[string]any{
			"type": "panel",
			"size": []int{0, 0},
			"bindings": []any{map[string]any{
				"binding_name":            "#item_id_aux",
				"binding_type":            "collection",
				"binding_collection_name": "container_items",
			}},
		},
		// The quick move of an inventory slot, shown (and so pressed) only
		// while some container slot holds the same item: removing |id| from
		// the joined string changes it. The removal and the comparison are
		// separate bindings.
		"match_move": map[string]any{
			"type":                     "button",
			"size":                     []string{"100%", "100%"},
			"layer":                    90,
			"always_listen_to_input":   true,
			"consume_hover_events":     false,
			"default_focus_precedence": -2,
			"property_bag":             map[string]any{"#sg_ids": "|", "#sg_rest": "|"},
			"bindings": []any{
				map[string]any{"binding_type": "collection_details", "binding_collection_name": "$moves_collection"},
				map[string]any{
					"binding_name":            "#item_id_aux",
					"binding_name_override":   "#ia",
					"binding_type":            "collection",
					"binding_collection_name": "$moves_collection",
				},
				map[string]any{
					"binding_type":         "view",
					"source_control_name":  ids,
					"source_property_name": "#sg_ids",
					"target_property_name": "#sg_ids",
				},
				map[string]any{
					"binding_type":         "view",
					"source_property_name": "(#sg_ids - ('|' + #ia + '|'))",
					"target_property_name": "#sg_rest",
				},
				map[string]any{
					"binding_type":         "view",
					"source_property_name": "(not (#sg_rest = #sg_ids))",
					"target_property_name": "#visible",
				},
			},
			"button_mappings": []any{
				map[string]any{"from_button_id": "button.menu_select", "to_button_id": "button.container_auto_place", "mapping_type": "pressed", "consume_event": false},
				map[string]any{"from_button_id": "button.menu_ok", "to_button_id": "button.container_auto_place", "mapping_type": "focused", "consume_event": false},
			},
		},
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
