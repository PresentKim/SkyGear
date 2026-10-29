package progress

import (
	"strings"

	"github.com/presentkim/skygear/common/i18n"
)

// ResearchText explains what a research is for in its screen; where to get
// the cost and what to do next is the advancements' job. The texts are in
// content/skygear/locales under "research.<id>.short|about".
type ResearchText struct {
	// Short is one line shown under the name in the hover text of the list.
	Short string
	// About says what the research opens and why it matters.
	About string
}

// TextFields are the text parts of every research.
var TextFields = []string{"short", "about"}

// Text returns a research's text in lang (a missing text shows the
// default language's, a missing research its keys).
func Text(lang, id string) ResearchText {
	id, _, _ = strings.Cut(id, "#") // a test copy reads its original's texts
	t := func(field string) string { return i18n.T(lang, "research."+id+"."+field) }
	return ResearchText{Short: t("short"), About: t("about")}
}
