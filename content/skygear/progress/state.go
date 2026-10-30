package progress

import (
	"github.com/presentkim/skygear/content/skygear/core"
)

// State is the progress a save keeps: its fields are part of the save's
// state document (core.Saved).
type State struct {
	// Researched are the done researches.
	Researched map[string]bool `json:"researched,omitempty"`
	// ResearchPaid counts the items delivered for researches not done yet,
	// by research and item key (old saves only: research is paid whole).
	ResearchPaid map[string]map[string]int `json:"research_paid,omitempty"`
	// Advancements are the done advancements.
	Advancements map[string]bool `json:"advancements,omitempty"`
	// Tracked is the advancement the quest panel follows instead of the
	// first open one.
	Tracked string `json:"tracked,omitempty"`
	// Unseen are the done advancements nobody has looked at on the
	// advancement screen yet, in the order they were done (NEW tag).
	Unseen []string `json:"unseen_advancements,omitempty"`
}

var saved = core.NewSaved[State](nil)

// StateOf is the progress of the save of s.
func StateOf(s *core.Session) *State { return saved.Of(s) }

// Local is the progress of a save that is not saved.
type Local struct {
	// Copies are test copies of researches for the research screen and
	// CopiesDone the ones marked done (Research.IsCopy).
	Copies     []*Research
	CopiesDone map[string]bool
	// RecipesFree opens every recipe until the save closes (/sgadmin
	// research free).
	RecipesFree bool
}

var local = core.NewPart[Local](nil)

// LocalOf is the unsaved progress of the save of s.
func LocalOf(s *core.Session) *Local { return local.Of(s) }
