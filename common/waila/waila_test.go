package waila

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
)

// TestFormat: the title keeps the pack's fixed layout: the prefix, a
// 10-character tool field, then the text with the anchor token.
func TestFormat(t *testing.T) {
	title, sub := Format(Info{
		Item: block.Stone{}, Name: "돌", Lines: []string{"진행 40%"}, Source: "Minecraft",
		Tools: []ToolIcon{{PickaxeWood, MarkRight}},
	})
	if !strings.HasPrefix(title, "_waila:A:aa,a;zz,z:") {
		t.Fatalf("title %q", title)
	}
	if len("_waila:A:aa,a;zz,z:") != 19 {
		t.Fatal("header must be 19 characters (the pack cuts the first 19)")
	}
	for _, want := range []string{"돌", "\n§7진행 40%", "\n§9§oMinecraft", anchor} {
		if !strings.Contains(title, want) {
			t.Errorf("title %q misses %q", title, want)
		}
	}
	if len(sub) != 1+iconWidth || sub == "i"+empty {
		t.Errorf("subtitle %q, want an item icon", sub)
	}
	if _, sub := Format(Info{Name: "물"}); sub != "i"+empty {
		t.Errorf("no item: subtitle %q, want i and an empty field", sub)
	}
}

// TestToolsFor: stone wants a wooden pickaxe; the mark follows the held item.
func TestToolsFor(t *testing.T) {
	got := ToolsFor(block.Stone{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1))
	if len(got) == 0 || got[0] != (ToolIcon{PickaxeWood, MarkRight}) {
		t.Fatalf("stone with an iron pickaxe: %v", got)
	}
	if got := ToolsFor(block.Stone{}, item.Stack{}); len(got) == 0 || got[0].Mark != MarkWrong {
		t.Fatalf("stone by hand: %v", got)
	}
	if got := ToolsFor(block.Dirt{}, item.Stack{}); len(got) == 0 || got[0] != (ToolIcon{Shovel, MarkNone}) {
		t.Fatalf("dirt by hand: %v", got)
	}
	if got := ToolsFor(block.DiamondOre{}, item.NewStack(item.Pickaxe{Tier: item.ToolTierStone}, 1)); len(got) == 0 || got[0] != (ToolIcon{PickaxeIron, MarkWrong}) {
		t.Fatalf("diamond ore with a stone pickaxe: %v", got)
	}
}

// TestFormatContainer: a chest adds up stacks of a kind into fixed-width
// slot fields after the icon and names its layout in the title.
func TestFormatContainer(t *testing.T) {
	items := []item.Stack{item.NewStack(block.Dirt{}, 64), {}, item.NewStack(block.Dirt{}, 10), item.NewStack(block.Stone{}, 1)}
	title, sub := Format(Info{Name: "상자", Item: block.NewChest(), Container: &Container{Layout: LayoutChest, Items: items}})
	if !strings.Contains(title, "__waila:inv.chest__") || strings.Contains(title, "inv.size_") {
		t.Fatalf("title %q", title)
	}
	if want := 1 + iconWidth + panelSlots*(iconWidth+countWidth); len(sub) != want {
		t.Fatalf("subtitle is %d long, want %d", len(sub), want)
	}
	slot := func(i int) (string, string) {
		start := 1 + iconWidth + i*(iconWidth+countWidth)
		return sub[start : start+iconWidth], sub[start+iconWidth : start+iconWidth+countWidth]
	}
	if it, n := slot(0); it == empty || n != "§f74~~~" {
		t.Errorf("slot 0 = %q %q, want dirt ×74", it, n)
	}
	if it, n := slot(1); it == empty || n != "~~~~~~~~" {
		t.Errorf("slot 1 = %q %q, want one stone", it, n)
	}
	if it, _ := slot(2); it != empty {
		t.Errorf("slot 2 = %q, want empty", it)
	}
	// More kinds than a row: they go on in the next rows, up to 27.
	var many []item.Stack
	for i := range 30 {
		many = append(many, item.NewStack(block.Wool{Colour: item.Colours()[i%16]}, 1).WithCustomName(string(rune('a'+i))))
	}
	title, sub = Format(Info{Container: &Container{Layout: LayoutChest, Items: many}})
	if strings.Contains(title, "inv.size_") {
		t.Errorf("30 kinds: title %q, want no +n", title)
	}
	for _, i := range []int{9, 10, 18, 19, 27} {
		if it, _ := slot(i); it == empty {
			t.Errorf("30 kinds: slot %d is empty", i)
		}
	}
	if it, _ := slot(8); it != empty {
		t.Errorf("30 kinds: slot 8 (the icon's) = %q, want empty", it)
	}
	if len(sub) < tailStart {
		t.Errorf("subtitle is %d bytes, want the %d of the slots", len(sub), tailStart)
	}
}

