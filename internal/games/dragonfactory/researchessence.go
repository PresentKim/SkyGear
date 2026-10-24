package dragonfactory

import (
	"image/color"
	"slices"

	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// Research essences (docs/design/11-research-essences.md): a research of a
// high enough level costs essences next to its materials. Each kind is one
// era of the factory, made in the laboratory (lab.go) from that era's
// materials. They are not the essences of mobs (essence.go).
const (
	EssenceEngineering = "engineering"
	EssenceAlchemy     = "alchemy"
	EssenceFarming     = "farming"
	EssenceHunting     = "hunting"
	EssenceMagic       = "magic"
)

// researchEssenceKinds are the kinds in the order they come in the game.
var researchEssenceKinds = []string{EssenceEngineering, EssenceAlchemy, EssenceFarming, EssenceHunting, EssenceMagic}

// ResearchEssence is the research essence of a kind.
type ResearchEssence struct{ Kind string }

func (e ResearchEssence) EncodeItem() (string, int16) {
	return "dragonfactory:research_essence_" + e.Kind, 0
}
func (e ResearchEssence) Name() string {
	return i18n.T(i18n.Default, "name.ResearchEssence."+e.Kind)
}
func (ResearchEssence) Category() category.Category { return category.Items() }

func init() {
	for _, k := range researchEssenceKinds {
		world.RegisterItem(ResearchEssence{Kind: k})
	}
}

// researchEssenceCosts are the research essences the researches cost next to
// their materials, by research: how many of the essence of each kind
// (docs/design/11-research-essences.md §5). Researches before the laboratory,
// and the tools and armour, cost none.
var researchEssenceCosts = map[string]map[string]int{
	// the first machines and conduits
	"energy_conduit": {EssenceEngineering: 2}, "item_conduit": {EssenceEngineering: 2}, "drawer_upgrades": {EssenceEngineering: 2},
	// automation
	"crusher": {EssenceEngineering: 4}, "electric_furnace": {EssenceEngineering: 4}, "auto_sieve": {EssenceEngineering: 4},
	"auto_miner": {EssenceEngineering: 4}, "item_vacuum": {EssenceEngineering: 4}, "tree_feller": {EssenceEngineering: 4},
	"energy_cell": {EssenceEngineering: 4}, "tinker_table": {EssenceEngineering: 4},
	"auto_crafter": {EssenceEngineering: 6}, "fluid_conduit": {EssenceEngineering: 6}, "fluid_tank": {EssenceEngineering: 6},
	"alloy_smelter": {EssenceEngineering: 6}, "storage_controller": {EssenceEngineering: 6},
	// crops
	"harvester": {EssenceEngineering: 4, EssenceFarming: 4},
	// alloys and fluids
	"conduit_upgrade_2": {EssenceEngineering: 4, EssenceAlchemy: 4}, "conduit_upgrade_3": {EssenceEngineering: 6, EssenceAlchemy: 8},
	"crucible": {EssenceEngineering: 6, EssenceAlchemy: 4}, "ore_washer": {EssenceEngineering: 6, EssenceAlchemy: 4},
	"mob_simulator": {EssenceEngineering: 6, EssenceAlchemy: 4}, "lava_generator": {EssenceEngineering: 6, EssenceAlchemy: 6},
	// monsters
	"bone_craft": {EssenceHunting: 2}, "slime_block": {EssenceHunting: 2},
	"blaze_craft": {EssenceAlchemy: 4, EssenceHunting: 4}, "ender_chest": {EssenceAlchemy: 4, EssenceHunting: 4},
	"diamond_armor":  {EssenceAlchemy: 2, EssenceHunting: 4},
	"dimension_gate": {EssenceEngineering: 8, EssenceAlchemy: 6, EssenceHunting: 4},
	// magic plants
	"magic_botany": {EssenceAlchemy: 6, EssenceHunting: 4, EssenceFarming: 4}, "dyeing": {EssenceMagic: 4},
	"magic_gems":    {EssenceAlchemy: 6, EssenceHunting: 4, EssenceMagic: 4},
	"magic_essence": {EssenceAlchemy: 6, EssenceHunting: 6, EssenceMagic: 6},
	"magic_sieve":   {EssenceEngineering: 8, EssenceAlchemy: 8, EssenceHunting: 6, EssenceMagic: 6},
	"config_copier": {EssenceEngineering: 8, EssenceAlchemy: 6},
	// machine tiers
	"machine_tier_1": {EssenceEngineering: 8, EssenceAlchemy: 8},
	"machine_tier_2": {EssenceEngineering: 16, EssenceAlchemy: 16, EssenceHunting: 8},
	"machine_tier_3": {EssenceEngineering: 24, EssenceAlchemy: 24, EssenceFarming: 16, EssenceHunting: 16, EssenceMagic: 16},
	// the projects
	"project_industry": {EssenceEngineering: 16, EssenceAlchemy: 16},
	"project_hunter":   {EssenceEngineering: 8, EssenceAlchemy: 8, EssenceHunting: 16},
	"project_botanist": {EssenceFarming: 16, EssenceHunting: 8, EssenceMagic: 16},
	"project_dragon":   {EssenceEngineering: 32, EssenceAlchemy: 32, EssenceFarming: 32, EssenceHunting: 32, EssenceMagic: 32},
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
		for _, kind := range researchEssenceKinds {
			if n := costs[kind]; n > 0 {
				r.Cost = append(r.Cost, Requirement{ResearchEssence{Kind: kind}, n})
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
		if _, ok := c.Item.(ResearchEssence); ok {
			n += c.Count
		}
	}
	return n
}

// essenceColours are the colours of the essences, as splashed when a
// research is paid with them.
var essenceColours = map[string]color.RGBA{
	EssenceEngineering: {214, 52, 52, 255}, EssenceAlchemy: {236, 140, 36, 255}, EssenceFarming: {232, 202, 48, 255},
	EssenceHunting: {64, 172, 72, 255}, EssenceMagic: {156, 76, 214, 255},
}

// pourEssences shows the essences of a cost being poured into a research:
// a brewing sound and a splash of the colour of each kind.
func pourEssences(tx *world.Tx, p *player.Player, cost []Requirement) {
	poured := false
	at := p.Position().Add(mgl64.Vec3{0, 1, 0})
	for _, c := range cost {
		if e, ok := c.Item.(ResearchEssence); ok {
			tx.AddParticle(at, particle.Splash{Colour: essenceColours[e.Kind]})
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
		if e, ok := c.Item.(ResearchEssence); ok && e.Kind == kind {
			n += c.Count
		}
	}
	return n
}
