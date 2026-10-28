package skygear

import (
	"os"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/internal/games/skygear/naming"
	"github.com/presentkim/skygear/internal/games/skygear/sgitem"
	"github.com/presentkim/skygear/internal/games/skygear/wintext"
)

func TestWindowItemTextsInDefaultLanguage(t *testing.T) {
	st := newMachine(KindCrusher, "", "").st
	st.tier = 2
	info := st.infoItem()
	name, lore, ok := wintext.Localize(i18n.Default, info)
	if !ok || name != info.CustomName() || strings.Join(lore, "\n") != strings.Join(info.Lore(), "\n") {
		t.Fatalf("localized %q %q, item %q %q", name, lore, info.CustomName(), info.Lore())
	}
	if !strings.Contains(name, tierName(i18n.Default, 2)) || !strings.Contains(lore[len(lore)-1], tierName(i18n.Default, 2)) {
		t.Errorf("tier missing in %q %q", name, lore)
	}
	if _, _, ok := wintext.Localize(i18n.Default, item.NewStack(item.Apple{}, 1)); ok {
		t.Error("localized an item without window texts")
	}
}

func TestWindowItemNamesInPlayerLanguage(t *testing.T) {
	saved := naming.PackNames
	t.Cleanup(func() { naming.PackNames = saved })
	if err := naming.LoadNames(os.DirFS(packDir)); err != nil {
		t.Fatal(err)
	}
	essence := item.NewStack(sgitem.ResearchEssence{Kind: sgitem.EssenceEngineering}, 1)
	st := wintext.With(item.NewStack(item.Paper{}, 1), wintext.Text{wintext.Raw("§r"), wintext.Item(essence)}, wintext.Text{wintext.Key("machine.energy.gauge", 5, 10)})
	name, lore, _ := wintext.Localize("en", st)
	if name != "§r"+naming.ItemName("en", essence) || name == st.CustomName() {
		t.Errorf("english name %q, korean %q", name, st.CustomName())
	}
	if len(lore) != 1 || !strings.Contains(lore[0], "5") || !strings.Contains(lore[0], "10") {
		t.Errorf("lore %q", lore)
	}
}
