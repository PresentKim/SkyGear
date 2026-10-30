export type Id = string // the id a press sends; '' = not pressable
export type Icon = string // '' none, 'tex:<texture path>', 'aux:<runtime id * 65536>', 'geo:<item name>' or a bare item name
export type Text = string // may carry § colour codes and client spans (U+FDD0 kind:value U+FDD1 fallback U+FDD2)

export type ScreenName =
  | 'menu' | 'cards' | 'grid' | 'board' | 'sieve' | 'split' | 'search' | 'listsplit' | 'input'
  | 'slotmenu' | 'choice' | 'sides' | 'plane' | 'quest' | 'home' | 'loading' | 'console' | 'probe'

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
export interface ListItem extends Button { toggle?: boolean; on?: boolean } // toggle: a row with a switch
export interface Tab { id: Id; tip: Text; icon: Icon; on: boolean } // on: the selected tab, not pressable
export interface BarButton { id: Id; tip: Text; icon: Icon; glyph: string; quick: boolean } // icon or glyph ('?')
export interface Chrome { title: Text; tabs: Tab[]; bar: BarButton[] } // bar: pagers, help, back (left to right); the close button is always last
export interface Slot { id: Id; alt?: Id; icon: Icon; count: Text; tip: Text; deco: Icon; selected: boolean }
export interface SlotRow { heading: Text; slots: Slot[] }
export interface Cell { k: 'blank' | 'slot' | 'mini' | 'frame' | 'arrow' | 'text'; slot?: Slot; text?: Text; span?: number }

export interface MenuData { chrome: Chrome; body: Text; items: ListItem[] }
export interface CardsData { chrome: Chrome; body: Text; items: Button[] }
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
export interface SideCard { x: number; y: number; arrow: string; name: Text; icon: Icon; tip: Text; focus: boolean; ports: Port[]; none: Text }
export interface Picker { x: number; y: number; w: number; choices: { id: Id; colour: string; icon: Icon; tip: Text; on: boolean }[] }
export interface SidesData { chrome: Chrome; center: Icon; cards: SideCard[]; picker: Picker | null } // x, y, w in GUI px
export interface Spot { id: Id; at: [number, number, number, number]; on: boolean; tip: Text }
export interface PlaneData { fov: number; dist: number; spots: Spot[]; info: Text; actions: Button[]; corner: Text }
export interface Chapter { id: Id; name: Text; icon: Icon; count: Text; on: boolean }
export interface QuestLine { x1: number; y1: number; x2: number; y2: number; cls: string } // rem
export interface QuestNode { id: Id; x: number; y: number; shape: string; state: string; tracked: boolean; tip: Text; icon: Icon } // rem
export interface QuestPanel { icon: Icon; title: Text; status: Text; body: Text; rows: SlotRow[]; actions: Button[]; close: Id }
export interface QuestData { bar: BarButton[]; chapters: Chapter[]; heading: Text; progress: Text; key: string; focusX: number; focusY: number; lines: QuestLine[]; nodes: QuestNode[]; panel: QuestPanel | null }
export interface HomeButton { id: Id; icon: Icon; label: Text; badge: number }
export interface HomeProfile { id: Id; face: string[]; icon: Icon; name: Text; title: Text; fill: number; line: Text } // face: 64 'rrggbb' ('' none) or []
export interface HomeSave { head: Text; name: Text; icon: Icon; lines: Text[]; arrows: boolean; prev: Id; next: Id }
export interface HomeOverlay { title: Text; body: Text; buttons: { id: Id; label: Text }[] }
export interface HomeData { profile: HomeProfile; corner: HomeButton[]; side: HomeButton[]; feed: Text[]; save: HomeSave; play: { id: Id; label: Text; sub: Text }; manage: { id: Id; label: Text } | null; overlay: HomeOverlay | null }
export interface LoadingData { icon: Icon; title: Text; sub: Text; line: Text; tip: Text }
export interface ConsoleData { chrome: Chrome; lines: Text[]; field: Id; hint: Text; submit: Button }
export interface ProbeNode { t: 'input' | 'box' | 'probe' | 'restest'; [field: string]: unknown } // the legacy node fields
export interface ProbeData { chrome: Chrome; info: Text; nodes: ProbeNode[] }

export interface ScreenData {
  menu: MenuData; cards: CardsData; grid: GridData; board: BoardData; sieve: SieveData; split: SplitData
  search: SearchData; listsplit: ListSplitData; input: InputData; slotmenu: SlotMenuData; choice: ChoiceData
  sides: SidesData; plane: PlaneData; quest: QuestData; home: HomeData; loading: LoadingData
  console: ConsoleData; probe: ProbeData
}
