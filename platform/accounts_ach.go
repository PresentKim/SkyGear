package platform

import (
	_ "github.com/presentkim/skygear/internal/locales" // this server's texts
	"github.com/presentkim/skygear/platform/account"
)

// The achievements of the account itself, not of a game: coming back.
var loginTiers = []int64{1, 7, 30, 100, 365}

func init() {
	var tiers []account.Tier
	for _, n := range loginTiers {
		id := "pf_login_" + itoa(n)
		account.Register(account.Keyed(id, "achgroup.account"))
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
