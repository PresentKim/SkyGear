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

// TestLoadConfigIgnoresRetiredTables: a platform.toml written when it had an
// [Experimental] table still loads, and the other keys in it are kept.
func TestLoadConfigIgnoresRetiredTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "platform.toml")
	old := "[Instance]\n  LoadRadius = 7\n\n[Experimental]\n  VoxelShapes = true\n  BlockActions = true\n  LocatorBar = true\n  CameraSpline = true\n  RecipeUnlock = \"book\"\n"
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if c.Instance.LoadRadius != 7 {
		t.Fatalf("LoadRadius = %d, want 7", c.Instance.LoadRadius)
	}
}
