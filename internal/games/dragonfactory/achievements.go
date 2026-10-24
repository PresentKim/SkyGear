package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/account"
	_ "github.com/presentkim/dragonfactory/internal/locales" // this server's texts
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
		account.Achievement{ID: achMachine, NameKey: "dragonfactory.achievements.init.1", DescKey: "dragonfactory.achievements.init.2", TitleKey: "dragonfactory.achievements.init.3", GroupKey: achievementGroup("factory")},
		account.Achievement{ID: achChunks, NameKey: "dragonfactory.achievements.init.4", DescKey: "dragonfactory.achievements.init.5", TitleKey: "dragonfactory.achievements.init.6", GroupKey: achievementGroup("island")},
		account.Achievement{ID: achResearch, NameKey: "dragonfactory.achievements.init.7", DescKey: "dragonfactory.achievements.init.8", TitleKey: "dragonfactory.achievements.init.9", GroupKey: achievementGroup("progress")},
	)
}
