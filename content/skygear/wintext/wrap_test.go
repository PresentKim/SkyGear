package wintext

import (
	"reflect"
	"strings"
	"testing"
)

func TestWrapKeepsShortLines(t *testing.T) {
	in := []string{"§7short line"}
	if got := wrap(in); !reflect.DeepEqual(got, in) {
		t.Fatalf("got %q", got)
	}
}

func TestWrapBreaksLongLinesWithTheirCodes(t *testing.T) {
	line := "§7" + strings.Repeat("가나다라 ", 20)
	got := wrap([]string{line})
	if len(got) < 2 {
		t.Fatalf("got %d lines", len(got))
	}
	for _, l := range got {
		if !strings.HasPrefix(l, "§7") || textWidth(l) > loreWidth {
			t.Fatalf("line %q", l)
		}
	}
}
