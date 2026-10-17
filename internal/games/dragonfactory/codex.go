package dragonfactory

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// The codex (docs/design/10-user-flow.md §4): what the game's tables say,
// asked from the game. Players of the genre expect a recipe viewer: where
// does this come from, what is it for, what does this mesh sift. It reads the
// same tables the machines use, so it cannot disagree with them.

// codexSieveInputs are the blocks a sieve takes, in the order the codex shows
// them.
func codexSieveInputs() []world.Item {
	return []world.Item{block.Gravel{}, block.Sand{}, block.Dirt{}, Dust{}}
}

// openCodex opens the codex, or the page of the item p holds.
func (s *session) openCodex(p *player.Player) {
	count(p, "codex_open", 1)
	if held, _ := p.HeldItems(); !held.Empty() && !isMenuItem(held) {
		s.showCodexItem(p, held)
		return
	}
	s.showCodex(p)
}

// showCodex is the front page of the codex.
func (s *session) showCodex(p *player.Player) {
	lang := i18n.Of(p)
	opts := []ui.Option{
		s.opt(i18n.T(lang, "codex.search"), iconOf(item.NewStack(item.Compass{}, 1)), func(p *player.Player, _ *world.Tx) { s.askCodexSearch(p, "") }),
		s.opt(i18n.T(lang, "codex.sieve"), iconOf(machineItem(KindSieve)), func(p *player.Player, _ *world.Tx) { s.showCodexSieve(p) }),
		s.opt(i18n.T(lang, "codex.process"), iconOf(machineItem(KindAlloy)), func(p *player.Player, _ *world.Tx) { s.showCodexText(p, "codex.process", s.codexProcessLines(lang)) }),
		s.opt(i18n.T(lang, "codex.mobs"), iconOf(machineItem(KindGate)), func(p *player.Player, _ *world.Tx) { s.showCodexText(p, "codex.mobs", s.codexMobLines(lang)) }),
		s.opt(i18n.T(lang, "codex.plants"), iconOf(item.NewStack(Petal{Colour: "red"}, 1)), func(p *player.Player, _ *world.Tx) { s.showCodexPlants(p, 0) }),
		s.opt(i18n.T(lang, "codex.gear"), iconOf(machineItem(KindTinker)), func(p *player.Player, _ *world.Tx) { s.showCodexText(p, "codex.gear", codexGearLines(lang)) }),
		s.opt(i18n.T(lang, "codex.hand"), iconOf(item.NewStack(item.Book{}, 1)), func(p *player.Player, _ *world.Tx) {
			if held, _ := p.HeldItems(); !held.Empty() {
				s.showCodexItem(p, held)
				return
			}
			s.showCodexText(p, "codex.hand", []string{i18n.T(lang, "codex.hand.none")})
		}),
	}
	ui.CardMenu(p, i18n.T(lang, "codex.title"), i18n.T(lang, "codex.body"), append(opts, s.backOption())...)
}

// showCodexText shows lines under a title key with a way back to the codex.
func (s *session) showCodexText(p *player.Player, key string, lines []string) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "codex.title")+" · "+i18n.T(lang, key), strings.Join(lines, "\n"),
		s.opt(i18n.T(lang, "codex.back"), iconBack, func(p *player.Player, _ *world.Tx) { s.showCodex(p) }))
}

// showCodexSieve asks which block to sift.
func (s *session) showCodexSieve(p *player.Player) {
	lang := i18n.Of(p)
	var opts []ui.Option
	for _, in := range codexSieveInputs() {
		opts = append(opts, s.opt("§f"+itemName(item.NewStack(in, 1)), iconOf(item.NewStack(in, 1)), func(p *player.Player, _ *world.Tx) { s.showCodexMesh(p, in) }))
	}
	opts = append(opts, s.opt(i18n.T(lang, "codex.back"), iconBack, func(p *player.Player, _ *world.Tx) { s.showCodex(p) }))
	ui.Menu(p, i18n.T(lang, "codex.title")+" · "+i18n.T(lang, "codex.sieve"), i18n.T(lang, "codex.sieve.pick"), opts...)
}

