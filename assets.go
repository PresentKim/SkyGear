package main

import "embed"

// packFS holds the resource packs (assets/resourcepack/<name>), so the
// server runs from any folder without the assets next to it.
//
//go:embed all:assets/resourcepack
var packFS embed.FS
