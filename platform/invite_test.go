package platform

import (
	"strings"
	"testing"

	"github.com/presentkim/skygear/platform/db"
	"github.com/presentkim/skygear/platform/instance"
)

func TestInviteCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		c := newCode()
		if len(c) != 6 || strings.Trim(c, codeAlphabet) != "" {
			t.Fatalf("bad code %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 190 {
		t.Fatalf("codes repeat too often: %d unique of 200", len(seen))
	}
}

// TestInviteCodes: turning the code on again gives a new one, and the old
// one stops working; turning it off drops it.
func TestInviteCodes(t *testing.T) {
	var c inviteCodes
	inst := &instance.Instance{Save: db.Save{ID: 7}}
	first := c.enable(inst)
	if id, ok := c.save(first); !ok || id != 7 {
		t.Fatalf("code %q -> %d %v", first, id, ok)
	}
	second := c.enable(inst)
	if second == first {
		t.Fatal("re-enabling kept the code")
	}
	if _, ok := c.save(first); ok {
		t.Fatal("old code still works")
	}
	if l, ok := c.of(7); !ok || l.code != second || l.inst != inst {
		t.Fatalf("of = %+v %v", l, ok)
	}
	c.disable(7)
	if _, ok := c.save(second); ok {
		t.Fatal("disabled code still works")
	}
	if _, ok := c.of(7); ok {
		t.Fatal("disabled save keeps a code")
	}
}

// A zero inviteCodes value must work without any enable.
func TestInviteCodesZeroValue(t *testing.T) {
	var c inviteCodes
	c.disable(1)
	if _, ok := c.save("ABCDEF"); ok {
		t.Fatal("zero value has codes")
	}
}

// Codes of a host are dropped when it leaves, so no closed instance is kept.
func TestInviteCodesDropHost(t *testing.T) {
	var c inviteCodes
	inst := &instance.Instance{Save: db.Save{ID: 3}}
	code := c.enable(inst)
	c.dropHost(inst.Host())
	if _, ok := c.save(code); ok {
		t.Fatal("code kept after the host left")
	}
}
