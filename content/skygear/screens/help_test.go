package screens

import (
	"testing"

	"github.com/presentkim/skygear/common/i18n"
)

// TestHelpTopics: every help topic is in one tab, every tab has topics, and
// the texts of the tabs and topics are in the catalog.
func TestHelpTopics(t *testing.T) {
	seen := map[string]bool{}
	for _, h := range HelpTopics {
		if seen[h.id] {
			t.Errorf("help topic %q listed twice", h.id)
		}
		seen[h.id] = true
		if h.tab < 0 || h.tab >= HelpTabCount {
			t.Errorf("help topic %q: no tab", h.id)
		}
		for _, k := range []string{"help." + h.id + ".title", "help." + h.id + ".text"} {
			if !i18n.Has(k) {
				t.Errorf("help topic %q: no %s", h.id, k)
			}
		}
	}
	for tab := HelpTab(0); tab < HelpTabCount; tab++ {
		if len(HelpTopicsOf(tab)) == 0 {
			t.Errorf("help tab %d has no topics", tab)
		}
		if !i18n.Has(HelpTabKeys[tab]) {
			t.Errorf("help tab %d: no %s", tab, HelpTabKeys[tab])
		}
	}
}
