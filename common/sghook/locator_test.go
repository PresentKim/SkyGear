package sghook

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestStartLocatorRulesTurnTheBarOn(t *testing.T) {
	got := startLocatorRules([]protocol.GameRule{{Name: "naturalregeneration", Value: false}, {Name: "locatorBar", Value: false}})
	on := 0
	for _, r := range got {
		switch r.Name {
		case "naturalregeneration":
			if r.Value != false {
				t.Fatalf("other rule changed: %+v", r)
			}
		case "locatorbar", "locatorBar":
			if r.Value != true {
				t.Fatalf("%s is off", r.Name)
			}
			on++
		}
	}
	if on != 2 || len(got) != 3 {
		t.Fatalf("rules %+v", got)
	}
}
