package dragonfactory

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func TestChequeRedeemOnce(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	a, b := spawnPlayer(s), spawnPlayer(s)
	for _, h := range []*world.EntityHandle{a, b} {
		if _, err := s.g.db.TouchAccount(h.UUID().String(), "", "p"); err != nil {
			t.Fatal(err)
		}
	}
	var cheque item.Stack
	withPlayer(t, s, a, func(p *player.Player, tx *world.Tx) {
		s.money[p.UUID().String()] = 1000
		if !s.TakeMoney(p, 700) {
			t.Fatal("take money")
		}
		var err error
		if cheque, err = s.newCheque(700, p.UUID().String()); err != nil {
			t.Fatal(err)
		}
	})
	// The cheque is handed to b, which redeems it; a copy is worthless.
	withPlayer(t, s, b, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(cheque, item.Stack{})
		s.redeemCheque(p)
		if s.Money(p) != 700 {
			t.Fatalf("redeemed money = %d", s.Money(p))
		}
		if held, _ := p.HeldItems(); !held.Empty() {
			t.Fatal("cheque not consumed")
		}
		p.SetHeldItems(cheque, item.Stack{})
		s.redeemCheque(p)
		if s.Money(p) != 700 {
			t.Fatal("a copied cheque was redeemed twice")
		}
	})
	withPlayer(t, s, a, func(p *player.Player, tx *world.Tx) {
		if s.Money(p) != 300 {
			t.Fatalf("issuer money = %d", s.Money(p))
		}
	})
}
