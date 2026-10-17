// Package platform implements the server around the games: accounts, the
// server guide, the lobby and save slot selection (docs/design/01-platform.md).
package platform

import (
	"fmt"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/account"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
	"github.com/presentkim/dragonfactory/internal/waila"
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
	account.Init(d, log)
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
		Handler:  func(*player.Player) player.Handler { return lobbyHandler{pf: *pf} },
		ChatName: account.DisplayName,
		Quit: func(p *player.Player) {
			account.Forget(p)
			waila.Forget(p)
		},
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
		p.Disconnect(i18n.T(i18n.Of(p), "platform.platform.Join.1"))
		return
	}
	pf.m.Join(p)
	account.Count(p, "login", 1)
	account.Refresh(p)
	if acc.GuideVersion < GuideVersion {
		pf.showGuide(p, true)
	} else {
		p.Messagef(i18n.T(i18n.Of(p), "platform.platform.Join.2"), pf.game.Name())
	}
	pf.notifyRequests(p)
}

const lobbyItemKey = "platform_lobby_item"

// arriveLobby gives the lobby menu items.
func (pf *Platform) arriveLobby(p *player.Player) {
	p.SetGameMode(world.GameModeAdventure)
	_ = p.Inventory().SetItem(0, item.NewStack(item.Compass{}, 1).WithCustomName(i18n.T(i18n.Of(p), "platform.platform.arriveLobby.1")).WithValue(lobbyItemKey, "menu"))
	_ = p.Inventory().SetItem(8, item.NewStack(item.Book{}, 1).WithCustomName(i18n.T(i18n.Of(p), "platform.platform.arriveLobby.2")).WithValue(lobbyItemKey, "guide"))
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
		pf.fail(p, i18n.T(i18n.Of(p), "platform.platform.showSaves.1"), err)
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
				Label: fmt.Sprintf(i18n.T(i18n.Of(p), "platform.platform.showSaves.2"), slot, s.Name, s.Day, formatDuration(s.InstanceTime)),
				Icon:  iconSave,
				Run:   func(p *player.Player, _ *world.Tx) { pf.showSave(p, s.ID) },
			})
			continue
		}
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf(i18n.T(i18n.Of(p), "platform.platform.showSaves.3"), slot),
			Icon:  iconNewSave,
			Run:   func(p *player.Player, _ *world.Tx) { pf.askNewSave(p, slot) },
		})
	}
	ui.Menu(p, i18n.T(i18n.Of(p), "platform.platform.showSaves.4"), pf.game.Name()+i18n.T(i18n.Of(p), "platform.platform.showSaves.5"), opts...)
}

func (pf *Platform) showSave(p *player.Player, saveID int64) {
	s, err := pf.db.Save(saveID)
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.platform.showSave.1"), err)
		return
	}
	body := fmt.Sprintf(i18n.T(i18n.Of(p), "platform.platform.showSave.2"), s.Name, s.Day, formatDuration(s.InstanceTime), s.CreatedAt.Format("2006-01-02"))
	ui.Menu(p, s.Name, body,
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.platform.showSave.3"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) { pf.enter(p, saveID) }},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.platform.showSave.4"), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.showInviteCode(p, saveID) }},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.platform.showSave.5"), Icon: iconMembers, Run: func(p *player.Player, _ *world.Tx) { pf.showMembers(p, saveID) }},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.platform.showSave.6"), Icon: iconRename, Run: func(p *player.Player, _ *world.Tx) { pf.askRename(p, s) }},
		ui.Option{Label: i18n.T(i18n.Of(p), "platform.platform.showSave.7"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) { pf.askDelete(p, s, 0) }},
		ui.Back(i18n.T(i18n.Of(p), "platform.platform.showSave.8"), func(p *player.Player, _ *world.Tx) { pf.showSaves(p) }),
	)
}

func (pf *Platform) enter(p *player.Player, saveID int64) {
	if pf.m.Location(instance.AccountID(p)) != nil {
		return
	}
	if err := pf.m.Enter(p, saveID); err != nil {
		switch err {
		case instance.ErrHostOffline:
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.platform.enter.1"))
		case instance.ErrStateLocked:
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.platform.enter.2"))
		case instance.ErrTooManyActive:
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.platform.enter.3"))
		default:
			pf.fail(p, i18n.T(i18n.Of(p), "platform.platform.enter.4"), err)
		}
	}
}

func (pf *Platform) askNewSave(p *player.Player, slot int) {
	ui.Input(p, i18n.T(i18n.Of(p), "platform.platform.askNewSave.1"), i18n.T(i18n.Of(p), "platform.platform.askNewSave.2"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.platform.askNewSave.3"), slot), func(p *player.Player, _ *world.Tx, name string) {
		name, ok := cleanName(name)
		if !ok {
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.platform.askNewSave.4"))
			return
		}
		s, err := pf.db.CreateSave(instance.AccountID(p), pf.game.ID(), slot, name)
		if err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.platform.askNewSave.5"), err)
			return
		}
		pf.enter(p, s.ID)
	})
}

