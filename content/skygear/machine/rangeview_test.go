package machine

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

func TestRangeSettingIsCappedByTier(t *testing.T) {
	st := New(KindFeller, "", "").State
	if got := st.Area(KindFeller); got != (Area{fellerReach, fellerReach, fellerDown, fellerUp}) {
		t.Fatalf("unset range: %+v", got)
	}
	st.RangeX, st.RangeY, st.RangeZ = 2, 4, 99
	if got := st.Area(KindFeller); got != (Area{2, fellerReach, fellerDown, 4}) {
		t.Errorf("set range: %+v", got)
	}
}

func TestPeekShowsRangeForAMoment(t *testing.T) {
	m := New(KindHarvester, "", "")
	m.State.curTick = 100
	PeekRange(nil, cube.Pos{}, m)
	if !m.State.rangeShown() || m.State.rangeToggled() {
		t.Fatal("a peek does not show the range on its own")
	}
	m.State.curTick += peekTicks + 1
	if m.State.rangeShown() {
		t.Error("the peek outlasts its ticks")
	}
	crusher := New(KindCrusher, "", "")
	PeekRange(nil, cube.Pos{}, crusher)
	if crusher.State.peekUntil != 0 {
		t.Error("a crusher peeks a range")
	}
}

func TestRangeSettingIsSaved(t *testing.T) {
	m := New(KindVacuum, "", "")
	m.State.RangeX, m.State.RangeY, m.State.RangeZ = 3, 1, 2
	back := m.DecodeNBT(m.EncodeNBT()).(Machine).State
	if back.RangeX != 3 || back.RangeY != 1 || back.RangeZ != 2 {
		t.Errorf("loaded range %d %d %d", back.RangeX, back.RangeY, back.RangeZ)
	}
}

func TestRangeBoxFollowsKind(t *testing.T) {
	st := New(KindVacuum, "", "").State
	st.RangeX, st.RangeY, st.RangeZ = 2, 2, 4
	lo, hi := st.rangeBox(KindVacuum, cube.Pos{0, 10, 0})
	if lo != (mgl64.Vec3{-2, 8, -4}) || hi != (mgl64.Vec3{3, 13, 5}) {
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
