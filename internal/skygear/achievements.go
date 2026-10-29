package skygear

import (
	"github.com/presentkim/skygear/internal/account"
	_ "github.com/presentkim/skygear/internal/locales" // this server's texts
	"github.com/presentkim/skygear/internal/skygear/achievement"
)

func init() {
	account.Register(
		account.Keyed(achievement.FirstMachine, achievement.Group("factory")),
		account.Keyed(achievement.Chunks10, achievement.Group("island")),
		account.Keyed(achievement.FirstResearch, achievement.Group("progress")),
	)
}
