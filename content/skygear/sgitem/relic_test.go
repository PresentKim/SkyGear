package sgitem

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

// TestRelicStack: a relic carries no name of its own (the pack names it in
// each player's language), only its grade as stars; an old relic named in
// the server's language is made again with its grade.
func TestRelicStack(t *testing.T) {
	st := RelicStack(RelicCrossbow, 3)
	if st.CustomName() != "" || RelicGrade(st) != 3 || len(st.Lore()) != 1 {
		t.Fatalf("relic %q %v grade %d", st.CustomName(), st.Lore(), RelicGrade(st))
	}
	if !FixRelic(st).Equal(st) {
		t.Error("a new relic was made again")
	}
	old := item.NewStack(Relic{Kind: RelicCrossbow}, 1).WithValue(relicGradeKey, 4).WithCustomName("§r§d연사 석궁 §e★★★★").WithLore("가장 가까운 몬스터에게 화살을 쏩니다.")
	got := FixRelic(old)
	if got.CustomName() != "" || RelicGrade(got) != 4 || got.Lore()[0] != relicLore(4) {
		t.Errorf("an old relic became %q %v grade %d", got.CustomName(), got.Lore(), RelicGrade(got))
	}
}
