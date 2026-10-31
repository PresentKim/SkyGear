package dimension

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/instance"
)

// The second dimension's window, opened with the teleport stone: the
// monsters on the left (the open ones in order, then the closed ones) with
// their stars, and the chosen one's records and buttons on the right. While
// someone hunts, the others may watch instead.

// ShowWindow shows p the window with mob chosen (the first open monster
// when empty).
func ShowWindow(s *core.Session, p *player.Player, mob string) {
	lang := i18n.Of(p)
	if mob == "" || huntIndex(mob) < 0 {
		mob = Hunts[0].Mob
	}
	var open, closed []ui.ListRow
	for _, h := range Hunts {
		row := ui.ListRow{
			Icon: formicon.Of(item.NewStack(sgitem.Essence{Mob: h.Mob}, 1)), Selected: h.Mob == mob,
			Run: func(p *player.Player, _ *world.Tx) { ShowWindow(s, p, h.Mob) },
		}
		if IsOpen(s, h.Mob) {
			row.Label = ui.ListRowText(MobName(lang, h.Mob), starText(Stars(s, h.Mob)))
			open = append(open, row)
		} else {
			row.Label = ui.ListRowText("§7"+MobName(lang, h.Mob), i18n.T(lang, "dimension.window.closed"))
			closed = append(closed, row)
		}
	}
	sp := ui.ListSplit{Rows: append(open, closed...), Body: windowBody(s, lang, mob)}
	switch r := RunOf(s); {
	case r != nil:
		sp.Body += "\n\n" + fmt.Sprintf(i18n.T(lang, "dimension.window.busy"), hunterName(r), MobName(lang, r.Mob))
		sp.Actions = append(sp.Actions, s.Opt(i18n.T(lang, "dimension.window.watch"), "", func(p *player.Player, tx *world.Tx) {
			Watch(s, tx, p)
		}))
	case IsOpen(s, mob):
		sp.Actions = append(sp.Actions, s.Opt(i18n.T(lang, "dimension.window.enter"), "", func(p *player.Player, tx *world.Tx) {
			if !Begin(s, tx, p, mob) {
				ShowWindow(s, p, mob)
			}
		}), s.Opt(i18n.T(lang, "dimension.window.loadout"), "", func(p *player.Player, _ *world.Tx) {
			ShowLoadout(s, p, mob)
		}))
		sp.Body += "\n\n" + loadoutLine(s, p, lang)
	case CanOpen(s, mob):
		sp.Actions = append(sp.Actions, s.Opt(i18n.T(lang, "dimension.window.open"), formicon.Of(item.NewStack(sgitem.FairyEmerald{}, 1)), func(p *player.Player, tx *world.Tx) {
			if !Open(s, p, mob) {
				ui.Notify(p, "%s", i18n.T(i18n.Of(p), "dimension.window.noEmerald"))
			}
			ShowWindow(s, p, mob)
		}))
	}
	ui.ListSplitMenu(p, i18n.T(lang, "dimension.title"), sp)
}

// windowBody is the right side's text for mob: its records and stars, or
// what opens it.
func windowBody(s *core.Session, lang, mob string) string {
	h, _ := huntOf(mob)
	rec := RecordOf(s, mob)
	stars := Stars(s, mob)
	var b strings.Builder
	fmt.Fprintf(&b, i18n.T(lang, "dimension.window.hunt"), MobName(lang, mob), h.Count, h.Seconds/60, h.Seconds%60)
	b.WriteString("\n")
	if rec.Best > 0 {
		fmt.Fprintf(&b, i18n.T(lang, "dimension.window.best"), rec.Best/60, rec.Best%60)
	} else {
		b.WriteString(i18n.T(lang, "dimension.window.noBest"))
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, i18n.T(lang, "dimension.window.kills"), rec.Kills, rec.Clears)
	b.WriteString("\n\n")
	lines := []string{
		i18n.T(lang, "dimension.star.clear"),
		fmt.Sprintf(i18n.T(lang, "dimension.star.time"), h.Star2/60, h.Star2%60),
		fmt.Sprintf(i18n.T(lang, "dimension.star.kills"), h.Star3),
	}
	for i, line := range lines {
		mark := "§8☆ "
		if stars[i] {
			mark = "§e★ "
		}
		b.WriteString(mark + "§f" + line + "\n")
	}
	if !IsOpen(s, mob) {
		prev := Hunts[huntIndex(mob)-1].Mob
		b.WriteString("\n" + fmt.Sprintf(i18n.T(lang, "dimension.window.opens"), MobName(lang, prev)))
	}
	return strings.TrimRight(b.String(), "\n")
}

