package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/ui"
)

// slotPageSize is how many item slots a page of a grid holds: a form with many
// buttons is slow to open, so a long grid is split into pages
// (docs/tech/json-ui.md).
const slotPageSize = 33

// slotPage cuts the slots of a long grid into the page asked for and returns the
// buttons that turn the page (none when everything fits on one page).
// again shows the grid on another page.
func (s *session) slotPage(p *player.Player, slots []ui.ItemSlot, page int, again func(p *player.Player, page int)) ([]ui.ItemSlot, []ui.Option) {
	pages := max(1, (len(slots)+slotPageSize-1)/slotPageSize)
	page = min(max(page, 0), pages-1)
	slots = slots[min(page*slotPageSize, len(slots)):min((page+1)*slotPageSize, len(slots))]
	return slots, s.guard(ui.Pager(p, page, pages, func(p *player.Player, _ *world.Tx, to int) { again(p, to) }))
}
