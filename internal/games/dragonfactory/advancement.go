package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Advancements (docs/design/02-dragonfactory.md §3.5, 05-sky-progression.md
// §5): Bedrock has no advancements a server can use, so they are ours. An
// advancement is a step of the progression with a hint of how to do it; the
// quest panel shows the first one open. They belong to the save and are
// shared by everyone on the island like the researches. The tree is a
// draft.

// Advancement is one step of the advancement tree. It is done by having an
// item, by a research or by an event the game reports (advance).
type Advancement struct {
	ID, Name string
	// Hint says how to do it, one HUD row per line.
	Hint string
	// Parent must be done before it opens (empty for the first).
	Parent string
	// Item is the name of an item to have in the inventory (unlockName
	// form), or empty.
	Item string
	// Research is a research to complete, or empty.
	Research string
	// Icon is the item name shown in the advancement screen; Item when
	// empty.
	Icon string
}

// Advancements done by events (advance).
const (
	advGrowTree = "grow_tree"
	advSift     = "sift"
	advMachine  = "machine"
	advChunk    = "chunk"
)

// advancementTree is every advancement; the quest panel follows this
// order, so the main path comes first.
var advancementTree = []Advancement{
	{ID: "wood", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.1"), Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.2"), Item: "oak_log"},
	{ID: "planks", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.3"), Parent: "wood", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.4"), Item: "oak_planks"},
	{ID: "crafting_table", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.5"), Parent: "planks", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.6"), Item: "crafting_table"},
	{ID: "research_table", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.7"), Parent: "crafting_table", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.8"), Item: "dragonfactory:research_table"},
	{ID: "first_research", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.9"), Parent: "research_table", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.10"), Research: "stick", Icon: "stick"},
	{ID: "pebble", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.11"), Parent: "first_research", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.12"), Item: "dragonfactory:pebble_stone"},
	{ID: "cobble", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.13"), Parent: "pebble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.14"), Item: "cobblestone"},
	{ID: advGrowTree, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.15"), Parent: "wood", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.16"), Icon: "oak_sapling"},
	{ID: "wood_tools", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.17"), Parent: "first_research", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.18"), Item: "wooden_pickaxe"},
	{ID: "composter", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.19"), Parent: "wood_tools", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.20"), Item: "composter"},
	{ID: "barrel", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.21"), Parent: "composter", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.22"), Item: "dragonfactory:barrel_oak"},
	{ID: "stone_axe", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.23"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.24"), Item: "stone_axe"},
	{ID: "fiber", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.25"), Parent: "stone_axe", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.26"), Item: "dragonfactory:bark_fiber"},
	{ID: "string", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.27"), Parent: "fiber", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.28"), Item: "string"},
	{ID: "hammer", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.29"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.30"), Item: "dragonfactory:hammer"},
	{ID: "gravel", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.31"), Parent: "hammer", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.32"), Item: "gravel"},
	{ID: "sieve", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.33"), Parent: "string", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.34"), Item: "dragonfactory:sieve"},
	{ID: advSift, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.35"), Parent: "sieve", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.36"), Icon: "gravel"},
	{ID: "dust", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.37"), Parent: "gravel", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.38"), Item: "dragonfactory:dust"},
	{ID: "furnace", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.39"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.40"), Item: "furnace"},
	{ID: "iron", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.41"), Parent: advSift, Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.42"), Item: "iron_ingot"},
	{ID: advMachine, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.43"), Parent: "iron", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.44"), Icon: "dragonfactory:crusher"},
	{ID: "fluid_conduit", Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.47"), Parent: advMachine, Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.48"), Research: "fluid_conduit", Icon: "dragonfactory:conduit_fluid_1"},
	{ID: "ore_dust", Name: i18n.T(i18n.Default, "advancement.ore_dust.name"), Parent: advMachine, Hint: i18n.T(i18n.Default, "advancement.ore_dust.hint"), Item: "dragonfactory:metal_dust_iron"},
	{ID: "tin_ingot", Name: i18n.T(i18n.Default, "advancement.tin_ingot.name"), Parent: "ore_dust", Hint: i18n.T(i18n.Default, "advancement.tin_ingot.hint"), Item: "dragonfactory:ingot_tin"},
	{ID: "lead_ingot", Name: i18n.T(i18n.Default, "advancement.lead_ingot.name"), Parent: "tin_ingot", Hint: i18n.T(i18n.Default, "advancement.lead_ingot.hint"), Item: "dragonfactory:ingot_lead"},
	{ID: "silver_ingot", Name: i18n.T(i18n.Default, "advancement.silver_ingot.name"), Parent: "ore_dust", Hint: i18n.T(i18n.Default, "advancement.silver_ingot.hint"), Item: "dragonfactory:ingot_silver"},
	{ID: "nickel_ingot", Name: i18n.T(i18n.Default, "advancement.nickel_ingot.name"), Parent: "silver_ingot", Hint: i18n.T(i18n.Default, "advancement.nickel_ingot.hint"), Item: "dragonfactory:ingot_nickel"},
	{ID: "bronze_ingot", Name: i18n.T(i18n.Default, "advancement.bronze_ingot.name"), Parent: "tin_ingot", Hint: i18n.T(i18n.Default, "advancement.bronze_ingot.hint"), Item: "dragonfactory:ingot_bronze"},
	{ID: "electrum_ingot", Name: i18n.T(i18n.Default, "advancement.electrum_ingot.name"), Parent: "silver_ingot", Hint: i18n.T(i18n.Default, "advancement.electrum_ingot.hint"), Item: "dragonfactory:ingot_electrum"},
	{ID: "invar_ingot", Name: i18n.T(i18n.Default, "advancement.invar_ingot.name"), Parent: "nickel_ingot", Hint: i18n.T(i18n.Default, "advancement.invar_ingot.hint"), Item: "dragonfactory:ingot_invar"},
	{ID: "steel_ingot", Name: i18n.T(i18n.Default, "advancement.steel_ingot.name"), Parent: "bronze_ingot", Hint: i18n.T(i18n.Default, "advancement.steel_ingot.hint"), Item: "dragonfactory:ingot_steel"},
	{ID: "alloy_smelter", Name: i18n.T(i18n.Default, "advancement.alloy_smelter.name"), Parent: "tin_ingot", Hint: i18n.T(i18n.Default, "advancement.alloy_smelter.hint"), Item: "dragonfactory:alloy_smelter"},
	{ID: "energy_cell", Name: i18n.T(i18n.Default, "advancement.energy_cell.name"), Parent: "lead_ingot", Hint: i18n.T(i18n.Default, "advancement.energy_cell.hint"), Item: "dragonfactory:energy_cell"},
	{ID: "conduit_upgrade_2", Name: i18n.T(i18n.Default, "advancement.conduit_upgrade_2.name"), Parent: "electrum_ingot", Hint: i18n.T(i18n.Default, "advancement.conduit_upgrade_2.hint"), Item: "dragonfactory:conduit_energy_2"},
	{ID: "conduit_upgrade_3", Name: i18n.T(i18n.Default, "advancement.conduit_upgrade_3.name"), Parent: "steel_ingot", Hint: i18n.T(i18n.Default, "advancement.conduit_upgrade_3.hint"), Item: "dragonfactory:conduit_energy_3"},
	{ID: "hopper", Name: i18n.T(i18n.Default, "advancement.hopper.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.hopper.hint"), Research: "hopper", Icon: "hopper"},
	{ID: "generator", Name: i18n.T(i18n.Default, "advancement.generator.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.generator.hint"), Research: "generator", Icon: "dragonfactory:generator"},
	{ID: "energy_conduit", Name: i18n.T(i18n.Default, "advancement.energy_conduit.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.energy_conduit.hint"), Research: "energy_conduit", Icon: "dragonfactory:conduit_energy_1"},
	{ID: "item_conduit", Name: i18n.T(i18n.Default, "advancement.item_conduit.name"), Parent: "hopper", Hint: i18n.T(i18n.Default, "advancement.item_conduit.hint"), Research: "item_conduit", Icon: "dragonfactory:conduit_item_1"},
	{ID: "drawer", Name: i18n.T(i18n.Default, "advancement.drawer.name"), Parent: "wood_tools", Hint: i18n.T(i18n.Default, "advancement.drawer.hint"), Research: "drawer", Icon: "dragonfactory:drawer"},
	{ID: "auto_sieve", Name: i18n.T(i18n.Default, "advancement.auto_sieve.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.auto_sieve.hint"), Research: "auto_sieve", Icon: "dragonfactory:auto_sieve"},
	{ID: "auto_miner", Name: i18n.T(i18n.Default, "advancement.auto_miner.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.auto_miner.hint"), Research: "auto_miner", Icon: "dragonfactory:auto_miner"},
	{ID: "harvester", Name: i18n.T(i18n.Default, "advancement.harvester.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.harvester.hint"), Research: "harvester", Icon: "dragonfactory:harvester"},
	{ID: "copper", Name: i18n.T(i18n.Default, "advancement.copper.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.copper.hint"), Item: "copper_ingot", Icon: "copper_ingot"},
	{ID: "gold", Name: i18n.T(i18n.Default, "advancement.gold.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.gold.hint"), Item: "gold_ingot", Icon: "gold_ingot"},
	{ID: "mesh_diamond", Name: i18n.T(i18n.Default, "advancement.mesh_diamond.name"), Parent: "gold", Hint: i18n.T(i18n.Default, "advancement.mesh_diamond.hint"), Item: "dragonfactory:mesh_diamond", Icon: "dragonfactory:mesh_diamond"},
	{ID: "mesh_netherite", Name: i18n.T(i18n.Default, "advancement.mesh_netherite.name"), Parent: "mesh_diamond", Hint: i18n.T(i18n.Default, "advancement.mesh_netherite.hint"), Item: "dragonfactory:mesh_netherite", Icon: "dragonfactory:mesh_netherite"},
	{ID: "stone_barrel", Name: i18n.T(i18n.Default, "advancement.stone_barrel.name"), Parent: "barrel", Hint: i18n.T(i18n.Default, "advancement.stone_barrel.hint"), Item: "dragonfactory:barrel_stone", Icon: "dragonfactory:barrel_stone"},
	{ID: "lava", Name: i18n.T(i18n.Default, "advancement.lava.name"), Parent: "stone_barrel", Hint: i18n.T(i18n.Default, "advancement.lava.hint"), Item: "lava_bucket", Icon: "lava_bucket"},
	{ID: "clay", Name: i18n.T(i18n.Default, "advancement.clay.name"), Parent: "barrel", Hint: i18n.T(i18n.Default, "advancement.clay.hint"), Item: "clay_ball", Icon: "clay_ball"},
	{ID: "essence", Name: i18n.T(i18n.Default, "advancement.essence.name"), Parent: "iron", Hint: i18n.T(i18n.Default, "advancement.essence.hint"), Item: "dragonfactory:essence_zombie", Icon: "dragonfactory:essence_zombie"},
	{ID: "mob_simulator", Name: i18n.T(i18n.Default, "advancement.mob_simulator.name"), Parent: "essence", Hint: i18n.T(i18n.Default, "advancement.mob_simulator.hint"), Item: "dragonfactory:mob_simulator", Icon: "dragonfactory:mob_simulator"},
	{ID: "ender_pearl", Name: i18n.T(i18n.Default, "advancement.ender_pearl.name"), Parent: "mob_simulator", Hint: i18n.T(i18n.Default, "advancement.ender_pearl.hint"), Item: "ender_pearl", Icon: "ender_pearl"},
	{ID: "blaze_rod", Name: i18n.T(i18n.Default, "advancement.blaze_rod.name"), Parent: "ender_pearl", Hint: i18n.T(i18n.Default, "advancement.blaze_rod.hint"), Item: "blaze_rod", Icon: "blaze_rod"},
	{ID: "auto_composter", Name: i18n.T(i18n.Default, "advancement.auto_composter.name"), Parent: "generator", Hint: i18n.T(i18n.Default, "advancement.auto_composter.hint"), Research: "auto_composter", Icon: "dragonfactory:auto_composter"},
	{ID: advChunk, Name: i18n.T(i18n.Default, "dragonfactory.advancement.init.45"), Parent: "cobble", Hint: i18n.T(i18n.Default, "dragonfactory.advancement.init.46"), Icon: "grass_block"},
}

