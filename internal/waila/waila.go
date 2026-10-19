// Package waila shows what a player looks at in the WAILA HUD panel. The
// panel is the WAILA resource pack (assets/resourcepack/waila); this package does what its Script API behaviour pack did:
// it sends the panel's data as a title the pack draws instead of a vanilla
// title (docs/tech/waila.md). Games find the target and fill an Info.
package waila

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/title"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/dfhook"
)

// Tool is a tool icon of the panel (textures/waila/icons/tags/
// tile/<code>.png).
type Tool string

const (
	ToolNone       Tool = "zz"
	PickaxeWood    Tool = "aa"
	PickaxeStone   Tool = "ab"
	PickaxeIron    Tool = "ac"
	PickaxeDiamond Tool = "ad"
	AxeWood        Tool = "ba"
	AxeStone       Tool = "bb"
	AxeIron        Tool = "bc"
	AxeDiamond     Tool = "bd"
	Shovel         Tool = "ca"
	Hoe            Tool = "da"
	Sword          Tool = "ea"
	Crops          Tool = "fb"
	Shears         Tool = "fc"
)

// Mark is the small mark on a tool icon: whether the held item is right.
type Mark byte

const (
	MarkNone  Mark = 'z'
	MarkRight Mark = 'a'
	MarkWrong Mark = 'b'
)

// ToolIcon is a tool icon with its mark.
type ToolIcon struct {
	Tool Tool
	Mark Mark
}

// Info is what the panel shows about a block.
type Info struct {
	// Item is drawn as the icon; nil shows none.
	Item world.Item
	// Name is the first line.
	Name string
	// NameKey, if not empty, is a client translation key ("tile.stone.name")
	// shown as the first line instead of Name, in the player's own language
	// (for vanilla blocks and items the game has no name of its own for).
	NameKey string
	// Lines are shown under the name.
	Lines []string
	// Source is an optional last, blue line: where the block comes from.
	Source string
	// Tools are up to two tool icons at the top right.
	Tools []ToolIcon
	// Container, if not nil and not empty, shows the contents under the
	// name.
	Container *Container
	// Marks are small marks after the name.
	Marks Marks
	// Line, if not nil, is a line under the name with an item icon.
	Line *ItemLine
}

// Marks are small marks after the name (a drawer's lock and upgrades).
type Marks struct {
	Lock bool
	// Multiplier is a capacity upgrade: 2, 4 or 8 shows ×2, ×4 or ×8.
	Multiplier int
	Void       bool
}

// ItemLine is a line of an item's name, its icon, then Text (up to
// textWidth bytes; start it with a colour code, as text that looks like a
// number shows nothing), such as "흙 [icon] §f×3".
type ItemLine struct {
	Item item.Stack
	Name string
	Text string
	// Gauge, if ShowGauge, is a ring after the text filled in Gauge tenths.
	Gauge     int
	ShowGauge bool
}

// panelPrefix starts a block panel title ("A": blocks, "B": entities).
const panelPrefix = "_waila:A:"

// anchor places the panel at the top middle of the screen.
const anchor = "__waila:anchor.top_middle__"

// Format returns the title and subtitle that show info. The title is the
// prefix, a 10-character tool field ("aa,a;zz,z:"), the text, and tokens the
// pack removes (also the container layout); the subtitle is
// "i", the icon item's field (runtime ID × 65536 left padded with 'x' to 11
// characters; the pack draws it like the form item icons), for a container
// (or a tail) per panel slot the item's field and the count in 4
// characters, and the tail of marks and the item line.
func Format(info Info) (titleText, subtitle string) {
	tools := [2]ToolIcon{{ToolNone, MarkNone}, {ToolNone, MarkNone}}
	copy(tools[:], info.Tools)
	var b strings.Builder
	b.WriteString(panelPrefix)
	for i, t := range tools {
		if t.Tool == "" {
			t = ToolIcon{ToolNone, MarkNone}
		}
		if t.Mark == 0 {
			t.Mark = MarkNone
		}
		sep := byte(';')
		if i == 1 {
			sep = ':'
		}
		fmt.Fprintf(&b, "%s,%c%c", t.Tool, t.Mark, sep)
	}
	if info.NameKey != "" {
		b.WriteString("§f" + keyMark + info.NameKey + keyMark + "§r")
	} else {
		b.WriteString("§f" + clean(info.Name) + "§r")
	}
	for _, l := range info.Lines {
		b.WriteString("\n§7" + clean(l) + "§r")
	}
	if info.Source != "" {
		b.WriteString("\n§9§o" + clean(info.Source) + "§r")
	}
	b.WriteString(anchor)
	// The subtitle: "i" (keeps it text: the pack also reads it as a string
	// for block states, and a bare number broke that, a grey box) and the
	// icon field; for a container or a tail, each panel slot's item and
	// count fields; then the tail (tail.go).
	var sub strings.Builder
	sub.WriteString("i")
	var icon item.Stack
	if info.Item != nil {
		icon = item.NewStack(info.Item, 1)
	}
	sub.WriteString(itemField(icon))
	c := info.Container
	hasContainer := c != nil && !c.empty()
	hasTail := info.Marks != (Marks{}) || info.Line != nil
	if hasContainer || hasTail {
		var slots [panelSlots]item.Stack
		if hasContainer {
			slots = c.slots()
			b.WriteString(c.token())
		}
		for _, it := range slots {
			sub.WriteString(itemField(it))
			sub.WriteString(countField(it.Count()))
		}
	}
	if hasTail {
		if info.Line != nil {
			b.WriteString(lineToken)
		}
		sub.WriteString(tail(info.Marks, info.Line))
	}
	return b.String(), sub.String()
}

