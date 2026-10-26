package dragonfactory

import (
	"slices"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
)

func init() {
	for _, k := range dfitem.ResearchEssenceKinds {
		world.RegisterItem(dfitem.ResearchEssence{Kind: k})
	}
}

// researchEssenceCosts are the research essences the researches cost next to
// their materials, by research: how many of the essence of each kind.
// Researches before the laboratory, and the tools and armour, cost none.
var researchEssenceCosts = map[string]map[string]int{
	// the first machines and conduits
	"energy_conduit": {dfitem.EssenceEngineering: 2}, "item_conduit": {dfitem.EssenceEngineering: 2}, "drawer_upgrades": {dfitem.EssenceEngineering: 2},
	// automation
	"crusher": {dfitem.EssenceEngineering: 4}, "electric_furnace": {dfitem.EssenceEngineering: 4}, "auto_sieve": {dfitem.EssenceEngineering: 4},
	"auto_miner": {dfitem.EssenceEngineering: 4}, "item_vacuum": {dfitem.EssenceEngineering: 4}, "tree_feller": {dfitem.EssenceEngineering: 4},
	"energy_cell": {dfitem.EssenceEngineering: 4}, "tinker_table": {dfitem.EssenceEngineering: 4},
	"auto_crafter": {dfitem.EssenceEngineering: 6}, "fluid_conduit": {dfitem.EssenceEngineering: 6}, "fluid_tank": {dfitem.EssenceEngineering: 6},
	"alloy_smelter": {dfitem.EssenceEngineering: 6}, "storage_controller": {dfitem.EssenceEngineering: 6},
	// crops
	"harvester": {dfitem.EssenceEngineering: 4, dfitem.EssenceFarming: 4},
	// alloys and fluids
	"conduit_upgrade_2": {dfitem.EssenceEngineering: 4, dfitem.EssenceAlchemy: 4}, "conduit_upgrade_3": {dfitem.EssenceEngineering: 6, dfitem.EssenceAlchemy: 8},
	"crucible": {dfitem.EssenceEngineering: 6, dfitem.EssenceAlchemy: 4}, "ore_washer": {dfitem.EssenceEngineering: 6, dfitem.EssenceAlchemy: 4},
	"mob_simulator": {dfitem.EssenceEngineering: 6, dfitem.EssenceAlchemy: 4}, "lava_generator": {dfitem.EssenceEngineering: 6, dfitem.EssenceAlchemy: 6},
	// monsters
	"bone_craft": {dfitem.EssenceHunting: 2}, "slime_block": {dfitem.EssenceHunting: 2},
	"blaze_craft": {dfitem.EssenceAlchemy: 4, dfitem.EssenceHunting: 4}, "ender_chest": {dfitem.EssenceAlchemy: 4, dfitem.EssenceHunting: 4},
	"diamond_armor":  {dfitem.EssenceAlchemy: 2, dfitem.EssenceHunting: 4},
	"dimension_gate": {dfitem.EssenceEngineering: 8, dfitem.EssenceAlchemy: 6, dfitem.EssenceHunting: 4},
	// magic plants
	"magic_botany": {dfitem.EssenceAlchemy: 6, dfitem.EssenceHunting: 4, dfitem.EssenceFarming: 4}, "dyeing": {dfitem.EssenceMagic: 4},
	"magic_gems":    {dfitem.EssenceAlchemy: 6, dfitem.EssenceHunting: 4, dfitem.EssenceMagic: 4},
	"magic_essence": {dfitem.EssenceAlchemy: 6, dfitem.EssenceHunting: 6, dfitem.EssenceMagic: 6},
	"magic_sieve":   {dfitem.EssenceEngineering: 8, dfitem.EssenceAlchemy: 8, dfitem.EssenceHunting: 6, dfitem.EssenceMagic: 6},
	"config_copier": {dfitem.EssenceEngineering: 8, dfitem.EssenceAlchemy: 6},
	// machine tiers
	"machine_tier_1": {dfitem.EssenceEngineering: 8, dfitem.EssenceAlchemy: 8},
	"machine_tier_2": {dfitem.EssenceEngineering: 16, dfitem.EssenceAlchemy: 16, dfitem.EssenceHunting: 8},
	"machine_tier_3": {dfitem.EssenceEngineering: 24, dfitem.EssenceAlchemy: 24, dfitem.EssenceFarming: 16, dfitem.EssenceHunting: 16, dfitem.EssenceMagic: 16},
	// the projects
	"project_industry": {dfitem.EssenceEngineering: 16, dfitem.EssenceAlchemy: 16},
	"project_hunter":   {dfitem.EssenceEngineering: 8, dfitem.EssenceAlchemy: 8, dfitem.EssenceHunting: 16},
	"project_botanist": {dfitem.EssenceFarming: 16, dfitem.EssenceHunting: 8, dfitem.EssenceMagic: 16},
	"project_dragon":   {dfitem.EssenceEngineering: 32, dfitem.EssenceAlchemy: 32, dfitem.EssenceFarming: 32, dfitem.EssenceHunting: 32, dfitem.EssenceMagic: 32},
}

