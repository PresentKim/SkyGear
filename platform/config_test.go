package platform

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/presentkim/skygear/platform/instance"
)

func TestOperatorByXUIDOrName(t *testing.T) {
	var c Config
	c.Admin.Operators = []string{"2535400000000001", "Alice"}
	if !c.IsOperator("Whoever", "2535400000000001") || c.OperatorByNameOnly("Whoever", "2535400000000001") {
		t.Fatal("an XUID entry matches the XUID, not the name")
	}
	if !c.IsOperator("alice", "42") || !c.OperatorByNameOnly("alice", "42") {
		t.Fatal("a name entry still works and is reported")
	}
	if c.IsOperator("Bob", "42") || c.IsOperator("2535400000000001", "") {
		t.Fatal("an XUID is not matched as a name")
	}
}

func TestDefaultConfigPerf(t *testing.T) {
	c := DefaultConfig()
	if c.Debug.Pprof != "" || c.Runtime.MemoryLimitMB != 0 || c.Runtime.GCPercent != 0 ||
		c.Instance.StateFlushSeconds != 10 || c.ResourcePack.Encrypt || c.Network.CompressionLevel != 3 {
		t.Fatalf("unexpected perf defaults: %+v", c)
	}
}

func TestStateFlushIntervalDefault(t *testing.T) {
	m := instance.NewManager(slog.Default(), nil, instance.Config{}, instance.Lobby{}, nil)
	if got := m.StateFlushInterval(); got != 10*time.Second {
		t.Fatalf("StateFlushInterval = %v, want 10s", got)
	}
}

func TestRecipeUnlockMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platform.toml")
	for mode, ok := range map[string]bool{"": true, "toast": true, "book": true, "all": false} {
		if err := os.WriteFile(path, []byte("[Experimental]\n  RecipeUnlock = \""+mode+"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); (err == nil) != ok {
			t.Errorf("mode %q: err %v", mode, err)
		}
	}
}
