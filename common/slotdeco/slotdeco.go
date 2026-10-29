// Package slotdeco marks item names so that the slotdeco resource pack can
// draw a background (rarity) and a front badge (tier pips) on the slots that
// hold the item.
//
// The client's JSON UI (the container screens) cannot read an item's data,
// only the text of its hover box (the name and lore); the DDUI screens draw
// the rarity from the same marks (common/ui slotDeco). So the marks are invisible colour-code sequences
// inside the name that the pack's bindings look for. A mark must not appear
// anywhere else in item texts.
package slotdeco

import (
	"fmt"
	"strings"
)

// Rarity is the rarity level of an item, drawn as a coloured slot background
// and a name colour.
type Rarity int

const (
	Common Rarity = iota
	Uncommon
	Rare
	Epic
	Legendary
)

// rarityMarks are the marks of the levels: an obfuscation switch undone at
// once (nothing shows), then the colour the name continues in. The colour is
// part of the mark so that the name is coloured like its slot.
var rarityMarks = [...]string{"", "§k§r§a", "§k§r§b", "§k§r§d", "§k§r§6"}

// The front image of a slot is chosen by a number: 1 to MaxTier are tier pips,
// then come the badges.
const (
	// MaxTier is the highest tier of the pips.
	MaxTier = 4
	// BadgeWarn is a red warning mark: something is wrong (a stalled machine).
	BadgeWarn = 5
	// BadgeLock is a lock: the item is locked.
	BadgeLock = 6
	// MaxFront is the highest front image number.
	MaxFront = 6
)

// Mark returns the mark of the rarity, "" for Common.
func (r Rarity) Mark() string {
	if r < Common || int(r) >= len(rarityMarks) {
		return ""
	}
	return rarityMarks[r]
}

// TierMark returns the mark of the front image n (1 to MaxFront: tier pips
// and badges), "" otherwise. It is put after the name; the colour code in it
// is undone at once.
func TierMark(tier int) string {
	if tier < 1 || tier > MaxFront {
		return ""
	}
	return tierMarks[tier]
}

const markPrefix = "§k"

var tierMarks = func() (m [MaxFront + 1]string) {
	for t := 1; t <= MaxFront; t++ {
		m[t] = fmt.Sprintf("§k§l§r§%d§r", t)
	}
	return m
}()

// Decorate returns name with the rarity mark in front and the front image
// mark after it.
func Decorate(r Rarity, tier int, name string) string {
	return r.Mark() + name + TierMark(tier)
}

// Strip removes every mark from s.
func Strip(s string) string {
	if !strings.Contains(s, markPrefix) {
		return s
	}
	for _, m := range tierMarks[1:] {
		s = strings.ReplaceAll(s, m, "")
	}
	for _, m := range rarityMarks[1:] {
		s = strings.ReplaceAll(s, m, "")
	}
	return s
}

// Of returns the rarity and tier marked in s.
func Of(s string) (Rarity, int) {
	if !strings.Contains(s, markPrefix) {
		return Common, 0
	}
	var r Rarity
	for i := Uncommon; i <= Legendary; i++ {
		if strings.Contains(s, i.Mark()) {
			r = i
		}
	}
	tier := 0
	for t := 1; t <= MaxFront; t++ {
		if strings.Contains(s, tierMarks[t]) {
			tier = t
		}
	}
	return r, tier
}
