package i18n

import (
	"strconv"
	"strings"
	"time"
)

// A client span is a piece of a text that the Ore UI page puts in the
// client's own language (packs/oreui/hbui/oui-ddui.js localize): a
// translation key the client knows (the vanilla and the pack's texts/*.lang),
// a date or the time since a moment. It carries the server's text in the
// player's language for the screens that cannot translate it (the JSON UI
// forms), Plain. The marks are Unicode noncharacters, which no font or text
// uses:
//
//	U+FDD0 <kind>:<value> U+FDD1 <fallback> U+FDD2
const (
	spanOpen  = "﷐"
	spanSep   = "﷑"
	spanClose = "﷒"
)

func span(kind, value, fallback string) string {
	return spanOpen + kind + ":" + value + spanSep + fallback + spanClose
}

// ClientKey is fallback as a span the client shows as its text of the
// translation key key ("item.diamond.name"). An empty key is fallback.
func ClientKey(key, fallback string) string {
	if key == "" {
		return fallback
	}
	return span("k", key, fallback)
}

// ClientDate is fallback as a span the client shows as the date of t in its
// own format.
func ClientDate(t time.Time, fallback string) string {
	return span("d", strconv.FormatInt(t.Unix(), 10), fallback)
}

// ClientAgo is fallback as a span the client shows as the time since t
// ("1h ago").
func ClientAgo(t time.Time, fallback string) string {
	return span("a", strconv.FormatInt(t.Unix(), 10), fallback)
}

// Plain is s with its client spans replaced by their fallback texts.
func Plain(s string) string {
	if !strings.Contains(s, spanOpen) {
		return s
	}
	var b strings.Builder
	for {
		i := strings.Index(s, spanOpen)
		if i < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i])
		s = s[i+len(spanOpen):]
		if j := strings.Index(s, spanSep); j >= 0 {
			s = s[j+len(spanSep):]
		}
	}
	return strings.ReplaceAll(b.String(), spanClose, "")
}

// PlainBytes is Plain for a JSON text (a form): the marks and keys need no
// escaping, so the result stays valid JSON.
func PlainBytes(data []byte) []byte {
	if !strings.Contains(string(data), spanOpen) {
		return data
	}
	return []byte(Plain(string(data)))
}
