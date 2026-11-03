package core

import "testing"

func BenchmarkPartOf(b *testing.B) {
	p := NewPart[int](nil)
	s := New(Config{})
	for b.Loop() {
		*p.Of(s)++
	}
}
