package platform

import (
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/internal/account"
	"github.com/presentkim/skygear/internal/db"
	"github.com/presentkim/skygear/internal/instance"
)

// The lobby's main screen (ui.HomeScreen) is open for as long as the player
// is in the lobby: its buttons show their screens in its place, and those
// screens come back to it (lobbyIdle). Esc opens the system menu over it
// instead of closing it, and leaving the server is a button of that menu.

// Overlays of the main screen.
const (
	homeNoOverlay = iota
	homeSystem
	homeExit
)

// homeState is what one player's main screen shows: the save chosen on its
// card (0: the last one played) and the overlay, and whether it is the
// screen shown (not one of its buttons' screens).
type homeState struct {
	save    int64
	overlay int
	shown   bool
}

var homes = struct {
	sync.Mutex
	m map[uuid.UUID]homeState
}{m: map[uuid.UUID]homeState{}}

func homeOf(p *player.Player) homeState {
	homes.Lock()
	defer homes.Unlock()
	return homes.m[p.UUID()]
}

func setHome(p *player.Player, f func(s *homeState)) {
	homes.Lock()
	s := homes.m[p.UUID()]
	f(&s)
	homes.m[p.UUID()] = s
	homes.Unlock()
}

// forgetHome drops what p's main screen showed (p left the server).
func forgetHome(p *player.Player) {
	homes.Lock()
	delete(homes.m, p.UUID())
	homes.Unlock()
	forgetFace(p)
	forgetConsole(p)
}

// lobbyIdle is ui.Idle: p answered a screen and has none open, or closed one.
// In the lobby the main screen comes back at once.
func (pf *Platform) lobbyIdle(p *player.Player, _ *world.Tx, _ string, _ bool) {
	if pf.m.InLobby(instance.AccountID(p)) {
		pf.showLobbyMenu(p)
	}
}

// showLobbyMenu shows p the lobby's main screen.
func (pf *Platform) showLobbyMenu(p *player.Player) {
	if !pf.m.InLobby(instance.AccountID(p)) {
		return
	}
	closeConsole(p)
	setHome(p, func(s *homeState) { s.shown = true })
	ui.HomeScreen(p, pf.home(p))
}

// refreshHome draws p's main screen again (a notice came) when p is looking
// at it.
func (pf *Platform) refreshHome(p *player.Player) {
	if homeOf(p).shown {
		pf.showLobbyMenu(p)
	}
}

// homeRun is the run of a button that leaves the main screen: the overlay is
// gone when the screen comes back.
func homeRun(f func(p *player.Player)) func(p *player.Player, tx *world.Tx) {
	return func(p *player.Player, _ *world.Tx) {
		setHome(p, func(s *homeState) { s.overlay, s.shown = homeNoOverlay, false })
		f(p)
	}
}

// overlay shows the main screen again with overlay o.
func (pf *Platform) overlay(o int) func(p *player.Player, tx *world.Tx) {
	return func(p *player.Player, _ *world.Tx) {
		setHome(p, func(s *homeState) { s.overlay = o })
		pf.showLobbyMenu(p)
	}
}

// home is the content of p's main screen.
func (pf *Platform) home(p *player.Player) ui.Home {
	lang := i18n.Of(p)
	id := instance.AccountID(p)
	st := homeOf(p)
	h := ui.Home{
		Profile: pf.homeProfile(p),
		Close: func(p *player.Player, tx *world.Tx) {
			o := homeSystem
			if homeOf(p).overlay != homeNoOverlay {
				o = homeNoOverlay
			}
			pf.overlay(o)(p, tx)
		},
	}

	notices := pf.notifies(p)
	h.Corner = []ui.HomeButton{
		{Icon: iconRequests, Label: i18n.T(lang, "platform.notify.title"), Badge: len(notices), Run: homeRun(func(p *player.Player) { pf.showNotify(p, "") })},
		{Icon: iconSettings, Label: i18n.T(lang, "platform.home.settings"), Run: homeRun(pf.showSettings)},
		{Icon: iconExit, Label: i18n.T(lang, "platform.home.exit"), Run: pf.overlay(homeExit)},
	}
	operator := pf.isOperator != nil && pf.isOperator(p)
	if operator {
		console := ui.HomeButton{Icon: iconConsole, Label: i18n.T(lang, "platform.home.console"), Run: homeRun(pf.showConsole)}
		h.Corner = append([]ui.HomeButton{console}, h.Corner...)
	}
	h.Side = []ui.HomeButton{
		{Icon: "textures/ui/sg/icon_people", Label: i18n.T(lang, "platform.home.multi"), Run: homeRun(pf.showMultiplayer)},
		{Icon: "textures/ui/sg/icon_person", Label: i18n.T(lang, "platform.home.friends"), Run: homeRun(pf.showFriends)},
		{Icon: "textures/ui/sg/icon_crown", Label: i18n.T(lang, "platform.home.cosmetics"), Run: homeRun(pf.showCosmetics)},
		{Icon: "textures/ui/sg/icon_help", Label: i18n.T(lang, "platform.home.help"), Run: homeRun(func(p *player.Player) { pf.showGuide(p, false) })},
	}
	h.Feed = pf.homeFeed(lang, id, notices)

	saves, err := pf.db.SavesOf(id)
	if err != nil {
		pf.log.Error("load saves", "player", p.Name(), "err", err)
	}
	pf.homeSaves(&h, lang, saves, st.save)

	switch st.overlay {
	case homeSystem:
		h.Overlay = &ui.HomeOverlay{Title: i18n.T(lang, "platform.home.menu"), Buttons: []ui.HomeButton{
			{Label: i18n.T(lang, "platform.home.resume"), Run: pf.overlay(homeNoOverlay)},
			{Label: i18n.T(lang, "platform.home.settings"), Run: homeRun(pf.showSettings)},
			{Label: i18n.T(lang, "platform.home.guide"), Run: homeRun(func(p *player.Player) { pf.showGuide(p, false) })},
			{Label: i18n.T(lang, "platform.home.exit"), Run: pf.overlay(homeExit)},
		}}
		if operator {
			last := len(h.Overlay.Buttons) - 1
			h.Overlay.Buttons = append(h.Overlay.Buttons[:last:last],
				ui.HomeButton{Label: i18n.T(lang, "platform.home.console"), Run: homeRun(pf.showConsole)}, h.Overlay.Buttons[last])
		}
	case homeExit:
		h.Overlay = &ui.HomeOverlay{Title: i18n.T(lang, "platform.home.exit"), Body: i18n.T(lang, "platform.home.exit.ask"), Buttons: []ui.HomeButton{
			{Label: i18n.T(lang, "platform.home.exit.yes"), Run: func(p *player.Player, _ *world.Tx) {
				p.Disconnect(i18n.T(i18n.Of(p), "platform.home.exit.bye"))
			}},
			{Label: i18n.T(lang, "platform.home.exit.no"), Run: pf.overlay(homeNoOverlay)},
		}}
	}
	return h
}

