package ui

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
)

// minLoading is the least time a loading screen stays, so a save that is
// ready at once does not flash it.
const minLoading = 1200 * time.Millisecond

// Loading is the content of LoadingScreen: the picture Icon, the Title with
// the small Sub line under it, the Line that ends in moving dots and the Tip
// at the bottom ("": none).
type Loading struct {
	Icon, Title, Sub, Line, Tip string
}

// LoadingScreen shows p a loading screen over the whole screen: a sky with
// drifting clouds, the picture bobbing in the middle, the texts and a moving
// bar. It has the size of the windows, so it takes the place of the lobby's
// main screen. Esc does not close it: it stays until CloseLoading or another
// screen.
func LoadingScreen(p *player.Player, ld Loading) {
	l := newOreLayout()
	l.size = fmt.Sprintf("%dx%d", windowW, windowH)
	l.loading = true
	l.close = func(p *player.Player, _ *world.Tx) { LoadingScreen(p, ld) }
	clouds := make([]any, 0, 5)
	for i := range 5 {
		clouds = append(clouds, box(fmt.Sprintf("mc-load-cloud mc-load-cloud-%d", i)))
	}
	dots := []any{txt("mc-load-dot mc-load-dot-0", "."), txt("mc-load-dot mc-load-dot-1", "."), txt("mc-load-dot mc-load-dot-2", ".")}
	mid := []any{
		box("mc-load-isle", "children", []any{box("mc-load-glow"), img("mc-load-icon", ld.Icon)}),
		txt("mc-load-title", ld.Title),
	}
	if ld.Sub != "" {
		mid = append(mid, txt("mc-load-sub", ld.Sub))
	}
	mid = append(mid,
		box("mc-load-bar", "children", []any{box("mc-load-bar-fill")}),
		box("mc-load-line", "children", append([]any{txt("mc-load-line-text", ld.Line)}, dots...)),
	)
	children := []any{
		box("mc-load-sky"),
		box("mc-load-clouds", "children", clouds),
		box("mc-load-mid", "children", mid),
	}
	if ld.Tip != "" {
		children = append(children, txt("mc-load-tip", ld.Tip))
	}
	showOre(p, l, box("mc-screen mc-load", "children", children))
}

// CloseLoading closes p's loading screen, if it still shows one, once it has
// been up for minLoading.
func CloseLoading(p *player.Player) {
	if oreShown != nil {
		return
	}
	id := p.UUID()
	oreScreens.Lock()
	st := oreScreens.m[id]
	var wait time.Duration
	if st != nil && st.form != nil && !st.loading.IsZero() {
		wait = minLoading - time.Since(st.loading)
	}
	oreScreens.Unlock()
	if wait > 0 {
		time.AfterFunc(wait, func() { closeLoading(id) })
		return
	}
	closeLoading(id)
}

// closeLoading closes the screen of the player id if it is a loading screen.
func closeLoading(id uuid.UUID) {
	oreScreens.Lock()
	defer oreScreens.Unlock()
	st := oreScreens.m[id]
	if st == nil || st.form == nil || st.loading.IsZero() {
		return
	}
	_ = st.form.Close()
	st.form = nil
	st.loading = time.Time{}
}
