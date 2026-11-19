package invui

import "fmt"

// A screen is written in the slots of Size (the constants of this package),
// and a window holds only the slots its layout uses: the common slots first,
// then the layout's own. The pack (tools/invuigen) reads the same places.

// Common are the slots every layout uses, at the front of the window in this
// order.
var Common = concat(
	span(SlotKey, SlotNext+1),
	span(TabIcon0, TabBtn0+Tabs),
	span(BarIcon0, BarBtn0+Bars),
	span(SlotLogo, SlotRight+1),
)

// ListActions is how many action buttons a list-split screen draws, in the
// rows from ListRows.
const ListActions = 12

// own are the slots each layout uses after the common ones.
var own = map[string][]int{
	KeyMenu:      rows(Rows),
	KeyCards:     rows(Rows),
	KeyGrid:      concat(rows(24), field(), cells(Cells)),
	KeyBoard:     concat(rows(24), cells(Cells)),
	KeyPick:      concat(rows(24), cells(2*PickCards)),
	KeyResult:    concat(rows(24), cells(ResultReward0+ResultRewards)),
	KeySplit:     concat(rows(1+SplitActions), cells(Cells)),
	KeyListSplit: concat(rows(ListRows+ListActions), cells(ListRows)),
	KeyRange:     concat(rows(RangeRow0+4), cells(RangeView+1)),
	KeyHome:      concat(rows(HomeOverRow0+HomeOvers), cells(HomeSideBadge0+HomeSides)),
	KeyLoading:   concat(rows(1), cells(1)),
}

func init() {
	for n := 1; n <= RadialItems; n++ {
		own[KeyRadial+fmt.Sprint(n)] = concat(rows(RadialItems+1), cells(RadialItems))
	}
	for g := range QuestGrids {
		own[KeyQuest+fmt.Sprint(g)] = concat(span(RowIcon0, RowIcon0+QuestChapters), span(RowBtn0, RowBtn0+2*QuestChapters), cells(Cells))
	}
	for layout, slots := range own {
		places[layout] = map[int]int{}
		start := len(Common) + offset(layout)
		for i, v := range slots {
			places[layout][v] = start + i
		}
	}
}

// places are the window slots of the slots of each layout.
var places = map[string]map[int]int{}

// offset is where a layout's own slots start after the common ones: the
// lobby's main screen goes after the most the screens of its buttons use,
// so its slots stay while one of them is shown over it.
func offset(layout string) int {
	if layout != KeyHome {
		return 0
	}
	most := 0
	for _, l := range Shared[KeyHome] {
		if l != KeyHome {
			most = max(most, len(own[l]))
		}
	}
	return most
}

// Place returns the window slot of slot in a window drawing layout, false if
// the layout does not use it.
func Place(layout string, slot int) (int, bool) {
	for i, v := range Common {
		if v == slot {
			return i, true
		}
	}
	p, ok := places[layout][slot]
	return p, ok
}

// Region returns the window slots a layout's own slots take.
func Region(layout string) (first, end int) {
	first = len(Common) + offset(layout)
	return first, first + len(own[layout])
}

// Layouts returns every layout with slots of its own.
func Layouts() []string {
	out := make([]string, 0, len(own))
	for l := range own {
		out = append(out, l)
	}
	return out
}

func span(from, to int) []int {
	out := make([]int, 0, to-from)
	for i := from; i < to; i++ {
		out = append(out, i)
	}
	return out
}

func rows(n int) []int { return concat(span(RowIcon0, RowIcon0+n), span(RowBtn0, RowBtn0+n)) }

func cells(n int) []int { return span(Cell0, Cell0+n) }

func field() []int { return []int{RowIcon0 + RowField, RowBtn0 + RowField} }

func concat(parts ...[]int) []int {
	var out []int
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
