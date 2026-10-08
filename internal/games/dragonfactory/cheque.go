package dragonfactory

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Cheques turn money into items that can be stored, traded and given as
// rewards (docs/design/02-dragonfactory.md §7). Every cheque has an ID in
// the save's ledger, so a cheque can only be redeemed once and only in the
// save that issued it.
const (
	chequeIDKey   = "cheque_id"
	chequeSaveKey = "cheque_save"
)

// newCheque records and returns a cheque item. issuedBy is "" for rewards.
func (s *session) newCheque(amount int64, issuedBy string) (item.Stack, error) {
	id := uuid.NewString()
	if err := s.g.db.IssueCheque(id, s.saveID, amount, issuedBy); err != nil {
		return item.Stack{}, err
	}
	return item.NewStack(item.Paper{}, 1).
		WithCustomName("§r§a수표 · "+formatMoney(amount)).
		WithLore("§7사용하면 소지금에 더해집니다.", "§8"+s.saveName+" 전용").
		WithValue(chequeIDKey, id).
		WithValue(chequeSaveKey, strconv.FormatInt(s.saveID, 10)), nil
}

func isCheque(st item.Stack) bool {
	_, ok := st.Value(chequeIDKey)
	return ok
}

func (s *session) askIssueCheque(p *player.Player) {
	ui.Input(p, "수표 발행", fmt.Sprintf("금액 (소지금 %s)", formatMoney(s.Money(p))), "1000", func(p *player.Player, tx *world.Tx, v string) {
		amount, err := strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(v), ",", ""), 10, 64)
		if err != nil || amount <= 0 {
			p.Message("§c금액을 숫자로 입력해 주세요.")
			return
		}
		if !s.TakeMoney(p, amount) {
			p.Message("§c돈이 부족합니다.")
			return
		}
		c, err := s.newCheque(amount, instance.AccountID(p))
		if err != nil {
			s.AddMoney(p, amount)
			s.g.log.Error("issue cheque", "err", err)
			p.Message("§c수표 발행에 실패했습니다.")
			return
		}
		if _, err := p.Inventory().AddItem(c); err != nil {
			s.dropAt(tx, p.Position(), c)
		}
		s.storeMoney(instance.AccountID(p))
		p.Messagef("§a%s 수표를 발행했습니다.", formatMoney(amount))
	})
}

// redeemCheque cashes the cheque held in p's main hand.
func (s *session) redeemCheque(p *player.Player) {
	held, off := p.HeldItems()
	id, _ := held.Value(chequeIDKey)
	save, _ := held.Value(chequeSaveKey)
	if save != strconv.FormatInt(s.saveID, 10) {
		p.Message("§c다른 세이브에서 발행된 수표입니다.")
		return
	}
	idStr, _ := id.(string)
	amount, err := s.g.db.RedeemCheque(idStr, s.saveID, instance.AccountID(p))
	switch {
	case errors.Is(err, db.ErrChequeUsed), errors.Is(err, db.ErrNotFound):
		p.SetHeldItems(held.Grow(-1), off)
		p.Message("§c이미 사용되었거나 유효하지 않은 수표입니다. 수표를 회수합니다.")
		return
	case err != nil:
		s.g.log.Error("redeem cheque", "err", err)
		p.Message("§c수표를 사용하지 못했습니다.")
		return
	}
	p.SetHeldItems(held.Grow(-1), off)
	s.AddMoney(p, amount)
	s.storeMoney(instance.AccountID(p))
	p.Messagef("§a수표 %s를 사용했습니다.", formatMoney(amount))
}
