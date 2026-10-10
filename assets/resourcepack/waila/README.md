# WAILA (resource pack)

From [WAILA](https://github.com/r4isen1920/WAILA) v5.1.1 by r4isen1920, MIT License (`LICENSE`).
Only the resource pack is used; the server (Go) does what the Script API behaviour pack did.
Changes for dragonfactory (see `docs/tech/waila.md`):

- `manifest.json`: new UUIDs, no dependency on the behaviour pack.
- `ui/hud_screen.json`: `"namespace": "hud"` (vanilla namespace, for the pack validator).
- `main.r4ui`: the top middle panel sits 26 below the top (under the boss bar); the block states label is off (unused, it warned "expected a string").
- `tile.r4ui` `chest_inventory`: one row only (no second row, no "+n").
- `tile.r4ui` `generic_container_inventory_item`: no slot border images (an empty panel of the same size).
- `tile.r4ui` `item_inventory`: container slots read their item and count from the subtitle (fixed-width fields per slot) instead of player inventory slots. Generated: one literal binding set per slot index.
- The block icon is drawn from the subtitle ("i" + item runtime ID × 65536 left padded with "x" to 11 characters; the "i" keeps it text) (`tile.r4ui` `df_subtitle_item_renderer`, used by `main.r4ui`) instead of an inventory slot.
- `tile.r4ui` `df_marks`, `df_item_line` and `main.r4ui` `df_name_row`: small marks after the name and a line with an item icon, read from the subtitle tail (server `internal/waila/tail.go`).
