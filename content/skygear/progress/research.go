package progress

import (
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/platform/account"
)

// Research is one node of the research tree.
type Research struct {
	ID string
	// Needs are the researches that must be done first.
	Needs []string
	// Cost are the items to deliver.
	Cost []Requirement
	// Unlocks are the item names (without "minecraft:") whose recipes the
	// research opens; every variant of a name opens together.
	Unlocks []string
	// Icon is the item shown for a research that opens no recipe (a project,
	// project.go).
	Icon string
}

// Title is the name of the research in lang. A test copy (isCopy) has its
// original's name and its number.
func (r *Research) Title(lang string) string {
	id, n, copied := strings.Cut(r.ID, "#")
	name := i18n.T(lang, "research."+id+".name")
	if copied {
		name += " #" + n
	}
	return name
}

// IsCopy reports whether the research is a test copy of another one: the
// operator's /sgdebug research load makes hundreds of them to try the research
// screen with a long list. They live in the session only, are never saved and
// cost nothing.
func (r *Research) IsCopy() bool { return strings.Contains(r.ID, "#") }

// Tree is every research in menu order (a research's needs come
// before it), with the essence costs of researchEssenceCosts added.
var Tree []Research

// index is the index in researchTree of each ID's first research.
var index map[string]int

// SetTree makes t the research tree: every research in menu order,
// a research's needs before it. The game's content sets it once, before
// anything asks for researches.
func SetTree(t []Research) bool {
	Tree = t
	index = map[string]int{}
	for i := range t {
		if _, ok := index[t[i].ID]; !ok {
			index[t[i].ID] = i
		}
	}
	return true
}

func ByID(id string) (*Research, bool) {
	if i, ok := index[id]; ok {
		return &Tree[i], true
	}
	return nil, false
}

// Researched reports whether the save has done a research.
func Researched(s *core.Session, id string) bool {
	if strings.Contains(id, "#") {
		return LocalOf(s).CopiesDone[id]
	}
	return StateOf(s).Researched[id]
}

// Open reports whether a research can be done: not done yet and
// everything it needs is done.
func Open(s *core.Session, r *Research) bool {
	if Researched(s, r.ID) {
		return false
	}
	for _, n := range r.Needs {
		if !Researched(s, n) {
			return false
		}
	}
	return true
}

// Blocker returns the one research that is still to do before r
// when r is only one step from open: that research is open itself, and it is
// the only one r needs that is not done.
func Blocker(s *core.Session, r *Research) (*Research, bool) {
	if Researched(s, r.ID) {
		return nil, false
	}
	var left *Research
	for _, n := range r.Needs {
		if Researched(s, n) {
			continue
		}
		if left != nil {
			return nil, false
		}
		need, ok := ByID(n)
		if !ok {
			return nil, false
		}
		left = need
	}
	if left == nil || !Open(s, left) {
		return nil, false
	}
	return left, true
}

// Paid is how many of an item of a research's cost the save records
// as delivered in parts. Research is paid whole; only old saves hold parts.
func Paid(s *core.Session, r *Research, it world.Item) int {
	return StateOf(s).ResearchPaid[r.ID][naming.ItemKey(it)]
}

// Need is how many of an item of a research's cost are still to be
// spent.
func Need(s *core.Session, r *Research, c Requirement) int {
	return max(0, c.Count-Paid(s, r, c.Item))
}

// Missing lists what p lacks to do a research now: the cost items
// with how many more they need.
func Missing(s *core.Session, p *player.Player, r *Research) []Requirement {
	var missing []Requirement
	for _, c := range r.Cost {
		if short := Need(s, r, c) - core.CountPlain(p.Inventory(), c.Item); short > 0 {
			missing = append(missing, Requirement{c.Item, short})
		}
	}
	return missing
}

// Do spends the whole cost from p's inventory and completes the
// research. A research is never paid in parts: when p lacks anything it
// takes nothing and returns what is missing.
func Do(s *core.Session, p *player.Player, tx *world.Tx, r *Research) (missing []Requirement, done bool) {
	if !Open(s, r) {
		return nil, false
	}
	if missing = Missing(s, p, r); len(missing) > 0 {
		return missing, false
	}
	for _, c := range r.Cost {
		core.TakePlain(p.Inventory(), c.Item, Need(s, r, c))
	}
	PourEssences(tx, p, r.Cost)
	Complete(s, tx, r)
	GiveReward(s, p, tx, r)
	account.Grant(p, achievement.FirstResearch)
	return nil, true
}

// Complete marks a research done and opens its recipes for
// everyone on the island.
func Complete(s *core.Session, tx *world.Tx, r *Research) {
	if StateOf(s).Researched == nil {
		StateOf(s).Researched = map[string]bool{}
	}
	StateOf(s).Researched[r.ID] = true
	achievement.CountAll(tx, "research_done", 1)
	if spent := EssencesIn(r.Cost); spent > 0 {
		Advance(s, tx, advancement.EssenceResearch)
		achievement.CountAll(tx, "essence_spent", int64(spent))
	}
	if strings.HasPrefix(r.ID, "project_") && r.ID != "project_dragon" {
		achievement.CountAll(tx, "project_done", 1)
	}
	if len(StateOf(s).Researched) >= len(Tree) {
		for e := range tx.Players() {
			account.Grant(e.(*player.Player), achievement.AllResearch)
		}
	}
	delete(StateOf(s).ResearchPaid, r.ID)
	s.MarkDirty()
	SendRecipes(s, tx)
	OnChange(s, tx)
	for e := range tx.Players() {
		p := e.(*player.Player)
		if len(r.Unlocks) == 0 {
			// a project: it opens no recipe of its own
			p.Messagef(i18n.T(i18n.Of(p), "researchdone.plain"), r.Title(i18n.Of(p)))
			continue
		}
		p.Messagef(i18n.T(i18n.Of(p), "researchdone.unlocks"), r.Title(i18n.Of(p)), UnlockNames(s, i18n.Of(p), r))
	}
}

// UnlockNames lists a research's results in lang.
func UnlockNames(s *core.Session, lang string, r *Research) string {
	var names []string
	for _, n := range r.Unlocks {
		if it, ok := world.ItemByName(naming.UnlockName(n), 0); ok {
			names = append(names, naming.ItemName(lang, item.NewStack(it, 1)))
		} else {
			names = append(names, n)
		}
	}
	return strings.Join(names, ", ")
}

// Requirement is an item asked for: a research's cost, an expansion's
// price.
type Requirement struct {
	Item  world.Item
	Count int
}
