package platform

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

// MaxMembers is the maximum number of members of a save, host included
// (balancing draft).
const MaxMembers = 10

// inviteTTL is how long a friend invite stays valid. It is stored, so a
// friend who is offline finds it when they come back.
const inviteTTL = 7 * 24 * time.Hour

// codeAlphabet avoids look-alike characters (0/O, 1/I).
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}

// liveCode is the invite code a host turned on for one session of a save:
// it belongs to that instance, so it is gone once the instance closes.
type liveCode struct {
	code string
	inst *instance.Instance
}

// inviteCodes are the invite codes turned on, by save. A save has none until
// its host turns one on inside it; turning it on again makes a new one.
type inviteCodes struct {
	mu     sync.Mutex
	bySave map[int64]liveCode
	byCode map[string]int64
}

// init must be called with mu held.
func (c *inviteCodes) init() {
	if c.bySave == nil {
		c.bySave, c.byCode = map[int64]liveCode{}, map[string]int64{}
	}
}

// enable gives inst's save a new code, replacing the one it had.
func (c *inviteCodes) enable(inst *instance.Instance) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init()
	c.dropLocked(inst.Save.ID)
	code := newCode()
	for _, taken := c.byCode[code]; taken; _, taken = c.byCode[code] {
		code = newCode()
	}
	c.bySave[inst.Save.ID] = liveCode{code: code, inst: inst}
	c.byCode[code] = inst.Save.ID
	return code
}

// disable turns the code of a save off.
func (c *inviteCodes) disable(saveID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init()
	c.dropLocked(saveID)
}

func (c *inviteCodes) dropLocked(saveID int64) {
	if old, ok := c.bySave[saveID]; ok {
		delete(c.byCode, old.code)
		delete(c.bySave, saveID)
	}
}

// dropHost turns off the codes of the saves host hosts, when host leaves
// them, so a closed instance is not kept.
func (c *inviteCodes) dropHost(host string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init()
	for id, l := range c.bySave {
		if l.inst.Host() == host {
			c.dropLocked(id)
		}
	}
}

// of returns the code of a save and the instance it was turned on in.
func (c *inviteCodes) of(saveID int64) (liveCode, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init()
	l, ok := c.bySave[saveID]
	return l, ok
}

// save returns the save a code is turned on for.
func (c *inviteCodes) save(code string) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init()
	id, ok := c.byCode[code]
	return id, ok
}

// inviteCode returns the code turned on in inst, if any. A code turned on in
// an earlier session of the save (an instance that closed) is dropped.
func (pf *Platform) inviteCode(inst *instance.Instance) (string, bool) {
	l, ok := pf.codes.of(inst.Save.ID)
	if !ok {
		return "", false
	}
	if l.inst != inst || pf.m.Active(inst.Save.ID) != inst {
		pf.codes.disable(inst.Save.ID)
		return "", false
	}
	return l.code, true
}

// codeSave returns the save a code lets one join: only while the session it
// was turned on in is open.
func (pf *Platform) codeSave(code string) (int64, bool) {
	id, ok := pf.codes.save(code)
	if !ok {
		return 0, false
	}
	l, ok := pf.codes.of(id)
	if !ok || pf.m.Active(id) != l.inst {
		pf.codes.disable(id)
		return 0, false
	}
	return id, true
}

// askInviteCode asks for an invite code and joins its save; closing or going
// back shows the multiplayer screen again.
func (pf *Platform) askInviteCode(p *player.Player) {
	lang := i18n.Of(p)
	ui.Input(p, i18n.T(lang, "platform.invite.code.enter"), i18n.T(lang, "platform.invite.code.label"), "", func(p *player.Player, _ *world.Tx, code string) {
		if !pf.joinCode(p, code) {
			pf.showMultiplayerAt(p, "")
		}
	}, func(p *player.Player, _ *world.Tx) { pf.showMultiplayerAt(p, "") })
}

