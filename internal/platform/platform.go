// Package platform implements the server around the games: accounts, the
// server guide, the lobby and save slot selection (docs/design/01-platform.md).
package platform

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Slots is the number of save slots per account.
const Slots = 3

// LobbySpawn is the block players stand on top of in the lobby.
var LobbySpawn = cube.Pos{0, 64, 0}

// Platform ties accounts, the lobby and instances together.
type Platform struct {
	log *slog.Logger
	db  *db.DB
	m   *instance.Manager
	// game is the only game type offered to players for now; game selection
	// is hidden until more games exist (docs/design/01-platform.md §1).
	game    instance.Game
	online  Online
	invites invites
}

// New creates the platform. The lobby world must already be built.
func New(log *slog.Logger, d *db.DB, m *instance.Manager, game instance.Game, online Online) *Platform {
	pf := &Platform{log: log, db: d, m: m, game: game, online: online}
	m.Register(game)
	registerCommands(pf)
	return pf
}

// Lobby returns the lobby description for the instance manager.
func Lobby(w *world.World, pf **Platform) instance.Lobby {
	return instance.Lobby{
		World: w,
		Spawn: LobbySpawn.Vec3Middle(),
		Arrive: func(p *player.Player) {
			(*pf).arriveLobby(p)
		},
		Handler: func(*player.Player) player.Handler { return lobbyHandler{pf: *pf} },
	}
}

// BuildLobby places the lobby platform. It runs in the lobby world.
func BuildLobby(tx *world.Tx) {
	for x := -7; x <= 7; x++ {
		for z := -7; z <= 7; z++ {
			var b world.Block = block.Grass{}
			if x == -7 || x == 7 || z == -7 || z == 7 {
				b = block.StoneBricks{}
			}
			tx.SetBlock(LobbySpawn.Add(cube.Pos{x, -1, z}), b, nil)
		}
	}
}

// Join handles a player connecting. It runs in the lobby world.
func (pf *Platform) Join(p *player.Player) {
	acc, err := pf.db.TouchAccount(instance.AccountID(p), p.XUID(), p.Name())
	if err != nil {
		pf.log.Error("touch account", "player", p.Name(), "err", err)
		p.Disconnect("계정 정보를 불러오지 못했습니다. 잠시 후 다시 접속해 주세요.")
		return
	}
	pf.m.Join(p)
	if acc.GuideVersion < GuideVersion {
		pf.showGuide(p, true)
	} else {
		p.Messagef("§a%s에 오신 것을 환영합니다! §7나침반으로 메뉴를 엽니다.", pf.game.Name())
	}
	pf.notifyRequests(p)
}

const lobbyItemKey = "platform_lobby_item"

// arriveLobby gives the lobby menu items.
func (pf *Platform) arriveLobby(p *player.Player) {
	p.SetGameMode(world.GameModeAdventure)
	_ = p.Inventory().SetItem(0, item.NewStack(item.Compass{}, 1).WithCustomName("§r§a메뉴").WithValue(lobbyItemKey, "menu"))
	_ = p.Inventory().SetItem(8, item.NewStack(item.Book{}, 1).WithCustomName("§r§e서버 안내").WithValue(lobbyItemKey, "guide"))
	_ = p.SetHeldSlot(0)
}

// useLobbyItem runs the action of a lobby item held by p.
func (pf *Platform) useLobbyItem(p *player.Player) bool {
	held, _ := p.HeldItems()
	switch v, _ := held.Value(lobbyItemKey); v {
	case "menu", "saves":
		pf.showLobbyMenu(p)
	case "guide":
		pf.showGuide(p, false)
	default:
		return false
	}
	return true
}

