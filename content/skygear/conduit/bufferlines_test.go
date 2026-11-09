package conduit

import (
	"reflect"
	"testing"
)

func TestBufferLinesKo(t *testing.T) {
	got := bufferLines("ko", bufferValues{300, 600, true}, bufferValues{0, 2000, true})
	want := []string{"§7망 저장량 300 / 600 RE", "§7망 저장량 0 / 2000 mB"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBufferLinesEn(t *testing.T) {
	got := bufferLines("en", bufferValues{1, 2, true}, bufferValues{})
	want := []string{"§7Network buffer 1 / 2 RE"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBufferLinesEmptyBundle(t *testing.T) {
	if got := bufferLines("ko", bufferValues{}, bufferValues{}); got != nil {
		t.Fatalf("got %q", got)
	}
}
