package dragonfactory

import (
	"strings"

	"github.com/presentkim/dragonfactory/common/i18n"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/achievement"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/advancement"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
)

// Advancements (docs/design/02-dragonfactory.md §10, 05-sky-progression.md
// §5): Bedrock has no advancements a server can use, so they are ours. An
// advancement is a step of the progression with a hint of how to do it; the
// quest panel shows the first one open. They belong to the save and are
// shared by everyone on the island like the researches. Every advancement
// is shown from the start, in its chapter; one whose parents are not all
// done yet is drawn darker but can be done all the same. The tree is a
// draft.

// advanced reports whether the save has done an advancement.
func (s *session) advanced(id string) bool { return s.state.Advancements[id] }

// advancementOpen reports whether an advancement is not done yet and its
// parents all are.
func (s *session) advancementOpen(a *advancement.Advancement) bool {
	if s.advanced(a.ID) {
		return false
	}
	for _, id := range a.Parents {
		if !s.advanced(id) {
			return false
		}
	}
	return true
}

// nextAdvancement is the tracked advancement while it is not done, else the
// first open one in tree order.
func (s *session) nextAdvancement() (*advancement.Advancement, bool) {
	if a, ok := advancement.ByID(s.state.Tracked); ok && !s.advanced(a.ID) {
		return a, true
	}
	for i := range advancement.Tree {
		if a := &advancement.Tree[i]; s.advancementOpen(a) {
			return a, true
		}
	}
	return nil, false
}

// track makes the quest panel follow an advancement not done yet; tracking the one
// followed already goes back to the tree order.
func (s *session) track(tx *world.Tx, a *advancement.Advancement) {
	if s.state.Tracked == a.ID {
		s.state.Tracked = ""
	} else {
		s.state.Tracked = a.ID
	}
	s.storeState()
	s.updateBoards(tx)
}

// goalResearch is the research an advancement waits on: the one it asks
// for, the one that unlocks the recipe of the item it asks for, or its
// guide.
func goalResearch(a *advancement.Advancement) (*Research, bool) {
	if a.Research != "" {
		return researchByID(a.Research)
	}
	if a.Item != "" {
		for i := range researchTree {
			for _, u := range researchTree[i].Unlocks {
				if naming.UnlockName(u) == naming.UnlockName(a.Item) {
					return &researchTree[i], true
				}
			}
		}
	}
	if a.Guide != "" {
		return researchByID(a.Guide)
	}
	return nil, false
}

// researchStep is the research to do now on the way to r: r itself when it
// is open, else the first open research among what it needs.
func (s *session) researchStep(r *Research) (*Research, bool) {
	if s.researched(r.ID) {
		return nil, false
	}
	if s.researchOpen(r) {
		return r, true
	}
	for _, n := range r.Needs {
		if need, ok := researchByID(n); ok {
			if step, ok := s.researchStep(need); ok {
				return step, true
			}
		}
	}
	return nil, false
}

// advance completes an advancement reported by an event.
func (s *session) advance(tx *world.Tx, id string) {
	if a, ok := advancement.ByID(id); ok && !s.advanced(id) {
		s.completeAdvancements(tx, []*advancement.Advancement{a})
	}
}

// checkAdvancementsHeld completes the item and research advancements met by
// the players with these item names (inventoryNames). It runs every second.
func (s *session) checkAdvancementsHeld(tx *world.Tx, held []advancement.PlayerItems) {
	var have map[string]bool
	var done []*advancement.Advancement
	for i := range advancement.Tree {
		a := &advancement.Tree[i]
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
				have = advancement.HeldItemNames(held)
			}
			if have[advancement.Items[i]] {
				done = append(done, a)
			}
		}
	}
	if len(done) > 0 {
		s.completeAdvancements(tx, done)
	}
}

func init() {
	account.RegisterSetting(
		account.Setting{Key: advancement.SettingToast, NameKey: "setting.advancement.toast", Default: true},
		account.Setting{Key: advancement.SettingChat, NameKey: "setting.advancement.chat", Default: true},
		account.Setting{Key: advancement.SettingSound, NameKey: "setting.advancement.sound", Default: true},
		account.Setting{Key: advancement.SettingQuestPanel, NameKey: "setting.advancement.panel", Default: true},
		account.Setting{Key: veinDiagonalSetting, NameKey: "setting.vein.diagonal"},
	)
}

// completeAdvancements marks advancements done, whether their parents are
// or not, and tells everyone on the island what they did, as each one's
// settings say.
func (s *session) completeAdvancements(tx *world.Tx, as []*advancement.Advancement) {
	if s.state.Advancements == nil {
		s.state.Advancements = map[string]bool{}
	}
	var newly []string
	for _, a := range as {
		if !s.advanced(a.ID) {
			s.state.Advancements[a.ID] = true
			newly = append(newly, a.ID)
		}
	}
	s.grantAdvancementAchievements(tx, newly)
	countAll(tx, "adv_done", int64(len(newly)))
	if len(s.state.Advancements) >= len(advancement.Tree) {
		for e := range tx.Players() {
			account.Grant(e.(*player.Player), achievement.AllAdvancements)
		}
	}
	s.storeState()
	s.updateBoards(tx)
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