// showSaves shows the save slots of p.
func (pf *Platform) showSaves(p *player.Player) {
	saves, err := pf.db.SavesOf(instance.AccountID(p), pf.game.ID())
	if err != nil {
		pf.fail(p, "세이브 목록 불러오기", err)
		return
	}
	bySlot := map[int]db.Save{}
	for _, s := range saves {
		bySlot[s.Slot] = s
	}
	var opts []ui.Option
	for slot := 1; slot <= Slots; slot++ {
		if s, ok := bySlot[slot]; ok {
			opts = append(opts, ui.Option{
				Label: fmt.Sprintf("§l%d. %s§r\n§8%d일차 · 플레이 %s", slot, s.Name, s.Day, formatDuration(s.InstanceTime)),
				Run:   func(p *player.Player, _ *world.Tx) { pf.showSave(p, s.ID) },
			})
			continue
		}
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf("§l%d. 새 게임§r\n§8비어 있는 슬롯", slot),
			Run:   func(p *player.Player, _ *world.Tx) { pf.askNewSave(p, slot) },
		})
	}
	ui.Menu(p, "세이브 선택", pf.game.Name()+" 세이브를 선택하세요.", opts...)
}

func (pf *Platform) showSave(p *player.Player, saveID int64) {
	s, err := pf.db.Save(saveID)
	if err != nil {
		pf.fail(p, "세이브 불러오기", err)
		return
	}
	body := fmt.Sprintf("§f%s\n§7%d일차 · 플레이 %s\n만든 날: %s", s.Name, s.Day, formatDuration(s.InstanceTime), s.CreatedAt.Format("2006-01-02"))
	ui.Menu(p, s.Name, body,
		ui.Option{Label: "§a입장", Run: func(p *player.Player, _ *world.Tx) { pf.enter(p, saveID) }},
		ui.Option{Label: "초대 코드", Run: func(p *player.Player, _ *world.Tx) { pf.showInviteCode(p, saveID) }},
		ui.Option{Label: "이름 변경", Run: func(p *player.Player, _ *world.Tx) { pf.askRename(p, s) }},
		ui.Option{Label: "§c삭제", Run: func(p *player.Player, _ *world.Tx) { pf.askDelete(p, s, 0) }},
		ui.Option{Label: "뒤로", Run: func(p *player.Player, _ *world.Tx) { pf.showSaves(p) }},
	)
}

func (pf *Platform) enter(p *player.Player, saveID int64) {
	if pf.m.Location(instance.AccountID(p)) != nil {
		return
	}
	if err := pf.m.Enter(p, saveID); err != nil {
		switch err {
		case instance.ErrHostOffline:
			p.Message("§c호스트가 접속 중일 때만 입장할 수 있습니다.")
		case instance.ErrTooManyActive:
			p.Message("§c서버가 혼잡합니다. 잠시 후 다시 시도해 주세요.")
		default:
			pf.fail(p, "입장", err)
		}
	}
}

func (pf *Platform) askNewSave(p *player.Player, slot int) {
	ui.Input(p, "새 게임", "세이브 이름", fmt.Sprintf("세이브 %d", slot), func(p *player.Player, _ *world.Tx, name string) {
		name, ok := cleanName(name)
		if !ok {
			p.Message("§c이름은 1~20자로 입력해 주세요.")
			return
		}
		s, err := pf.db.CreateSave(instance.AccountID(p), pf.game.ID(), slot, name)
		if err != nil {
			pf.fail(p, "세이브 만들기", err)
			return
		}
		pf.enter(p, s.ID)
	})
}

func (pf *Platform) askRename(p *player.Player, s db.Save) {
	ui.Input(p, "이름 변경", "새 이름", s.Name, func(p *player.Player, _ *world.Tx, name string) {
		name, ok := cleanName(name)
		if !ok {
			p.Message("§c이름은 1~20자로 입력해 주세요.")
			return
		}
		if err := pf.db.RenameSave(s.ID, name); err != nil {
			pf.fail(p, "이름 변경", err)
			return
		}
		pf.showSave(p, s.ID)
	})
}

