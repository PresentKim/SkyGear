package ui

import (
	"testing"
	"time"

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

// TestResent: a form sent right after a button press is a form sent again
// (the window does not slide); one sent later, or one with no press before it,
// is a first form and slides up.
func TestResent(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	if resentAt(id, now) {
		t.Error("no press before it")
	}
	submittedAt(id, now)
	if !resentAt(id, now.Add(50*time.Millisecond)) {
		t.Error("sent right after a press")
	}
	if resentAt(id, now.Add(60*time.Millisecond)) {
		t.Error("a press counts for one form")
	}
	submittedAt(id, now)
	if resentAt(id, now.Add(2*time.Second)) {
		t.Error("sent long after the press")
	}
}
