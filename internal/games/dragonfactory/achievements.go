package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/account"
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
		account.Achievement{ID: achMachine, Name: "첫 기계", Desc: "기계를 처음 설치", Title: "기술자"},
		account.Achievement{ID: achChunks, Name: "땅 부자", Desc: "섬 청크 10개 소유", Title: "지주"},
		account.Achievement{ID: achResearch, Name: "첫 연구", Desc: "연구대에서 연구 1개 완료", Title: "연구자"},
	)
}
