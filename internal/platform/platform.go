// Package platform implements the server around Sky Gear: accounts, the
// server guide, the lobby and save slot selection.
package platform

import (
	"fmt"
	"github.com/presentkim/skygear/common/i18n"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/internal/account"
	"github.com/presentkim/skygear/internal/db"
	"github.com/presentkim/skygear/internal/instance"
)

// Slots is the number of save slots per account.
const Slots = 3

// LobbySpawn is the block players stand on top of in the lobby.
var LobbySpawn = cube.Pos{0, 64, 0}

// Platform ties accounts, the lobby and instances together.
type Platform struct {
	log     *slog.Logger
	db      *db.DB
	m       *instance.Manager
	online  Online
	invites invites
	// isOperator reports whether a player is a server operator (platform.toml
	// [Admin]); operators manage the members of any save they are in.
	isOperator func(p *player.Player) bool
}

// New creates the platform. The lobby world must already be built.
func New(log *slog.Logger, d *db.DB, m *instance.Manager, online Online, isOperator func(*player.Player) bool) *Platform {
	pf := &Platform{log: log, db: d, m: m, online: online, isOperator: isOperator}
	// The DDUI screens shown in the lobby take its look (ui.lobbySkin).
	ui.InLobby = func(p *player.Player) bool { return m.Location(instance.AccountID(p)) == nil }
	ui.Idle = pf.lobbyIdle
	// The game's name and logo at the top of every lobby screen.
	ui.LobbyHeader = func(p *player.Player) (string, string) {
		return i18n.T(i18n.Of(p), "name.Game"), "textures/blocks/grass_side_carried"
	}
	account.Init(d, log)
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
		Handler:   func(*player.Player) player.Handler { return lobbyHandler{} },
		ChatName:  account.DisplayName,
		UsePlayer: func(p, target *player.Player) bool { return (*pf).usePlayer(p, target) },
		Quit: func(p *player.Player) {
			account.Forget(p)
			waila.Forget(p)
			forgetLobbyHUD(p)
			stopLobbyMotion(p)
			forgetHome(p)
		},
		Entered:     entered,
		EnterFailed: func(p *player.Player, err error) { (*pf).enterFailed(p, err) },
	}
}

// BuildLobby builds the lobby island (buildLobbyIsland). It runs in the
// lobby world.
func BuildLobby(tx *world.Tx) { buildLobbyIsland(tx) }

// Join handles a player connecting. It runs in the lobby world.
func (pf *Platform) Join(p *player.Player) {
	acc, err := pf.db.TouchAccount(instance.AccountID(p), p.XUID(), p.Name())
	if err != nil {
		pf.log.Error("touch account", "player", p.Name(), "err", err)
		p.Disconnect(i18n.T(i18n.Of(p), "platform.accountLoadFailed"))
		return
	}
	pf.keepClientSettings(p)
	pf.m.Join(p)
	account.Count(p, "login", 1)
	account.Refresh(p)
	if acc.GuideVersion < GuideVersion {
		sghook.AfterClientReady(p, func(p *player.Player, _ *world.Tx) { pf.showGuide(p, true) })
	}
	pf.notifyPending(p)
}

// arriveLobby shows the lobby scene and, once the client can show it, the
// main screen (the guide comes first the first time: Join shows it).
func (pf *Platform) arriveLobby(p *player.Player) {
	p.SetGameMode(world.GameModeAdventure)
	p.Inventory().Clear()
	showLobbyScene(p, p.Tx())
	forgetLobbyHUD(p)
	startLobbyMotion(p)
	sendLobbyHUD(p, "")
	if acc, err := pf.db.Account(instance.AccountID(p)); err == nil && acc.GuideVersion >= GuideVersion {
		sghook.AfterClientReady(p, func(p *player.Player, _ *world.Tx) { pf.showLobbyMenu(p) })
	}
}

