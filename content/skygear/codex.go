package skygear

import (
	"fmt"
	"strings"

	"github.com/presentkim/skygear/content/skygear/achievement"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// The codex is the game's recipe viewer: where an item comes from, what it is
// for, what a mesh sifts. It reads the same tables the machines use, so it
// cannot disagree with them.

// codexSieveInputs are the blocks a sieve takes, in the order the codex shows
// them.
func codexSieveInputs() []world.Item {
	return []world.Item{
		block.Gravel{}, block.Sand{}, block.Dirt{}, sgitem.Dust{}, block.Sand{Red: true}, block.SoulSand{},
		sgitem.CrushedBlock{Kind: sgitem.CrushedDeepslate}, sgitem.CrushedBlock{Kind: sgitem.CrushedNetherrack},
		sgitem.CrushedBlock{Kind: sgitem.CrushedBlackstone}, sgitem.CrushedBlock{Kind: sgitem.CrushedEndStone},
	}
}

// openCodex opens the codex: the page of the item p holds, else the page of
// the block p looks at (the one the WAILA panel shows), else the first page.
func (s *session) openCodex(p *player.Player) {
	achievement.Count(p, "codex_open", 1)
	s.codexStart(p)
	if held, _ := p.HeldItems(); !held.Empty() && !isMenuItem(held) {
		s.showCodexItem(p, held)
		return
	}
	if tx := p.Tx(); tx != nil {
		if _, b, ok := lookedAt(p, tx, p.Position(), p.Rotation()); ok {
			if it, isItem := b.(world.Item); isItem {
				if e, found := codexItemByKey(naming.ItemKey(it)); found {
					s.showCodexItem(p, e)
					return
				}
			}
		}
	}
	s.showCodex(p)
}

// showCodex is the front page of the codex: the search tab.
func (s *session) showCodex(p *player.Player) { s.showCodexSearch(p, "") }

func n1(lang string, w world.Item) string { return naming.UIItemName(lang, item.NewStack(w, 1)) }

// codexSources lists where an item comes from, read from the tables of the
// game.
func (s *session) codexSources(lang string, it item.Stack) []string {
	k := naming.ItemKey(it.Item())
	var out []string
	// the sieve: the first mesh that finds it, per block
	for _, in := range codexSieveInputs() {
		for _, t := range sgitem.MeshTiers {
			found := false
			for _, d := range sievetable.Tables[t.Tier][naming.ItemKey(in)] {
				if naming.ItemKey(d.Item.Item()) == k {
					out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.sieve"), sgitem.MeshName(lang, t.Tier), n1(lang, in), textfmt.Permille(d.Permille)))
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
		if got, ok := crusherOutput(item.NewStack(in, 1)); ok && naming.ItemKey(got.Item()) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.crusher"), n1(lang, in)))
		}
	}
	if _, ok := it.Item().(sgitem.MetalDust); ok {
		out = append(out, i18n.T(lang, "codex.src.washer"))
	}
	for _, r := range machinerecipe.Alloys {
		if naming.ItemKey(r.Out) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.alloy"), fmt.Sprintf("%s ×%d + %s ×%d", n1(lang, r.A), r.CountA, n1(lang, r.B), r.CountB)))
		}
	}
	for i := range machinerecipe.Labs {
		if r := &machinerecipe.Labs[i]; naming.ItemKey(r.Out) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.lab"), machinerecipe.LabIngredientsText(lang, r)))
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, d := range sgitem.SimDrops[mob] {
			if naming.ItemKey(d.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.sim"), mobName(lang, mob), d.Chance))
			}
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, in := range codexSieveInputs() {
			for _, d := range sievetable.MagicThemes[mob].Adds[naming.ItemKey(in)] {
				if naming.ItemKey(d.Item.Item()) == k {
					out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.magic"), mobName(lang, mob), n1(lang, in), textfmt.Permille(d.Permille)))
				}
			}
		}
	}
	if _, ok := it.Item().(sgitem.Essence); ok {
		out = append(out, i18n.T(lang, "codex.src.waveDrop"))
	}
	for i, w := range waveLevels {
		for _, r := range w.Reward {
			if naming.ItemKey(r.Item()) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.wave"), i+1))
			}
		}
	}
	for _, pk := range plantKinds {
		for _, y := range pk.yield {
			if naming.ItemKey(y.item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.plant"), naming.UIItemName(lang, item.NewStack(PlantSeed{Seed: pk.seed}, 1))))
			}
		}
	}
	crafts := 0
	for _, cr := range machinerecipe.Crafters() {
		if cr.OutKey == k && crafts < 3 {
			var parts []string
			for _, req := range cr.Reqs {
				parts = append(parts, fmt.Sprintf("%s ×%d", machinerecipe.ReqName(lang, req), req.Count))
			}
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.craft"), strings.Join(parts, " + ")))
			crafts++
		}
	}
	return out
}

// codexUses lists what an item is for.
func (s *session) codexUses(lang string, it item.Stack) []string {
	k := naming.ItemKey(it.Item())
	var out []string
	for i := range researchTree {
		r := &researchTree[i]
		for _, c := range r.Cost {
			if naming.ItemKey(c.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.research"), r.Title(lang), c.Count))
			}
		}
	}
	var made []string
	for _, cr := range machinerecipe.Crafters() {
		if cr.Wants(it) {
			made = append(made, naming.UIItemName(lang, cr.Out))
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
	for _, r := range machinerecipe.Alloys {
		if naming.ItemKey(r.A) == k || naming.ItemKey(r.B) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.alloy"), n1(lang, r.Out)))
		}
	}
	for i := range machinerecipe.Labs {
		for _, in := range machinerecipe.Labs[i].In {
			if naming.ItemKey(in.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.lab"), n1(lang, machinerecipe.Labs[i].Out)))
			}
		}
	}
	if got, ok := crusherOutput(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.crusher"), naming.UIItemName(lang, got)))
	}
	if e, ok := it.Item().(sgitem.Essence); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.sim"), mobName(lang, e.Mob)))
	}
	if fuelDuration(it) > 0 {
		out = append(out, i18n.T(lang, "codex.use.fuel"))
	}
	if sm, ok := smeltOf(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.smelt"), naming.UIItemName(lang, sm)))
	}
	if plantMatter(it) {
		out = append(out, i18n.T(lang, "codex.use.compost"))
	}
	for _, h := range tinker.HeadMaterials {
		if naming.ItemKey(h.Item) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmat."+h.ID), i18n.T(lang, "codex.use.head")))
		}
	}
	for _, h := range tinker.HandleMaterials {
		if naming.ItemKey(h.Item) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "thandle."+h.ID), i18n.T(lang, "codex.use.handle")))
		}
	}
	for _, m := range tinker.ToolModifiers {
		if naming.ItemKey(m.Item) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.tool"), i18n.T(lang, "tmod."+m.ID), fmt.Sprintf("×%d", m.Cost)))
		}
	}
	for _, t := range sgitem.MeshTiers {
		if t.Material != nil && naming.ItemKey(t.Material) == k {
			out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.mesh"), sgitem.MeshName(lang, t.Tier)))
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
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return
	}
	s.openCodex(p)
}
