package dragonfactory

import "github.com/presentkim/dragonfactory/internal/i18n"

// ResearchText explains a research in its screen (docs/design/02-dragonfactory.md
// §3.4): what it is for, how to get what it costs, and what to do once it is
// done. A player must never be left without a next step, so every research
// has all of them. The texts are in the catalog (internal/i18n/locales) under
// "research.<id>.short|about|hint|after".
type ResearchText struct {
	// Short is one line shown under the name in the hover text of the list.
	Short string
	// About says what the research opens and why it matters.
	About string
	// Hint says where the cost comes from (shown while it is not done).
	Hint string
	// After says what to do with the new recipes (shown once it is done, and
	// in the message of the completion).
	After string
}

// researchTextFields are the text parts of every research.
var researchTextFields = []string{"short", "about", "hint", "after"}

// researchText returns a research's text in lang (a missing text shows the
// default language's, a missing research its keys).
func researchText(lang, id string) ResearchText {
	t := func(field string) string { return i18n.T(lang, "research."+id+"."+field) }
	return ResearchText{Short: t("short"), About: t("about"), Hint: t("hint"), After: t("after")}
}
