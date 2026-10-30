package progress

import (
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/platform/account"
)

// OnChange runs when a save's progress changed, to redraw what
// shows it (the quest panel). The game sets it.
var OnChange = func(s *core.Session, tx *world.Tx) {}

// Advanced reports whether the save has done an advancement.
func Advanced(s *core.Session, id string) bool { return StateOf(s).Advancements[id] }

// AdvancementOpen reports whether an advancement is not done yet and its
// parents all are.
func AdvancementOpen(s *core.Session, a *advancement.Advancement) bool {
	if Advanced(s, a.ID) {
		return false
	}
	for _, id := range a.Parents {
		if !Advanced(s, id) {
			return false
		}
	}
	return true
}

// NextAdvancement is the tracked advancement while it is not done, else the
// first open one in tree order.
func NextAdvancement(s *core.Session) (*advancement.Advancement, bool) {
	if a, ok := advancement.ByID(StateOf(s).Tracked); ok && !Advanced(s, a.ID) {
		return a, true
	}
	for i := range advancement.Tree {
		if a := &advancement.Tree[i]; AdvancementOpen(s, a) {
			return a, true
		}
	}
	return nil, false
}

// Track makes the quest panel follow an advancement not done yet; tracking the one
// followed already goes back to the tree order.
func Track(s *core.Session, tx *world.Tx, a *advancement.Advancement) {
	if StateOf(s).Tracked == a.ID {
		StateOf(s).Tracked = ""
	} else {
		StateOf(s).Tracked = a.ID
	}
	s.MarkDirty()
	OnChange(s, tx)
}

// GoalResearch is the research an advancement waits on: the one it asks
// for, the one that unlocks the recipe of the item it asks for, or its
// guide.
func GoalResearch(a *advancement.Advancement) (*Research, bool) {
	if a.Research != "" {
		return ByID(a.Research)
	}
	if a.Item != "" {
		for i := range Tree {
			for _, u := range Tree[i].Unlocks {
				if naming.UnlockName(u) == naming.UnlockName(a.Item) {
					return &Tree[i], true
				}
			}
		}
	}
	if a.Guide != "" {
		return ByID(a.Guide)
	}
	return nil, false
}

// Step is the research to do now on the way to r: r itself when it
// is open, else the first open research among what it needs.
func Step(s *core.Session, r *Research) (*Research, bool) {
	if Researched(s, r.ID) {
		return nil, false
	}
	if Open(s, r) {
		return r, true
	}
	for _, n := range r.Needs {
		if need, ok := ByID(n); ok {
			if step, ok := Step(s, need); ok {
				return step, true
			}
		}
	}
	return nil, false
}

// Advance completes an advancement reported by an event.
func Advance(s *core.Session, tx *world.Tx, id string) {
	if a, ok := advancement.ByID(id); ok && !Advanced(s, id) {
		CompleteAdvancements(s, tx, []*advancement.Advancement{a})
	}
}

// CheckHeld completes the item and research advancements met by
// the players with these item names (advancement.InventoryNames). It runs every second.
func CheckHeld(s *core.Session, tx *world.Tx, held []advancement.PlayerItems) {
	var have map[string]bool
	var done []*advancement.Advancement
	for i := range advancement.Tree {
		a := &advancement.Tree[i]
		if Advanced(s, a.ID) {
			continue
		}
		switch {
		case a.Research != "":
			if Researched(s, a.Research) {
				done = append(done, a)
			}
		case a.Item != "":
			if have == nil {
				have = advancement.HeldItemNames(held)
			}
			if have[advancement.Items[i]] {
				done = append(done, a)
			}
		}
	}
	if len(done) > 0 {
		CompleteAdvancements(s, tx, done)
	}
}

// CompleteAdvancements marks advancements done, whether their parents are
// or not, and tells everyone on the island what they did, as each one's
// settings say.
func CompleteAdvancements(s *core.Session, tx *world.Tx, as []*advancement.Advancement) {
	if StateOf(s).Advancements == nil {
		StateOf(s).Advancements = map[string]bool{}
	}
	var newly []string
	for _, a := range as {
		if !Advanced(s, a.ID) {
			StateOf(s).Advancements[a.ID] = true
			newly = append(newly, a.ID)
		}
	}
	StateOf(s).Unseen = append(StateOf(s).Unseen, newly...)
	grantAdvancementAchievements(s, tx, newly)
	achievement.CountAll(tx, "adv_done", int64(len(newly)))
	if len(StateOf(s).Advancements) >= len(advancement.Tree) {
		for e := range tx.Players() {
			account.Grant(e.(*player.Player), achievement.AllAdvancements)
		}
	}
	s.MarkDirty()
	OnChange(s, tx)
	for e := range tx.Players() {
		p := e.(*player.Player)
		lang := i18n.Of(p)
		var names []string
		for _, a := range as {
			names = append(names, a.Title(lang))
		}
		if account.On(p, advancement.SettingToast) {
			p.SendToast(i18n.T(lang, "advancement.done.toast"), strings.Join(names, ", "))
		}
		if account.On(p, advancement.SettingChat) {
			p.Message(i18n.T(lang, "advancement.done.chat") + strings.Join(names, ", "))
		}
		if account.On(p, advancement.SettingSound) {
			p.PlaySound(sound.LevelUp{})
		}
	}
}

// Unseen reports whether an advancement is done and was not looked at yet on
// the advancement screen.
func Unseen(s *core.Session, id string) bool {
	return slices.Contains(StateOf(s).Unseen, id)
}

// LatestUnseen is the advancement done last of those not looked at yet.
func LatestUnseen(s *core.Session) (*advancement.Advancement, bool) {
	u := StateOf(s).Unseen
	for i := len(u) - 1; i >= 0; i-- {
		if a, ok := advancement.ByID(u[i]); ok {
			return a, true
		}
	}
	return nil, false
}

// See marks an advancement looked at: its NEW tag goes.
func See(s *core.Session, id string) {
	st := StateOf(s)
	if i := slices.Index(st.Unseen, id); i >= 0 {
		st.Unseen = slices.Delete(st.Unseen, i, i+1)
		s.MarkDirty()
	}
}

// AdvAchievements gives an account achievement to everyone on the island when
// an advancement is done.
var AdvAchievements = map[string]string{
	advancement.Wave1:    achievement.Wave1,
	advancement.Wave8:    achievement.Wave8,
	advancement.ToolMade: achievement.Tinker,
	advancement.Bloom:    achievement.Bloom,
	advancement.Dragon:   achievement.Dragon,
}

// grantAdvancementAchievements gives the achievements of advancements just done.
func grantAdvancementAchievements(s *core.Session, tx *world.Tx, ids []string) {
	for _, id := range ids {
		ach, ok := AdvAchievements[id]
		if !ok {
			continue
		}
		for e := range tx.Players() {
			account.Grant(e.(*player.Player), ach)
		}
	}
}
