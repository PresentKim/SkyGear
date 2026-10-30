package main

import (
	"os"
	"regexp"
	"testing"
)

// TestChangelogHasVersion checks that the newest entry of CHANGELOG.md is
// the release Version names.
func TestChangelogHasVersion(t *testing.T) {
	if !regexp.MustCompile(`^\d{2}\.(1[0-2]|[1-9])\.[1-9]\d*$`).MatchString(Version) {
		t.Fatalf("Version %q is not year.month.build", Version)
	}
	b, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^## (\S+)`).FindSubmatch(b)
	if m == nil {
		t.Fatal("CHANGELOG.md has no entry")
	}
	if string(m[1]) != Version {
		t.Fatalf("the newest CHANGELOG.md entry is %s, want %s", m[1], Version)
	}
}
