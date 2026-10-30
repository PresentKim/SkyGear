package screens

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/hud"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/perf"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/platform/account"
)

// UpdateBoard refreshes p's HUD: day and play time (and the server load for
// an operator who turned it on) and the quest panel. Rows go out through
// common/hud only when they changed.
func UpdateBoard(s *core.Session, p *player.Player) {
	total, day, _ := s.Clock.Now()
	lang := i18n.Of(p)
	var b hud.Board
	b.Add(hud.TopLeft, fmt.Sprintf(i18n.T(lang, "sidebar.day"), day, textfmt.PlayTime(lang, total)))
	if showsLoad(s, p) {
		b.Add(hud.TopLeft, perf.Line(s.World))
	}
	if _, expanding := expand.Of(s, p); !expanding && account.On(p, advancement.SettingQuestPanel) {
		title, rows := QuestRows(s, lang, p.Inventory())
		b.Add(hud.QuestTitle, title)
		for _, r := range rows {
			b.Add(hud.Quest, r)
		}
	}
	if key := b.Key(); Boards.Of(s)[p.H()] != key {
		Boards.Of(s)[p.H()] = key
		b.Send(p)
	}
}

// nextResearch is the first research in tree order that can be done now.
func nextResearch(s *core.Session) (*progress.Research, bool) {
	for i := range progress.Tree {
		if r := &progress.Tree[i]; progress.Open(s, r) {
			return r, true
		}
	}
	return nil, false
}

// QuestRows are the quest panel: the next advancement and its hint (with
// the research cost for a research advancement), then once every
// advancement is done the next research and its cost. The cost rows count
// what inv holds of each item, when inv is not nil.
func QuestRows(s *core.Session, lang string, inv *inventory.Inventory) (string, []string) {
	if a, ok := progress.NextAdvancement(s); ok {
		rows := strings.Split(a.HintText(lang), "\n")
		for i := range rows {
			rows[i] = "§f" + rows[i]
		}
		if g, ok := progress.GoalResearch(a); ok {
			if r, ok := progress.Step(s, g); ok {
				if r != g || a.Research == "" {
					rows = append(rows, fmt.Sprintf(i18n.T(lang, "sidebar.quest.research"), r.Title(lang)))
				}
				rows = append(rows, ResearchCostRows(s, lang, r, inv)...)
			}
		}
		return i18n.T(lang, "sidebar.quest.advancement") + a.Title(lang), rows
	}
	r, ok := nextResearch(s)
	if !ok {
		return i18n.T(lang, "sidebar.quest.allResearch"), nil
	}
	return fmt.Sprintf(i18n.T(lang, "sidebar.quest.researchNamed"), r.Title(lang)), ResearchCostRows(s, lang, r, inv)
}

// ResearchCostRows are what a research spends, one row per item: how many
// of it inv holds out of the number still needed (green once there are
// enough), or the number alone when inv is nil.
func ResearchCostRows(s *core.Session, lang string, r *progress.Research, inv *inventory.Inventory) []string {
	var rows []string
	for _, c := range r.Cost {
		name, need := naming.ItemName(lang, item.NewStack(c.Item, 1)), progress.Need(s, r, c)
		if inv == nil {
			rows = append(rows, fmt.Sprintf("§f%s §e×%d", name, need))
			continue
		}
		have, colour := min(core.CountPlain(inv, c.Item), need), "§e"
		if have >= need {
			colour = "§a"
		}
		rows = append(rows, fmt.Sprintf("§f%s %s%d§7/%d", name, colour, have, need))
	}
	return rows
}

func showsLoad(s *core.Session, p *player.Player) bool {
	return s.IsOperator(p) && perf.Shown(p.Name())
}

// UpdateBoards refreshes every player's HUD. It runs every second (the play
// time, the server load and the counts of the cost rows change) and after
// the events that change the quest panel.
func UpdateBoards(s *core.Session, tx *world.Tx) {
	for e := range tx.Players() {
		UpdateBoard(s, e.(*player.Player))
	}
}
