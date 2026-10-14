package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// Achievements of the game (docs/design/01-platform.md §5). Drafts. The
// level, mission, urgent and money achievements went with those systems
// (13th playtest).
const (
	achMachine  = "df_first_machine"
	achChunks   = "df_chunks_10"
	achResearch = "df_research_first"
)

func init() {
	account.Register(
		account.Achievement{ID: achMachine, Name: i18n.T(i18n.Default, "dragonfactory.achievements.init.1"), Desc: i18n.T(i18n.Default, "dragonfactory.achievements.init.2"), Title: i18n.T(i18n.Default, "dragonfactory.achievements.init.3")},
		account.Achievement{ID: achChunks, Name: i18n.T(i18n.Default, "dragonfactory.achievements.init.4"), Desc: i18n.T(i18n.Default, "dragonfactory.achievements.init.5"), Title: i18n.T(i18n.Default, "dragonfactory.achievements.init.6")},
		account.Achievement{ID: achResearch, Name: i18n.T(i18n.Default, "dragonfactory.achievements.init.7"), Desc: i18n.T(i18n.Default, "dragonfactory.achievements.init.8"), Title: i18n.T(i18n.Default, "dragonfactory.achievements.init.9")},
	)
}
