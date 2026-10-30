package perf

import (
	"testing"
)

func TestToggle(t *testing.T) {
	if Shown("a") || !Toggle("a") || !Shown("a") || Toggle("a") || Shown("a") {
		t.Fatal("toggle")
	}
}
