package platform

import (
	"errors"
	"fmt"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/internal/db"
	"github.com/presentkim/skygear/internal/instance"
)

// copySaveCmd lets an operator copy another player's save into a free slot of
// its own account, for testing: with only a player name it lists that
// player's saves, with a slot it copies the save in it.
type copySaveCmd struct {
	pf     *Platform
	Player string            `cmd:"player"`
	Slot   cmd.Optional[int] `cmd:"slot"`
}

func (c copySaveCmd) Allow(src cmd.Source) bool {
	p, ok := src.(*player.Player)
	return ok && c.pf.isOperator != nil && c.pf.isOperator(p)
}

func (c copySaveCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	lang := i18n.Of(p)
	acc, err := c.pf.db.AccountByName(c.Player)
	if err != nil {
		o.Errorf(i18n.T(lang, "platform.copysave.noPlayer"), c.Player)
		return
	}
	saves, err := c.pf.db.SavesOf(acc.ID)
	if err != nil {
		o.Error(err)
		return
	}
	slot, ok := c.Slot.Load()
	if !ok {
		if len(saves) == 0 {
			o.Errorf(i18n.T(lang, "platform.copysave.noSaves"), acc.Name)
			return
		}
		for _, s := range saves {
			o.Printf(i18n.T(lang, "platform.copysave.entry"), s.Slot, s.Name, s.Day)
		}
		return
	}
	var from db.Save
	for _, s := range saves {
		if s.Slot == slot {
			from = s
		}
	}
	if from.ID == 0 {
		o.Errorf(i18n.T(lang, "platform.copysave.noSlot"), acc.Name, slot)
		return
	}
	free, ok := c.pf.freeSlot(instance.AccountID(p))
	if !ok {
		o.Error(i18n.T(lang, "platform.copysave.full"))
		return
	}
	name := fmt.Sprintf(i18n.T(lang, "platform.copysave.name"), from.Name)
	s, err := c.pf.m.CopySave(from.ID, instance.AccountID(p), free, name)
	switch {
	case errors.Is(err, instance.ErrActive):
		o.Error(i18n.T(lang, "platform.copysave.active"))
	case err != nil:
		c.pf.log.Error("copy save", "save", from.ID, "err", err)
		o.Error(err)
	default:
		o.Printf(i18n.T(lang, "platform.copysave.done"), acc.Name, from.Name, s.Slot)
	}
}

// freeSlot is the first save slot of the account id with no save in it.
func (pf *Platform) freeSlot(id string) (int, bool) {
	saves, err := pf.db.SavesOf(id)
	if err != nil {
		return 0, false
	}
	used := map[int]bool{}
	for _, s := range saves {
		used[s.Slot] = true
	}
	for n := 1; n <= Slots; n++ {
		if !used[n] {
			return n, true
		}
	}
	return 0, false
}