// showCodexMesh asks which mesh sifts the block.
func (s *session) showCodexMesh(p *player.Player, in world.Item) {
	lang := i18n.Of(p)
	var opts []ui.Option
	for _, t := range meshTiers {
		opts = append(opts, s.opt("§f"+meshName(t.tier), iconOf(item.NewStack(Mesh{Tier: t.tier}, 1)), func(p *player.Player, _ *world.Tx) { s.showCodexDrops(p, in, t.tier) }))
	}
	opts = append(opts, s.opt(i18n.T(lang, "codex.back"), iconBack, func(p *player.Player, _ *world.Tx) { s.showCodexSieve(p) }))
	ui.Menu(p, i18n.T(lang, "codex.title")+" · "+itemName(item.NewStack(in, 1)), i18n.T(lang, "codex.mesh.pick"), opts...)
}

// showCodexDrops lists what a mesh finds in a block.
func (s *session) showCodexDrops(p *player.Player, in world.Item, tier string) {
	lang := i18n.Of(p)
	drops := append([]SieveDrop(nil), sieveTables[tier][itemKey(in)]...)
	sort.SliceStable(drops, func(i, j int) bool { return drops[i].Permille > drops[j].Permille })
	var slots []ui.ItemSlot
	for _, d := range drops {
		slots = append(slots, ui.ItemSlot{
			Icon: iconOf(d.Item), Count: formatPermille(d.Permille) + "%",
			Tip: fmt.Sprintf("§f%s\n§7%s", itemName(d.Item), fmt.Sprintf(i18n.T(lang, "codex.sieve.chance"), formatPermille(d.Permille))),
			Run: func(p *player.Player, _ *world.Tx) { s.showCodexItem(p, d.Item) },
		})
	}
	title := fmt.Sprintf(i18n.T(lang, "codex.sieve.list"), itemName(item.NewStack(in, 1)), meshName(tier))
	ui.ItemGrid(p, title, i18n.T(lang, "codex.sieve.listBody"), slots,
		s.opt(i18n.T(lang, "codex.back"), iconBack, func(p *player.Player, _ *world.Tx) { s.showCodexMesh(p, in) }))
}

// codexProcessLines are the crusher chain, the ore doubling and the alloys.
func (s *session) codexProcessLines(lang string) []string {
	n := func(w world.Item) string { return itemName(item.NewStack(w, 1)) }
	lines := []string{
		"§e" + i18n.T(lang, "codex.process.crusher"),
		"§f" + strings.Join([]string{n(block.Stone{}), n(block.Cobblestone{}), n(block.Gravel{}), n(block.Sand{}), n(Dust{})}, " → "),
		"§7" + i18n.T(lang, "codex.process.doubling"),
		"",
		"§e" + i18n.T(lang, "codex.process.alloy"),
	}
	for _, r := range alloyRecipes {
		lines = append(lines, fmt.Sprintf("§f%s ×%d §7←§f %s ×%d + %s ×%d", n(r.Out), r.OutCount, n(r.A), r.CountA, n(r.B), r.CountB))
	}
	lines = append(lines, "", "§e"+i18n.T(lang, "codex.process.lab"))
	for i := range labRecipes {
		r := &labRecipes[i]
		lines = append(lines, fmt.Sprintf(i18n.T(lang, "codex.process.labRecipe"), n(r.Out), r.OutCount, labIngredientsText(r), r.Ticks/20))
	}
	return lines
}

