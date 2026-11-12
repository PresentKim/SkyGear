package netstore

import (
	"fmt"
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
)

// The window of the pattern encoder (a 27 slot chest window):
//
//	slots 0-8   input samples of a processing pattern (any item; they stay)
//	slots 9-11  output samples
//	slot 12     the blank pattern to engrave
//	slot 13     the progress arrow's place (unused)
//	slot 14     the engraved pattern (output; take it)
//
// Using it with sneaking opens the recipe picker (the recipes the research
// opened, like the auto crafter's): a pick engraves the crafting recipe, the
// "process" button engraves the samples. The blank pattern comes from slot
// 12, else from the hand.
const (
	encIn    = 0
	encOut   = 9
	encBlank = 12
	encDone  = 14

	encoderPage = 36
)

// encoderAccepts says what a slot of the encoder takes: any item as a sample,
// a blank pattern on the blank slot.
func encoderAccepts(slot int, st item.Stack) bool {
	if slot >= encIn && slot < encBlank {
		return true
	}
	return isBlank(st)
}

// clickEncoder is the click on an encoder: a sneaking click opens the recipe
// list, any other the machine window.
func clickEncoder(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
	if !p.Sneaking() {
		return false
	}
	m, ok := tx.Block(pos).(machine.Machine)
	if ok && m.State != nil && !s.Debounced(p) {
		showEncoder(s, p, m.State, 0)
	}
	return true
}

// isBlank reports whether a stack is a pattern with no recipe on it.
func isBlank(st item.Stack) bool {
	_, ok := st.Item().(Pattern)
	return ok && stringValue(st, patKindKey) == ""
}

// recipeOpener says which crafting recipes the island's research opened.
func recipeOpener(s *core.Session) func(*machinerecipe.Crafter) bool {
	if progress.LocalOf(s).RecipesFree {
		return func(*machinerecipe.Crafter) bool { return true }
	}
	filter := progress.RecipeFilter(s)
	return func(cr *machinerecipe.Crafter) bool { return filter(cr.Recipe) }
}

func showEncoder(s *core.Session, p *player.Player, st *machine.State, page int) {
	lang := i18n.Of(p)
	open := recipeOpener(s)
	var all []*machinerecipe.Crafter
	for _, cr := range machinerecipe.Crafters() {
		if open(cr) {
			all = append(all, cr)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return naming.ItemName(lang, all[i].Out) < naming.ItemName(lang, all[j].Out) })
	pages := max(1, (len(all)+encoderPage-1)/encoderPage)
	page = min(max(page, 0), pages-1)
	var slots []ui.ItemSlot
	for _, cr := range all[min(page*encoderPage, len(all)):min((page+1)*encoderPage, len(all))] {
		count := ""
		if cr.Out.Count() > 1 {
			count = fmt.Sprint(cr.Out.Count())
		}
		slots = append(slots, ui.ItemSlot{
			Icon: formicon.Of(cr.Out), Count: count, Tip: decor.TipName(lang, cr.Out) + "\n" + strings.Join(cr.Tip(lang), "\n"),
			Run: func(p *player.Player, tx *world.Tx) {
				engrave(s, p, tx, st, func(blank item.Stack) item.Stack { return EncodeCrafting(blank, cr.ID) }, "")
			},
		})
	}
	actions := ui.Pager(p, page, pages, func(p *player.Player, _ *world.Tx, to int) { showEncoder(s, p, st, to) })
	actions = append(actions, ui.Option{Label: i18n.T(lang, "netstore.encoder.processing"), Run: func(p *player.Player, tx *world.Tx) {
		engrave(s, p, tx, st, func(blank item.Stack) item.Stack {
			return EncodeProcessing(blank, sampleSlots(st, encIn, 9), sampleSlots(st, encOut, 3))
		}, "netstore.encoder.noSamples")
	}})
	body := i18n.T(lang, "netstore.encoder.body") + "\n" + fmt.Sprintf(i18n.T(lang, "netstore.encoder.page"), page+1, pages, len(all))
	ui.ItemGrid(p, i18n.T(lang, "netstore.encoder.title"), body, slots, actions...)
}

// sampleSlots reads n sample slots from first.
func sampleSlots(st *machine.State, first, n int) []item.Stack {
	var out []item.Stack
	for i := first; i < first+n; i++ {
		it, _ := st.Inv().Item(i)
		out = append(out, it)
	}
	return out
}

// engrave writes a blank pattern with write: the one in the blank slot (the
// result goes to the done slot if it is free) or else the one in hand (the
// result goes to the player). failKey is the message if write changes nothing.
func engrave(s *core.Session, p *player.Player, tx *world.Tx, st *machine.State, write func(item.Stack) item.Stack, failKey string) {
	if tx.World() != s.World {
		return
	}
	lang := i18n.Of(p)
	inv := st.Inv()
	blank, _ := inv.Item(encBlank)
	fromSlot := isBlank(blank)
	held, left := p.HeldItems()
	if !fromSlot {
		blank = held
	}
	if !isBlank(blank) {
		p.Message(i18n.T(lang, "netstore.encoder.noBlank"))
		return
	}
	made := write(blank.Grow(1 - blank.Count()))
	if isBlank(made) {
		if failKey != "" {
			p.Message(i18n.T(lang, failKey))
		}
		return
	}
	if done, _ := inv.Item(encDone); fromSlot && done.Empty() {
		_ = inv.SetItem(encBlank, item.Stack{})
		_ = inv.SetItem(encDone, made)
	} else {
		if fromSlot {
			_ = inv.SetItem(encBlank, blank.Grow(-1))
		} else {
			p.SetHeldItems(held.Grow(-1), left)
		}
		s.GiveItem(p, tx, p.Position(), made)
	}
	p.Message(i18n.T(lang, "netstore.encoder.done"))
}
