package platform

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/presentkim/dragonfactory/common/i18n"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// MaxMembers is the maximum number of members of a save, host included
// (docs/design/01-platform.md §4, balancing draft).
const MaxMembers = 10

// inviteTTL is how long a friend invite stays valid.
const inviteTTL = 10 * time.Minute

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

// invite is a pending friend invite to a save.
type invite struct {
	saveID   int64
	saveName string
	from     string
	at       time.Time
}

type invites struct {
	mu sync.Mutex
	m  map[string][]invite // invited account -> invites
}

// init must be called with mu held.
func (iv *invites) init() {
	if iv.m == nil {
		iv.m = map[string][]invite{}
	}
}

func (iv *invites) add(to string, in invite) {
	iv.mu.Lock()
	defer iv.mu.Unlock()
	iv.init()
	list := iv.m[to][:0]
	for _, o := range iv.m[to] {
		if o.saveID != in.saveID && time.Since(o.at) < inviteTTL {
			list = append(list, o)
		}
	}
	iv.m[to] = append(list, in)
}

func (iv *invites) list(to string) []invite {
	iv.mu.Lock()
	defer iv.mu.Unlock()
	iv.init()
	var out []invite
	for _, o := range iv.m[to] {
		if time.Since(o.at) < inviteTTL {
			out = append(out, o)
		}
	}
	iv.m[to] = out
	return out
}

func (iv *invites) remove(to string, saveID int64) {
	iv.mu.Lock()
	defer iv.mu.Unlock()
	iv.init()
	list := iv.m[to][:0]
	for _, o := range iv.m[to] {
		if o.saveID != saveID {
			list = append(list, o)
		}
	}
	iv.m[to] = list
}

// inviteCode returns the save's invite code, creating one if needed (a new
// one when renew). The code is read from the database: s may be an older copy
// (the save of a running instance keeps the code it was opened with, and the
// friends tab showed the old code after it was renewed).
func (pf *Platform) inviteCode(s db.Save, renew bool) (string, error) {
	if !renew {
		if cur, err := pf.db.Save(s.ID); err == nil {
			s = cur
		}
		if s.InviteCode != "" {
			return s.InviteCode, nil
		}
	}
	for range 10 {
		code := newCode()
		err := pf.db.SetInviteCode(s.ID, code)
		if errors.Is(err, db.ErrCodeTaken) {
			continue
		}
		return code, err
	}
	return "", errors.New("could not find a free invite code")
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
	pf.invites.remove(acc, saveID)
	if pf.m.Active(saveID) == nil {
		// the save waits on the multiplayer screen until its host plays
		pf.showMultiplayerAt(p, fmt.Sprintf("s%d", saveID))
		return
	}
	pf.enter(p, saveID)
}

// showMultiplayer is the lobby's multiplayer screen (docs/design/01-platform.md
// §1, §4): the row that asks for an invite code first (pressing it asks at
// once: no button on the right does the same), then the invites received and the
// saves p joined as a guest, the list on the left and what can be done with
// the chosen one on the right.
func (pf *Platform) showMultiplayer(p *player.Player) { pf.showMultiplayerAt(p, "") }