// showSaves shows the save slots of p and the actions of the chosen one
// (ui.ListSplitMenu). Pressing a save chooses it; pressing an empty slot asks
// for the new save's name at once (no button on the right does the same).
// With 0 the first save is chosen.
func (pf *Platform) showSaves(p *player.Player, sel int64) {
	saves, err := pf.db.SavesOf(instance.AccountID(p))
	if err != nil {
		pf.fail(p, i18n.T(i18n.Of(p), "platform.saves.loading"), err)
		return
	}
	lang := i18n.Of(p)
	var chosen db.Save
	bySlot := map[int]db.Save{}
	for _, s := range saves {
		bySlot[s.Slot] = s
		if s.ID == sel {
			chosen = s
		}
	}
	if chosen.ID == 0 && len(saves) > 0 {
		chosen = saves[0]
	}
	var rows []ui.ListRow
	for n := 1; n <= Slots; n++ {
		s, ok := bySlot[n]
		if !ok {
			rows = append(rows, ui.ListRow{
				Label: ui.ListRowText(i18n.T(lang, "platform.saves.new"), ""),
				Icon:  iconNewSave,
				Run:   func(p *player.Player, _ *world.Tx) { pf.askNewSave(p, n) },
			})
			continue
		}
		info := fmt.Sprintf(i18n.T(lang, "platform.saves.info"), s.Day, formatDuration(lang, s.InstanceTime), i18n.ClientDate(s.CreatedAt, s.CreatedAt.Format("2006-01-02")))
		rows = append(rows, ui.ListRow{
			Label:    ui.ListRowText(s.Name, info),
			Icon:     iconSave,
			Selected: s.ID == chosen.ID,
			Run:      func(p *player.Player, _ *world.Tx) { pf.showSaves(p, s.ID) },
		})
	}
	sp := ui.ListSplit{Rows: rows}
	switch {
	case chosen.ID != 0:
		code := chosen.InviteCode
		if code == "" {
			if code, err = pf.inviteCode(chosen, false); err != nil {
				pf.fail(p, i18n.T(lang, "platform.invite.renew"), err)
				return
			}
		}
		n, _ := pf.db.ActiveMembers(chosen.ID)
		// the invite code (a friend enters it on their multiplayer screen;
		// pressing makes a new one) and the member count sit on their
		// buttons, so the actions fill the column without a body above them
		sp.Actions = []ui.Option{
			{Label: i18n.T(lang, "platform.save.enter"), Icon: iconEnter, Run: func(p *player.Player, _ *world.Tx) { pf.enter(p, chosen.ID) }},
			{Label: fmt.Sprintf(i18n.T(lang, "platform.save.code"), code), Icon: iconCode, Run: func(p *player.Player, _ *world.Tx) {
				pf.confirm(p, i18n.T(lang, "platform.invite.renew"), i18n.T(lang, "platform.invite.renew.ask"), i18n.T(lang, "platform.invite.renew"), func(p *player.Player, _ *world.Tx) {
					s, err := pf.db.Save(chosen.ID)
					if err == nil {
						_, err = pf.inviteCode(s, true)
					}
					if err != nil {
						pf.fail(p, i18n.T(lang, "platform.invite.renew"), err)
						return
					}
					pf.showSaves(p, chosen.ID)
				}, func(p *player.Player, _ *world.Tx) { pf.showSaves(p, chosen.ID) })
			}},
			{Label: fmt.Sprintf(i18n.T(lang, "platform.save.members"), n, MaxMembers), Icon: iconMembers, Run: func(p *player.Player, _ *world.Tx) { pf.showMembers(p, chosen.ID) }},
			{Label: i18n.T(lang, "platform.save.rename"), Icon: iconRename, Run: func(p *player.Player, _ *world.Tx) { pf.askRename(p, chosen) }},
			{Label: i18n.T(lang, "platform.save.delete"), Icon: iconDelete, Run: func(p *player.Player, _ *world.Tx) { pf.askDelete(p, chosen, 0) }},
		}
	default:
		sp.Body = i18n.T(lang, "platform.saves.none")
	}
	// The back button sits in the control box left of the close button, like
	// every other screen's.
	sp.Actions = append(sp.Actions, ui.Back(i18n.T(lang, "platform.saves.lobby"), func(p *player.Player, _ *world.Tx) { pf.showLobbyMenu(p) }))
	ui.ListSplitMenu(p, i18n.T(lang, "platform.saves.title"), sp)
}

func (pf *Platform) enter(p *player.Player, saveID int64) {
	if pf.m.Location(instance.AccountID(p)) != nil {
		return
	}
	if err := pf.m.Enter(p, saveID); err != nil {
		pf.enterFailed(p, err)
	}
}

// entered runs when p left the lobby for a save: the save has its own view,
// so the lobby's camera, HUD, locks and frame go.
func entered(p *player.Player) {
	leaveLobbyScene(p)
	forgetLobbyHUD(p)
}

// enterFailed tells p in the lobby why it could not enter a save.
func (pf *Platform) enterFailed(p *player.Player, err error) {
	switch err {
	case instance.ErrHostOffline:
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.enter.hostOffline"))
	case instance.ErrStateLocked:
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.enter.hostBusy"))
	case instance.ErrTooManyActive:
		ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.enter.serverBusy"))
	default:
		pf.fail(p, i18n.T(i18n.Of(p), "platform.enter.entering"), err)
	}
}

func (pf *Platform) askNewSave(p *player.Player, slot int) {
	ui.Input(p, ui.CreateToken+i18n.T(i18n.Of(p), "platform.newSave.title"), i18n.T(i18n.Of(p), "platform.newSave.label"), i18n.T(i18n.Of(p), "platform.newSave.placeholder"), func(p *player.Player, _ *world.Tx, name string) {
		name, ok := cleanName(name)
		if !ok {
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.save.nameLength"))
			return
		}
		s, err := pf.db.CreateSave(instance.AccountID(p), slot, name)
		if err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.newSave.creating"), err)
			return
		}
		pf.showSaves(p, s.ID)
	}, func(p *player.Player, _ *world.Tx) {
		pf.showSaves(p, 0)
	})
}

