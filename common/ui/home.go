package ui

import (
	"fmt"
	"strconv"

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
	// Hint is the small line at the bottom middle.
	Hint string
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

// HomeScreen shows p the lobby's main screen, like a game's lobby: the
// profile at the top left, icon buttons at the top right and down the left,
// recent lines at the bottom left and the chosen save with the play button
// at the bottom right, leaving the middle to the lobby island. It has the
// size of the windows, so the screens its buttons show take its place in the
// open screen and come back to it the same way, without the screen closing
// in between.
func HomeScreen(p *player.Player, h Home) {
	l := newOreLayout()
	l.size = fmt.Sprintf("%dx%d", windowW, windowH)
	l.close = h.Close
	children := []any{
		homeProfile(l, h.Profile),
		box("mc-home-corner", "children", homeButtons(l, "mc-home-sq", h.Corner, false)),
		box("mc-home-side", "children", homeButtons(l, "mc-home-tile", h.Side, true)),
	}
	if len(h.Feed) > 0 {
		var feed []any
		for _, line := range h.Feed {
			feed = append(feed, txt("mc-home-feed-line", line))
		}
		children = append(children, box("mc-home-feed", "children", feed))
	}
	play := []any{homeSave(l, h.Save)}
	label := []any{txt("mc-home-play-label", h.Play.Label)}
	if h.PlaySub != "" {
		label = append(label, txt("mc-home-play-sub", h.PlaySub))
	}
	play = append(play, l.press(box("mc-home-play", "children", label), h.Play.Run))
	if h.Manage.Run != nil {
		play = append(play, l.press(box("mc-btn mc-home-manage", "children", []any{txt("mc-btn-text", h.Manage.Label)}), h.Manage.Run))
	}
	children = append(children, box("mc-home-start", "children", play))
	if h.Hint != "" {
		children = append(children, txt("mc-home-hint", h.Hint))
	}
	if o := h.Overlay; o != nil {
		panel := []any{txt("mc-home-ov-title", o.Title)}
		if o.Body != "" {
			panel = append(panel, txt("mc-home-ov-body", o.Body))
		}
		for _, b := range o.Buttons {
			panel = append(panel, l.press(box("mc-btn mc-home-ov-btn", "children", []any{txt("mc-btn-text", b.Label)}), b.Run))
		}
		children = append(children, box("mc-home-ov", "children", []any{box("mc-home-ov-panel", "children", panel)}))
	}
	showOre(p, l, box("mc-screen mc-home", "children", children))
}

// homeProfile is the player's card: the head, then the name, the title, the
// bar and its line.
func homeProfile(l *oreLayout, pr HomeProfile) map[string]any {
	fill := min(max(pr.Fill, 0), 1)
	info := []any{txt("mc-home-name", pr.Name)}
	if pr.Title != "" {
		info = append(info, txt("mc-home-title", pr.Title))
	}
	info = append(info,
		box("mc-home-bar", "children", []any{box("mc-home-bar-fill", "style", map[string]any{"width": strconv.Itoa(int(fill*100+0.5)) + "%"})}),
		txt("mc-home-bar-line", pr.Line),
	)
	head := img("mc-home-head", pr.Icon)
	if len(pr.Face) == 64 {
		head = node("pixels", "class", "mc-home-head", "w", 8, "h", 8, "px", pr.Face)
	}
	n := box("mc-btn mc-home-profile", "children", []any{head, box("mc-home-info", "children", info)})
	return l.press(n, pr.Run)
}

// homeButtons are icon buttons of class: the label under the icon (labelled)
// or as the tooltip, and the badge.
func homeButtons(l *oreLayout, class string, bs []HomeButton, labelled bool) []any {
	var out []any
	for _, b := range bs {
		ch := []any{img("mc-home-ic", b.Icon)}
		if labelled {
			ch = append(ch, txt("mc-home-tile-label", b.Label))
		}
		if b.Badge > 0 {
			ch = append(ch, txt("mc-home-badge", strconv.Itoa(b.Badge)))
		}
		n := box("mc-btn "+class, "children", ch)
		if !labelled {
			n["tip"] = b.Label
			quick(n)
		}
		out = append(out, l.press(n, b.Run))
	}
	return out
}

// homeSave is the save card: the picture, the lines and the arrows.
func homeSave(l *oreLayout, s HomeSave) map[string]any {
	text := []any{txt("mc-home-save-head", s.Head), txt("mc-home-save-name", s.Name)}
	for _, line := range s.Lines {
		text = append(text, txt("mc-home-save-line", line))
	}
	ch := []any{img("mc-home-save-pic", s.Icon), box("mc-home-save-text", "children", text)}
	if s.Prev != nil || s.Next != nil {
		ch = append(ch, box("mc-home-arrows", "children", []any{
			homeArrow(l, "textures/ui/sg/chevron_left", s.Prev),
			homeArrow(l, "textures/ui/sg/chevron_right", s.Next),
		}))
	}
	return box("mc-panel mc-home-save", "children", ch)
}

func homeArrow(l *oreLayout, icon string, run func(p *player.Player, tx *world.Tx)) map[string]any {
	class := "mc-btn mc-home-arrow"
	if run == nil {
		class += " off"
	}
	return quick(l.press(box(class, "children", []any{img("", icon)}), run))
}
