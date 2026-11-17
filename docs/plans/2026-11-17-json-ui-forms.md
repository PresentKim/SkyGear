# Menus in a chest window

The client no longer loads the Ore UI page from a server pack, so every DDUI screen of `common/ui` is drawn again with JSON UI. The screens show in one chest window per player that stays open while the server rewrites its slots (the InvUI technique of the research repository: `invui-components`, `invui-input-probe`). The look follows the Ore UI screens (`web/oreui`, `assets/resourcepack/oreui/hbui/oui-mc.css`), not the JSON UI theme removed in d149ab9e.

## Decisions

- The builders of `common/ui` and their callers stay. `showScreen` turns the screen name and its data into the slots of the menu window (`common/ui/menu.go`) instead of sending a DDUI root.
- `common/invui` keeps the menu window: a `chestui.Storage` of `invui.Size` slots titled with `invui.Marker`. Changing the screen or updating it writes only the slots that changed; the window is not reopened.
- A press is a drop of one item: every input of a slot button maps to `button.drop_one` (guide INP-17). The storage's handler cancels every move and runs the slot's element for a drop. The slot buttons keep focus enabled and the `button.container_slot_hovered` mapping, like the vanilla container slot button, or the client sends no request.
- Text input: an `input` is a custom form (`common/formui`, `sgui/ui/sg_form.json`). The codex and storage searches are removed. Island expansion (`plane`) works by clicks only.
- Inventory screens (`ShowInventory`) become real container windows.

## Slots

`common/invui/layout.go` holds the layout, and `tools/invuigen` writes `assets/resourcepack/inventoryui/ui/sg_menu.json` from it. Slot 0 names the screen, 1 holds the title and 2 the body. Tabs, control bar buttons and list rows take an icon slot and a button slot each.

- A button slot holds a paper named with its text and a kind marker (`invui.Kind*`) at the end.
- An icon slot holds the item itself for an item icon, or a paper named with the texture path and the kind of its element.
- The client puts `§f` in front of a slot's `#hover_text`, so the pack takes `§o`, `§r`, `§f` and the kind markers off before using a name as a texture path.

## Pack

- `inventoryui/ui/chest_screen.json` routes a window whose title holds the marker to the vanilla small chest panel, inserts `sg_menu.overlay` into it and hides the vanilla window under it (the overlay's `sg_menu_hide_vanilla` property). Replacing `$screen_content` with a panel of our own made the client close the window at once.
- `sg_menu.overlay` is gated on the title at load time (`ignored`) and draws the window inside a modal input panel.

## Phases

1. The menu window, the window chrome (title, tabs, control bar, close) and the list rows; other screens show a row per element they can press. Inputs as custom forms.
2. Layouts for `cards`, `grid`, `board`, `split`, `listsplit`, `slotmenu`, `choice`, `stepper`, `sieve`, `sides`, `pick`, `result`, `equip`, `console`.
3. `home`, `loading`, `radial`, `plane` (a chunk map), `quest` (nodes on fixed cells, panning by buttons).
4. The touch layout (`pocket_containers`), and a client closing the window by opening another screen (chestui ignores window 255 closes).
5. Inventory screens to container windows.
6. Remove the Ore UI transport, the oreui pack, `web/oreui` and the pack encryption requirement; update the docs.

Each phase is checked in the game (Windows client) before the next.