// homeProfile is p's card: name, title, and the achievements owned of all.
func (pf *Platform) homeProfile(p *player.Player) ui.HomeProfile {
	lang := i18n.Of(p)
	id := instance.AccountID(p)
	pr := ui.HomeProfile{Face: faceOf(p), Icon: iconHead, Name: p.Name(), Title: i18n.T(lang, "platform.home.noTitle"), Run: homeRun(pf.showCosmetics)}
	if acc, err := pf.db.Account(id); err == nil {
		pr.Name = account.ColorOf(acc.NameColor).Code + p.Name()
		if a, ok := account.Get(acc.Title); ok && a.Title(lang) != "" {
			pr.Title = "§6" + a.Title(lang)
		}
	}
	owned, total := len(account.Owned(id)), len(account.All())
	if total > 0 {
		pr.Fill = float64(owned) / float64(total)
	}
	pr.Line = fmt.Sprintf(i18n.T(lang, "platform.home.achievements"), owned, total)
	return pr
}

// homeFeed are the recent lines: p's newest notices, then the friends
// online, at most homeFeedLines, oldest first.
func (pf *Platform) homeFeed(lang, id string, notices []Notify) []string {
	var lines []string
	for _, n := range notices {
		if n.Feed != "" {
			lines = append(lines, n.Feed)
		}
	}
	friends, _ := pf.db.Friends(id)
	for _, f := range friends {
		if pf.isOnline(f.ID) {
			lines = append(lines, "§f"+f.Name+"§7\u00a0·\u00a0"+pf.statusOf(lang, f.ID, true))
		}
	}
	if len(lines) > homeFeedLines {
		lines = lines[:homeFeedLines]
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines
}

// homeFeedLines is how many lines the feed has at most.
const homeFeedLines = 3

// homeSaves fills the save card and the play buttons from p's saves: the one
// chosen (sel, else the last played), the arrows to the others, and play to
// enter it or, with no save, to make the first.
func (pf *Platform) homeSaves(h *ui.Home, lang string, saves []db.Save, sel int64) {
	h.Manage = ui.HomeButton{Label: i18n.T(lang, "platform.home.manage"), Run: homeRun(func(p *player.Player) { pf.showSaves(p, sel) })}
	if len(saves) == 0 {
		h.Save = ui.HomeSave{Head: fmt.Sprintf(i18n.T(lang, "platform.home.saveHead"), 0, Slots), Name: i18n.T(lang, "platform.home.noSave"), Icon: iconNewSave,
			Lines: []string{i18n.T(lang, "platform.home.noSave.line")}}
		h.Play = ui.HomeButton{Label: i18n.T(lang, "platform.home.newGame"), Run: homeRun(func(p *player.Player) { pf.askNewSave(p, 1) })}
		return
	}
	at := 0
	for i, s := range saves {
		if s.LastPlayed.After(saves[at].LastPlayed) {
			at = i
		}
	}
	for i, s := range saves {
		if s.ID == sel {
			at = i
		}
	}
	s := saves[at]
	members, _ := pf.db.ActiveMembers(s.ID)
	h.Save = ui.HomeSave{
		Head: fmt.Sprintf(i18n.T(lang, "platform.home.saveHead"), at+1, len(saves)),
		Name: s.Name,
		Icon: iconSave,
		Lines: []string{
			fmt.Sprintf(i18n.T(lang, "platform.home.saveInfo"), s.Day, formatDuration(lang, s.InstanceTime)),
			fmt.Sprintf(i18n.T(lang, "platform.home.saveLast"), members, i18n.ClientDate(s.LastPlayed, s.LastPlayed.Format("2006-01-02"))),
		},
	}
	if len(saves) > 1 {
		choose := func(i int) func(p *player.Player, tx *world.Tx) {
			id := saves[(i+len(saves))%len(saves)].ID
			return func(p *player.Player, _ *world.Tx) {
				setHome(p, func(s *homeState) { s.save = id })
				pf.showLobbyMenu(p)
			}
		}
		h.Save.Prev, h.Save.Next = choose(at-1), choose(at+1)
	}
	h.Play = ui.HomeButton{Label: i18n.T(lang, "platform.home.play"), Run: homeRun(func(p *player.Player) { pf.enter(p, s.ID) })}
	h.PlaySub = s.Name
	h.Manage.Run = homeRun(func(p *player.Player) { pf.showSaves(p, s.ID) })
}
