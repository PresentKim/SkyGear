package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/achievement"
	_ "github.com/presentkim/dragonfactory/internal/locales" // this server's texts
)

func init() {
	account.Register(
		account.Keyed(achievement.FirstMachine, achievement.Group("factory")),
		account.Keyed(achievement.Chunks10, achievement.Group("island")),
		account.Keyed(achievement.FirstResearch, achievement.Group("progress")),
	)
}
