// Package hud sends the HUD texts (top left, top right, quest panel) as the
// rows of the sidebar scoreboard. The resource pack's scoreboards.json
// (assets/resourcepack/dragonfactory/ui/scoreboards.json) draws a sidebar
// whose objective name is Title as these parts instead of the sidebar: each
// row starts with the marker of its part, which the pack removes. Rows keep
// their order within a part. docs/tech/json-ui.md "HUD".
package hud

import (
	"strings"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/scoreboard"
)

// Title is the objective name that tells the pack to draw the HUD parts.
const Title = "§d§h§u§d"

// Part markers: each row starts with one.
const (
	TopLeft    = "§d§1"
	TopRight   = "§d§2"
	QuestTitle = "§d§3"
	Quest      = "§d§4"
)

// maxRows is the most rows a sidebar shows.
const maxRows = 15

// Board collects the rows of one player's HUD.
type Board struct{ rows []string }

// Add adds a row of text to a part (TopLeft, TopRight, QuestTitle, Quest).
// Rows past the sidebar's 15 are dropped.
func (b *Board) Add(part, text string) {
	if len(b.rows) < maxRows {
		b.rows = append(b.rows, part+text)
	}
}

// Rows returns the rows as sent: rows the same as an earlier one get an
// invisible "§r" each so the client keeps them apart (it merges rows of the
// same text).
func (b *Board) Rows() []string {
	rows := make([]string, len(b.rows))
	seen := map[string]bool{}
	for i, r := range b.rows {
		for seen[r] {
			r += "§r"
		}
		seen[r], rows[i] = true, r
	}
	return rows
}

// Send shows the board to p, or removes p's sidebar if the board is empty.
func (b *Board) Send(p *player.Player) {
	if len(b.rows) == 0 {
		p.RemoveScoreboard()
		return
	}
	sb := scoreboard.New(Title)
	sb.RemovePadding()
	for i, r := range b.Rows() {
		sb.Set(i, r)
	}
	p.SendScoreboard(sb)
}

// Strip returns a row as the pack shows it: without its part marker.
func Strip(row string) string {
	for _, m := range []string{TopLeft, TopRight, QuestTitle, Quest} {
		if strings.HasPrefix(row, m) {
			row = strings.TrimPrefix(row, m)
			for strings.HasSuffix(row, "§r") {
				row = strings.TrimSuffix(row, "§r")
			}
			return row
		}
	}
	return row
}