// advancementByID returns an advancement of the tree.
func advancementByID(id string) (*Advancement, bool) {
	for i := range advancementTree {
		if advancementTree[i].ID == id {
			return &advancementTree[i], true
		}
	}
	return nil, false
}

// advanced reports whether the save has done an advancement.
func (s *session) advanced(id string) bool { return s.state.Advancements[id] }

// advancementOpen reports whether an advancement is not done yet and its
// parent is.
func (s *session) advancementOpen(a *Advancement) bool {
	return !s.advanced(a.ID) && (a.Parent == "" || s.advanced(a.Parent))
}

// nextAdvancement is the first open advancement in tree order.
func (s *session) nextAdvancement() (*Advancement, bool) {
	for i := range advancementTree {
		if a := &advancementTree[i]; s.advancementOpen(a) {
			return a, true
		}
	}
	return nil, false
}

// advance completes an advancement reported by an event.
func (s *session) advance(tx *world.Tx, id string) {
	if a, ok := advancementByID(id); ok && !s.advanced(id) {
		s.completeAdvancements(tx, []*Advancement{a})
	}
}

// checkAdvancements completes the item and research advancements met by
// the players in tx. It runs every second.
func (s *session) checkAdvancements(tx *world.Tx) {
	var have map[string]bool
	var done []*Advancement
	for i := range advancementTree {
		a := &advancementTree[i]
		if s.advanced(a.ID) {
			continue
		}
		switch {
		case a.Research != "":
			if s.researched(a.Research) {
				done = append(done, a)
			}
		case a.Item != "":
			if have == nil {
				have = heldItemNames(tx)
			}
			if have[unlockName(a.Item)] {
				done = append(done, a)
			}
		}
	}
	if len(done) > 0 {
		s.completeAdvancements(tx, done)
	}
}

