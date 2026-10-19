package ui

import (
	"testing"

	"github.com/google/uuid"
)

func TestOnPanel(t *testing.T) {
	for in, want := range map[string]string{
		"§f이름 §7설명 §8희미 §e제목 §a좋음 §c나쁨": "§0이름 §8설명 §7희미 §1제목 §2좋음 §4나쁨",
		"§r§l굵게": "§r§l굵게",
		"":       "",
	} {
		if got := OnPanel(in); got != want {
			t.Errorf("OnPanel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestOnPanelLabel: texts drawn on the window are changed after their marker;
// texts on a button face or in a tooltip keep their codes.
func TestOnPanelLabel(t *testing.T) {
	if got := onPanelLabel(CellToken + "§f값"); got != CellToken+"§0값" {
		t.Errorf("cell %q", got)
	}
	if got := onPanelLabel(detailMarker + "§6이름\n§a상태"); got != detailMarker+"§4이름\n§2상태" {
		t.Errorf("detail %q", got)
	}
	for _, label := range []string{"§f버튼", TabToken + "§f탭", slotWToken + "l" + "§f~~~~~~~~~" + "§e툴팁", treeCellToken + "abs~~~~~~~~§e팁"} {
		if got := onPanelLabel(label); got != label {
			t.Errorf("%q changed to %q", label, got)
		}
	}
}

// TestResending: a form sent while a button callback of another form runs is a
// form sent again (the window does not slide); one sent outside a callback, or
// after it returned, is a new screen and slides up.
func TestResending(t *testing.T) {
	id := uuid.New()
	if resendingOf(id) {
		t.Error("no press before it")
	}
	enterPressOf(id)
	if !resendingOf(id) {
		t.Error("sent during the callback")
	}
	enterPressOf(id)
	leavePressOf(id)
	if !resendingOf(id) {
		t.Error("the outer callback still runs")
	}
	leavePressOf(id)
	if resendingOf(id) {
		t.Error("the callback finished")
	}
	leavePressOf(id) // an extra leave must not go negative
	if resendingOf(id) {
		t.Error("left behind by an extra leave")
	}
}
