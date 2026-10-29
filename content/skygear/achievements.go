package skygear

import (
	"github.com/presentkim/skygear/content/skygear/achievement"
	_ "github.com/presentkim/skygear/content/skygear/locales" // this server's texts
	"github.com/presentkim/skygear/platform/account"
)

func init() {
	account.Register(
		account.Keyed(achievement.FirstMachine, achievement.Group("factory")),
		account.Keyed(achievement.Chunks10, achievement.Group("island")),
		account.Keyed(achievement.FirstResearch, achievement.Group("progress")),
	)
}
