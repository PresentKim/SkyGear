package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// The codex (docs/design/10-user-flow.md §4): what the game's tables say,
// asked from the game. Players of the genre expect a recipe viewer: where
// does this come from, what is it for, what does this mesh sift. It reads the
// same tables the machines use, so it cannot disagree with them.

// codexSieveInputs are the blocks a sieve takes, in the order the codex shows
// them.
func codexSieveInputs() []world.Item {
	return []world.Item{
		block.Gravel{}, block.Sand{}, block.Dirt{}, Dust{}, block.Sand{Red: true}, block.SoulSand{},
		CrushedBlock{Kind: CrushedDeepslate}, CrushedBlock{Kind: CrushedNetherrack},
		CrushedBlock{Kind: CrushedBlackstone}, CrushedBlock{Kind: CrushedEndStone},
	}
}

// openCodex opens the codex: the page of the item p holds, else the page of
// the block p looks at (the one the WAILA panel shows), else the first page.
func (s *session) openCodex(p *player.Player) {
	count(p, "codex_open", 1)
	s.codexStart(p)
	if held, _ := p.HeldItems(); !held.Empty() && !isMenuItem(held) {
		s.showCodexItem(p, held)
		return
	}
	if tx := p.Tx(); tx != nil {
		if _, b, ok := lookedAt(p, tx, p.Position(), p.Rotation()); ok {
			if it, isItem := b.(world.Item); isItem {
				if e, found := codexItemByKey(itemKey(it)); found {
					s.showCodexItem(p, e)
					return
				}
			}
		}
	}
	s.showCodex(p)
}

// showCodex is the front page of the codex: the search tab.
func (s *session) showCodex(p *player.Player) { s.showCodexSearch(p, "", 0) }

// formatPermille writes a per mille chance as a percent ("1.2", "40").
func formatPermille(p int) string {
	if p%10 == 0 {
		return fmt.Sprint(p / 10)
	}
	return fmt.Sprintf("%.1f", float64(p)/10)
}

func n1(lang string, w world.Item) string { return itemName(lang, item.NewStack(w, 1)) }

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
					out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.sieve"), meshName(lang, t.tier), n1(lang, in), formatPermille(d.Permille)))
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	for _, in := range crushInputs() {
		if got, ok := crusherOutput(item.NewStack(in, 1)); ok && itemKey(got.Item()) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.crusher"), n1(lang, in)))
		}
	}
	if _, ok := it.Item().(MetalDust); ok {
		out = append(out, i18n.T(lang, "codex.src.washer"))
	}
	for _, r := range alloyRecipes {
		if itemKey(r.Out) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.alloy"), fmt.Sprintf("%s ×%d + %s ×%d", n1(lang, r.A), r.CountA, n1(lang, r.B), r.CountB)))
		}
	}
	for i := range labRecipes {
		if r := &labRecipes[i]; itemKey(r.Out) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.lab"), labIngredientsText(lang, r)))
		}
	}
	for _, mob := range mobKinds {
		for _, d := range simDrops[mob] {
			if itemKey(d.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.sim"), mobName(lang, mob), d.Chance))
			}
		}
	}
	for _, mob := range mobKinds {
		for _, in := range codexSieveInputs() {
			for _, d := range magicThemes[mob].Adds[itemKey(in)] {
				if itemKey(d.Item.Item()) == k {
					out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.magic"), mobName(lang, mob), n1(lang, in), formatPermille(d.Permille)))
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
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.plant"), itemName(lang, item.NewStack(PlantSeed{Seed: pk.seed}, 1))))
			}
		}
	}
	crafts := 0
	for _, cr := range crafterRecipes() {
		if cr.outKey == k && crafts < 3 {
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
			made = append(made, itemName(lang, cr.out))
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
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.alloy"), n1(lang, r.Out)))
		}
	}
	for i := range labRecipes {
		for _, in := range labRecipes[i].In {
			if itemKey(in.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.lab"), n1(lang, labRecipes[i].Out)))
			}
		}
	}
	if got, ok := crusherOutput(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.crusher"), itemName(lang, got)))
	}
	if e, ok := it.Item().(Essence); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.sim"), mobName(lang, e.Mob)))
	}
	if fuelDuration(it) > 0 {
		out = append(out, i18n.T(lang, "codex.use.fuel"))
	}
	if sm, ok := smeltOf(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.smelt"), itemName(lang, sm)))
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
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.mesh"), meshName(lang, t.tier)))
		}
	}
	return out
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
