package skygear

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/presentkim/skygear/content/skygear/fairy"
	"github.com/presentkim/skygear/content/skygear/progress"
)

func TestGiftsComeFromDoneResearch(t *testing.T) {
	s := testSession(t)
	progress.StateOf(s.Session).Researched = map[string]bool{}
	var reach func(id string)
	reach = func(id string) {
		progress.StateOf(s.Session).Researched[id] = true
		r, _ := progress.ByID(id)
		for _, n := range r.Needs {
			reach(n)
		}
	}
	reach("generator")
	generator, _ := progress.ByID("generator")
	costs, rewards := 0, 0
	for range 1000 {
		gift := fairy.RollGift(s.Session)
		switch {
		case gift[0].Comparable(progress.Reward(generator)[0]):
			rewards++
		case gift[0].Comparable(fairy.HalfCost(generator)[0]):
			costs++
		}
	}
	if rewards == 0 || costs == 0 || rewards > costs {
		t.Errorf("%d reward gifts and %d cost gifts in 1000", rewards, costs)
	}
}

func TestGiftsSkipProjectsAndResearchDoneOutOfOrder(t *testing.T) {
	s := testSession(t)
	progress.StateOf(s.Session).Researched = map[string]bool{"project_dragon": true, "iron_tools": true, "stick": true}
	for range 500 {
		gift := fairy.RollGift(s.Session)
		for _, st := range gift {
			switch st.Item().(type) {
			case item.Diamond, item.NetheriteIngot, item.Emerald, item.IronIngot:
				t.Fatalf("gift %v from research done out of order", gift)
			}
		}
	}
	stick, _ := progress.ByID("stick")
	if !fairy.ReachedResearch(s.Session, stick) {
		t.Error("stick, done and needing nothing, is not reached")
	}
	project, _ := progress.ByID("project_dragon")
	if fairy.ReachedResearch(s.Session, project) {
		t.Error("project_dragon counted as reached without its needs")
	}
}

func TestDeeperResearchIsDeeper(t *testing.T) {
	stick, _ := progress.ByID("stick")
	generator, _ := progress.ByID("generator")
	if fairy.ResearchDepth(stick) != 0 || fairy.ResearchDepth(generator) <= fairy.ResearchDepth(stick) {
		t.Errorf("depths: stick %d, generator %d", fairy.ResearchDepth(stick), fairy.ResearchDepth(generator))
	}
}