// join makes p a member of a save and enters it if its host is playing.
// It must run in p's lobby transaction.
func (pf *Platform) join(p *player.Player, saveID int64) {
	acc := instance.AccountID(p)
	s, err := pf.db.Save(saveID)
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.joining"), err)
		return
	}
	if s.Owner == acc {
		pf.enter(p, saveID)
		return
	}
	if blocked, _ := pf.db.Blocked(s.Owner, acc); blocked {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.invite.denied"))
		return
	}
	if _, err := pf.db.MemberRole(saveID, acc); errors.Is(err, db.ErrNotFound) {
		if n, _ := pf.db.ActiveMembers(saveID); n >= MaxMembers {
			ui.Notify(p, i18n.T(i18n.Of(p), "platform.invite.full"), MaxMembers)
			return
		}
		if err := pf.db.AddMember(saveID, acc, db.RoleGuest); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.joining"), err)
			return
		}
		host, _ := pf.db.Account(s.Owner)
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.invite.joined"), host.Name, s.Name)
		pf.tell(s.Owner, "platform.invite.memberJoined", p.Name(), s.Name)
	}
	_ = pf.db.RemoveInvite(saveID, acc)
	if pf.m.Active(saveID) == nil {
		// the save waits on the multiplayer screen until its host plays
		pf.showMultiplayerAt(p, fmt.Sprintf("s%d", saveID))
		return
	}
	pf.enter(p, saveID)
}

// showMultiplayer is the lobby's multiplayer screen: a row that asks for an
// invite code, the saves p joined as a guest, then the open saves of the
// friends who made p their close friend. The invites p got are notices
// (notify.go).
func (pf *Platform) showMultiplayer(p *player.Player) { pf.showMultiplayerAt(p, "") }

// showMultiplayerAt shows the screen with the row sel chosen ("s<save>",
// "h<host>"; the first one when empty).
func (pf *Platform) showMultiplayerAt(p *player.Player, sel string) {
	lang := i18n.Of(p)
	acc := instance.AccountID(p)
	saves, err := pf.db.JoinedSaves(acc)
	if err != nil {
		pf.fail(p, i18n.T(lang, "platform.invite.load"), err)
		return
	}
	if sel == "" && len(saves) > 0 {
		sel = fmt.Sprintf("s%d", saves[0].ID)
	}
	again := func(id string) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) { pf.showMultiplayerAt(p, id) }
	}
	sp := ui.ListSplit{Rows: []ui.ListRow{{
		Label: ui.ListRowText(i18n.T(lang, "platform.invite.code.enter"), ""), Icon: iconCode,
		Run: func(p *player.Player, _ *world.Tx) { pf.askInviteCode(p) },
	}}}
	hosts := map[string]db.Account{}
	for _, s := range saves {
		id := fmt.Sprintf("s%d", s.ID)
		host, ok := hosts[s.Owner]
		if !ok {
			host, _ = pf.db.Account(s.Owner)
			hosts[s.Owner] = host
		}
		open := pf.m.Active(s.ID) != nil
		state := i18n.T(lang, "platform.invite.save.offline")
		if open {
			state = i18n.T(lang, "platform.invite.save.open")
		}
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label:    ui.ListRowText(s.Name, fmt.Sprintf(i18n.T(lang, "platform.invite.save.row"), host.Name, state, s.Day)),
			Icon:     iconSave,
			Selected: sel == id, Run: again(id),
		})
		if sel != id {
			continue
		}
		if open {
			sp.Actions = append(sp.Actions, ui.Option{Label: i18n.T(lang, "platform.save.enter"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) { pf.enter(p, s.ID) }})
		} else {
			sp.Body = fmt.Sprintf(i18n.T(lang, "platform.invite.save.wait"), host.Name)
		}
		sp.Actions = append(sp.Actions, ui.Option{Label: i18n.T(lang, "platform.invite.leave"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) {
			pf.confirm(p, i18n.T(lang, "platform.invite.leave"), fmt.Sprintf(i18n.T(lang, "platform.invite.leave.ask"), host.Name, s.Name), i18n.T(lang, "platform.invite.leave"), func(p *player.Player, _ *world.Tx) {
				if err := pf.db.LeaveMember(s.ID, instance.AccountID(p)); err != nil {
					pf.fail(p, i18n.T(lang, "platform.invite.leave"), err)
					return
				}
				pf.tell(s.Owner, "platform.invite.left", p.Name(), s.Name)
				pf.showMultiplayerAt(p, "")
			}, again(id))
		}})
	}
	pf.closeFriendRows(p, &sp, saves, sel)
	if len(sp.Rows) == 1 {
		sp.Body = i18n.T(lang, "platform.invite.code.body")
	}
	sp.Actions = append(sp.Actions, pf.lobbyBack(p, pf.showLobbyMenu))
	ui.ListSplitMenu(p, i18n.T(lang, "platform.invite.title"), sp)
}

