package waila

import (
	"math/bits"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
)

// The tail of the subtitle, after the panel slots: the item line's icon
// field, four flags (lock 'L', capacity upgrades 'a'+n, void 'V', the line's
// gauge 'a'+tenths; '-' for none), the item line's text in textWidth characters
// padded with '~', and the item line's name (the rest, the only part that is
// not ASCII: the pack cuts by position). The pack reads it in
// tile.r4ui df_marks and df_item_line.
const (
	tailStart  = 1 + iconWidth + panelSlots*(iconWidth+countWidth)
	flagsStart = tailStart + iconWidth
	textStart  = flagsStart + 4
	textWidth  = 24
	nameStart  = textStart + textWidth
)

// lineToken in the title shows the item line.
const lineToken = "__r4ui:df.line__"

// tail is the subtitle tail for marks and line.
func tail(m Marks, line *ItemLine) string {
	var b strings.Builder
	lineIcon, text, name := itemField(item.Stack{}), "", ""
	if line != nil {
		lineIcon, text, name = itemField(line.Item), ascii(line.Text, textWidth), line.Name
	}
	b.WriteString(lineIcon)
	b.WriteByte(flag(m.Lock, 'L'))
	// Letters, not digits: a one-digit flag reads as a number in the pack
	// and never equals the digit text (the ×2 mark did not show).
	mult := byte('a')
	if m.Multiplier > 1 {
		mult = byte('a' + min(bits.Len(uint(m.Multiplier))-1, 9))
	}
	b.WriteByte(mult)
	b.WriteByte(flag(m.Void, 'V'))
	gauge := byte('-')
	if line != nil && line.ShowGauge {
		gauge = byte('a' + min(max(line.Gauge, 0), 10))
	}
	b.WriteByte(gauge)
	b.WriteString(text + strings.Repeat("~", textWidth-len(text)))
	b.WriteString(name)
	return b.String()
}

func flag(on bool, c byte) byte {
	if on {
		return c
	}
	return '-'
}

// ascii keeps the printable ASCII of s but '~', up to n characters.
func ascii(s string, n int) string {
	var b strings.Builder
	for _, r := range s {
		if r >= ' ' && r < '~' && b.Len() < n {
			b.WriteRune(r)
		}
	}
	return b.String()
}