// heldItemNames are the names of the items in the inventories of the
// players in tx.
func heldItemNames(tx *world.Tx) map[string]bool {
	names := map[string]bool{}
	for e := range tx.Players() {
		for _, st := range e.(*player.Player).Inventory().Items() {
			name, _ := st.Item().EncodeItem()
			names[name] = true
		}
	}
	return names
}

// completeAdvancements marks advancements done, with the ones before them
// (an item used up within a second would otherwise leave a gap), and tells
// everyone on the island what they did and what comes next.
func (s *session) completeAdvancements(tx *world.Tx, as []*Advancement) {
	if s.state.Advancements == nil {
		s.state.Advancements = map[string]bool{}
	}
	var names []string
	for _, a := range as {
		for cur, ok := a, true; ok && !s.advanced(cur.ID); cur, ok = advancementByID(cur.Parent) {
			s.state.Advancements[cur.ID] = true
		}
		names = append(names, a.Name)
	}
	s.storeState()
	s.updateBoards(tx)
	msg := i18n.T(i18n.Default, "dragonfactory.advancement.completeAdvancements.1") + strings.Join(names, ", ")
	if next, ok := s.nextAdvancement(); ok {
		msg += fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.advancement.completeAdvancements.2"), next.Name, flatHint(next))
	}
	for e := range tx.Players() {
		p := e.(*player.Player)
		p.SendToast(i18n.T(i18n.Of(p), "dragonfactory.advancement.completeAdvancements.3"), strings.Join(names, ", "))
		p.Message(msg)
		p.PlaySound(sound.LevelUp{})
	}
}

