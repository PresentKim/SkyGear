package dragonfactory

// Glyphs drawn by the resource pack font
// (assets/resourcepack/dragonfactory/font/glyph_E1.png): a padlock in the
// first cell and the void upgrade's mark in the second; the second row
// holds the cycle gauge rings (textures/items/cycle_0..10, gaugeGlyph).
const (
	lockGlyph = "\ue100"
	voidGlyph = "\ue101"
)

// gaugeGlyph is a ring filled in step tenths (0 to 10), for text such as
// the WAILA panel's machine progress.
func gaugeGlyph(step int) string {
	return string(rune(0xe110 + min(max(step, 0), 10)))
}
