package dragonfactory

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// Research rewards (docs/design/02-dragonfactory.md §9): a research that
// opens a facility or a machine gives the player who did it some of what it
// opens, so the first one costs no more materials (playtest: research and
// the first machines spent so much iron that players stopped before
// automating it). The amounts are drafts.

// researchRewards are the rewards by research: how many of each item it
// opens (unlock name) the researcher gets.
var researchRewards = map[string]map[string]int{
	"chest":              {"chest": 2},
	"compost":            {"composter": 1},
	"barrel":             {"dragonfactory:barrel_oak": 1},
	"furnace":            {"furnace": 1},
	"sieve":              {"dragonfactory:sieve": 1, "dragonfactory:mesh_string": 1},
	"stone_barrel":       {"dragonfactory:barrel_stone": 1},
	"bucket":             {"bucket": 1},
	"hopper":             {"hopper": 2},
	"drawer":             {"dragonfactory:drawer": 2},
	"generator":          {"dragonfactory:generator": 1},
	"laboratory":         {"dragonfactory:laboratory": 1},
	"energy_conduit":     {"dragonfactory:conduit_energy_1": 8},
	"item_conduit":       {"dragonfactory:conduit_item_1": 8},
	"fluid_conduit":      {"dragonfactory:conduit_fluid_1": 8},
	"crusher":            {"dragonfactory:crusher": 1},
	"auto_sieve":         {"dragonfactory:auto_sieve": 1},
	"auto_miner":         {"dragonfactory:auto_miner": 1},
	"iron_hammer":        {"dragonfactory:hammer": 1},
	"harvester":          {"dragonfactory:harvester": 1},
	"electric_furnace":   {"dragonfactory:electric_furnace": 1},
	"item_vacuum":        {"dragonfactory:item_vacuum": 1},
	"tree_feller":        {"dragonfactory:tree_feller": 1},
	"auto_crafter":       {"dragonfactory:auto_crafter": 1},
	"alloy_smelter":      {"dragonfactory:alloy_smelter": 1},
	"energy_cell":        {"dragonfactory:energy_cell": 1},
	"fluid_tank":         {"dragonfactory:fluid_tank": 1},
	"crucible":           {"dragonfactory:crucible": 1},
	"lava_generator":     {"dragonfactory:lava_generator": 1},
	"ore_washer":         {"dragonfactory:ore_washer": 1},
	"mob_simulator":      {"dragonfactory:mob_simulator": 1},
	"storage_controller": {"dragonfactory:storage_controller": 1, "dragonfactory:storage_link": 2},
	"tinker_table":       {"dragonfactory:tinker_table": 1},
	"magic_sieve":        {"dragonfactory:magic_sieve": 1},
	"dimension_gate":     {"dragonfactory:dimension_gate": 1},
}

// researchReward is what a research gives the researcher, in the order of
// its unlocks.
func researchReward(r *Research) []item.Stack {
	id, _, _ := strings.Cut(r.ID, "#")
	counts := researchRewards[id]
	var out []item.Stack
	for _, n := range r.Unlocks {
		c, ok := counts[n]
		if !ok {
			continue
		}
		if it, ok := world.ItemByName(unlockName(n), 0); ok {
			out = append(out, item.NewStack(it, c))
		}
	}
	return out
}

// rewardCount is how many of an unlocked item a research gives (0: none).
func rewardCount(r *Research, unlock string) int {
	id, _, _ := strings.Cut(r.ID, "#")
	return researchRewards[id][unlock]
}

// giveResearchReward gives p the reward of r and tells what it got.
func (s *session) giveResearchReward(p *player.Player, tx *world.Tx, r *Research) {
	reward := researchReward(r)
	if len(reward) == 0 {
		return
	}
	var names []string
	for _, st := range reward {
		s.giveItem(p, tx, p.Position(), st)
		names = append(names, fmt.Sprintf("%s ×%d", itemName(st), st.Count()))
	}
	p.Messagef(i18n.T(i18n.Of(p), "researchreward.given"), strings.Join(slices.Clip(names), ", "))
}