// flatHint is an advancement's hint on one line.
func flatHint(a *Advancement) string { return strings.ReplaceAll(a.Hint, "\n", " ") }

// advancementIcon is the icon of an advancement in the advancement screen.
func advancementIcon(a *Advancement) string {
	name := a.Icon
	if name == "" {
		name = a.Item
	}
	if it, ok := world.ItemByName(unlockName(name), 0); ok {
		return iconOf(item.NewStack(it, 1))
	}
	return iconMission
}

// showAdvancements shows the advancement tree: done ones in green, open
// ones with their hint, the rest hidden.
func (s *session) showAdvancements(p *player.Player) {
	body, opts := s.advancementMenu()
	ui.Menu(p, i18n.T(i18n.Of(p), "dragonfactory.advancement.showAdvancements.1"), body, opts...)
}

func (s *session) advancementMenu() (string, []ui.Option) {
	done := 0
	var opts []ui.Option
	for i := range advancementTree {
		a := &advancementTree[i]
		var label, icon string
		switch {
		case s.advanced(a.ID):
			done++
			label, icon = "§a✔ "+a.Name, advancementIcon(a)
		case s.advancementOpen(a):
			label, icon = "§e"+a.Name+"\n§7"+flatHint(a), advancementIcon(a)
		default:
			label, icon = "§8???", iconLocked
		}
		opts = append(opts, s.opt(label, icon, func(p *player.Player, _ *world.Tx) { s.showAdvancementNode(p, a) }))
	}
	body := fmt.Sprintf(i18n.T(i18n.Default, "dragonfactory.advancement.advancementMenu.1"), done, len(advancementTree))
	return body, append(opts, s.backOption())
}

// showAdvancementNode shows one advancement; a hidden one shows only what
// comes before it.
func (s *session) showAdvancementNode(p *player.Player, a *Advancement) {
	title, body := a.Name, ""
	switch {
	case s.advanced(a.ID):
		body = i18n.T(i18n.Of(p), "dragonfactory.advancement.showAdvancementNode.1") + flatHint(a)
	case s.advancementOpen(a):
		body = i18n.T(i18n.Of(p), "dragonfactory.advancement.showAdvancementNode.2") + flatHint(a)
	default:
		title, body = "???", i18n.T(i18n.Of(p), "dragonfactory.advancement.showAdvancementNode.3")
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "dragonfactory.advancement.showAdvancementNode.4")+title, body, ui.Option{Label: i18n.T(i18n.Of(p), "dragonfactory.advancement.showAdvancementNode.5"), Icon: iconBack, Run: func(p *player.Player, _ *world.Tx) { s.showAdvancements(p) }})
}
