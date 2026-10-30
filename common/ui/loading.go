package ui

import (
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

// loadingData is the data of the loading screen.
type loadingData struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Sub   string `json:"sub"`
	Line  string `json:"line"`
	Tip   string `json:"tip"`
}

// LoadingScreen shows p a loading screen over the whole screen: a sky with
// drifting clouds, the picture bobbing in the middle, the texts and a moving
// bar. It has the size of the windows, so it takes the place of the lobby's
// main screen. Esc does not close it: it stays until CloseLoading or another
// screen.
func LoadingScreen(p *player.Player, ld Loading) {
	l := newOreLayout()
	l.loading = true
	l.close = func(p *player.Player, _ *world.Tx) { LoadingScreen(p, ld) }
	showScreen(p, l, "loading", loadingData{Icon: oreIcon(ld.Icon), Title: ld.Title, Sub: ld.Sub, Line: ld.Line, Tip: ld.Tip})
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
