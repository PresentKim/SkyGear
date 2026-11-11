// Package clientdef adds to the item entries a server sends its clients the
// components that Dragonfly leaves out: what a custom item mines (the
// minecraft:digger component), that it can be used on the air
// (minecraft:use_modifiers) and what it burns for (minecraft:fuel). An item
// declares them by implementing Digger, TagDigger and the Dragonfly interfaces
// item.Usable and item.Fuel; ItemEntry reads them from the registered custom
// items. The package knows no game: hand ItemEntry to sghook.EditCustomItems
// (inside the game's own edit) and every custom item gets its components.
package clientdef