// showMultiplayerAt shows the screen with the row sel chosen: "i<save>" for
// an invite or "s<save>" for a joined save; the first invite, else the first
// save when empty.
func (pf *Platform) showMultiplayerAt(p *player.Player, sel string) {
	lang := i18n.Of(p)
	acc := instance.AccountID(p)
	saves, err := pf.db.JoinedSaves(acc)
	if err != nil {
		pf.fail(p, i18n.T(lang, "platform.invite.load"), err)
		return
	}
	invites := pf.invites.list(acc)
	if sel == "" {
		switch {
		case len(invites) > 0:
			sel = fmt.Sprintf("i%d", invites[0].saveID)
		case len(saves) > 0:
			sel = fmt.Sprintf("s%d", saves[0].ID)
		}
	}
	again := func(id string) func(p *player.Player, _ *world.Tx) {
		return func(p *player.Player, _ *world.Tx) { pf.showMultiplayerAt(p, id) }
	}
	sp := ui.ListSplit{Rows: []ui.ListRow{{
		Label: ui.ListRowText(i18n.T(lang, "platform.invite.code.enter"), ""), Icon: iconCode,
		Run: func(p *player.Player, _ *world.Tx) { pf.askInviteCode(p) },
	}}}
	if len(invites) == 0 && len(saves) == 0 {
		sp.Body = i18n.T(lang, "platform.invite.code.body")
	}
	for _, in := range invites {
		id := fmt.Sprintf("i%d", in.saveID)
		left := max(1, int((inviteTTL-time.Since(in.at)).Minutes()+0.5))
		sp.Rows = append(sp.Rows, ui.ListRow{
			Label:    ui.ListRowText(in.saveName, fmt.Sprintf(i18n.T(lang, "platform.invite.row"), in.from, left)),
			Icon:     iconInvite,
			Selected: sel == id, Run: again(id),
		})
		if sel == id {
			sp.Body = fmt.Sprintf(i18n.T(lang, "platform.invite.body"), in.from, in.saveName)
			sp.Actions = []ui.Option{
				{Label: i18n.T(lang, "platform.invite.accept"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) { pf.join(p, in.saveID) }},
				{Label: i18n.T(lang, "platform.invite.decline"), Icon: iconDecline, Run: func(p *player.Player, _ *world.Tx) {
					pf.invites.remove(instance.AccountID(p), in.saveID)
					pf.showMultiplayerAt(p, "")
				}},
			}
		}
	}
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
	sp.Actions = append(sp.Actions, pf.lobbyBack(p, pf.showLobbyMenu))
	ui.ListSplitMenu(p, i18n.T(lang, "platform.invite.title"), sp)
}

// askAcceptInvite asks p in the lobby to join the save they were invited to;
// not now keeps the invite on the multiplayer screen.
func (pf *Platform) askAcceptInvite(p *player.Player, in invite) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "platform.invite.title"), fmt.Sprintf(i18n.T(lang, "platform.invite.body"), in.from, in.saveName),
		ui.Option{Label: i18n.T(lang, "platform.invite.accept"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) {
			if pf.m.Location(instance.AccountID(p)) == nil {
				pf.join(p, in.saveID)
			}
		}},
		ui.Option{Label: i18n.T(lang, "platform.invite.later"), Icon: iconInvite, Run: func(p *player.Player, _ *world.Tx) {
			pf.showMultiplayerAt(p, fmt.Sprintf("i%d", in.saveID))
		}},
	)
}

// inviteCmd opens the friends screen inside the save p hosts, where an online
// friend is invited and the invite code shows.
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
	c.pf.showFriendsAt(p, friendsTab, codeRow)
}

// sendInvite invites a friend to the save p hosts: in the lobby they are
// asked at once, in a save they are told it waits on the multiplayer screen.
func (pf *Platform) sendInvite(p *player.Player, inst *instance.Instance, f db.Account) {
	in := invite{saveID: inst.Save.ID, saveName: inst.Save.Name, from: p.Name(), at: time.Now()}
	pf.invites.add(f.ID, in)
	ui.Flash(p, i18n.T(i18n.Of(p), "platform.invite.sent"), f.Name)
	pf.with(f.ID, func(fp *player.Player) {
		if pf.m.Location(f.ID) == nil {
			pf.askAcceptInvite(fp, in)
			return
		}
		ui.Tell(fp, "", fmt.Sprintf(i18n.T(i18n.Of(fp), "platform.invite.received"), p.Name(), inst.Save.Name))
	})
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
// there is no such code, so the caller can ask again.
func (pf *Platform) joinCode(p *player.Player, code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	s, err := pf.db.SaveByInviteCode(code)
	if errors.Is(err, db.ErrNotFound) || code == "" {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.invite.code.invalid"))
		return false
	} else if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.code.checking"), err)
		return true
	}
	pf.join(p, s.ID)
	return true
}
