import { Fragment, useRef } from 'react';
import type { ReactNode } from 'react';
import type { BoardData, Cell, ListItem, SearchData, SieveData, SlotRow } from '../protocol';
import { Window } from '../components/Window';
import { Scroll } from '../components/Scroll';
import { Field } from '../components/Field';
import { Frame, Gap, SlotView } from '../components/Slot';
import { ListButton } from '../components/Buttons';
import { Icon } from '../components/Icon';
import { Rich } from '../text/Rich';

// Board columns per row, and the width of one column in rem (18 GUI units).
const COLUMNS = 11;
const COLUMN_REM = 3.6;

/** BoardRow is a row of COLUMNS cells; a text cell runs over the blank cells after it. */
function BoardRow(props: { row: Cell[] }): JSX.Element {
  const { row } = props;
  const at = (i: number): Cell => row[i] ?? { k: 'blank' };
  const cells: ReactNode[] = [];
  for (let i = 0; i < COLUMNS; i++) {
    const c = at(i);
    switch (c.k) {
      case 'slot':
        cells.push(<SlotView key={i} slot={c.slot!} />);
        break;
      case 'mini':
        cells.push(<SlotView key={i} slot={c.slot!} variant="mini" />);
        break;
      case 'frame':
        cells.push(<Frame key={i} />);
        break;
      case 'arrow':
        cells.push(
          <div key={i} className="oui-box mc-arrow">
            <div className="oui-box mc-arrow-img" />
          </div>,
        );
        break;
      case 'text': {
        let span = 1;
        while (i + span < COLUMNS && at(i + span).k === 'blank') span++;
        const width = Math.round(COLUMN_REM * span * 10) / 10 + 'rem';
        cells.push(
          <Rich key={i} text={c.text ?? ''} className="oui-text mc-cell-text" style={{ width, flexShrink: 0 }} />,
        );
        i += span - 1;
        break;
      }
      default:
        cells.push(<Gap key={i} />);
    }
  }
  return <div className="oui-box mc-board-row">{cells}</div>;
}

/** Actions is a list button per item, each under a small gap. */
function Actions(props: { items: ListItem[] }): JSX.Element {
  return (
    <>
      {props.items.map((item) => (
        <Fragment key={item.id}>
          <Gap size={3} />
          <ListButton item={item} />
        </Fragment>
      ))}
    </>
  );
}

/** BoardScreen is an item board: the body text, the rows and the list buttons. */
export function BoardScreen(props: { data: BoardData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <div className="oui-box mc-board">
          {data.body ? (
            <>
              <Rich text={data.body} className="oui-text mc-body-text" />
              <Gap size={3} />
            </>
          ) : null}
          {data.rows.map((r, i) => (
            <BoardRow key={i} row={r} />
          ))}
        </div>
        <Actions items={data.items} />
      </Scroll>
    </Window>
  );
}

/** SieveScreen is the board rows and under them the recipes of each sieve. */
export function SieveScreen(props: { data: SieveData }): JSX.Element {
  const { data } = props;
  const total = Math.max(1, data.lines) * 9;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <div className="oui-box mc-board">
          {data.top.map((r, i) => (
            <BoardRow key={i} row={r} />
          ))}
          {data.groups.map((g, gi) => (
            <Fragment key={gi}>
              <Gap size={6} />
              <div className="oui-box mc-board-row">
                <SlotView slot={g.head} />
                <Rich text={'§l' + g.name} className="oui-text mc-cell-text" />
              </div>
              <Gap size={3} />
              {g.recipes.map((r, ri) => (
                <Fragment key={ri}>
                  <div className="oui-box mc-jei-recipe">
                    <div className="oui-box mc-jei-top">
                      <SlotView slot={r.input} />
                      <SlotView slot={r.mesh} variant="jei-mesh" />
                    </div>
                    <div className="oui-box mc-jei-drops">
                      {r.drops.map((d, di) => (
                        <SlotView key={di} slot={{ ...d, count: '' }} />
                      ))}
                      {Array.from({ length: Math.max(0, total - r.drops.length) }, (_, fi) => (
                        <Frame key={'f' + fi} />
                      ))}
                    </div>
                  </div>
                  <Gap size={6} />
                </Fragment>
              ))}
            </Fragment>
          ))}
        </div>
        <Actions items={data.items} />
      </Scroll>
    </Window>
  );
}

/** Sections is the headed grids of slots. */
function Sections(props: { sections: SlotRow[] }): JSX.Element {
  return (
    <div className="oui-box mc-col">
      {props.sections.map((sec, i) => (
        <Fragment key={i}>
          {sec.heading ? <Rich text={sec.heading} className="oui-text mc-head" /> : null}
          <div className="oui-box mc-grid">
            {sec.slots.map((s, si) => (
              <SlotView key={si} slot={s} />
            ))}
          </div>
          <Gap size={3} />
        </Fragment>
      ))}
    </div>
  );
}

/** SearchScreen is an input box over the results the server sends as the player types. */
export function SearchScreen(props: { data: SearchData }): JSX.Element {
  const { data } = props;
  // The field owns what was typed; later data only changes the results.
  const typed = useRef(data.query);
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <div className="oui-box mc-board">
          <div className="oui-box mc-search">
            <Icon icon="tex:textures/ui/magnifyingGlass" />
            <Field id={data.field} value={typed.current} placeholder={data.placeholder} delay={250} />
          </div>
          <Rich text={data.info} className="oui-text mc-body-text" />
          <Gap size={3} />
          <div className="oui-region">
            <Sections sections={data.sections} />
          </div>
          <Actions items={data.items} />
        </div>
      </Scroll>
    </Window>
  );
}
