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
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
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
		pf.fail(p, "세이브 불러오기", err)
		return
	}
	code, err := pf.inviteCode(s, false)
	if err != nil {
		pf.fail(p, "초대 코드 만들기", err)
		return
	}
	n, _ := pf.db.ActiveMembers(saveID)
	ui.Menu(p, "초대 코드", fmt.Sprintf("§f'%s' 초대 코드\n\n§l§e%s§r\n\n§7로비 메뉴의 '초대 코드 입력'에 이 코드를 넣으면 참여자가 됩니다.\n참여자 %d/%d명", s.Name, code, n, MaxMembers),
		ui.Option{Label: "새 코드 발급 (이전 코드 무효)", Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) {
			s, _ := pf.db.Save(saveID)
			if _, err := pf.inviteCode(s, true); err != nil {
				pf.fail(p, "초대 코드 발급", err)
				return
			}
			pf.showInviteCode(p, saveID)
		}},
	)
}

func (pf *Platform) askInviteCode(p *player.Player) {
	ui.Input(p, "초대 코드 입력", "초대 코드 6자리", "", func(p *player.Player, _ *world.Tx, code string) {
		pf.joinCode(p, code)
	})
}

// join makes p a member of a save and enters it if its host is playing.
// It must run in p's lobby transaction.
func (pf *Platform) join(p *player.Player, saveID int64) {
	acc := instance.AccountID(p)
	s, err := pf.db.Save(saveID)
	if err != nil {
		pf.fail(p, "세이브 참여", err)
		return
	}
	if s.Owner == acc {
		pf.enter(p, saveID)
		return
	}
	if blocked, _ := pf.db.Blocked(s.Owner, acc); blocked {
		p.Message("§c이 세이브에 참여할 수 없습니다.")
		return
	}
	if _, err := pf.db.MemberRole(saveID, acc); errors.Is(err, db.ErrNotFound) {
		if n, _ := pf.db.ActiveMembers(saveID); n >= MaxMembers {
			p.Messagef("§c참여 인원이 가득 찼습니다. (최대 %d명)", MaxMembers)
			return
		}
		if err := pf.db.AddMember(saveID, acc, db.RoleGuest); err != nil {
			pf.fail(p, "세이브 참여", err)
			return
		}
		host, _ := pf.db.Account(s.Owner)
		p.Messagef("§a%s 님의 '%s'에 참여했습니다.", host.Name, s.Name)
		pf.tell(s.Owner, fmt.Sprintf("§a%s 님이 '%s'에 참여했습니다.", p.Name(), s.Name))
	}
	pf.invites.remove(acc, saveID)
	if pf.m.Active(saveID) == nil {
		p.Message("§e호스트가 접속하면 '참여 중인 세이브'에서 입장할 수 있습니다.")
		return
	}
	pf.enter(p, saveID)
}

// showJoined lists the saves p joined as a guest.
func (pf *Platform) showJoined(p *player.Player) {
	saves, err := pf.db.JoinedSaves(instance.AccountID(p))
	if err != nil {
		pf.fail(p, "참여 중인 세이브 불러오기", err)
		return
	}
	var opts []ui.Option
	for _, s := range saves {
		host, _ := pf.db.Account(s.Owner)
		state := "§7호스트 오프라인"
		if pf.m.Active(s.ID) != nil {
			state = "§a입장 가능"
		}
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf("%s §8(%s)\n%s · %d일차", s.Name, host.Name, state, s.Day),
			Icon:  iconSave,
			Run: func(p *player.Player, _ *world.Tx) {
				ui.Menu(p, s.Name, fmt.Sprintf("§f%s 님의 '%s'\n%s", host.Name, s.Name, state),
					ui.Option{Label: "§a입장", Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) { pf.enter(p, s.ID) }},
					ui.Option{Label: "§c탈퇴", Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) { pf.askLeave(p, s, host) }},
					ui.Back("뒤로", func(p *player.Player, _ *world.Tx) { pf.showJoined(p) }),
				)
			},
		})
	}
	body := "초대받아 참여한 세이브입니다. 호스트가 플레이 중일 때만 입장할 수 있습니다."
	if len(opts) == 0 {
		body = "§7참여 중인 세이브가 없습니다. 초대 코드를 입력하거나 친구의 초대를 받아 보세요."
	}
	ui.Menu(p, "참여 중인 세이브", body, opts...)
}