func (pf *Platform) askRename(p *player.Player, s db.Save) {
	ui.Input(p, i18n.T(i18n.Of(p), "platform.rename.title"), i18n.T(i18n.Of(p), "platform.rename.label"), s.Name, func(p *player.Player, _ *world.Tx, name string) {
		name, ok := cleanName(name)
		if !ok {
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "platform.save.nameLength"))
			return
		}
		if err := pf.db.RenameSave(s.ID, name); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.rename.renaming"), err)
			return
		}
		pf.showSaves(p, s.ID)
	}, func(p *player.Player, _ *world.Tx) {
		pf.showSaves(p, s.ID)
	})
}

// askDelete confirms twice before deleting a save; cancelling shows the save
// again.
func (pf *Platform) askDelete(p *player.Player, s db.Save, step int) {
	back := func(p *player.Player, _ *world.Tx) { pf.showSaves(p, s.ID) }
	if step == 0 {
		pf.confirm(p, i18n.T(i18n.Of(p), "platform.delete.title"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.delete.ask"), s.Name), i18n.T(i18n.Of(p), "platform.save.delete"), func(p *player.Player, _ *world.Tx) {
			pf.askDelete(p, s, 1)
		}, back)
		return
	}
	pf.confirm(p, i18n.T(i18n.Of(p), "platform.delete.confirmTitle"), fmt.Sprintf(i18n.T(i18n.Of(p), "platform.delete.confirmAsk"), s.Name), i18n.T(i18n.Of(p), "platform.delete.forever"), func(p *player.Player, _ *world.Tx) {
		if err := pf.m.DeleteSave(s.ID); err != nil {
			pf.fail(p, i18n.T(i18n.Of(p), "platform.delete.deleting"), err)
			return
		}
		ui.Flash(p, i18n.T(i18n.Of(p), "platform.delete.done"), s.Name)
		pf.showSaves(p, 0)
	}, back)
}

func (pf *Platform) fail(p *player.Player, what string, err error) {
	pf.log.Error(what, "player", p.Name(), "err", err)
	ui.Notify(p, i18n.T(i18n.Of(p), "platform.failed"), what)
}

func cleanName(s string) (string, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "§", ""))
	n := utf8.RuneCountInString(s)
	return s, n >= 1 && n <= 20
}

func registerCommands(pf *Platform) {
	cmd.Register(cmd.New("lobby", i18n.T(i18n.Default, "platform.cmd.lobby"), []string{"spawn"}, lobbyCmd{pf: pf}))
	cmd.Register(cmd.New("saves", i18n.T(i18n.Default, "platform.cmd.saves"), nil, savesCmd{pf: pf}))
	cmd.Register(cmd.New("guide", i18n.T(i18n.Default, "platform.cmd.guide"), nil, guideCmd{pf: pf}))
	cmd.Register(cmd.New("notify", i18n.T(i18n.Default, "platform.cmd.notify"), []string{"notices", "inbox"}, notifyCmd{pf: pf}))
	cmd.Register(cmd.New("friends", i18n.T(i18n.Default, "platform.cmd.friends"), []string{"friend"}, friendsCmd{pf: pf}))
	cmd.Register(cmd.New("invite", i18n.T(i18n.Default, "platform.cmd.invite"), nil, inviteCmd{pf: pf}))
	cmd.Register(cmd.New("members", i18n.T(i18n.Default, "platform.cmd.members"), nil, membersCmd{pf: pf}))
	cmd.Register(cmd.New("join", i18n.T(i18n.Default, "platform.cmd.join"), nil, joinCmd{pf: pf}))
	cmd.Register(cmd.New("title", i18n.T(i18n.Default, "platform.cmd.title"), []string{"titles"}, titleCmd{pf: pf}))
	cmd.Register(cmd.New("uidebug", i18n.T(i18n.Default, "platform.uidebug.command"), nil, uiDebugCmd{}))
	cmd.Register(cmd.New("settings", i18n.T(i18n.Default, "setting.command"), []string{"setting", "options"}, settingsCmd{pf: pf}))
	cmd.Register(cmd.New("copysave", i18n.T(i18n.Default, "platform.cmd.copysave"), nil, copySaveCmd{pf: pf}))
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
	p.Message(i18n.T(i18n.Of(p), "platform.toLobby"))
	c.pf.m.ToLobby(p)
}

type savesCmd struct {
	playerOnly
	pf *Platform
}

func (c savesCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	if c.pf.m.Location(instance.AccountID(p)) != nil {
		o.Error(i18n.T(i18n.Of(p), "platform.lobbyOnly"))
		return
	}
	c.pf.showSaves(p, 0)
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

// lobbyBack is the back button of a lobby screen: it shows back. The screens
// that also open in a save go back with menuBack.
func (pf *Platform) lobbyBack(p *player.Player, back func(p *player.Player)) ui.Option {
	return ui.Back(i18n.T(i18n.Of(p), "platform.saves.lobby"), func(p *player.Player, _ *world.Tx) { back(p) })
}
