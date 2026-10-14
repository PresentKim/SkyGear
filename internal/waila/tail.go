package waila

import (
	"math/bits"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
)

// The tail of the subtitle, after the panel slots: the item line's icon
// field, four flags (lock 'L', capacity upgrades 'a'+n, void 'V', the line's
// gauge 'a'+tenths; '-' for none), the item line's text in textWidth bytes
// padded with '~', and the item line's name (the rest). The pack reads it
// in tile.json df_marks and df_item_line.
const (
	tailStart  = 1 + iconWidth + panelSlots*(iconWidth+countWidth)
	flagsStart = tailStart + iconWidth
	textStart  = flagsStart + 4
	textWidth  = 24
	nameStart  = textStart + textWidth
)

// lineToken in the title shows the item line.
const lineToken = "__waila:df.line__"

// tail is the subtitle tail for marks and line.
func tail(m Marks, line *ItemLine) string {
	var b strings.Builder
	lineIcon, text, name := itemField(item.Stack{}), "", ""
	if line != nil {
		lineIcon, text, name = itemField(line.Item), line.Text, line.Name
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
	b.WriteString(pad(text, textWidth))
	b.WriteString(name)
	return b.String()
}

func flag(on bool, c byte) byte {
	if on {
		return c
	}
	return '-'
}
