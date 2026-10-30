// Package codex is the game's recipe viewer: where an item comes from, what
// it is for, what a mesh sifts. It reads the same tables the machines use, so
// it cannot disagree with them. The front page searches (codexsearch.go),
// the tabs list items by kind (codextabs.go) and an item's page shows its
// cards (codexpages.go, codexcards.go, codexwork.go); favourites and goals
// are kept per account (codexgoal.go).
package codex

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/sievetable"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wave"
)

// SieveInputs are the blocks a sieve takes, in the order the codex shows
// them.
func SieveInputs() []world.Item {
	return []world.Item{
		block.Gravel{}, block.Sand{}, block.Dirt{}, sgitem.Dust{}, block.Sand{Red: true}, block.SoulSand{},
		sgitem.CrushedBlock{Kind: sgitem.CrushedDeepslate}, sgitem.CrushedBlock{Kind: sgitem.CrushedNetherrack},
		sgitem.CrushedBlock{Kind: sgitem.CrushedBlackstone}, sgitem.CrushedBlock{Kind: sgitem.CrushedEndStone},
	}
}

// Show is the front page of the codex: the search tab.
func Show(s *core.Session, p *player.Player) { showCodexSearch(s, p, "") }

func n1(lang string, w world.Item) string { return naming.UIItemName(lang, item.NewStack(w, 1)) }

// Sources lists where an item comes from, read from the tables of the
// game.
func Sources(s *core.Session, lang string, it item.Stack) []string {
	k := naming.ItemKey(it.Item())
	var out []string
	// the sieve: the first mesh that finds it, per block
	for _, in := range SieveInputs() {
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
	for _, in := range CrushInputs() {
		if got, ok := machine.CrusherOutput(item.NewStack(in, 1)); ok && naming.ItemKey(got.Item()) == k {
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
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.sim"), wave.MobName(lang, mob), d.Chance))
			}
		}
	}
	for _, mob := range sgitem.MobKinds {
		for _, in := range SieveInputs() {
			for _, d := range sievetable.MagicThemes[mob].Adds[naming.ItemKey(in)] {
				if naming.ItemKey(d.Item.Item()) == k {
					out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.magic"), wave.MobName(lang, mob), n1(lang, in), textfmt.Permille(d.Permille)))
				}
			}
		}
	}
	if _, ok := it.Item().(sgitem.Essence); ok {
		out = append(out, i18n.T(lang, "codex.src.waveDrop"))
	}
	for i, w := range wave.Levels {
		for _, r := range w.Reward {
			if naming.ItemKey(r.Item()) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.wave"), i+1))
			}
		}
	}
	for _, pk := range farm.PlantKinds {
		for _, y := range pk.Yield {
			if naming.ItemKey(y.Item) == k {
				out = append(out, fmt.Sprintf(i18n.T(lang, "codex.src.plant"), naming.UIItemName(lang, item.NewStack(farm.PlantSeed{Seed: pk.Seed}, 1))))
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

// Uses lists what an item is for.
func Uses(s *core.Session, lang string, it item.Stack) []string {
	k := naming.ItemKey(it.Item())
	var out []string
	for i := range progress.Tree {
		r := &progress.Tree[i]
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
	if got, ok := machine.CrusherOutput(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.crusher"), naming.UIItemName(lang, got)))
	}
	if e, ok := it.Item().(sgitem.Essence); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.sim"), wave.MobName(lang, e.Mob)))
	}
	if machine.FuelDuration(it) > 0 {
		out = append(out, i18n.T(lang, "codex.use.fuel"))
	}
	if sm, ok := machine.SmeltOf(it); ok {
		out = append(out, fmt.Sprintf(i18n.T(lang, "codex.use.smelt"), naming.UIItemName(lang, sm)))
	}
	if farm.PlantMatter(it) {
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

// codexState is what a save keeps of its codex: its fields are part of the
// save's state document (core.Saved).
type codexState struct {
	// Favourites are each account's favourite item keys in the codex,
	// oldest first (codexgoal.go).
	Favourites map[string][]string `json:"favourites,omitempty"`
}

var codexSaved = core.NewSaved[codexState](nil)

// codexStateOf is what the save of s keeps of its codex.
func codexStateOf(s *core.Session) *codexState { return codexSaved.Of(s) }

// codexNavs is where each player is in the codex and came from.
var codexNavs = core.NewPlayers[*codexNav]()

// codexRecent are the items each player looked at last in the codex.
var codexRecent = core.NewPlayers[[]item.Stack]()
