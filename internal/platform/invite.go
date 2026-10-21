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
const MaxMembers = 4

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

// inviteCode returns the save's invite code, creating one if needed.
func (pf *Platform) inviteCode(s db.Save, renew bool) (string, error) {
	if s.InviteCode != "" && !renew {
		return s.InviteCode, nil
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

// showInviteCode shows and lets the host renew the invite code of a save.
func (pf *Platform) showInviteCode(p *player.Player, saveID int64) {
	s, err := pf.db.Save(saveID)
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.showInviteCode.1"), err)
		return
	}
	code, err := pf.inviteCode(s, false)
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.showInviteCode.2"), err)
		return
	}
	n, _ := pf.db.ActiveMembers(saveID)
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.invite.showInviteCode.3"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.showInviteCode.4"), s.Name, code, n, MaxMembers),
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.invite.showInviteCode.5"), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) {
			s, _ := pf.db.Save(saveID)
			if _, err := pf.inviteCode(s, true); err != nil {
				pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.showInviteCode.6"), err)
				return
			}
			pf.showInviteCode(p, saveID)
		}},
	)
}

func (pf *Platform) askInviteCode(p *player.Player) {
	ui.Input(p, i18n.T(i18n.Of(p), "platform.invite.askInviteCode.1"), i18n.T(i18n.Of(p), "platform.invite.askInviteCode.2"), "", func(p *player.Player, _ *world.Tx, code string) {
		pf.joinCode(p, code)
	}, func(p *player.Player, _ *world.Tx) { pf.showMultiplayer(p) })
}

// join makes p a member of a save and enters it if its host is playing.
// It must run in p's lobby transaction.
func (pf *Platform) join(p *player.Player, saveID int64) {
	acc := instance.AccountID(p)
	s, err := pf.db.Save(saveID)
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.join.1"), err)
		return
	}
	if s.Owner == acc {
		pf.enter(p, saveID)
		return
	}
	if blocked, _ := pf.db.Blocked(s.Owner, acc); blocked {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.invite.join.2"))
		return
	}
	if _, err := pf.db.MemberRole(saveID, acc); errors.Is(err, db.ErrNotFound) {
		if n, _ := pf.db.ActiveMembers(saveID); n >= MaxMembers {
			ui.Notify(p, i18n.T(i18n.Of(p), "platform.invite.join.3"), MaxMembers)
			return
		}
		if err := pf.db.AddMember(saveID, acc, db.RoleGuest); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.join.4"), err)
			return
		}
		host, _ := pf.db.Account(s.Owner)
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.invite.join.5"), host.Name, s.Name)
		pf.tell(s.Owner, fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.join.6"), p.Name(), s.Name))
	}
	pf.invites.remove(acc, saveID)
	if pf.m.Active(saveID) == nil {
		ui.Flash(p, "%s", i18n.T(i18n.Of(p), "platform.invite.join.7"))
		return
	}
	pf.enter(p, saveID)
}

// showJoined lists the saves p joined as a guest.
func (pf *Platform) showJoined(p *player.Player) {
	saves, err := pf.db.JoinedSaves(instance.AccountID(p))
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.showJoined.1"), err)
		return
	}
	var opts []ui.Option
	for _, s := range saves {
		host, _ := pf.db.Account(s.Owner)
		state := i18n.T(i18n.Of(p), "platform.invite.showJoined.2")
		if pf.m.Active(s.ID) != nil {
			state = i18n.T(i18n.Of(p), "platform.invite.showJoined.3")
		}
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.showJoined.4"), s.Name, host.Name, state, s.Day),
			Icon:  iconSave,
			Run: func(p *player.Player, _ *world.Tx) {
				ui.Menu(p, s.Name, fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.showJoined.5"), host.Name, s.Name, state),
					ui.Option{Label: i18n.T(i18n.Of(p), "platform.invite.showJoined.6"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) { pf.enter(p, s.ID) }},
					ui.Option{Label: i18n.T(i18n.Of(p), "platform.invite.showJoined.7"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) { pf.askLeave(p, s, host) }},
					ui.Back(i18n.T(i18n.Of(p), "platform.invite.showJoined.8"), func(p *player.Player, _ *world.Tx) { pf.showJoined(p) }),
				)
			},
		})
	}
	body := i18n.T(i18n.Of(p), "platform.invite.showJoined.9")
	if len(opts) == 0 {
		body = i18n.T(i18n.Of(p), "platform.invite.showJoined.10")
	}
	opts = append(opts, pf.lobbyBack(p, pf.showMultiplayer))
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.invite.showJoined.11"), body, opts...)
}