// loadoutLine names the relics p would hunt with now.
func loadoutLine(s *core.Session, p *player.Player, lang string) string {
	var names []string
	for _, kind := range Loadout(s, p) {
		g, _ := carried(p, kind)
		names = append(names, fmt.Sprintf("%s%s", i18n.T(lang, "relic."+kind), strings.Repeat("★", g)))
	}
	if len(names) == 0 {
		return i18n.T(lang, "dimension.window.noRelics")
	}
	return fmt.Sprintf(i18n.T(lang, "dimension.window.relics"), strings.Join(names, ", "))
}

// starText draws three stars, the earned ones lit.
func starText(stars [3]bool) string {
	var b strings.Builder
	for _, ok := range stars {
		if ok {
			b.WriteString("§e★")
		} else {
			b.WriteString("§8☆")
		}
	}
	return b.String()
}

// hunterName is the name of the hunter of r, if it is still around.
func hunterName(r *Run) string {
	if r.name != "" {
		return r.name
	}
	return "?"
}

// watcher is a player watching a run, and how it was before.
type watcher struct {
	Return mgl64.Vec3
	Mode   world.GameMode
}

// Watching reports whether p watches the save's run.
func Watching(s *core.Session, p *player.Player) bool {
	_, ok := runPart.Of(s).watchers[p.H()]
	return ok
}

// Watch takes p over the hunting ground as a spectator of the run under way.
func Watch(s *core.Session, tx *world.Tx, p *player.Player) {
	r := RunOf(s)
	l := runPart.Of(s)
	if r == nil || r.Hunter == p.H() || Watching(s, p) {
		return
	}
	if l.watchers == nil {
		l.watchers = map[*world.EntityHandle]watcher{}
	}
	ret := p.Position()
	if core.InArena(ret) {
		ret = island.Spawn(s)
	}
	l.watchers[p.H()] = watcher{Return: ret, Mode: p.GameMode()}
	p.SetGameMode(world.GameModeSpectator)
	over := arenaStart().Add(mgl64.Vec3{0, 8, 0})
	p.Teleport(over)
	instance.PlaceAt(p, over, over)
}

// StopWatching brings a watcher back where it was, as it was.
func StopWatching(s *core.Session, p *player.Player) {
	l := runPart.Of(s)
	w, ok := l.watchers[p.H()]
	if !ok {
		return
	}
	delete(l.watchers, p.H())
	p.SetGameMode(w.Mode)
	p.Teleport(w.Return)
	instance.PlaceAt(p, w.Return, island.Spawn(s))
}

// releaseWatchers brings every watcher of the save back (the run ended).
func releaseWatchers(s *core.Session, tx *world.Tx) {
	for h := range runPart.Of(s).watchers {
		if e, ok := h.Entity(tx); ok {
			StopWatching(s, e.(*player.Player))
		} else {
			delete(runPart.Of(s).watchers, h)
		}
	}
}

// ShowWatchMenu is the menu a watcher opens: stop watching.
func ShowWatchMenu(s *core.Session, p *player.Player) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "dimension.title"), i18n.T(lang, "dimension.watch.body"),
		s.Opt(i18n.T(lang, "dimension.watch.stop"), "", func(p *player.Player, _ *world.Tx) { StopWatching(s, p) }))
}
