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
		account.Keyed(achMachine, achievementGroup("factory")),
		account.Keyed(achChunks, achievementGroup("island")),
		account.Keyed(achResearch, achievementGroup("progress")),
	)
}
