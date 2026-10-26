package dragonfactory

import (
	"strings"

	"github.com/presentkim/dragonfactory/common/i18n"
)

// ResearchText explains what a research is for in its screen; where to get
// the cost and what to do next is the advancements' job. The texts are in
// internal/locales under "research.<id>.short|about".
type ResearchText struct {
	// Short is one line shown under the name in the hover text of the list.
	Short string
	// About says what the research opens and why it matters.
	About string
}

// researchTextFields are the text parts of every research.
var researchTextFields = []string{"short", "about"}

// researchText returns a research's text in lang (a missing text shows the
// default language's, a missing research its keys).
func researchText(lang, id string) ResearchText {
	id, _, _ = strings.Cut(id, "#") // a test copy reads its original's texts
	t := func(field string) string { return i18n.T(lang, "research."+id+"."+field) }
	return ResearchText{Short: t("short"), About: t("about")}
}
