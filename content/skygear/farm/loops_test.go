package farm

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestKeyPosCache(t *testing.T) {
	for _, p := range []cube.Pos{{1, 2, 3}, {-5, 64, 900}, {0, 0, 0}} {
		k := CropKey(p)
		for range 2 {
			got, ok := keyPos(k)
			if !ok || got != p {
				t.Fatalf("keyPos(%q) = %v %v", k, got, ok)
			}
		}
	}
	if _, ok := keyPos("1,2"); ok {
		t.Fatal("bad key parsed")
	}
	if _, ok := keyPos("a,b,c"); ok {
		t.Fatal("bad key parsed")
	}
}

func TestPlantCheckSchedule(t *testing.T) {
	r := &PlantRecord{Kind: "x", ReadyAt: 100 * time.Second}
	if !plantCheckDue(r, 0) {
		t.Fatal("new record must be checked once")
	}
	scheduleCheck(r, 10*time.Second)
	if plantCheckDue(r, 20*time.Second) {
		t.Fatal("not due before the ripening or sweep")
	}
	if !plantCheckDue(r, 10*time.Second+plantSweep) {
		t.Fatal("sweep must re-check a plant that may be gone")
	}
	// ripens before the sweep: checked exactly at ReadyAt
	r.ReadyAt = 30 * time.Second
	scheduleCheck(r, 10*time.Second)
	if plantCheckDue(r, 29*time.Second) || !plantCheckDue(r, 30*time.Second) {
		t.Fatal("ripe plant must be checked at ReadyAt")
	}
}

func TestBloomedPlantNotCheckedEachSecond(t *testing.T) {
	r := &PlantRecord{ReadyAt: 5 * time.Second, bloomed: true}
	scheduleCheck(r, 10*time.Second)
	if plantCheckDue(r, 11*time.Second) {
		t.Fatal("a bloomed plant was checked again at once")
	}
	r.bloomed = false // a harvest resets it
	if !plantCheckDue(r, 11*time.Second) {
		t.Fatal("a plant past ReadyAt must be checked")
	}
}
