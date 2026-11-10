export type Id = string // the id a press sends; '' = not pressable
export type Icon = string // '' none, 'tex:<texture path>', 'aux:<runtime id * 65536>', 'geo:<item name>' or a bare item name
export type Text = string // may carry § colour codes and client spans (U+FDD0 kind:value U+FDD1 fallback U+FDD2)

export type ScreenName =
  | 'menu' | 'cards' | 'pick' | 'result' | 'equip' | 'grid' | 'board' | 'sieve' | 'split' | 'search' | 'listsplit' | 'input'
  | 'slotmenu' | 'choice' | 'sides' | 'plane' | 'quest' | 'home' | 'loading' | 'console' | 'probe' | 'inventory' | 'stepper' | 'radial'

export interface Root {
  screen: ScreenName
  data: unknown // the ScreenData of screen
  keys?: Keys // the screen answers these keys itself (else they move the keyboard selection)
  lobby?: Lobby // set while the player is in the lobby
  debug?: boolean // draw the renderer's debug line
}
export interface Keys { up?: Id; down?: Id; left?: Id; right?: Id; ok?: Id }
export interface Lobby { title: Text; logo: Icon; touch: boolean }

export interface Button { id: Id; label: Text; icon: Icon }
export interface ListItem extends Button { toggle?: boolean; on?: boolean; kind?: 'heading' | 'divider' } // toggle: a row with a switch; kind: a row that is no button
export interface Tab { id: Id; tip: Text; icon: Icon; on: boolean } // on: the selected tab, not pressable
export interface BarButton { id: Id; tip: Text; icon: Icon; glyph: string; quick: boolean } // icon or glyph ('?')
export interface Chrome { title: Text; tabs: Tab[]; bar: BarButton[] } // bar: pagers, help, back (left to right); the close button is always last
export interface Slot { id: Id; alt?: Id; icon: Icon; count: Text; tip: Text; rare: number; selected: boolean } // rare: the item's rarity (0 common to 4 legendary), the tooltip's border
export interface SlotRow { heading: Text; slots: Slot[] }
export interface Cell { k: 'blank' | 'slot' | 'mini' | 'frame' | 'arrow' | 'text'; slot?: Slot; text?: Text; span?: number }

export interface MenuData { chrome: Chrome; body: Text; items: ListItem[] }
export interface CardsData { chrome: Chrome; body: Text; items: Button[] }
export interface PickCard { id: Id; icon: Icon; title: Text; text: Text; tag: Text; rarity: 'common' | 'rare' | 'epic' | '' }
export interface PickData { chrome: Chrome; body: Text; cards: PickCard[]; items: ListItem[] }
export interface ResultLine { label: Text; value: Text }
export interface ResultData { chrome: Chrome; headline: Text; win: boolean; lines: ResultLine[]; stars: { on: boolean; new: boolean }[]; rewards: Slot[]; ok: Button }
export interface EquipSlot extends Slot { empty: Icon } // empty: the slot's silhouette, shown while icon is ''
export interface EquipGroup { heading: Text; slots: EquipSlot[] }
export interface EquipData { chrome: Chrome; note: Text; groups: EquipGroup[]; bagHeading: Text; bag: Slot[]; items: ListItem[] }
export interface GridData { chrome: Chrome; body: Text; sections: SlotRow[]; items: ListItem[] }
export interface BoardData { chrome: Chrome; body: Text; rows: Cell[][]; items: ListItem[] }
export interface SieveRecipe { input: Slot; mesh: Slot; drops: Slot[] }
export interface SieveGroup { head: Slot; name: Text; recipes: SieveRecipe[] }
export interface SieveData { chrome: Chrome; top: Cell[][]; groups: SieveGroup[]; lines: number; items: ListItem[] }
export interface Detail { icon: Icon; name: Text; status: Text; body: Text }
export interface SplitData { chrome: Chrome; slots: Slot[]; prev: Id; next: Id; detail: Detail; actions: Button[]; rows: SlotRow[] }
export interface SearchData { chrome: Chrome; field: Id; query: Text; placeholder: Text; info: Text; sections: SlotRow[]; items: ListItem[] }
export interface ListRow { id: Id; icon: Icon; name: Text; info: Text; selected: boolean }
export interface ListSplitData { chrome: Chrome; rows: ListRow[]; body: Text; actions: Button[] }
export interface InputData { chrome: Chrome; label: Text; field: Id; value: Text; submit: Button }
export interface SlotMenuData { chrome: Chrome; store: Slot; info: Text; left: Slot[]; right: Slot[]; items: ListItem[] }
export interface Stat { icon: Icon; label: Text; value: Text }
export interface Choice { id: Id; label: Text; icon: Icon; tip: Text; on: boolean; dim: boolean }
export interface ChoiceGroup { title: Text; choices: Choice[] }
export interface ChoiceData { chrome: Chrome; stats: Stat[]; note: Text; groups: ChoiceGroup[]; items: ListItem[] }
export interface Swatch { id: Id; alt: Id; colour: string; icon: Icon; tip: Text } // colour '#rrggbb', or '' and an icon
export interface Port { label: Text; swatch: Swatch | null; plus: { id: Id; on: boolean } }
export interface SideCard { name: Text; icon: Icon; tip: Text; focus: boolean; ports: Port[]; none: Text }
export interface Picker { choices: { id: Id; colour: string; icon: Icon; tip: Text; on: boolean }[] }
export interface SidesData { chrome: Chrome; cards: SideCard[]; guide: Text; picker: Picker | null } // cards three to a row
export interface Spot { id: Id; at: [number, number, number, number]; on: boolean; tip: Text }
export interface PlaneData { fov: number; dist: number; spots: Spot[]; info: Text; actions: Button[]; corner: Text }
export interface Chapter { id: Id; name: Text; icon: Icon; count: Text; on: boolean; new: boolean } // new: it has quests tagged new
export interface QuestLine { x1: number; y1: number; x2: number; y2: number; cls: string } // rem
export interface QuestNode { id: Id; x: number; y: number; shape: string; state: string; tracked: boolean; new: boolean; seen: Id; tip: Text; icon: Icon } // rem; new: the NEW tag, seen: sent ('seen') once it is looked at
export interface QuestPanel { icon: Icon; title: Text; status: Text; body: Text; rows: SlotRow[]; actions: Button[]; close: Id }
export interface QuestData { bar: BarButton[]; chapters: Chapter[]; heading: Text; progress: Text; key: string; focusX: number; focusY: number; recenter: boolean; lines: QuestLine[]; nodes: QuestNode[]; panel: QuestPanel | null }
export interface HomeButton { id: Id; icon: Icon; label: Text; badge: number }
export interface HomeProfile { id: Id; face: string[]; icon: Icon; name: Text; title: Text; fill: number; line: Text } // face: 64 'rrggbb' ('' none) or []
export interface HomeSave { head: Text; name: Text; icon: Icon; lines: Text[]; arrows: boolean; prev: Id; next: Id }
export interface HomeOverlay { title: Text; body: Text; buttons: { id: Id; label: Text; leave?: boolean }[] } // leave: the press leaves the server from the client
export interface HomeData { profile: HomeProfile; corner: HomeButton[]; side: HomeButton[]; feed: Text[]; save: HomeSave; play: { id: Id; label: Text; sub: Text }; manage: { id: Id; label: Text } | null; overlay: HomeOverlay | null }
export interface LoadingData { icon: Icon; title: Text; sub: Text; line: Text; tip: Text }
export interface ConsoleData { chrome: Chrome; lines: Text[]; field: Id; hint: Text; submit: Button }
export interface ProbeNode { t: 'input' | 'box' | 'probe' | 'restest'; [field: string]: unknown } // the legacy node fields
export interface ProbeData { chrome: Chrome; info: Text; nodes: ProbeNode[] }