func (pf *Platform) askLeave(p *player.Player, s db.Save, host db.Account) {
	ui.Confirm(p, i18n.T(i18n.Of(p), "platform.invite.askLeave.1"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.askLeave.2"), host.Name, s.Name), i18n.T(i18n.Of(p), "platform.invite.askLeave.3"), func(p *player.Player, _ *world.Tx) {
		if err := pf.db.LeaveMember(s.ID, instance.AccountID(p)); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.askLeave.4"), err)
			return
		}
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.invite.askLeave.5"), s.Name)
		pf.tell(s.Owner, fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.askLeave.6"), p.Name(), s.Name))
	})
}

// showInvites lists pending friend invites.
func (pf *Platform) showInvites(p *player.Player) {
	list := pf.invites.list(instance.AccountID(p))
	var opts []ui.Option
	for _, in := range list {
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.showInvites.1"), in.saveName, in.from),
			Icon:  iconInvite,
			Run:   func(p *player.Player, _ *world.Tx) { pf.askAcceptInvite(p, in) },
		})
	}
	body := i18n.T(i18n.Of(p), "platform.invite.showInvites.2")
	if len(opts) == 0 {
		body = i18n.T(i18n.Of(p), "platform.invite.showInvites.3")
	}
	opts = append(opts, pf.lobbyBack(p, pf.showMultiplayer))
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.invite.showInvites.4"), body, opts...)
}

func (pf *Platform) askAcceptInvite(p *player.Player, in invite) {
	ui.Confirm(p, i18n.T(i18n.Of(p), "platform.invite.askAcceptInvite.1"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.invite.askAcceptInvite.2"), in.from, in.saveName), i18n.T(i18n.Of(p), "platform.invite.askAcceptInvite.3"), func(p *player.Player, _ *world.Tx) {
		if pf.m.Location(instance.AccountID(p)) != nil {
			ui.Flash(p, "%s", i18n.T(i18n.Of(p), "platform.invite.askAcceptInvite.4"))
			return
		}
		pf.join(p, in.saveID)
	})
}

// inviteCmd lets the host invite online friends from inside the instance.
type inviteCmd struct {
	playerOnly
	pf *Platform
}

func (c inviteCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	inst := c.pf.m.Location(instance.AccountID(p))
	if inst == nil || inst.Host() != instance.AccountID(p) {
		o.Error(i18n.T(i18n.Of(p), "platform.invite.Run.1"))
		return
	}
	c.pf.showInviteMenu(p, inst)
}

func (pf *Platform) showInviteMenu(p *player.Player, inst *instance.Instance) {
	friends, err := pf.db.Friends(instance.AccountID(p))
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.showInviteMenu.1"), err)
		return
	}
	saveID := inst.Save.ID
	opts := []ui.Option{{Label: i18n.T(i18n.Of(p), "platform.invite.showInviteMenu.2"), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.showInviteCode(p, saveID) }}}
	for _, f := range friends {
		if !pf.isOnline(f.ID) {
			continue
		}
		if role, err := pf.db.MemberRole(saveID, f.ID); err == nil && role != "" && pf.m.Location(f.ID) == inst {
			continue // already here
		}
		opts = append(opts, ui.Option{Label: fmt.Sprintf("%s\n%s", f.Name, pf.status(f.ID)), Icon: iconPlayer, Run: func(p *player.Player, _ *world.Tx) {
			pf.sendInvite(p, inst, f)
		}})
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.invite.showInviteMenu.3"), i18n.T(i18n.Of(p), "platform.invite.showInviteMenu.4"), opts...)
}

func (pf *Platform) sendInvite(p *player.Player, inst *instance.Instance, f db.Account) {
	in := invite{saveID: inst.Save.ID, saveName: inst.Save.Name, from: p.Name(), at: time.Now()}
	pf.invites.add(f.ID, in)
	ui.Flash(p, i18n.T(i18n.Of(p), "platform.invite.sendInvite.1"), f.Name)
	pf.with(f.ID, func(fp *player.Player) {
		if pf.m.Location(f.ID) == nil {
			pf.askAcceptInvite(fp, in)
			return
		}
		ui.Tell(fp, "", fmt.Sprintf(i18n.T(i18n.Of(fp), "platform.invite.sendInvite.2"), p.Name(), inst.Save.Name))
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
		o.Error(i18n.T(i18n.Of(p), "platform.invite.Run.2"))
		return
	}
	code, ok := c.Code.Load()
	if !ok {
		c.pf.askInviteCode(p)
		return
	}
	c.pf.joinCode(p, code)
}

// joinCode joins the save with the given invite code.
func (pf *Platform) joinCode(p *player.Player, code string) {
	code = strings.ToUpper(strings.TrimSpace(code))
	s, err := pf.db.SaveByInviteCode(code)
	if errors.Is(err, db.ErrNotFound) || code == "" {
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.invite.joinCode.1"))
		return
	} else if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.invite.joinCode.2"), err)
		return
	}
	pf.join(p, s.ID)
}