// askDelete confirms twice before deleting a save.
func (pf *Platform) askDelete(p *player.Player, s db.Save, step int) {
	if step == 0 {
		ui.Confirm(p, "세이브 삭제", fmt.Sprintf("'%s' 세이브를 삭제할까요?\n삭제한 세이브는 되돌릴 수 없습니다.", s.Name), "§c삭제", func(p *player.Player, _ *world.Tx) {
			pf.askDelete(p, s, 1)
		})
		return
	}
	ui.Confirm(p, "정말 삭제할까요?", fmt.Sprintf("'%s'의 섬과 모든 참여자의 진행 상황이 사라집니다.", s.Name), "§c영구 삭제", func(p *player.Player, _ *world.Tx) {
		if err := pf.m.DeleteSave(s.ID); err != nil {
			pf.fail(p, "세이브 삭제", err)
			return
		}
		p.Messagef("§e'%s' 세이브를 삭제했습니다.", s.Name)
		pf.showSaves(p)
	})
}

func (pf *Platform) fail(p *player.Player, what string, err error) {
	pf.log.Error(what, "player", p.Name(), "err", err)
	p.Messagef("§c%s에 실패했습니다.", what)
}

func cleanName(s string) (string, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "§", ""))
	n := utf8.RuneCountInString(s)
	return s, n >= 1 && n <= 20
}

func registerCommands(pf *Platform) {
	cmd.Register(cmd.New("lobby", "로비로 이동합니다", []string{"spawn"}, lobbyCmd{pf: pf}))
	cmd.Register(cmd.New("saves", "세이브 선택 화면을 엽니다", nil, savesCmd{pf: pf}))
	cmd.Register(cmd.New("guide", "서버 안내를 봅니다", nil, guideCmd{pf: pf}))
	cmd.Register(cmd.New("friends", "친구 목록을 엽니다", []string{"friend"}, friendsCmd{pf: pf}))
	cmd.Register(cmd.New("invite", "친구를 세이브에 초대합니다", nil, inviteCmd{pf: pf}))
}

type playerOnly struct{}

func (playerOnly) Allow(src cmd.Source) bool {
	_, ok := src.(*player.Player)
	return ok
}

type lobbyCmd struct {
	playerOnly
	pf *Platform
}

func (c lobbyCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	if c.pf.m.Location(instance.AccountID(p)) == nil {
		p.Teleport(LobbySpawn.Vec3Middle())
		return
	}
	p.Message("§e로비로 이동합니다.")
	c.pf.m.ToLobby(p)
}

type savesCmd struct {
	playerOnly
	pf *Platform
}

func (c savesCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	if c.pf.m.Location(instance.AccountID(p)) != nil {
		o.Error("로비에서만 사용할 수 있습니다. /lobby 로 이동하세요.")
		return
	}
	c.pf.showSaves(p)
}

type guideCmd struct {
	playerOnly
	pf *Platform
}

func (c guideCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	c.pf.showGuide(src.(*player.Player), false)
}

type friendsCmd struct {
	playerOnly
	pf *Platform
}

func (c friendsCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	c.pf.showFriends(src.(*player.Player))
}

// showLobbyMenu is the main menu of the lobby.
func (pf *Platform) showLobbyMenu(p *player.Player) {
	reqs, _ := pf.db.FriendRequests(instance.AccountID(p))
	ui.Menu(p, pf.game.Name(), "원하는 항목을 고르세요.",
		ui.Option{Label: "내 세이브", Run: func(p *player.Player, _ *world.Tx) { pf.showSaves(p) }},
		ui.Option{Label: "참여 중인 세이브", Run: func(p *player.Player, _ *world.Tx) { pf.showJoined(p) }},
		ui.Option{Label: fmt.Sprintf("받은 초대 §8(%d)", len(pf.invites.list(instance.AccountID(p)))), Run: func(p *player.Player, _ *world.Tx) { pf.showInvites(p) }},
		ui.Option{Label: "초대 코드 입력", Run: func(p *player.Player, _ *world.Tx) { pf.askInviteCode(p) }},
		ui.Option{Label: fmt.Sprintf("친구 §8(요청 %d)", len(reqs)), Run: func(p *player.Player, _ *world.Tx) { pf.showFriends(p) }},
		ui.Option{Label: "서버 안내", Run: func(p *player.Player, _ *world.Tx) { pf.showGuide(p, false) }},
	)
}