// An inventory screen: slots worked like the vanilla inventory. A slot id is "<area>:<index>"; a click
// sends 'slot' {b: 0 left | 1 right, shift, dbl} and a drag 'drag' {b, ids}.
export interface InvSlot { id: Id; icon: Icon; count: Text; tip: Text; rare: number; empty: Icon; dim?: boolean; off?: boolean } // empty: shown while icon is ''; dim: drawn faded; off: a slot that is not there, dark and pressless
export interface InvTab { id: Id; tip: Text; icon: Icon; on: boolean; end: boolean } // end: at the right end of the tabs
export interface InvBook { tabs: InvTab[]; title: Text; field: Id; query: Text; placeholder: Text; filter: BarButton | null; slots: InvSlot[]; note: Text } // the vanilla recipe book left of the window (slots "book:<i>"); field: the search tab's box ('' none)
export interface InvGauge { kind: string; fill: number; tip: Text } // a tall bar filled fill% from the bottom: "energy" red, "fluid" blue
export interface InvProgress { icon: Icon; tip: Text } // a machine's progress arrow or ring
export interface InvPart { area: string; cols: number; slots: InvSlot[]; big: boolean; arrow: boolean; button: BarButton | null; off?: boolean; gauge?: InvGauge; progress?: InvProgress } // off: dark slots that take no presses
export interface InvRow { heading: Text; parts: InvPart[]; grow: boolean; note: Text; bar?: BarButton[]; barTop?: boolean; center?: boolean } // grow: takes the height left and scrolls; bar: buttons outside the right edge, level with the row's bottom (its top with barTop); center: the parts in the middle of the window, the gauges at its left edge
export interface InventoryData { chrome: Chrome; field: Id; query: Text; placeholder: Text; info: Text; rows: InvRow[]; side: BarButton[]; tools?: BarButton[]; quick: Text; book?: InvBook | null; cursor: InvSlot | null; hold?: boolean; height?: number } // quick: the tip of the quick move switch ('' none); hold: the screen hears when buttons, fingers and shift are held and let go; height: the window's height in GUI pixels (absent: the vanilla inventory's)

export interface StepperItem { label: Text; value: Text; up: BarButton; down: BarButton } // an arrow with id '' is dim
export interface StepperData { chrome: Chrome; line: Text; steppers: StepperItem[]; side: BarButton[] }

export interface RadialItem { id: Id; label: Text; value: Text; info: Text; icon: Icon; on: boolean } // id '' is dim; on: the current choice
export interface RadialData { title: Text; line: Text; info: Text; items: RadialItem[]; back: Id; backTip: Text } // back 'close': the middle button closes the menu

export interface ScreenData {
  menu: MenuData; cards: CardsData; pick: PickData; result: ResultData; equip: EquipData; grid: GridData; board: BoardData; sieve: SieveData; split: SplitData
  search: SearchData; listsplit: ListSplitData; input: InputData; slotmenu: SlotMenuData; choice: ChoiceData
  sides: SidesData; plane: PlaneData; quest: QuestData; home: HomeData; loading: LoadingData
  console: ConsoleData; probe: ProbeData; inventory: InventoryData; stepper: StepperData; radial: RadialData
}