// empty is the item field of nothing.
const empty = "xxxxxxxxxx0"

// TestItemField: item fields are the runtime ID × 65536 left padded with
// 'x' to iconWidth.
func TestItemField(t *testing.T) {
	for _, it := range []item.Stack{{}, item.NewStack(block.Stone{}, 1), item.NewStack(item.Diamond{}, 1), item.NewStack(block.NewChest(), 1)} {
		f := itemField(it)
		digits := strings.TrimLeft(f, "x")
		if len(f) != iconWidth || strings.Contains(digits, "x") || (len(digits) > 1 && digits[0] == '0') {
			t.Errorf("%v: field %q", it, f)
		}
		v, err := strconv.Atoi(digits)
		if err != nil || v%65536 != 0 || (v == 0) != it.Empty() {
			t.Errorf("%v: field %q is no item", it, f)
		}
	}
}

func TestPrecomputedFields(t *testing.T) {
	if itemField(item.Stack{}) != empty || emptyItemField != empty {
		t.Errorf("empty field %q", itemField(item.Stack{}))
	}
	for _, n := range []int{-1, 0, 1} {
		if got := countField(n); got != "~~~~~~~~" {
			t.Errorf("countField(%d) = %q", n, got)
		}
	}
	for _, tools := range [][]ToolIcon{nil, {{PickaxeWood, MarkRight}}, {{AxeIron, MarkWrong}, {Shears, 0}}, {{"", Mark(200)}, {Hoe, Mark(255)}}} {
		title, _ := Format(Info{Tools: tools})
		want := panelPrefix
		padded := [2]ToolIcon{{ToolNone, MarkNone}, {ToolNone, MarkNone}}
		copy(padded[:], tools)
		for i, ti := range padded {
			if ti.Tool == "" {
				ti = ToolIcon{ToolNone, MarkNone}
			}
			if ti.Mark == 0 {
				ti.Mark = MarkNone
			}
			want += fmt.Sprintf("%s,%c%c", ti.Tool, ti.Mark, ";:"[i])
		}
		if !strings.HasPrefix(title, want) {
			t.Errorf("tools %v: title %q, want prefix %q", tools, title, want)
		}
	}
	var items []item.Stack
	for range 30 {
		items = append(items, item.NewStack(block.Dirt{}, 1))
	}
	out := Container{Layout: LayoutHopper, Items: items}.slots()
	for i, it := range out {
		if it.Empty() != (i == 8) {
			t.Errorf("hopper slot %d empty = %v", i, it.Empty())
		}
	}
}

// TestPackFields: the pack cuts the subtitle where Format puts the fields.
func TestPackFields(t *testing.T) {
	data, err := os.ReadFile("../packs/waila/ui/waila/tile.json")
	if err != nil {
		t.Fatal(err)
	}
	pack := string(data)
	const v = "$var_subtitle_text"
	// A field is width characters from start.
	cut := func(start, width int) string {
		return fmt.Sprintf("('%%.%ds' * (%s - ('%%.%ds' * %s)))", width, v, start, v)
	}
	wants := []string{cut(1, iconWidth)}
	for k := range panelSlots {
		start := 1 + iconWidth + k*(iconWidth+countWidth)
		wants = append(wants, fmt.Sprintf("($index = %d)", k), cut(start, iconWidth), cut(start+iconWidth, countWidth))
	}
	for _, w := range wants {
		if !strings.Contains(pack, w) {
			t.Errorf("tile.json misses %s", w)
		}
	}
}

