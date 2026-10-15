package platform

import (
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

// The achievements of the account itself, not of a game: coming back.
var loginTiers = []int64{1, 7, 30, 100, 365}

func init() {
	var tiers []account.Tier
	for _, n := range loginTiers {
		id := "pf_login_" + itoa(n)
		account.Register(account.Achievement{
			ID:    id,
			Name:  i18n.T(i18n.Default, "achievement."+id+".name"),
			Desc:  i18n.T(i18n.Default, "achievement."+id+".desc"),
			Title: i18n.T(i18n.Default, "achievement."+id+".title"),
			Group: i18n.T(i18n.Default, "achgroup.account"),
		})
		tiers = append(tiers, account.Tier{Need: n, ID: id})
	}
	account.RegisterCounter("login", tiers...)
}

func itoa(n int64) string {
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}