// codexMobLines are the drops of each mob in the simulator and the levels of the
// gate.
func (s *session) codexMobLines(lang string) []string {
	lines := []string{"§e" + i18n.T(lang, "codex.mobs.sim")}
	for _, mob := range mobKinds {
		var parts []string
		for _, d := range simDrops[mob] {
			amount := fmt.Sprint(d.Min)
			if d.Max > d.Min {
				amount = fmt.Sprintf("%d~%d", d.Min, d.Max)
			}
			parts = append(parts, fmt.Sprintf("%s ×%s (%d%%)", n1(d.Item), amount, d.Chance))
		}
		lines = append(lines, fmt.Sprintf("§f%s §7: %s", mobName(mob), strings.Join(parts, ", ")))
	}
	lines = append(lines, "", "§e"+i18n.T(lang, "codex.mobs.magic"))
	lines = append(lines, codexThemeLines(lang)...)
	lines = append(lines, "", "§e"+i18n.T(lang, "codex.mobs.waves"))
	for i, w := range waveLevels {
		var groups, rewards []string
		for _, g := range w.Groups {
			groups = append(groups, fmt.Sprintf("%s ×%d", mobName(g.Mob), g.N))
		}
		for _, r := range w.Reward {
			rewards = append(rewards, fmt.Sprintf("%s ×%d", itemName(r), r.Count()))
		}
		lines = append(lines, fmt.Sprintf("§f%d §7%s (%ds) → §f%s", i+1, strings.Join(groups, ", "), w.Seconds, strings.Join(rewards, ", ")))
	}
	return lines
}

// codexThemeLines are the themes of the magic sieve: for each mob's essence,
// what each block gives (the rare drops starred).
func codexThemeLines(lang string) []string {
	var lines []string
	for _, mob := range mobKinds {
		th := magicThemes[mob]
		lines = append(lines, fmt.Sprintf("§f%s §7(%s)", itemName(item.NewStack(Essence{Mob: mob}, 1)), fmt.Sprintf(i18n.T(lang, "codex.mobs.difficulty"), mobDifficulty[mob])))
		for _, in := range codexSieveInputs() {
			drops := th.Adds[itemKey(in)]
			if len(drops) == 0 {
				continue
			}
			var parts []string
			for _, d := range drops {
				star := ""
				if d.Rare {
					star = "★"
				}
				parts = append(parts, fmt.Sprintf("%s %s%%%s", itemName(d.Item), formatPermille(d.Permille), star))
			}
			head := n1(in)
			if th.Replace[itemKey(in)] {
				head += i18n.T(lang, "codex.mobs.replace")
			}
			lines = append(lines, fmt.Sprintf("  §7%s: %s", head, strings.Join(parts, ", ")))
		}
	}
	return lines
}

// formatPermille writes a per mille chance as a percent ("1.2", "40").
func formatPermille(p int) string {
	if p%10 == 0 {
		return fmt.Sprint(p / 10)
	}
	return fmt.Sprintf("%.1f", float64(p)/10)
}

func n1(w world.Item) string { return itemName(item.NewStack(w, 1)) }

// showCodexPlants lists the magic plants.
func (s *session) showCodexPlants(p *player.Player, page int) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, k := range plantKinds {
		var yields []string
		for _, y := range k.yield {
			amount := fmt.Sprint(y.min)
			if y.max > y.min {
				amount = fmt.Sprintf("%d~%d", y.min, y.max)
			}
			yields = append(yields, fmt.Sprintf("%s ×%s", n1(y.item), amount))
		}
		seed := item.NewStack(PlantSeed{Seed: k.seed}, 1)
		slots = append(slots, ui.ItemSlot{
			Icon: iconOf(item.NewStack(k.yield[0].item, 1)),
			Tip: fmt.Sprintf("§f%s\n§7%s: %s\n§7%s: %s", itemName(seed), i18n.T(lang, "codex.plants.yield"), strings.Join(yields, ", "),
				i18n.T(lang, "codex.plants.grow"), (k.grow / time.Second * time.Second).String()),
			Run: func(p *player.Player, _ *world.Tx) { s.showCodexItem(p, seed) },
		})
	}
	slots, turns := s.slotPage(p, slots, page, s.showCodexPlants)
	ui.ItemGrid(p, i18n.T(lang, "codex.title")+" · "+i18n.T(lang, "codex.plants"), i18n.T(lang, "codex.plants.body"), slots,
		append(turns, s.opt(i18n.T(lang, "codex.back"), iconBack, func(p *player.Player, _ *world.Tx) { s.showCodex(p) }))...)
}

