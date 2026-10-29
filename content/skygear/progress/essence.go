package progress

import (
	"slices"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// EssenceCosts are the research essences the researches cost next to
// their materials, by research: how many of the essence of each kind.
// Researches before the laboratory, and the tools and armour, cost none.
var EssenceCosts = map[string]map[string]int{
	// the first machines and conduits
	"energy_conduit": {sgitem.EssenceEngineering: 2}, "item_conduit": {sgitem.EssenceEngineering: 2}, "drawer_upgrades": {sgitem.EssenceEngineering: 2},
	// automation
	"crusher": {sgitem.EssenceEngineering: 4}, "electric_furnace": {sgitem.EssenceEngineering: 4}, "auto_sieve": {sgitem.EssenceEngineering: 4},
	"auto_miner": {sgitem.EssenceEngineering: 4}, "item_vacuum": {sgitem.EssenceEngineering: 4}, "tree_feller": {sgitem.EssenceEngineering: 4},
	"energy_cell": {sgitem.EssenceEngineering: 4}, "tinker_table": {sgitem.EssenceEngineering: 4},
	"auto_crafter": {sgitem.EssenceEngineering: 6}, "fluid_conduit": {sgitem.EssenceEngineering: 6}, "fluid_tank": {sgitem.EssenceEngineering: 6},
	"alloy_smelter": {sgitem.EssenceEngineering: 6}, "storage_controller": {sgitem.EssenceEngineering: 6},
	// crops
	"harvester": {sgitem.EssenceEngineering: 4, sgitem.EssenceFarming: 4},
	// alloys and fluids
	"conduit_upgrade_2": {sgitem.EssenceEngineering: 4, sgitem.EssenceAlchemy: 4}, "conduit_upgrade_3": {sgitem.EssenceEngineering: 6, sgitem.EssenceAlchemy: 8},
	"crucible": {sgitem.EssenceEngineering: 6, sgitem.EssenceAlchemy: 4}, "ore_washer": {sgitem.EssenceEngineering: 6, sgitem.EssenceAlchemy: 4},
	"mob_simulator": {sgitem.EssenceEngineering: 6, sgitem.EssenceAlchemy: 4}, "lava_generator": {sgitem.EssenceEngineering: 6, sgitem.EssenceAlchemy: 6},
	// monsters
	"bone_craft": {sgitem.EssenceHunting: 2}, "slime_block": {sgitem.EssenceHunting: 2},
	"blaze_craft": {sgitem.EssenceAlchemy: 4, sgitem.EssenceHunting: 4}, "ender_chest": {sgitem.EssenceAlchemy: 4, sgitem.EssenceHunting: 4},
	"diamond_armor":  {sgitem.EssenceAlchemy: 2, sgitem.EssenceHunting: 4},
	"dimension_gate": {sgitem.EssenceEngineering: 8, sgitem.EssenceAlchemy: 6, sgitem.EssenceHunting: 4},
	// magic plants
	"magic_botany": {sgitem.EssenceAlchemy: 6, sgitem.EssenceHunting: 4, sgitem.EssenceFarming: 4}, "dyeing": {sgitem.EssenceMagic: 4},
	"magic_gems":    {sgitem.EssenceAlchemy: 6, sgitem.EssenceHunting: 4, sgitem.EssenceMagic: 4},
	"magic_essence": {sgitem.EssenceAlchemy: 6, sgitem.EssenceHunting: 6, sgitem.EssenceMagic: 6},
	"magic_sieve":   {sgitem.EssenceEngineering: 8, sgitem.EssenceAlchemy: 8, sgitem.EssenceHunting: 6, sgitem.EssenceMagic: 6},
	"config_copier": {sgitem.EssenceEngineering: 8, sgitem.EssenceAlchemy: 6},
	// machine tiers
	"machine_tier_1": {sgitem.EssenceEngineering: 8, sgitem.EssenceAlchemy: 8},
	"machine_tier_2": {sgitem.EssenceEngineering: 16, sgitem.EssenceAlchemy: 16, sgitem.EssenceHunting: 8},
	"machine_tier_3": {sgitem.EssenceEngineering: 24, sgitem.EssenceAlchemy: 24, sgitem.EssenceFarming: 16, sgitem.EssenceHunting: 16, sgitem.EssenceMagic: 16},
	// the projects
	"project_industry": {sgitem.EssenceEngineering: 16, sgitem.EssenceAlchemy: 16},
	"project_hunter":   {sgitem.EssenceEngineering: 8, sgitem.EssenceAlchemy: 8, sgitem.EssenceHunting: 16},
	"project_botanist": {sgitem.EssenceFarming: 16, sgitem.EssenceHunting: 8, sgitem.EssenceMagic: 16},
	"project_dragon":   {sgitem.EssenceEngineering: 32, sgitem.EssenceAlchemy: 32, sgitem.EssenceFarming: 32, sgitem.EssenceHunting: 32, sgitem.EssenceMagic: 32},
}

// WithEssenceCosts returns the researches with the essence costs added after
// their materials, and the laboratory among what they need unless something
// they already need leads to it: the essences come from the laboratory.
func WithEssenceCosts(base []Research) []Research {
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
		costs, ok := EssenceCosts[r.ID]
		if !ok {
			continue
		}
		r.Cost = slices.Clone(r.Cost)
		for _, kind := range sgitem.ResearchEssenceKinds {
			if n := costs[kind]; n > 0 {
				r.Cost = append(r.Cost, Requirement{sgitem.ResearchEssence{Kind: kind}, n})
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

// EssencesIn counts the research essences in a cost.
func EssencesIn(cost []Requirement) int {
	n := 0
	for _, c := range cost {
		if _, ok := c.Item.(sgitem.ResearchEssence); ok {
			n += c.Count
		}
	}
	return n
}

// PourEssences shows the essences of a cost being poured into a research:
// a brewing sound and a splash of the colour of each kind.
func PourEssences(tx *world.Tx, p *player.Player, cost []Requirement) {
	poured := false
	at := p.Position().Add(mgl64.Vec3{0, 1, 0})
	for _, c := range cost {
		if e, ok := c.Item.(sgitem.ResearchEssence); ok {
			tx.AddParticle(at, particle.Splash{Colour: sgitem.EssenceColours[e.Kind]})
			poured = true
		}
	}
	if poured {
		tx.PlaySound(at, sound.PotionBrewed{})
	}
}

// EssenceCost is how many essences of a kind a research costs.
func EssenceCost(r *Research, kind string) int {
	n := 0
	for _, c := range r.Cost {
		if e, ok := c.Item.(sgitem.ResearchEssence); ok && e.Kind == kind {
			n += c.Count
		}
	}
	return n
}
