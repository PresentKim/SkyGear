// Package wand is the construction wand, ported from the ConstructionWand mod
// by ThetaDev (MIT, https://github.com/Theta-Dev/ConstructionWand): a click on
// a block face places a copy of the block in front of every block of the
// same kind that shares that face with it, up to the wand's limit. The wand
// takes the blocks from the player's inventory (the hotbar last), wears a
// point a block, and its last placements can be undone.
//
// The settings live on the wand as the value optionsKey: the lock (which
// directions the face spreads), what counts as the same block, whether
// replaceable blocks (grass, water, snow) are built over, and whether the
// blocks are drawn at random from the hotbar.
package wand

import (
	"encoding/json"
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// optionsKey is the item value that holds a wand's settings (JSON).
const optionsKey = "sg_wand"

// Locks: the directions a face spreads in. The zero value spreads in all.
const (
	LockNone = iota
	LockHorizontal
	LockVertical
	LockNorthSouth
	LockEastWest
)

// Locks are the locks in the order the settings list them.
var Locks = []int{LockNone, LockHorizontal, LockVertical, LockNorthSouth, LockEastWest}

// Matches: which blocks behind a free spot count as the clicked one. The zero
// value treats similar blocks (the kinds of dirt) as the same.
const (
	MatchSimilar = iota
	MatchExact
	MatchAny
)

// Matches are the matches in the order the settings list them.
var Matches = []int{MatchExact, MatchSimilar, MatchAny}

// Options are the settings of a wand.
type Options struct {
	Lock  int `json:"l,omitempty"`
	Match int `json:"m,omitempty"`
	// NoReplace keeps replaceable blocks (grass, water, snow layers) from
	// being built over.
	NoReplace bool `json:"r,omitempty"`
	// Random draws every block at random from the block items of the hotbar.
	Random bool `json:"x,omitempty"`
}

// OptionsOf reads the settings of a wand.
func OptionsOf(st item.Stack) Options {
	var o Options
	if v, ok := st.Value(optionsKey); ok {
		if raw, ok := v.(string); ok {
			_ = json.Unmarshal([]byte(raw), &o)
		}
	}
	if o.Lock < LockNone || o.Lock > LockEastWest {
		o.Lock = LockNone
	}
	if o.Match < MatchSimilar || o.Match > MatchAny {
		o.Match = MatchSimilar
	}
	return o
}

// WithOptions returns the wand with the settings and the lore that lists them.
func WithOptions(st item.Stack, o Options, lang string) item.Stack {
	raw, _ := json.Marshal(o)
	w, _ := sgitem.WandOf(st)
	lore := []string{"§7" + fmt.Sprintf(i18n.T(lang, "wand.lore.limit"), sgitem.WandLimit(w.WandTier()))}
	for _, l := range o.Lines(lang) {
		lore = append(lore, "§8"+l)
	}
	return st.WithValue(optionsKey, string(raw)).WithLore(lore...)
}

// Lines are the settings as lines, "Lock: none" and so on.
func (o Options) Lines(lang string) []string {
	return []string{
		i18n.T(lang, "wand.opt.lock") + ": " + i18n.T(lang, LockKey(o.Lock)),
		i18n.T(lang, "wand.opt.match") + ": " + i18n.T(lang, MatchKey(o.Match)),
		i18n.T(lang, "wand.opt.replace") + ": " + i18n.T(lang, onOffKey(!o.NoReplace)),
		i18n.T(lang, "wand.opt.random") + ": " + i18n.T(lang, onOffKey(o.Random)),
	}
}

// NextLock is the lock after l in the order Locks lists them, the first
// after the last.
func NextLock(l int) int { return next(Locks, l) }

// NextMatch is the match after m in the order Matches lists them, the
// first after the last.
func NextMatch(m int) int { return next(Matches, m) }

func next(list []int, v int) int {
	for i, x := range list {
		if x == v {
			return list[(i+1)%len(list)]
		}
	}
	return list[0]
}

// LockKey is the catalog key of a lock's name.
func LockKey(l int) string {
	return "wand.lock." + [...]string{"none", "horizontal", "vertical", "northsouth", "eastwest"}[l]
}

// MatchKey is the catalog key of a match's name.
func MatchKey(m int) string {
	return "wand.match." + [...]string{"similar", "exact", "any"}[m]
}

func onOffKey(on bool) string {
	if on {
		return "wand.on"
	}
	return "wand.off"
}

// allows reports whether the lock lets a face spread along l.
func (o Options) allows(l int) bool { return o.Lock == LockNone || o.Lock == l }

// spread are the offsets a spot of a face clicked on face spreads to, as the
// mod has them: along the face's two axes as the lock allows, and the
// diagonals when both are allowed. start reports whether the lock lets the
// face be built on at all.
func (o Options) spread(face cube.Face) (offsets []cube.Pos, start bool) {
	var a, b []cube.Pos
	var la, lb int
	switch face.Axis() {
	case cube.Y:
		a, la = []cube.Pos{{0, 0, -1}, {0, 0, 1}}, LockNorthSouth
		b, lb = []cube.Pos{{1, 0, 0}, {-1, 0, 0}}, LockEastWest
	case cube.Z:
		a, la = []cube.Pos{{1, 0, 0}, {-1, 0, 0}}, LockHorizontal
		b, lb = []cube.Pos{{0, 1, 0}, {0, -1, 0}}, LockVertical
	default:
		a, la = []cube.Pos{{0, 0, -1}, {0, 0, 1}}, LockHorizontal
		b, lb = []cube.Pos{{0, 1, 0}, {0, -1, 0}}, LockVertical
	}
	okA, okB := o.allows(la), o.allows(lb)
	if okA {
		offsets = append(offsets, a...)
	}
	if okB {
		offsets = append(offsets, b...)
	}
	if okA && okB {
		for _, x := range a {
			for _, y := range b {
				offsets = append(offsets, x.Add(y))
			}
		}
	}
	return offsets, okA || okB
}
