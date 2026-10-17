package textsearch

import "testing"

func TestMatch(t *testing.T) {
	for _, tc := range []struct {
		query, name string
		want        bool
	}{
		{"", "아무거나", true},
		{"철", "§f철 주괴", true},
		{"주괴", "§f철 주괴", true},
		{"철 주괴", "철주괴", true},
		{"ㅊ", "철 주괴", true},
		{"ㅊㅈㄱ", "철 주괴", true},
		{"ㅈㅈㄱ", "철 주괴", false},
		{"ㅊ주", "철 주괴", true},
		{"iron", "Iron Ingot", true},
		{"ingot iron", "Iron Ingot", true},
		{"gold", "Iron Ingot", false},
		{"§f", "§f철", true},
		{"구리", "철 주괴", false},
		// A syllable in the query is not an initial: it must be written out.
		{"차", "철 주괴", false},
	} {
		if got := Match(tc.query, tc.name); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.query, tc.name, got, tc.want)
		}
	}
}