// codexGearLines are the head and handle materials and the modifiers of tools.
func codexGearLines(lang string) []string {
	lines := []string{"§e" + i18n.T(lang, "codex.gear.heads")}
	for _, h := range headMaterials {
		lines = append(lines, fmt.Sprintf("§f%s §7(%s) %s", i18n.T(lang, "tmat."+h.id), n1(h.item), fmt.Sprintf(i18n.T(lang, "tinker.headStats"), h.dur, h.eff, h.attack)))
	}
	lines = append(lines, "", "§e"+i18n.T(lang, "codex.gear.handles"))
	for _, h := range handleMaterials {
		lines = append(lines, fmt.Sprintf("§f%s §7(%s)", i18n.T(lang, "thandle."+h.id), n1(h.item)))
	}
	lines = append(lines, "", "§e"+i18n.T(lang, "codex.gear.mods"))
	for _, m := range toolModifiers {
		lines = append(lines, fmt.Sprintf("§f%s §7: %s ×%d (%s)", i18n.T(lang, "tmod."+m.id), n1(m.item), m.cost, i18n.T(lang, "tmodinfo."+m.id)))
	}
	return lines
}

// codexSources lists where an item comes from, read from the tables of the
// game.
func (s *session) codexSources(lang string, it item.Stack) []string {
	k := itemKey(it.Item())
	var out []string
	// the sieve: the first mesh that finds it, per block
	for _, in := range codexSieveInputs() {
		for _, t := range meshTiers {
			found := false
			for _, d := range sieveTables[t.tier][itemKey(in)] {
				if itemKey(d.Item.Item()) == k {
					out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.sieve"), meshName(t.tier), n1(in), formatPermille(d.Permille)))
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	for _, in := range append([]world.Item{block.Stone{}, block.Cobblestone{}, block.Gravel{}, block.Sand{}}, chunkItems()...) {
		if got, ok := crusherOutput(item.NewStack(in, 1)); ok && itemKey(got.Item()) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.crusher"), n1(in)))
		}
	}
	if _, ok := it.Item().(MetalDust); ok {
		out = append(out, i18n.T(lang, "codex.src.washer"))
	}
	for _, r := range alloyRecipes {
		if itemKey(r.Out) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.alloy"), fmt.Sprintf("%s ×%d + %s ×%d", n1(r.A), r.CountA, n1(r.B), r.CountB)))
		}
	}
	for i := range labRecipes {
		if r := &labRecipes[i]; itemKey(r.Out) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.lab"), labIngredientsText(r)))
		}
	}
	for _, mob := range mobKinds {
		for _, d := range simDrops[mob] {
			if itemKey(d.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.sim"), mobName(mob), d.Chance))
			}
		}
	}
	for _, mob := range mobKinds {
		for _, in := range codexSieveInputs() {
			for _, d := range magicThemes[mob].Adds[itemKey(in)] {
				if itemKey(d.Item.Item()) == k {
					out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.magic"), mobName(mob), n1(in), formatPermille(d.Permille)))
				}
			}
		}
	}
	if _, ok := it.Item().(Essence); ok {
		out = append(out, i18n.T(lang, "codex.src.waveDrop"))
	}
	for i, w := range waveLevels {
		for _, r := range w.Reward {
			if itemKey(r.Item()) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.wave"), i+1))
			}
		}
	}
	for _, pk := range plantKinds {
		for _, y := range pk.yield {
			if itemKey(y.item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.plant"), itemName(item.NewStack(PlantSeed{Seed: pk.seed}, 1))))
			}
		}
	}
	crafts := 0
	for _, cr := range crafterRecipes() {
		if itemKey(cr.out.Item()) == k && crafts < 3 {
			var parts []string
			for _, req := range cr.reqs {
				parts = append(parts, fmt.Sprintf("%s ×%d", reqName(lang, req), req.count))
			}
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.craft"), strings.Join(parts, " + ")))
			crafts++
		}
	}
	return out
}

// chunkItems are the ore chunks.
func chunkItems() []world.Item {
	var out []world.Item
	for _, o := range oreKinds {
		out = append(out, OreChunk{Ore: o.ore})
	}
	return out
}

