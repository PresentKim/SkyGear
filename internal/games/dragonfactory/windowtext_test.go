package dragonfactory

import (
	"os"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/dragonfactory/common/i18n"
)

func TestWindowItemTextsInDefaultLanguage(t *testing.T) {
	st := newMachine(KindCrusher, "", "").st
	st.tier = 2
	info := st.infoItem()
	name, lore, ok := localizeWindowItem(i18n.Default, info)
	if !ok || name != info.CustomName() || strings.Join(lore, "\n") != strings.Join(info.Lore(), "\n") {
		t.Fatalf("localized %q %q, item %q %q", name, lore, info.CustomName(), info.Lore())
	}
	if !strings.Contains(name, tierName(i18n.Default, 2)) || !strings.Contains(lore[len(lore)-1], tierName(i18n.Default, 2)) {
		t.Errorf("tier missing in %q %q", name, lore)
	}
	if _, _, ok := localizeWindowItem(i18n.Default, item.NewStack(item.Apple{}, 1)); ok {
		t.Error("localized an item without window texts")
	}
}

func TestWindowItemNamesInPlayerLanguage(t *testing.T) {
	saved := packNames
	t.Cleanup(func() { packNames = saved })
	if err := LoadNames(os.DirFS(packDir)); err != nil {
		t.Fatal(err)
	}
	essence := item.NewStack(ResearchEssence{Kind: EssenceEngineering}, 1)
	st := withWindowText(item.NewStack(item.Paper{}, 1), wtext{wraw("§r"), witem(essence)}, wtext{wkey("machine.energy.gauge", 5, 10)})
	name, lore, _ := localizeWindowItem("en", st)
	if name != "§r"+itemName("en", essence) || name == st.CustomName() {
		t.Errorf("english name %q, korean %q", name, st.CustomName())
	}
	if len(lore) != 1 || !strings.Contains(lore[0], "5") || !strings.Contains(lore[0], "10") {
		t.Errorf("lore %q", lore)
	}
}
