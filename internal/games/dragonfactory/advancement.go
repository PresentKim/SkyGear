package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Advancements (docs/design/02-dragonfactory.md §3.5, 05-sky-progression.md
// §5): Bedrock has no advancements a server can use, so they are ours. An
// advancement is a step of the progression with a hint of how to do it; the
// quest panel shows the first one open. They belong to the save and are
// shared by everyone on the island like the researches. The tree is a
// draft.

// Advancement is one step of the advancement tree. It is done by having an
// item, by a research or by an event the game reports (advance).
type Advancement struct {
	ID, Name string
	// Hint says how to do it, one HUD row per line.
	Hint string
	// Parent must be done before it opens (empty for the first).
	Parent string
	// Item is the name of an item to have in the inventory (unlockName
	// form), or empty.
	Item string
	// Research is a research to complete, or empty.
	Research string
	// Icon is the item name shown in the advancement screen; Item when
	// empty.
	Icon string
}

// Advancements done by events (advance).
const (
	advGrowTree = "grow_tree"
	advSift     = "sift"
	advMachine  = "machine"
	advChunk    = "chunk"
)

// advancementTree is every advancement; the quest panel follows this
// order, so the main path comes first.
var advancementTree = []Advancement{
	{ID: "wood", Name: "나무 캐기", Hint: "섬 가운데 나무를\n맨손으로 캐세요", Item: "oak_log"},
	{ID: "planks", Name: "판자", Parent: "wood", Hint: "원목으로\n참나무 판자를 만드세요", Item: "oak_planks"},
	{ID: "crafting_table", Name: "조합대", Parent: "planks", Hint: "판자 4개로\n조합대를 만드세요", Item: "crafting_table"},
	{ID: "research_table", Name: "연구대", Parent: "crafting_table", Hint: "조합대 2개로\n연구대를 만드세요", Item: "dragonfactory:research_table"},
	{ID: "first_research", Name: "첫 연구", Parent: "research_table", Hint: "연구대를 놓고 사용해\n막대기를 연구하세요", Research: "stick", Icon: "stick"},
	{ID: "pebble", Name: "돌멩이", Parent: "first_research", Hint: "막대기를 들고\n잔디·흙을 우클릭하세요", Item: "dragonfactory:pebble_stone"},
	{ID: "cobble", Name: "조약돌", Parent: "pebble", Hint: "돌 다지기를 연구하고\n돌멩이 4개로 만드세요", Item: "cobblestone"},
	{ID: advGrowTree, Name: "나무 키우기", Parent: "wood", Hint: "묘목을 심고 옆에서\n웅크리기를 반복하세요", Icon: "oak_sapling"},
	{ID: "wood_tools", Name: "나무 곡괭이", Parent: "first_research", Hint: "나무 도구를 연구해\n나무 곡괭이를 만드세요", Item: "wooden_pickaxe"},
	{ID: "composter", Name: "퇴비통", Parent: "wood_tools", Hint: "연구로 퇴비통을 열어\n잎·묘목으로 흙을 만드세요", Item: "composter"},
	{ID: "barrel", Name: "배럴", Parent: "composter", Hint: "연구로 배럴을 열어\n식물로 물을 만드세요", Item: "dragonfactory:barrel_oak"},
	{ID: "stone_axe", Name: "돌 도끼", Parent: "cobble", Hint: "연구로 돌 도끼 조합법을\n열어 만드세요", Item: "stone_axe"},
	{ID: "fiber", Name: "나무껍질 섬유", Parent: "stone_axe", Hint: "돌 도끼로 원목을\n우클릭해 벗기세요", Item: "dragonfactory:bark_fiber"},
	{ID: "string", Name: "실", Parent: "fiber", Hint: "실을 연구하고 조합대에\n섬유 3개를 한 줄로 놓으세요", Item: "string"},
	{ID: "hammer", Name: "망치", Parent: "cobble", Hint: "망치를 연구해\n만드세요", Item: "dragonfactory:hammer"},
	{ID: "gravel", Name: "자갈", Parent: "hammer", Hint: "망치로 조약돌을\n캐세요", Item: "gravel"},
	{ID: "sieve", Name: "체", Parent: "string", Hint: "연구로 체 조합법을\n열어 만드세요", Item: "dragonfactory:sieve"},
	{ID: advSift, Name: "체질", Parent: "sieve", Hint: "체에 자갈을 얹고\n우클릭을 반복하세요", Icon: "gravel"},
	{ID: "dust", Name: "먼지", Parent: "gravel", Hint: "망치로 모래를\n캐세요", Item: "dragonfactory:dust"},
	{ID: "furnace", Name: "화로", Parent: "cobble", Hint: "화로를 연구해\n만드세요", Item: "furnace"},
	{ID: "iron", Name: "철", Parent: advSift, Hint: "체로 자갈을 걸러 철 원석 조각을\n모으고 연구로 철을 가공하세요", Item: "iron_ingot"},
	{ID: advMachine, Name: "첫 기계", Parent: "iron", Hint: "연구로 기계를 열어\n처음 하나를 설치하세요", Icon: "dragonfactory:crusher"},
	{ID: advChunk, Name: "섬 넓히기", Parent: "cobble", Hint: "메뉴 → 섬 확장에서\n청크를 사세요", Icon: "grass_block"},
}

// advancementByID returns an advancement of the tree.
func advancementByID(id string) (*Advancement, bool) {
	for i := range advancementTree {
		if advancementTree[i].ID == id {
			return &advancementTree[i], true
		}
	}
	return nil, false
}

// advanced reports whether the save has done an advancement.
func (s *session) advanced(id string) bool { return s.state.Advancements[id] }

// advancementOpen reports whether an advancement is not done yet and its
// parent is.
func (s *session) advancementOpen(a *Advancement) bool {
	return !s.advanced(a.ID) && (a.Parent == "" || s.advanced(a.Parent))
}

// nextAdvancement is the first open advancement in tree order.
func (s *session) nextAdvancement() (*Advancement, bool) {
	for i := range advancementTree {
		if a := &advancementTree[i]; s.advancementOpen(a) {
			return a, true
		}
	}
	return nil, false
}

// advance completes an advancement reported by an event.
func (s *session) advance(tx *world.Tx, id string) {
	if a, ok := advancementByID(id); ok && !s.advanced(id) {
		s.completeAdvancements(tx, []*Advancement{a})
	}
}

// checkAdvancements completes the item and research advancements met by
// the players in tx. It runs every second.
func (s *session) checkAdvancements(tx *world.Tx) {
	var have map[string]bool
	var done []*Advancement
	for i := range advancementTree {
		a := &advancementTree[i]
		if s.advanced(a.ID) {
			continue
		}
		switch {
		case a.Research != "":
			if s.researched(a.Research) {
				done = append(done, a)
			}
		case a.Item != "":
			if have == nil {
				have = heldItemNames(tx)
			}
			if have[unlockName(a.Item)] {
				done = append(done, a)
			}
		}
	}
	if len(done) > 0 {
		s.completeAdvancements(tx, done)
	}
}

// heldItemNames are the names of the items in the inventories of the
// players in tx.
func heldItemNames(tx *world.Tx) map[string]bool {
	names := map[string]bool{}
	for e := range tx.Players() {
		for _, st := range e.(*player.Player).Inventory().Items() {
			name, _ := st.Item().EncodeItem()
			names[name] = true
		}
	}
	return names
}

// completeAdvancements marks advancements done, with the ones before them
// (an item used up within a second would otherwise leave a gap), and tells
// everyone on the island what they did and what comes next.
func (s *session) completeAdvancements(tx *world.Tx, as []*Advancement) {
	if s.state.Advancements == nil {
		s.state.Advancements = map[string]bool{}
	}
	var names []string
	for _, a := range as {
		for cur, ok := a, true; ok && !s.advanced(cur.ID); cur, ok = advancementByID(cur.Parent) {
			s.state.Advancements[cur.ID] = true
		}
		names = append(names, a.Name)
	}
	s.storeState()
	s.updateBoards(tx)
	msg := "§a발전과제 달성: §f" + strings.Join(names, ", ")
	if next, ok := s.nextAdvancement(); ok {
		msg += fmt.Sprintf("\n§7다음: §e%s §7— %s", next.Name, flatHint(next))
	}
	for e := range tx.Players() {
		p := e.(*player.Player)
		p.SendToast("§a발전과제 달성", strings.Join(names, ", "))
		p.Message(msg)
		p.PlaySound(sound.LevelUp{})
	}
}

// flatHint is an advancement's hint on one line.
func flatHint(a *Advancement) string { return strings.ReplaceAll(a.Hint, "\n", " ") }

// advancementIcon is the icon of an advancement in the advancement screen.
func advancementIcon(a *Advancement) string {
	name := a.Icon
	if name == "" {
		name = a.Item
	}
	if it, ok := world.ItemByName(unlockName(name), 0); ok {
		return iconOf(item.NewStack(it, 1))
	}
	return iconMission
}

// showAdvancements shows the advancement tree: done ones in green, open
// ones with their hint, the rest hidden.
func (s *session) showAdvancements(p *player.Player) {
	body, opts := s.advancementMenu()
	ui.Menu(p, "발전과제", body, opts...)
}

func (s *session) advancementMenu() (string, []ui.Option) {
	done := 0
	var opts []ui.Option
	for i := range advancementTree {
		a := &advancementTree[i]
		var label, icon string
		switch {
		case s.advanced(a.ID):
			done++
			label, icon = "§a✔ "+a.Name, advancementIcon(a)
		case s.advancementOpen(a):
			label, icon = "§e"+a.Name+"\n§7"+flatHint(a), advancementIcon(a)
		default:
			label, icon = "§8???", iconLocked
		}
		opts = append(opts, s.opt(label, icon, func(p *player.Player, _ *world.Tx) { s.showAdvancementNode(p, a) }))
	}
	body := fmt.Sprintf("§7섬의 모두가 함께 진행합니다. §f%d§7/%d 달성", done, len(advancementTree))
	return body, append(opts, s.backOption())
}

// showAdvancementNode shows one advancement; a hidden one shows only what
// comes before it.
func (s *session) showAdvancementNode(p *player.Player, a *Advancement) {
	title, body := a.Name, ""
	switch {
	case s.advanced(a.ID):
		body = "§a달성했습니다.\n\n§7" + flatHint(a)
	case s.advancementOpen(a):
		body = "§e진행 중\n\n§f" + flatHint(a)
	default:
		title, body = "???", "§8아직 보이지 않습니다. 앞의 발전과제를 먼저 달성하세요."
	}
	ui.Menu(p, "발전과제 · "+title, body, ui.Option{Label: "§f뒤로", Icon: iconBack, Run: func(p *player.Player, _ *world.Tx) { s.showAdvancements(p) }})
}