// closeFriendRows adds to the multiplayer screen the saves p may enter
// without an invite: those of the hosts that marked p as their close friend,
// while they play in them (a save p joined already has its row). The row
// "h<host>" chosen shows the enter button.
func (pf *Platform) closeFriendRows(p *player.Player, sp *ui.ListSplit, joined []db.Save, sel string) {
	lang := i18n.Of(p)
	hosts, err := pf.db.CloseFriendOf(instance.AccountID(p))
	if err != nil {
		pf.log.Error("load close friend hosts", "player", p.Name(), "err", err)
		return
	}
	in := map[int64]bool{}
	for _, s := range joined {
		in[s.ID] = true
	}
	for _, h := range hosts {
		inst := pf.m.Location(h.ID)
		if inst == nil || inst.Host() != h.ID || in[inst.Save.ID] {
			continue
		}
		id := "h" + h.ID
		if sel == "" {
			sel = id
		}
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label:    ui.ListRowText(inst.Save.Name, fmt.Sprintf(i18n.T(lang, "platform.invite.close.row"), h.Name)),
			Icon:     iconFriend,
			Selected: sel == id,
			Run:      func(p *player.Player, _ *world.Tx) { pf.showMultiplayerAt(p, id) },
		})
		if sel != id {
			continue
		}
		sp.Body = fmt.Sprintf(i18n.T(lang, "platform.invite.close.body"), h.Name)
		sp.Actions = append(sp.Actions, ui.Option{Label: i18n.T(lang, "platform.save.enter"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) {
			pf.joinCloseFriend(p, h.ID)
		}})
	}
}

// inviteCmd opens the invite code tab of the island management inside the
// save p hosts.
type inviteCmd struct {
	playerOnly
	pf *Platform
}

func (c inviteCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	if c.pf.hostedSave(p) == nil {
		o.Error(i18n.T(i18n.Of(p), "platform.invite.hostOnly"))
		return
	}
	c.pf.showIsland(p, islandInviteTab, "")
}

// sendInvite invites a friend to the save p hosts. It is stored and waits in
// their notices, also while they are offline.
func (pf *Platform) sendInvite(p *player.Player, inst *instance.Instance, f db.Account) {
	if err := pf.db.AddInvite(inst.Save.ID, f.ID, instance.AccountID(p)); err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.friends.invite"), err)
		return
	}
	ui.Flash(p, i18n.T(i18n.Of(p), "platform.invite.sent"), f.Name)
	pf.notified(f.ID, "platform.invite.received", p.Name(), inst.Save.Name)
}

// joinCmd joins a save with an invite code: /join [code].
type joinCmd struct {
	playerOnly
	pf   *Platform
	Code cmd.Optional[string] `cmd:"code"`
}

func (c joinCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	if c.pf.m.Location(instance.AccountID(p)) != nil {
		o.Error(i18n.T(i18n.Of(p), "platform.lobbyOnly"))
		return
	}
	code, ok := c.Code.Load()
	if !ok {
		c.pf.askInviteCode(p)
		return
	}
	c.pf.joinCode(p, code)
}

// joinCode joins the save with the given invite code. It reports false when
// there is no such code turned on, so the caller can ask again.
func (pf *Platform) joinCode(p *player.Player, code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	saveID, ok := pf.codeSave(code)
	if !ok || code == "" {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.invite.code.invalid"))
		return false
	}
	pf.join(p, saveID)
	return true
}

// openCloseFriendSave is the save of host that p may enter as host's close
// friend: the one host is playing in, nil when host is not in their own save
// or did not mark p.
func (pf *Platform) openCloseFriendSave(p *player.Player, host string) *instance.Instance {
	inst := pf.m.Location(host)
	if inst == nil || inst.Host() != host {
		return nil
	}
	if ok, _ := pf.db.IsCloseFriend(host, instance.AccountID(p)); !ok {
		return nil
	}
	return inst
}

// joinCloseFriend enters the save host is playing in as host's close friend:
// p becomes a member when entering. It must run in p's lobby transaction.
func (pf *Platform) joinCloseFriend(p *player.Player, host string) {
	inst := pf.openCloseFriendSave(p, host)
	if inst == nil {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.invite.closeGone"))
		pf.showMultiplayerAt(p, "")
		return
	}
	pf.join(p, inst.Save.ID)
}
