import type { ScreenName } from '../protocol';
import { CardsScreen, GridScreen, MenuScreen, SlotMenuScreen } from './menus';
import { BoardScreen, SearchScreen, SieveScreen } from './boards';
import { ConsoleScreen, InputScreen, ListSplitScreen, SplitScreen } from './splits';
import { ChoiceScreen, LoadingScreen, PlaneScreen } from './boards2';
import { SidesScreen } from './sides';
import { QuestScreen } from './quest';
import { HomeScreen } from './home';
import { ProbeScreen } from './probe';

/** screens maps a screen name to its component; the data's type follows the name (Confirm is a menu). */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const screens: Record<ScreenName, (props: { data: any }) => JSX.Element> = {
  menu: MenuScreen,
  cards: CardsScreen,
  grid: GridScreen,
  board: BoardScreen,
  sieve: SieveScreen,
  split: SplitScreen,
  search: SearchScreen,
  listsplit: ListSplitScreen,
  input: InputScreen,
  slotmenu: SlotMenuScreen,
  choice: ChoiceScreen,
  sides: SidesScreen,
  plane: PlaneScreen,
  quest: QuestScreen,
  home: HomeScreen,
  loading: LoadingScreen,
  console: ConsoleScreen,
  probe: ProbeScreen,
};
