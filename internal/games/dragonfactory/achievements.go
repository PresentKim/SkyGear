package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
)

// Achievements of the game (docs/design/01-platform.md §5). Drafts.
const (
	achLevel3      = "df_level_3"
	achLevel5      = "df_level_5"
	achLevel8      = "df_level_8"
	achAllMissions = "df_all_missions"
	achMachine     = "df_first_machine"
	achUrgent      = "df_urgent_clear"
	achChunks      = "df_chunks_10"
	achResearch    = "df_research_first"
	achRich        = "df_money_100k"
)

// richMoney is the wallet that earns achRich.
const richMoney = 100_000

func init() {
	account.Register(
		account.Achievement{ID: achLevel3, Name: "기초 다지기", Desc: "드래곤팩토리 Lv.3 달성", Title: "새내기 공장장"},
		account.Achievement{ID: achLevel5, Name: "자동화의 시작", Desc: "드래곤팩토리 Lv.5 달성", Title: "공장장"},
		account.Achievement{ID: achLevel8, Name: "새 시대", Desc: "드래곤팩토리 Lv.8 달성", Title: "시대의 개척자"},
		account.Achievement{ID: achAllMissions, Name: "모든 미션 완료", Desc: "드래곤팩토리의 메인 미션을 모두 완료", Title: "드래곤 장인"},
		account.Achievement{ID: achMachine, Name: "첫 기계", Desc: "기계를 처음 설치", Title: "기술자"},
		account.Achievement{ID: achUrgent, Name: "긴급 해결사", Desc: "긴급 매입을 끝까지 채움", Title: "해결사"},
		account.Achievement{ID: achChunks, Name: "땅 부자", Desc: "섬 청크 10개 소유", Title: "지주"},
		account.Achievement{ID: achResearch, Name: "첫 연구", Desc: "연구대에서 연구 1개 완료", Title: "연구자"},
		account.Achievement{ID: achRich, Name: "부자", Desc: "소지금 100,000G 달성", Title: "부자"},
	)
}

// grantAll gives an achievement to every player in the instance (shared
// progress such as levels).
func (s *session) grantAll(tx *world.Tx, id string) {
	for e := range tx.Players() {
		account.Grant(e.(*player.Player), id)
	}
}

// levelAchievements grants the achievements of reaching the current level
// to everyone in the instance.
func (s *session) levelAchievements(tx *world.Tx) {
	for e := range tx.Players() {
		s.levelAchievementsFor(e.(*player.Player))
	}
}

// levelAchievementsFor grants p the achievements of the save's level, also
// to members who were away when it was reached (called on join).
func (s *session) levelAchievementsFor(p *player.Player) {
	lv := s.state.Level
	for _, a := range []struct {
		level int
		id    string
	}{{3, achLevel3}, {5, achLevel5}, {8, achLevel8}, {len(Levels) + 1, achAllMissions}} {
		if lv >= a.level {
			account.Grant(p, a.id)
		}
	}
}
