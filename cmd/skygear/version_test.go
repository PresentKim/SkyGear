package main

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var versionPattern = regexp.MustCompile(`^(\d{2})\.(1[0-2]|[1-9])\.([1-9]\d*)$`)

func versionKey(v string) (int, bool) {
	m := versionPattern.FindStringSubmatch(v)
	if m == nil {
		return 0, false
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	build, _ := strconv.Atoi(m[3])
	return (year*100+month)*10000 + build, true
}

func TestChangeLogHasVersion(t *testing.T) {
	want, ok := versionKey(Version)
	if !ok {
		t.Fatalf("Version %q is not year.month.build", Version)
	}
	entries, err := os.ReadDir("../../change_logs")
	if err != nil {
		t.Fatal(err)
	}
	newest := ""
	newestKey := 0
	for _, e := range entries {
		v, isLog := strings.CutSuffix(e.Name(), ".md")
		key, isVersion := versionKey(v)
		if !isLog || !isVersion {
			t.Errorf("change_logs/%s is not named <year.month.build>.md", e.Name())
			continue
		}
		if key > newestKey {
			newest, newestKey = v, key
		}
	}
	if newestKey != want {
		t.Fatalf("the newest change log is %q, want change_logs/%s.md", newest, Version)
	}
	b, err := os.ReadFile("../../change_logs/" + Version + ".md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "# "+Version+" ") {
		t.Fatalf("change_logs/%s.md does not start with \"# %s (date)\"", Version, Version)
	}
}
