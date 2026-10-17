// Package textsearch matches search words against names the way a recipe
// viewer does: case does not matter, colour codes are ignored, and a Korean
// word may be written as its initial consonants ("ㄷㅅㅌ" finds "도시탈").
package textsearch

import (
	"strings"
	"unicode"
)

// initials are the initial consonants of Korean syllables, in Unicode order.
var initials = []rune("ㄱㄲㄴㄷㄸㄹㅁㅂㅃㅅㅆㅇㅈㅉㅊㅋㅌㅍㅎ")

const (
	syllableFirst = 0xAC00
	syllableLast  = 0xD7A3
	// syllablesPerInitial is the number of syllables sharing an initial
	// (21 vowels × 28 finals).
	syllablesPerInitial = 588
)

// initialOf returns the initial consonant of a Korean syllable.
func initialOf(r rune) (rune, bool) {
	if r < syllableFirst || r > syllableLast {
		return 0, false
	}
	return initials[(r-syllableFirst)/syllablesPerInitial], true
}

func isInitial(r rune) bool {
	for _, c := range initials {
		if c == r {
			return true
		}
	}
	return false
}

// clean lower-cases s, drops colour codes ("§" and the character after it)
// and white space.
func clean(s string) []rune {
	var out []rune
	skip := false
	for _, r := range s {
		switch {
		case skip:
			skip = false
		case r == '§':
			skip = true
		case unicode.IsSpace(r):
		default:
			out = append(out, unicode.ToLower(r))
		}
	}
	return out
}

// Match reports whether name matches query. Every word of the query (split at
// white space) must occur in the name; an initial consonant in a word matches
// any syllable starting with it. An empty query matches everything.
func Match(query, name string) bool {
	n := clean(name)
	for _, word := range strings.Fields(query) {
		if !contains(n, clean(word)) {
			return false
		}
	}
	return true
}

func contains(name, word []rune) bool {
	if len(word) == 0 {
		return true
	}
	for i := 0; i+len(word) <= len(name); i++ {
		ok := true
		for j, w := range word {
			c := name[i+j]
			if c == w {
				continue
			}
			if ini, syl := initialOf(c); syl && isInitial(w) && ini == w {
				continue
			}
			ok = false
			break
		}
		if ok {
			return true
		}
	}
	return false
}
