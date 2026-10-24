// Command quickmatch writes packs/inventoryui/ui/df_quick_match.json, the
// part of the take all and put all bar (ui/df_quick.json) that puts only
// what the container already holds. JSON UI has no loops, so the controls
// that compare an inventory slot with every container slot are written out
// here. Run it from the repository root: go run ./tools/quickmatch
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const out = "packs/inventoryui/ui/df_quick_match.json"

// slots is how many container slots an inventory slot is compared with: a
// large chest.
const slots = 54

func main() {
	var ids []any
	var bindings []any
	var terms []string
	for k := range slots {
		name := fmt.Sprintf("df_quick_c%d", k)
		// collection_index is taken only where a definition is used, not
		// in a control defined in place.
		ids = append(ids, map[string]any{
			name + "@df_quick_match.container_id": map[string]any{"collection_index": k},
		})
		bindings = append(bindings, map[string]any{
			"binding_type":         "view",
			"source_control_name":  name,
			"source_property_name": "#item_id_aux",
			"target_property_name": fmt.Sprintf("#c%d", k),
		})
		terms = append(terms, fmt.Sprintf("(#ia = #c%d)", k))
	}
	moveBindings := append([]any{
		map[string]any{"binding_type": "collection_details", "binding_collection_name": "$moves_collection"},
		map[string]any{
			"binding_name":            "#item_id_aux",
			"binding_name_override":   "#ia",
			"binding_type":            "collection",
			"binding_collection_name": "$moves_collection",
		},
	}, bindings...)
	moveBindings = append(moveBindings, map[string]any{
		"binding_type":         "view",
		"source_property_name": "(" + strings.Join(terms, " or ") + ")",
		"target_property_name": "#visible",
	})
	doc := map[string]any{
		"namespace": "df_quick_match",
		// The item of each container slot, named so the slot moves below can
		// read it (source_control_name searches the screen from its root).
		"container_ids": map[string]any{
			"type":            "stack_panel",
			"size":            []int{0, 0},
			"collection_name": "container_items",
			"controls":        ids,
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
		// while some container slot holds the same item.
		"match_move": map[string]any{
			"type":                     "button",
			"size":                     []string{"100%", "100%"},
			"layer":                    90,
			"always_listen_to_input":   true,
			"consume_hover_events":     false,
			"default_focus_precedence": -2,
			"bindings":                 moveBindings,
			"button_mappings": []any{
				map[string]any{"from_button_id": "button.menu_select", "to_button_id": "button.container_auto_place", "mapping_type": "pressed", "consume_event": false},
				map[string]any{"from_button_id": "button.menu_ok", "to_button_id": "button.container_auto_place", "mapping_type": "focused", "consume_event": false},
			},
		},
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
