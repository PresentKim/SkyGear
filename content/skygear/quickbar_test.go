package skygear

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/presentkim/skygear/content/skygear/machine"
)

// machinePanels are the layouts of sg_machine.json by window title marker,
// as common assets/resourcepack/inventoryui ui/chest_screen.json picks them.
var machinePanels = map[string]string{
	"§d§m§1": "crusher_panel", "§d§m§2": "sieve_panel", "§d§m§3": "miner_panel",
	"§d§m§4": "sell_box_panel", "§d§m§5": "miner1_panel", "§d§m§6": "alloy_panel",
	"§d§m§7": "crafter_panel", "§d§m§8": "lab_panel", "§d§m§9": "magicsieve_panel",
}

func TestQuickBarTakesEveryOutputAndNothingElse(t *testing.T) {
	raw, err := os.ReadFile("../../assets/resourcepack/skygear/ui/sg_machine.json")
	if err != nil {
		t.Fatal(err)
	}
	src := regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAllString(string(raw), "")
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatal(err)
	}
	vars := func(name string) map[string]any {
		var v map[string]any
		_ = json.Unmarshal(doc[name], &v)
		return v
	}
	filterOf := func(panel string) []int {
		expr, ok := vars(panel + "@sg_machine.machine_panel")["$sg_machine_outputs"].(string)
		if !ok {
			expr, _ = vars("machine_panel")["$sg_machine_outputs|default"].(string)
		}
		var out []int
		for _, m := range regexp.MustCompile(`#collection_index = (-?\d+)`).FindAllStringSubmatch(expr, -1) {
			n, _ := strconv.Atoi(m[1])
			out = append(out, n)
		}
		return out
	}
	takeable := map[string][]int{}
	for name, k := range machine.MachineKinds {
		panel, ok := machinePanels[k.Marker]
		if !ok {
			continue
		}
		filter := filterOf(panel)
		for _, s := range k.Output {
			if !slices.Contains(filter, s) {
				t.Errorf("%s: take all skips output %d", name, s)
			}
		}
		takeable[panel] = append(append(append(takeable[panel], k.Output...), k.Input...), k.Charge...)
	}
	for panel, slots := range takeable {
		for _, s := range filterOf(panel) {
			if s >= 0 && !slices.Contains(slots, s) {
				t.Errorf("%s: take all moves slot %d, which no machine of it takes from", panel, s)
			}
		}
	}
}