// withEssenceCosts returns the researches with the essence costs added after
// their materials, and the laboratory among what they need unless something
// they already need leads to it: the essences come from the laboratory.
func withEssenceCosts(base []Research) []Research {
	out := slices.Clone(base)
	byID := map[string]*Research{}
	for i := range out {
		byID[out[i].ID] = &out[i]
	}
	// needsLab tells whether a research can only be done after the laboratory.
	needsLab := map[string]bool{"laboratory": true}
	var reachesLab func(id string) bool
	reachesLab = func(id string) bool {
		if v, ok := needsLab[id]; ok {
			return v
		}
		needsLab[id] = false
		if r, ok := byID[id]; ok {
			for _, n := range r.Needs {
				if reachesLab(n) {
					needsLab[id] = true
				}
			}
		}
		return needsLab[id]
	}
	for i := range out {
		r := &out[i]
		costs, ok := researchEssenceCosts[r.ID]
		if !ok {
			continue
		}
		r.Cost = slices.Clone(r.Cost)
		for _, kind := range dfitem.ResearchEssenceKinds {
			if n := costs[kind]; n > 0 {
				r.Cost = append(r.Cost, Requirement{dfitem.ResearchEssence{Kind: kind}, n})
			}
		}
		behind := false
		for _, n := range r.Needs {
			behind = behind || reachesLab(n)
		}
		if !behind {
			r.Needs = append(slices.Clone(r.Needs), "laboratory")
		}
		needsLab[r.ID] = true
	}
	return out
}

// essencesIn counts the research essences in a cost.
func essencesIn(cost []Requirement) int {
	n := 0
	for _, c := range cost {
		if _, ok := c.Item.(dfitem.ResearchEssence); ok {
			n += c.Count
		}
	}
	return n
}

// pourEssences shows the essences of a cost being poured into a research:
// a brewing sound and a splash of the colour of each kind.
func pourEssences(tx *world.Tx, p *player.Player, cost []Requirement) {
	poured := false
	at := p.Position().Add(mgl64.Vec3{0, 1, 0})
	for _, c := range cost {
		if e, ok := c.Item.(dfitem.ResearchEssence); ok {
			tx.AddParticle(at, particle.Splash{Colour: dfitem.EssenceColours[e.Kind]})
			poured = true
		}
	}
	if poured {
		tx.PlaySound(at, sound.PotionBrewed{})
	}
}

// essenceCostOf is how many essences of a kind a research costs.
func essenceCostOf(r *Research, kind string) int {
	n := 0
	for _, c := range r.Cost {
		if e, ok := c.Item.(dfitem.ResearchEssence); ok && e.Kind == kind {
			n += c.Count
		}
	}
	return n
}
