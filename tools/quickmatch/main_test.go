package main

import (
	"bytes"
	"os"
	"testing"
)

// The committed sg_quick_match.json is what the command writes.
func TestOutputIsGenerated(t *testing.T) {
	want, err := build()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../" + out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale: run go run ./tools/quickmatch from the repository root", out)
	}
}
