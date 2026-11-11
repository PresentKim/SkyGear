package conduit

import "github.com/presentkim/skygear/common/waila"

// Waila adds a bundle's lines to its panel (waila.Provider).
func (b Bundle) Waila(v waila.View, info waila.Info) waila.Info {
	info.Lines = BundleLines(v.Lang, v.Tx, v.Pos, b)
	return info
}
