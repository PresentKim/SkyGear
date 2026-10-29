package progress

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// Rewards are the rewards by research: how many of each item it
// opens (unlock name) the researcher gets.
var Rewards = map[string]map[string]int{
	"chest":              {"chest": 2},
	"compost":            {"composter": 1},
	"barrel":             {"skygear:barrel_oak": 1},
	"furnace":            {"furnace": 1},
	"sieve":              {"skygear:sieve": 1, "skygear:mesh_string": 1},
	"stone_barrel":       {"skygear:barrel_stone": 1},
	"bucket":             {"bucket": 1},
	"hopper":             {"hopper": 2},
	"drawer":             {"skygear:drawer": 2},
	"generator":          {"skygear:generator": 1},
	"laboratory":         {"skygear:laboratory": 1},
	"energy_conduit":     {"skygear:conduit_energy_1": 8},
	"item_conduit":       {"skygear:conduit_item_1": 8},
	"fluid_conduit":      {"skygear:conduit_fluid_1": 8},
	"crusher":            {"skygear:crusher": 1},
	"auto_sieve":         {"skygear:auto_sieve": 1},
	"auto_miner":         {"skygear:auto_miner": 1},
	"iron_hammer":        {"skygear:hammer": 1},
	"harvester":          {"skygear:harvester": 1},
	"electric_furnace":   {"skygear:electric_furnace": 1},
	"item_vacuum":        {"skygear:item_vacuum": 1},
	"tree_feller":        {"skygear:tree_feller": 1},
	"auto_crafter":       {"skygear:auto_crafter": 1},
	"alloy_smelter":      {"skygear:alloy_smelter": 1},
	"energy_cell":        {"skygear:energy_cell": 1},
	"fluid_tank":         {"skygear:fluid_tank": 1},
	"crucible":           {"skygear:crucible": 1},
	"lava_generator":     {"skygear:lava_generator": 1},
	"ore_washer":         {"skygear:ore_washer": 1},
	"mob_simulator":      {"skygear:mob_simulator": 1},
	"storage_controller": {"skygear:storage_controller": 1, "skygear:storage_link": 2},
	"tinker_table":       {"skygear:tinker_table": 1},
	"magic_sieve":        {"skygear:magic_sieve": 1},
	"dimension_gate":     {"skygear:dimension_gate": 1},
	"fairy_magic":        {"skygear:fairy_board": 1},
}

// Reward is what a research gives the researcher, in the order of
// its unlocks.
func Reward(r *Research) []item.Stack {
	id, _, _ := strings.Cut(r.ID, "#")
	counts := Rewards[id]
	var out []item.Stack
	for _, n := range r.Unlocks {
		c, ok := counts[n]
		if !ok {
			continue
		}
		if it, ok := world.ItemByName(naming.UnlockName(n), 0); ok {
			out = append(out, item.NewStack(it, c))
		}
	}
	return out
}

// RewardCount is how many of an unlocked item a research gives (0: none).
func RewardCount(r *Research, unlock string) int {
	id, _, _ := strings.Cut(r.ID, "#")
	return Rewards[id][unlock]
}

// GiveReward gives p the reward of r and tells what it got.
func GiveReward(s *core.Session, p *player.Player, tx *world.Tx, r *Research) {
	reward := Reward(r)
	if len(reward) == 0 {
		return
	}
	var names []string
	for _, st := range reward {
		s.GiveItem(p, tx, p.Position(), st)
		names = append(names, fmt.Sprintf("%s ×%d", naming.ItemName(i18n.Of(p), st), st.Count()))
	}
	p.Messagef(i18n.T(i18n.Of(p), "researchreward.given"), strings.Join(slices.Clip(names), ", "))
}