func (pf *Platform) askLeave(p *player.Player, s db.Save, host db.Account) {
	ui.Confirm(p, "세이브 탈퇴", fmt.Sprintf("%s 님의 '%s'에서 탈퇴할까요?\n이 세이브의 인벤토리와 돈은 세이브에 남아 호스트가 관리합니다.", host.Name, s.Name), "§c탈퇴", func(p *player.Player, _ *world.Tx) {
		if err := pf.db.LeaveMember(s.ID, instance.AccountID(p)); err != nil {
			pf.fail(p, "탈퇴", err)
			return
		}
		p.Messagef("§e'%s'에서 탈퇴했습니다.", s.Name)
		pf.tell(s.Owner, fmt.Sprintf("§e%s 님이 '%s'에서 탈퇴했습니다.", p.Name(), s.Name))
	})
}

// showInvites lists pending friend invites.
func (pf *Platform) showInvites(p *player.Player) {
	list := pf.invites.list(instance.AccountID(p))
	var opts []ui.Option
	for _, in := range list {
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf("%s\n§8%s 님의 초대", in.saveName, in.from),
			Icon:  iconInvite,
			Run:   func(p *player.Player, _ *world.Tx) { pf.askAcceptInvite(p, in) },
		})
	}
	body := "친구에게 받은 초대입니다. 10분 동안 유효합니다."
	if len(opts) == 0 {
		body = "§7받은 초대가 없습니다."
	}
	ui.Menu(p, "받은 초대", body, opts...)
}

func (pf *Platform) askAcceptInvite(p *player.Player, in invite) {
	ui.Confirm(p, "세이브 초대", fmt.Sprintf("%s 님이 '%s'에 초대했습니다.\n참여할까요?", in.from, in.saveName), "§a참여", func(p *player.Player, _ *world.Tx) {
		if pf.m.Location(instance.AccountID(p)) != nil {
			p.Message("§e로비에서 수락할 수 있습니다. 로비의 '받은 초대'에서 다시 수락하세요.")
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
		o.Error("자신의 세이브 안에서 호스트만 초대할 수 있습니다.")
		return
	}
	c.pf.showInviteMenu(p, inst)
}

func (pf *Platform) showInviteMenu(p *player.Player, inst *instance.Instance) {
	friends, err := pf.db.Friends(instance.AccountID(p))
	if err != nil {
		pf.fail(p, "친구 목록 불러오기", err)
		return
	}
	saveID := inst.Save.ID
	opts := []ui.Option{{Label: "초대 코드 보기", Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.showInviteCode(p, saveID) }}}
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
	ui.Menu(p, "초대하기", "초대할 온라인 친구를 고르세요.", opts...)
}

func (pf *Platform) sendInvite(p *player.Player, inst *instance.Instance, f db.Account) {
	in := invite{saveID: inst.Save.ID, saveName: inst.Save.Name, from: p.Name(), at: time.Now()}
	pf.invites.add(f.ID, in)
	p.Messagef("§a%s 님에게 초대를 보냈습니다.", f.Name)
	pf.with(f.ID, func(fp *player.Player) {
		if pf.m.Location(f.ID) == nil {
			pf.askAcceptInvite(fp, in)
			return
		}
		fp.Messagef("§e%s 님이 '%s'에 초대했습니다. §7로비 메뉴의 '받은 초대'에서 수락하세요.", p.Name(), inst.Save.Name)
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
		o.Error("로비에서만 사용할 수 있습니다. /lobby 로 이동하세요.")
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
		p.Message("§c올바르지 않은 초대 코드입니다.")
		return
	} else if err != nil {
		pf.fail(p, "초대 코드 확인", err)
		return
	}
	pf.join(p, s.ID)
}
