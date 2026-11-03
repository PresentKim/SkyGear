package storagenet

import "testing"

// TestPlanOK asserts that a plan is OK exactly when nothing is missing.
func TestPlanOK(t *testing.T) {
	if !(&Plan{}).OK() {
		t.Fatal("empty plan should be OK")
	}
	if (&Plan{Missing: map[Key]int64{"a": 1}}).OK() {
		t.Fatal("plan with missing keys should not be OK")
	}
}
