package machine

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

func TestRangeSettingIsCappedByTier(t *testing.T) {
	st := New(KindFeller, "", "").State
	if st.Reach(KindFeller) != fellerReach {
		t.Fatalf("unset range: %d", st.Reach(KindFeller))
	}
	st.Range = 2
	if st.Reach(KindFeller) != 2 {
		t.Errorf("set range: %d", st.Reach(KindFeller))
	}
	st.Range = 99
	if st.Reach(KindFeller) != fellerReach {
		t.Errorf("range over the tier: %d", st.Reach(KindFeller))
	}
}

func TestRangeBoxFollowsKind(t *testing.T) {
	st := New(KindVacuum, "", "").State
	st.Range = 2
	lo, hi := st.rangeBox(KindVacuum, cube.Pos{0, 10, 0})
	if lo != (mgl64.Vec3{-2, 10 - vacuumDown, -2}) || hi != (mgl64.Vec3{3, 11 + vacuumUp, 3}) {
		t.Errorf("box %v %v", lo, hi)
	}
	if _, ok := New(KindCrusher, "", "").State.rangeKind(); ok {
		t.Error("the crusher has a range")
	}
}

func TestBoxVariantPacksSize(t *testing.T) {
	v := boxVariant(mgl64.Vec3{-16, 0, -16}, mgl64.Vec3{17, 15, 17}, 3)
	if v%128 != 33 || v/128%128 != 15 || v/16384%128 != 33 || v/2097152 != 3 {
		t.Errorf("variant %d", v)
	}
}

func TestMinerFacingIsSaved(t *testing.T) {
	m := New(KindMiner, "", "")
	if m.State.Facing != cube.FaceDown {
		t.Fatalf("new miner faces %v", m.State.Facing)
	}
	m.State.Facing = cube.FaceNorth
	back := m.DecodeNBT(m.EncodeNBT()).(Machine)
	if back.State.Facing != cube.FaceNorth {
		t.Errorf("loaded miner faces %v", back.State.Facing)
	}
	if back := m.DecodeNBT(map[string]any{}).(Machine); back.State.Facing != cube.FaceDown {
		t.Errorf("miner saved without a facing faces %v", back.State.Facing)
	}
}
