package dragonfactory

import (
	"strings"

	"github.com/presentkim/dragonfactory/internal/i18n"
)

// ResearchText explains a research in its screen (docs/design/02-dragonfactory.md
// §9): what it is for. Where to get what it costs and what to do next is the
// advancements' job (§10). The texts are in the catalog
// (internal/i18n/locales) under "research.<id>.short|about".
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
