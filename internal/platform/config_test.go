package platform

import "testing"

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
