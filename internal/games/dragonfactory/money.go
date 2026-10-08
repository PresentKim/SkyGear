package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/account"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Money returns p's wallet in this save.
func (s *session) Money(p *player.Player) int64 { return s.money[instance.AccountID(p)] }

// AddMoney changes p's wallet by delta and refreshes its sidebar.
func (s *session) AddMoney(p *player.Player, delta int64) {
	s.money[instance.AccountID(p)] += delta
	s.checkRich(p)
	s.updateBoard(p)
}

// checkRich grants the money achievement when p's wallet reached it.
func (s *session) checkRich(p *player.Player) {
	if s.money[instance.AccountID(p)] >= richMoney {
		account.Grant(p, achRich)
	}
}

// TakeMoney removes amount from p's wallet if it has enough.
func (s *session) TakeMoney(p *player.Player, amount int64) bool {
	acc := instance.AccountID(p)
	if amount < 0 || s.money[acc] < amount {
		return false
	}
	s.money[acc] -= amount
	s.updateBoard(p)
	return true
}

// formatMoney formats an amount with thousands separators, e.g. 12,400G.
func formatMoney(v int64) string {
	return formatInt(v) + "G"
}

func formatInt(v int64) string {
	s := fmt.Sprint(v)
	neg := v < 0
	if neg {
		s = s[1:]
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
