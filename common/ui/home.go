package ui

import (
	"slices"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Home is the content of HomeScreen: the lobby's main screen, laid out like
// a game lobby around the player's character on the lobby island.
type Home struct {
	// Profile is the card at the top left.
	Profile HomeProfile
	// Corner are the icon buttons at the top right (the label is the
	// tooltip), Side the column of icon buttons on the left.
	Corner, Side []HomeButton
	// Feed are the recent lines at the bottom left, oldest first.
	Feed []string
	// Save is the card of the chosen save at the bottom right, over Play (the
	// large green button, Sub the small line under its label) and Manage.
	Save         HomeSave
	Play, Manage HomeButton
	PlaySub      string
	// Overlay, if set, is a panel over the dimmed screen: the system menu, a
	// question.
	Overlay *HomeOverlay
	// Close runs on Esc (the screen does not close by itself).
	Close func(p *player.Player, tx *world.Tx)
}

// HomeButton is a button of the main screen. Badge, if above 0, is the red
// count on its corner.
type HomeButton struct {
	Icon, Label string
	Badge       int
	Run         func(p *player.Player, tx *world.Tx)
}

// HomeProfile is the player's card: its face (Face, 8x8 colours "rrggbb"
// row by row, "" for none; when not 64 colours, the picture Icon), name,
// title, a bar filled to Fill (0..1) and the line under it.
type HomeProfile struct {
	Face              []string
	Icon, Name, Title string
	Fill              float64
	Line              string
	Run               func(p *player.Player, tx *world.Tx)
}

// HomeSave is the card of a save: a small line over its name (Head), the
// lines under the name, its picture, and the buttons to the save before and
// after it (nil: not drawn).
type HomeSave struct {
	Head, Name, Icon string
	Lines            []string
	Prev, Next       func(p *player.Player, tx *world.Tx)
}

// HomeOverlay is a panel over the main screen: a title, a body and a column
// of buttons.
type HomeOverlay struct {
	Title, Body string
	Buttons     []HomeButton
}

// homeData is the data of the home screen.
type homeData struct {
	Profile homeProfileD  `json:"profile"`
	Corner  []homeButtonD `json:"corner"`
	Side    []homeButtonD `json:"side"`
	Feed    []string      `json:"feed"`
	Save    homeSaveD     `json:"save"`
	Play    homePlayD     `json:"play"`
	Manage  *homeManageD  `json:"manage"`
	Overlay *homeOverlayD `json:"overlay"`
}

// homeProfileD is the player's card.
type homeProfileD struct {
	ID    string   `json:"id"`
	Face  []string `json:"face"`
	Icon  string   `json:"icon"`
	Name  string   `json:"name"`
	Title string   `json:"title"`
	Fill  float64  `json:"fill"`
	Line  string   `json:"line"`
}

// homeButtonD is an icon button of the home screen.
type homeButtonD struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"`
	Label string `json:"label"`
	Badge int    `json:"badge"`
}

// homeSaveD is the card of the chosen save.
type homeSaveD struct {
	Head   string   `json:"head"`
	Name   string   `json:"name"`
	Icon   string   `json:"icon"`
	Lines  []string `json:"lines"`
	Arrows bool     `json:"arrows"`
	Prev   string   `json:"prev"`
	Next   string   `json:"next"`
}

// homePlayD is the play button.
type homePlayD struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Sub   string `json:"sub"`
}

// homeManageD is the manage button.
type homeManageD struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// homeOverlayD is the panel over the dimmed screen.
type homeOverlayD struct {
	Title   string        `json:"title"`
	Body    string        `json:"body"`
	Buttons []homeManageD `json:"buttons"`
}

// homeButtonsOf is the data of icon buttons, [] when there are none.
func homeButtonsOf(l *oreLayout, bs []HomeButton) []homeButtonD {
	out := make([]homeButtonD, 0, len(bs))
	for _, b := range bs {
		out = append(out, homeButtonD{ID: l.id(b.Run), Icon: oreIcon(b.Icon), Label: b.Label, Badge: b.Badge})
	}
	return out
}

// HomeScreen shows p the lobby's main screen, like a game's lobby: the
// profile at the top left, icon buttons at the top right and down the left,
// recent lines at the bottom left and the chosen save with the play button
// at the bottom right, leaving the middle to the lobby island. It has the
// size of the windows, so the screens its buttons show take its place in the
// open screen and come back to it the same way, without the screen closing
// in between.
func HomeScreen(p *player.Player, h Home) {
	l := newOreLayout()
	l.close = h.Close
	pr := h.Profile
	d := homeData{
		Profile: homeProfileD{ID: l.id(pr.Run), Face: []string{}, Icon: oreIcon(pr.Icon), Name: pr.Name, Title: pr.Title,
			Fill: min(max(pr.Fill, 0), 1), Line: pr.Line},
		Corner: homeButtonsOf(l, h.Corner),
		Side:   homeButtonsOf(l, h.Side),
		Feed:   append([]string{}, h.Feed...),
	}
	if len(pr.Face) == 64 {
		d.Profile.Face, d.Profile.Icon = slices.Clone(pr.Face), ""
	}
	s := h.Save
	d.Save = homeSaveD{Head: s.Head, Name: s.Name, Icon: oreIcon(s.Icon), Lines: append([]string{}, s.Lines...),
		Arrows: s.Prev != nil || s.Next != nil, Prev: l.id(s.Prev), Next: l.id(s.Next)}
	d.Play = homePlayD{ID: l.id(h.Play.Run), Label: h.Play.Label, Sub: h.PlaySub}
	if h.Manage.Run != nil {
		d.Manage = &homeManageD{ID: l.id(h.Manage.Run), Label: h.Manage.Label}
	}
	if o := h.Overlay; o != nil {
		ov := &homeOverlayD{Title: o.Title, Body: o.Body, Buttons: []homeManageD{}}
		for _, b := range o.Buttons {
			ov.Buttons = append(ov.Buttons, homeManageD{ID: l.id(b.Run), Label: b.Label})
		}
		d.Overlay = ov
	}
	showScreen(p, l, "home", d)
}
