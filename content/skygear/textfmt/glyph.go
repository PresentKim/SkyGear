package textfmt

// Glyphs drawn by the resource pack font
// (assets/resourcepack/skygear/font/glyph_E1.png): a padlock in the
// first cell and the void upgrade's mark in the second; the second row
// holds the cycle gauge rings (textures/items/cycle_0..10, GaugeGlyph).
const (
	LockGlyph = "\ue100"
	VoidGlyph = "\ue101"
)

// GaugeGlyph is a ring filled in step tenths (0 to 10), for text such as
// the WAILA panel's machine progress.
func GaugeGlyph(step int) string {
	return string(rune(0xe110 + min(max(step, 0), 10)))
}