// codexUses lists what an item is for.
func (s *session) codexUses(lang string, it item.Stack) []string {
	k := itemKey(it.Item())
	var out []string
	for i := range researchTree {
		r := &researchTree[i]
		for _, c := range r.Cost {
			if itemKey(c.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.research"), r.Title(lang), c.Count))
			}
		}
	}
	var made []string
	for _, cr := range crafterRecipes() {
		if cr.wants(it) {
			made = append(made, itemName(cr.out))
		}
	}
	if len(made) > 0 {
		shown := made[:min(len(made), 4)]
		line := fmt.Sprintf(i18n.T(lang, "codex.use.craft"), strings.Join(shown, ", "))
		if len(made) > len(shown) {
			line += fmt.Sprintf(i18n.T(lang, "codex.use.more"), len(made)-len(shown))
		}
		out = append(out, line)
	}
	for _, r := range alloyRecipes {
		if itemKey(r.A) == k || itemKey(r.B) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.alloy"), n1(r.Out)))
		}
	}
	for i := range labRecipes {
		for _, in := range labRecipes[i].In {
			if itemKey(in.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.lab"), n1(labRecipes[i].Out)))
			}
		}
	}
	if got, ok := crusherOutput(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.crusher"), itemName(got)))
	}
	if e, ok := it.Item().(Essence); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.sim"), mobName(e.Mob)))
	}
	if fuelDuration(it) > 0 {
		out = append(out, i18n.T(lang, "codex.use.fuel"))
	}
	if sm, ok := smeltOf(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.smelt"), itemName(sm)))
	}
	if plantMatter(it) {
		out = append(out, i18n.T(lang, "codex.use.compost"))
	}
	for _, h := range headMaterials {
		if itemKey(h.item) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmat."+h.id), i18n.T(lang, "codex.use.head")))
		}
	}
	for _, h := range handleMaterials {
		if itemKey(h.item) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "thandle."+h.id), i18n.T(lang, "codex.use.handle")))
		}
	}
	for _, m := range toolModifiers {
		if itemKey(m.item) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmod."+m.id), fmt.Sprintf("×%d", m.cost)))
		}
	}
	for _, t := range meshTiers {
		if t.material != nil && itemKey(t.material) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.mesh"), meshName(t.tier)))
		}
	}
	return out
}

// showCodexItem shows where an item comes from and what it is for.
func (s *session) showCodexItem(p *player.Player, it item.Stack) {
	lang := i18n.Of(p)
	it = it.Grow(1 - it.Count())
	sources, uses := s.codexSources(lang, it), s.codexUses(lang, it)
	section := func(head string, lines []string) []string {
		out := []string{"§e" + head}
		if len(lines) == 0 {
			return append(out, "§7"+i18n.T(lang, "codex.item.nothing"))
		}
		for _, l := range lines {
			out = append(out, "§f- "+l)
		}
		return out
	}
	body := strings.Join(append(section(i18n.T(lang, "codex.item.sources"), sources), append([]string{""}, section(i18n.T(lang, "codex.item.uses"), uses)...)...), "\n")
	title := i18n.T(lang, "codex.title") + " · " + itemName(it)
	back := s.opt(i18n.T(lang, "codex.back"), iconBack, func(p *player.Player, _ *world.Tx) { s.showCodex(p) })
	search := s.opt(i18n.T(lang, "codex.search"), "", func(p *player.Player, _ *world.Tx) { s.askCodexSearch(p, "") })
	from, into := codexLinks(it)
	var sections []ui.GridSection
	if len(from) > 0 {
		sections = append(sections, ui.GridSection{Heading: i18n.T(lang, "codex.item.madeFrom"), Slots: s.codexLinkSlots(from)})
	}
	if len(into) > 0 {
		sections = append(sections, ui.GridSection{Heading: i18n.T(lang, "codex.item.madeInto"), Slots: s.codexLinkSlots(into)})
	}
	if len(sections) == 0 {
		ui.Menu(p, title, body, search, back)
		return
	}
	ui.ItemGridSections(p, title, body, sections, search, back)
}

// codexCmd is /도감 (and /codex).
type codexCmd struct{ g *Game }

func (c codexCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.menu.Run.1"))
		return
	}
	s.openCodex(p)
}
