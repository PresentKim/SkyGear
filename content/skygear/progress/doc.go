// Package progress is the progression of a save: researches, advancements
// and the recipes they open.
//
//   - Researches (research.go): every crafting recipe but BaseUnlocks is
//     locked. A research, paid whole with items at the research table, opens
//     the recipes of its results for everyone on the island. Researches form
//     a tree (Tree), which the game's content sets with SetTree. A research
//     that opens a facility or a machine gives the researcher some of what it
//     opens (rewards.go), and some researches also cost research essences
//     (essence.go).
//   - Advancements (advancement.go): Bedrock has no advancements a server
//     can use, so they are ours (the advancement package has the tree). Each
//     is a progression step with a hint; the quest panel shows the first open
//     one. They belong to the save like the researches, and one whose parents
//     are not done is drawn darker but can still be done.
//   - Recipe locking (recipes.go): players on the island only get the
//     recipes whose result is open (Player.SetRecipeFilter, Dragonfly patch).
//     The list is sent again at once when a research is done.
//
// The progress is saved with the save (State) and OnChange tells the game
// when it changed.
package progress

import (
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/platform/account"
)

func init() {
	account.RegisterSetting(
		account.Setting{Key: advancement.SettingToast, NameKey: "setting.advancement.toast", Default: true},
		account.Setting{Key: advancement.SettingChat, NameKey: "setting.advancement.chat", Default: true},
		account.Setting{Key: advancement.SettingSound, NameKey: "setting.advancement.sound", Default: true},
		account.Setting{Key: advancement.SettingQuestPanel, NameKey: "setting.advancement.panel", Default: true},
	)
}
