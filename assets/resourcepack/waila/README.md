# WAILA (resource pack)

From [WAILA](https://github.com/r4isen1920/WAILA) v5.1.1 by r4isen1920, MIT License (`LICENSE`).
Only the resource pack is used; the server (Go) does what the Script API behaviour pack did.
Changes for dragonfactory (see `docs/tech/waila.md`):

- `manifest.json`: new UUIDs, no dependency on the behaviour pack.
- `ui/hud_screen.json`: `"namespace": "hud"` (vanilla namespace, for the pack validator).
- The block icon is drawn from the subtitle (`tile.r4ui` `df_subtitle_item_renderer`, used by `main.r4ui`) instead of an inventory slot.
