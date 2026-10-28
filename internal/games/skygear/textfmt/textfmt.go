// Package textfmt formats numbers and durations for players and names the
// resource pack font glyphs used in texts.
package textfmt

import (
	"fmt"
	"strconv"
	"time"

	"github.com/presentkim/skygear/common/i18n"
	_ "github.com/presentkim/skygear/internal/locales" // this server's texts
)

// Rate writes a rate without trailing zeros.
func Rate(r float64) string { return strconv.FormatFloat(r, 'f', -1, 64) }

// Permille writes a per mille chance as a percent ("1.2", "40").
func Permille(p int) string {
	if p%10 == 0 {
		return fmt.Sprint(p / 10)
	}
	return fmt.Sprintf("%.1f", float64(p)/10)
}

// Duration formats a duration in minutes and seconds.
func Duration(lang string, d time.Duration) string {
	m, sec := int(d/time.Minute), int(d%time.Minute/time.Second)
	if sec == 0 {
		return fmt.Sprintf(i18n.T(lang, "crop.time.min"), m)
	}
	return fmt.Sprintf(i18n.T(lang, "crop.time.minSec"), m, sec)
}

// Span writes seconds as "45초", "3분", "2시간 5분".
func Span(lang string, secs int) string {
	switch {
	case secs < 60:
		return fmt.Sprintf(i18n.T(lang, "monitor.span.sec"), secs)
	case secs < 3600:
		return fmt.Sprintf(i18n.T(lang, "monitor.span.min"), secs/60)
	}
	return fmt.Sprintf(i18n.T(lang, "monitor.span.hour"), secs/3600, secs%3600/60)
}

// PlayTime formats the total instance time in whole minutes in lang.
func PlayTime(lang string, d time.Duration) string {
	m := int(d / time.Minute)
	if m < 60 {
		return fmt.Sprintf(i18n.T(lang, "sidebar.time.min"), m)
	}
	return fmt.Sprintf(i18n.T(lang, "sidebar.time.hourMin"), m/60, m%60)
}

// PerMin writes a per-minute amount: whole numbers from ten up, one
// decimal below.
func PerMin(v float64) string {
	if v >= 10 {
		return Int(int64(v + 0.5))
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// Int formats a number with thousands separators, e.g. 12,400.
func Int(v int64) string {
	s := fmt.Sprint(v)
	neg := v < 0
	if neg {
		s = s[1:]
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
