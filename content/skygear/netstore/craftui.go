package netstore

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// keyName is the display name of a key.
func keyName(n *Network, lang string, k storagenet.Key) string {
	if st := Proto(n.Sess, k); !st.Empty() {
		return naming.ItemName(lang, st)
	}
	return string(k)
}

func stackList(n *Network, lang string, m map[storagenet.Key]int64) string {
	keys := make([]storagenet.Key, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s ×%d", keyName(n, lang, k), m[k])
	}
	return strings.Join(parts, ", ")
}

func craftErr(lang string, err error) string {
	switch {
	case errors.Is(err, storagenet.ErrCycle):
		return i18n.T(lang, "netstore.craft.err.cycle")
	case errors.Is(err, storagenet.ErrBudget):
		return i18n.T(lang, "netstore.craft.err.budget")
	case errors.Is(err, storagenet.ErrNoPattern):
		return i18n.T(lang, "netstore.craft.err.nopattern")
	case errors.Is(err, ErrTaskLimit):
		return fmt.Sprintf(i18n.T(lang, "netstore.craft.limit"), MaxTasks)
	case errors.Is(err, storagenet.ErrShort):
		return i18n.T(lang, "netstore.craft.err.short")
	}
	return i18n.T(lang, "netstore.craft.err.other")
}

// RequestUI asks how many of k to craft, then shows what the craft would use
// and lets the player start it.
func RequestUI(p *player.Player, tx *world.Tx, n *Network, k storagenet.Key) {
	lang := i18n.Of(p)
	cs := craftOf(n)
	if cs == nil {
		return
	}
	run := func(count int64) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) { preview(p, n, k, count) }
	}
	opts := []ui.Option{}
	for _, c := range []int64{1, 8, 64} {
		opts = append(opts, ui.Option{Label: fmt.Sprintf(i18n.T(lang, "netstore.craft.amount"), c), Run: run(c)})
	}
	opts = append(opts, ui.Option{Label: i18n.T(lang, "netstore.craft.custom"), Run: func(p *player.Player, tx *world.Tx) {
		ui.Input(p, i18n.T(lang, "netstore.craft.custom"), "", "1", func(p *player.Player, tx *world.Tx, v string) {
			c, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err != nil || c <= 0 {
				ui.Notify(p, "%s", i18n.T(lang, "netstore.craft.invalid"))
				return
			}
			preview(p, n, k, c)
		})
	}})
	ui.Menu(p, fmt.Sprintf(i18n.T(lang, "netstore.craft.title"), keyName(n, lang, k)), i18n.T(lang, "netstore.craft.body"), opts...)
}

// playerCb runs f for the player of h once the calculation is applied.
func playerCb(p *player.Player, f func(p *player.Player, tx *world.Tx, plan *storagenet.Plan, t *storagenet.Task, err error)) func(tx *world.Tx, plan *storagenet.Plan, t *storagenet.Task, err error) {
	h := p.H()
	return func(tx *world.Tx, plan *storagenet.Plan, t *storagenet.Task, err error) {
		if e, ok := h.Entity(tx); ok {
			if pl, ok := e.(*player.Player); ok {
				f(pl, tx, plan, t, err)
			}
		}
	}
}

func preview(p *player.Player, n *Network, k storagenet.Key, count int64) {
	lang := i18n.Of(p)
	cs := craftOf(n)
	if cs == nil {
		return
	}
	ui.Toast(p, i18n.T(lang, "netstore.craft.calculating"), keyName(n, lang, k))
	err := cs.Request(k, count, false, playerCb(p, func(p *player.Player, tx *world.Tx, plan *storagenet.Plan, _ *storagenet.Task, err error) {
		showPreview(p, n, k, count, plan, err)
	}))
	if err != nil {
		ui.Notify(p, "%s", craftErr(lang, err))
	}
}