// TestTail: marks and the item line sit at fixed places after the slots.
func TestTail(t *testing.T) {
	title, sub := Format(Info{
		Name: "서랍", Item: block.NewChest(),
		Marks: Marks{Lock: true, Multiplier: 4},
		Line:  &ItemLine{Item: item.NewStack(block.Dirt{}, 1), Name: "흙", Text: "x 1,532/4,096", Gauge: 10, ShowGauge: true},
	})
	if !strings.Contains(title, lineToken) {
		t.Errorf("title %q has no line token", title)
	}
	if len(sub) < nameStart || sub[nameStart:] != "흙" {
		t.Fatalf("subtitle %q: name not at %d", sub, nameStart)
	}
	if got := sub[flagsStart:textStart]; got != "Lc-k" {
		t.Errorf("flags %q, want Lc-k", got)
	}
	if got := sub[textStart:nameStart]; got != "x 1,532/4,096~~~~~~~~~~~" {
		t.Errorf("text %q", got)
	}
	if got := sub[tailStart:flagsStart]; got == empty {
		t.Errorf("line icon is empty")
	}
	// Only marks: no line token, still the full layout.
	title, sub = Format(Info{Name: "서랍", Marks: Marks{Void: true}})
	if strings.Contains(title, lineToken) || len(sub) != nameStart || sub[flagsStart:textStart] != "-aV-" {
		t.Errorf("marks only: %q %q", title, sub)
	}
	// The pack cuts the same places.
	data, err := os.ReadFile("../packs/waila/ui/waila/tile.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []int{tailStart, flagsStart, flagsStart + 1, flagsStart + 2, flagsStart + 3, textStart, nameStart} {
		if w := fmt.Sprintf("('%%.%ds' * $var_subtitle_text)", at); !strings.Contains(string(data), w) {
			t.Errorf("tile.json does not cut at %d", at)
		}
	}
}

// TestFurnaceProgress: the progress item goes to the slot the pack draws in
// place of the arrow.
func TestFurnaceProgress(t *testing.T) {
	_, sub := Format(Info{Name: "화로", Container: &Container{Layout: LayoutFurnace,
		Items: []item.Stack{item.NewStack(block.Dirt{}, 1)}, Progress: item.NewStack(block.Stone{}, 1)}})
	start := 1 + iconWidth + progressSlot*(iconWidth+countWidth)
	if got := sub[start : start+iconWidth]; got == empty || got != itemField(item.NewStack(block.Stone{}, 1)) {
		t.Errorf("progress slot %q", got)
	}
	data, _ := os.ReadFile("../packs/waila/ui/waila/tile.json")
	if !strings.Contains(string(data), fmt.Sprintf(`"$index": %d`, progressSlot)) {
		t.Errorf("tile.json draws no slot %d", progressSlot)
	}
}

// TestNameKeyIsSentAsRawText: a vanilla block without a name of ours is sent
// as a translation key the client translates (raw JSON title).
func TestNameKeyIsSentAsRawText(t *testing.T) {
	title, _ := Format(Info{Name: "stripped_oak_log", NameKey: "tile.stripped_oak_log.name"})
	raw, ok := rawText(title)
	if !ok || !strings.Contains(raw, `{"translate":"tile.stripped_oak_log.name"}`) || !strings.Contains(raw, `"rawtext"`) {
		t.Fatalf("raw title %q (%v)", raw, ok)
	}
	if _, ok := rawText("plain"); ok {
		t.Error("a title without keys became JSON")
	}
	plain, _ := Format(Info{Name: "상자"})
	if strings.Contains(plain, keyMark) {
		t.Error("a named block carries a key mark")
	}
}