// keyMark surrounds a translation key in a title text (Format); Show sends
// such a title as raw JSON so the client translates it.
const keyMark = "\x01"

// rawText turns a title text with marked translation keys into the JSON the
// client translates, and reports whether there was any key.
func rawText(t string) (string, bool) {
	if !strings.Contains(t, keyMark) {
		return t, false
	}
	type part struct {
		Text      string `json:"text,omitempty"`
		Translate string `json:"translate,omitempty"`
	}
	var parts []part
	for i, seg := range strings.Split(t, keyMark) {
		switch {
		case seg == "":
		case i%2 == 1:
			parts = append(parts, part{Translate: seg})
		default:
			parts = append(parts, part{Text: seg})
		}
	}
	data, _ := json.Marshal(struct {
		Rawtext []part `json:"rawtext"`
	}{parts})
	return string(data), true
}

// clean removes what the pack would cut out of the text.
func clean(s string) string {
	for _, cut := range []string{"tile.", "entity.", ".name", "__waila:"} {
		s = strings.ReplaceAll(s, cut, "")
	}
	return s
}

// stay is how long a shown panel stays without being sent again; Show
// sends it again before it runs out.
const (
	stay    = 60 * time.Second
	refresh = 30 * time.Second
)

type state struct {
	shown, sub string
	sentAt     time.Time
	// pausedUntil is when an ordinary title sent with SendTitle is over.
	pausedUntil time.Time
}

var players = struct {
	sync.Mutex
	m map[uuid.UUID]*state
}{m: map[uuid.UUID]*state{}}

func stateOf(p *player.Player) *state {
	st, ok := players.m[p.UUID()]
	if !ok {
		st = &state{}
		players.m[p.UUID()] = st
	}
	return st
}

// Show shows info to p, unless the same panel is up or an ordinary title
// is showing. Call it whenever the target may have changed; it only sends
// what changed.
func Show(p *player.Player, info Info) {
	t, sub := Format(info)
	players.Lock()
	st := stateOf(p)
	now := time.Now()
	if now.Before(st.pausedUntil) || (st.shown == t && st.sub == sub && now.Sub(st.sentAt) < refresh) {
		players.Unlock()
		return
	}
	st.shown, st.sub, st.sentAt = t, sub, now
	players.Unlock()
	text, isJSON := rawText(t)
	tt := title.New(text).WithSubtitle(sub).WithFadeInDuration(0).WithFadeOutDuration(0).WithDuration(stay)
	// The subtitle (the panel's icons) must arrive with its own title.
	dfhook.SendTitle(p, tt, isJSON)
}

// Clear hides the panel of p if it is up.
func Clear(p *player.Player) {
	players.Lock()
	st := stateOf(p)
	if st.shown == "" || time.Now().Before(st.pausedUntil) {
		players.Unlock()
		return
	}
	st.shown, st.sub = "", ""
	players.Unlock()
	// A blank vanilla title, gone at once.
	p.SendTitle(title.New(" ").WithFadeInDuration(0).WithFadeOutDuration(0).WithDuration(0))
}

// SendTitle sends an ordinary title to p and keeps the panel from replacing
// it until it is over: a player sees one title at a time.
func SendTitle(p *player.Player, t title.Title) {
	players.Lock()
	st := stateOf(p)
	st.pausedUntil = time.Now().Add(t.FadeInDuration() + t.Duration() + t.FadeOutDuration())
	st.shown, st.sub = "", ""
	players.Unlock()
	dfhook.SendTitle(p, t, false)
}

// Forget drops what is kept about p (on quit).
func Forget(p *player.Player) {
	players.Lock()
	delete(players.m, p.UUID())
	players.Unlock()
}