func showPreview(p *player.Player, n *Network, k storagenet.Key, count int64, plan *storagenet.Plan, err error) {
	lang := i18n.Of(p)
	if err != nil {
		ui.Notify(p, "%s", craftErr(lang, err))
		return
	}
	var body []string
	if len(plan.Use) > 0 {
		body = append(body, fmt.Sprintf(i18n.T(lang, "netstore.craft.use"), stackList(n, lang, plan.Use)))
	}
	steps := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		name := s.PatternID
		if pat, ok := craftOf(n).repo.Get(s.PatternID); ok && len(pat.Outputs) > 0 {
			name = keyName(n, lang, pat.Outputs[0].Key)
		}
		steps = append(steps, fmt.Sprintf("%s ×%d", name, s.Times))
	}
	body = append(body, fmt.Sprintf(i18n.T(lang, "netstore.craft.steps"), strings.Join(steps, ", ")))
	opts := []ui.Option{}
	if plan.OK() {
		opts = append(opts, ui.Option{Label: i18n.T(lang, "netstore.craft.start"), Run: func(p *player.Player, tx *world.Tx) { start(p, n, k, count) }})
	} else {
		body = append(body, "§c"+fmt.Sprintf(i18n.T(lang, "netstore.craft.missing"), stackList(n, lang, plan.Missing)))
	}
	opts = append(opts, ui.Back(i18n.T(lang, "netstore.craft.back"), func(p *player.Player, tx *world.Tx) { RequestUI(p, tx, n, k) }))
	ui.Menu(p, fmt.Sprintf(i18n.T(lang, "netstore.craft.title"), fmt.Sprintf("%s ×%d", keyName(n, lang, k), count)), strings.Join(body, "\n"), opts...)
}

func start(p *player.Player, n *Network, k storagenet.Key, count int64) {
	lang := i18n.Of(p)
	cs := craftOf(n)
	if cs == nil {
		return
	}
	err := cs.Request(k, count, true, playerCb(p, func(p *player.Player, tx *world.Tx, plan *storagenet.Plan, t *storagenet.Task, err error) {
		switch {
		case err != nil:
			ui.Notify(p, "%s", craftErr(lang, err))
		case t == nil:
			ui.Notify(p, "%s", fmt.Sprintf(i18n.T(lang, "netstore.craft.missing"), stackList(n, lang, plan.Missing)))
		default:
			ui.Flash(p, i18n.T(lang, "netstore.craft.started"), keyName(n, lang, k), count)
		}
	}))
	if err != nil {
		ui.Notify(p, "%s", craftErr(lang, err))
	}
}

// MonitorUI lists the running tasks with their progress; each can be
// cancelled.
func MonitorUI(p *player.Player, tx *world.Tx, n *Network) {
	lang := i18n.Of(p)
	cs := craftOf(n)
	if cs == nil {
		return
	}
	var opts []ui.Option
	for _, t := range cs.Tasks() {
		id, st := t.ID(), t.Status()
		var done, total int64
		for _, s := range st.Steps {
			done += s.Done
			total += s.Total
		}
		label := fmt.Sprintf(i18n.T(lang, "netstore.craft.task"), keyName(n, lang, st.Want.Key), st.Want.Count, done, total)
		opts = append(opts, ui.Option{Label: label, Run: func(p *player.Player, tx *world.Tx) { taskUI(p, tx, n, id) }})
	}
	body := i18n.T(lang, "netstore.craft.monitor.body")
	if len(opts) == 0 {
		body = i18n.T(lang, "netstore.craft.monitor.empty")
	}
	ui.Menu(p, fmt.Sprintf(i18n.T(lang, "netstore.craft.monitor.title"), len(opts), MaxTasks), body, opts...)
}

func taskUI(p *player.Player, tx *world.Tx, n *Network, id string) {
	lang := i18n.Of(p)
	cs := craftOf(n)
	if cs == nil {
		return
	}
	for _, t := range cs.Tasks() {
		if t.ID() != id {
			continue
		}
		st := t.Status()
		var lines []string
		for _, s := range st.Steps {
			name := s.PatternID
			if pat, ok := cs.repo.Get(s.PatternID); ok && len(pat.Outputs) > 0 {
				name = keyName(n, lang, pat.Outputs[0].Key)
			}
			lines = append(lines, fmt.Sprintf("%s %d/%d", name, s.Done, s.Total))
		}
		if len(st.Waiting) > 0 {
			lines = append(lines, fmt.Sprintf(i18n.T(lang, "netstore.craft.waiting"), stackList(n, lang, st.Waiting)))
		}
		ui.Menu(p, fmt.Sprintf("%s ×%d", keyName(n, lang, st.Want.Key), st.Want.Count), strings.Join(lines, "\n"),
			ui.Option{Label: i18n.T(lang, "netstore.craft.cancel"), Run: func(p *player.Player, tx *world.Tx) {
				ui.Confirm(p, i18n.T(lang, "netstore.craft.cancel"), i18n.T(lang, "netstore.craft.cancel.confirm"), i18n.T(lang, "netstore.craft.cancel"), func(p *player.Player, tx *world.Tx) {
					if cs.Cancel(tx, id) {
						ui.Flash(p, "%s", i18n.T(lang, "netstore.craft.cancelled"))
					}
				})
			}},
			ui.Back(i18n.T(lang, "netstore.craft.back"), func(p *player.Player, tx *world.Tx) { MonitorUI(p, tx, n) }))
		return
	}
	MonitorUI(p, tx, n)
}