func (pf *Platform) askRename(p *player.Player, s db.Save) {
	ui.Input(p, i18n.T(i18n.Of(p), "platform.platform.askRename.1"), i18n.T(i18n.Of(p), "platform.platform.askRename.2"), s.Name, func(p *player.Player, _ *world.Tx, name string) {
		name, ok := cleanName(name)
		if !ok {
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.platform.askRename.3"))
			return
		}
		if err := pf.db.RenameSave(s.ID, name); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.platform.askRename.4"), err)
			return
		}
		pf.showSave(p, s.ID)
	})
}

// askDelete confirms twice before deleting a save.
func (pf *Platform) askDelete(p *player.Player, s db.Save, step int) {
	if step == 0 {
		ui.Confirm(p, i18n.T(i18n.Of(p), "platform.platform.askDelete.1"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.platform.askDelete.2"), s.Name), i18n.T(i18n.Of(p), "platform.platform.askDelete.3"), func(p *player.Player, _ *world.Tx) {
			pf.askDelete(p, s, 1)
		})
		return
	}
	ui.Confirm(p, i18n.T(i18n.Of(p), "platform.platform.askDelete.4"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.platform.askDelete.5"), s.Name), i18n.T(i18n.Of(p), "platform.platform.askDelete.6"), func(p *player.Player, _ *world.Tx) {
		if err := pf.m.DeleteSave(s.ID); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.platform.askDelete.7"), err)
			return
		}
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.platform.askDelete.8"), s.Name)
		pf.showSaves(p)
	})
}

func (pf *Platform) fail(p *player.Player, what string, err error) {
	pf.log.Error(what, "player", p.Name(), "err", err)
	ui.Notify(p, i18n.T(i18n.Of(p), "platform.platform.fail.1"), what)
}

func cleanName(s string) (string, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "§", ""))
	n := utf8.RuneCountInString(s)
	return s, n >= 1 && n <= 20
}

func registerCommands(pf *Platform) {
	cmd.Register(cmd.New("lobby", i18n.T(i18n.Default, "platform.platform.registerCommands.1"), []string{"spawn"}, lobbyCmd{pf: pf}))
	cmd.Register(cmd.New("saves", i18n.T(i18n.Default, "platform.platform.registerCommands.2"), nil, savesCmd{pf: pf}))
	cmd.Register(cmd.New("guide", i18n.T(i18n.Default, "platform.platform.registerCommands.3"), nil, guideCmd{pf: pf}))
	cmd.Register(cmd.New("friends", i18n.T(i18n.Default, "platform.platform.registerCommands.4"), []string{"friend"}, friendsCmd{pf: pf}))
	cmd.Register(cmd.New("invite", i18n.T(i18n.Default, "platform.platform.registerCommands.5"), nil, inviteCmd{pf: pf}))
	cmd.Register(cmd.New("members", i18n.T(i18n.Default, "platform.platform.registerCommands.6"), nil, membersCmd{pf: pf}))
	cmd.Register(cmd.New("join", i18n.T(i18n.Default, "platform.platform.registerCommands.7"), nil, joinCmd{pf: pf}))
	cmd.Register(cmd.New("title", i18n.T(i18n.Default, "platform.platform.registerCommands.8"), []string{"titles"}, titleCmd{pf: pf}))
}

type playerOnly struct{}

func (playerOnly) Allow(src cmd.Source) bool {
	_, ok := src.(*player.Player)
	return ok
}

type titleCmd struct {
	playerOnly
	pf *Platform
}

func (c titleCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	c.pf.showCosmetics(src.(*player.Player))
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
	p.Message(i18n.T(i18n.Of(p), "platform.platform.Run.1"))
	c.pf.m.ToLobby(p)
}

type savesCmd struct {
	playerOnly
	pf *Platform
}

func (c savesCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	if c.pf.m.Location(instance.AccountID(p)) != nil {
		o.Error(i18n.T(i18n.Of(p), "platform.platform.Run.2"))
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
	lang := i18n.Of(p)
	friends := i18n.T(lang, "platform.platform.showLobbyMenu.10")
	if len(reqs) > 0 {
		friends = fmt.Sprintf(i18n.T(lang, "platform.platform.showLobbyMenu.7"), len(reqs))
	}
	ui.CardMenu(p, pf.game.Name()+i18n.T(lang, "platform.platform.showLobbyMenu.1"), "",
		ui.Option{Label: i18n.T(lang, "platform.platform.showLobbyMenu.3"), Icon: iconSaves, Run: func(p *player.Player, _ *world.Tx) { pf.showSaves(p) }},
		ui.Option{Label: i18n.T(lang, "platform.platform.showLobbyMenu.4"), Icon: iconJoined, Run: func(p *player.Player, _ *world.Tx) { pf.showJoined(p) }},
		ui.Option{Label: i18n.T(lang, "platform.platform.showLobbyMenu.8"), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.showCosmetics(p) }},
		ui.Option{Label: friends, Icon: iconFriends, Run: func(p *player.Player, _ *world.Tx) { pf.showFriends(p) }},
		ui.Option{Label: i18n.T(lang, "platform.platform.showLobbyMenu.6"), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) { pf.askInviteCode(p) }},
		ui.Option{Label: fmt.Sprintf(i18n.T(lang, "platform.platform.showLobbyMenu.5"), len(pf.invites.list(instance.AccountID(p)))), Icon: iconInvite, Run: func(p *player.Player, _ *world.Tx) { pf.showInvites(p) }},
	)
}
