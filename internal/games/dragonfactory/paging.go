package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// slotPageSize is how many item slots a page of a grid holds: a form with many
// buttons is slow to open, so a long grid is split into pages
// (docs/tech/json-ui.md).
const slotPageSize = 33

// slotPage cuts the slots of a long grid into the page asked for and returns the
// tab buttons that turn the page (none when everything fits on one page).
// again shows the grid on another page.
func (s *session) slotPage(p *player.Player, slots []ui.ItemSlot, page int, again func(p *player.Player, page int)) ([]ui.ItemSlot, []ui.Option) {
	pages := max(1, (len(slots)+slotPageSize-1)/slotPageSize)
	page = min(max(page, 0), pages-1)
	slots = slots[min(page*slotPageSize, len(slots)):min((page+1)*slotPageSize, len(slots))]
	var turns []ui.Option
	turn := func(label string, to int) ui.Option {
		return ui.Tab(label, "", false, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				again(p, to)
			}
		})
	}
	if page > 0 {
		turns = append(turns, turn(fmt.Sprintf(i18n.T(i18n.Of(p), "researchpage.prev"), page, pages), page-1))
	}
	if page < pages-1 {
		turns = append(turns, turn(fmt.Sprintf(i18n.T(i18n.Of(p), "researchpage.next"), page+2, pages), page+1))
	}
	return slots, turns
}
