import { Fragment } from 'react';
import type { ConsoleData, InputData, ListRow, ListSplitData, SplitData } from '../protocol';
import { Window } from '../components/Window';
import { Scroll } from '../components/Scroll';
import { Field } from '../components/Field';
import { Frame, Gap, SlotView } from '../components/Slot';
import { ActionButton, ListButton, McButton } from '../components/Buttons';
import { Icon } from '../components/Icon';
import { Rich } from '../text/Rich';

// Slots in the grid of a split screen.
const SPLIT_CELLS = 48;

/** Pager is a page arrow: pressable with an id, else the dimmed one. */
function Pager(props: { id: string; icon: string }): JSX.Element {
  if (!props.id) {
    return (
      <div className="oui-box mc-btn off">
        <Icon icon={props.icon} />
      </div>
    );
  }
  return (
    <McButton id={props.id} className="mc-btn">
      <Icon icon={props.icon} />
    </McButton>
  );
}

/** SplitScreen is the grid of slots with page arrows on the left and the chosen slot's detail on the right. */
export function SplitScreen(props: { data: SplitData }): JSX.Element {
  const { data } = props;
  const frames = Math.max(0, SPLIT_CELLS - data.slots.length);
  const lines = data.detail.name.split('\n');
  return (
    <Window chrome={data.chrome}>
      <div className="oui-box mc-row" style={{ height: '100%' }}>
        <div className="oui-box mc-panel mc-split-left">
          <div className="oui-box mc-col">
            <div className="oui-box mc-split-grid">
              {data.slots.slice(0, SPLIT_CELLS).map((s, i) => (
                <SlotView key={i} slot={s} />
              ))}
              {Array.from({ length: frames }, (_, i) => (
                <Frame key={'f' + i} />
              ))}
            </div>
            <div className="oui-box mc-pager">
              <Pager id={data.prev} icon="tex:textures/ui/sg/arrow_left" />
              <Pager id={data.next} icon="tex:textures/ui/sg/arrow_right" />
            </div>
          </div>
        </div>
        <div className="oui-box mc-panel mc-split-right">
          <Scroll>
            <div className="oui-box mc-col">
              <div className="oui-box mc-row">
                <SlotView slot={{ id: '', icon: data.detail.icon, count: '', tip: '', deco: '', selected: false }} variant="big" />
                <div className="oui-box mc-col" style={{ flexGrow: 1, marginLeft: '0.8rem' }}>
                  {lines.map((line, i) => (
                    <Fragment key={i}>
                      {i > 0 ? <Gap size={3} /> : null}
                      <Rich text={line} className="oui-text mc-small-text" />
                    </Fragment>
                  ))}
                  {data.detail.status ? (
                    <>
                      <Gap size={3} />
                      <Rich text={data.detail.status} className="oui-text mc-small-text" />
                    </>
                  ) : null}
                  {data.actions.map((a, i) => (
                    <Fragment key={i}>
                      <Gap size={3} />
                      <ActionButton button={a} />
                    </Fragment>
                  ))}
                </div>
              </div>
              {data.detail.body ? (
                <>
                  <Gap size={3} />
                  <Rich text={data.detail.body} className="oui-text mc-small-text" />
                </>
              ) : null}
              {data.rows.map((r, i) => (
                <Fragment key={i}>
                  <Gap size={3} />
                  {r.heading ? <Rich text={r.heading} className="oui-text mc-head" /> : null}
                  <div className="oui-box mc-grid">
                    {r.slots.map((s, si) => (
                      <SlotView key={si} slot={s} />
                    ))}
                  </div>
                </Fragment>
              ))}
            </div>
          </Scroll>
        </div>
      </div>
    </Window>
  );
}

function Row(props: { row: ListRow }): JSX.Element {
  const { row } = props;
  return (
    <McButton id={row.id} className={row.selected ? 'mc-btn mc-lrow on' : 'mc-btn mc-lrow'}>
      <Icon icon={row.icon} />
      <div className="oui-box mc-lrow-text">
        <Rich text={row.name} className="oui-text mc-lrow-name" />
        {row.info ? <Rich text={row.info} className="oui-text mc-lrow-info" /> : null}
      </div>
    </McButton>
  );
}

/** ListSplitScreen is the rows on the left; on the right the body and the actions, the first full width and the rest in pairs. */
export function ListSplitScreen(props: { data: ListSplitData }): JSX.Element {
  const { data } = props;
  const pairs: number[] = [];
  for (let i = 1; i < data.actions.length; i += 2) pairs.push(i);
  return (
    <Window chrome={data.chrome}>
      <div className="oui-box mc-row" style={{ height: '100%' }}>
        <div className="oui-box mc-panel mc-lsplit-left">
          <Scroll>
            {data.rows.map((r, i) => (
              <Row key={i} row={r} />
            ))}
          </Scroll>
        </div>
        <div className="oui-box mc-lsplit-right">
          <Scroll>
            {data.body ? (
              <>
                <Rich text={data.body} className="oui-text mc-small-text" />
                <Gap size={6} />
              </>
            ) : null}
            {data.actions.length > 0 ? <ActionButton button={data.actions[0]} /> : null}
            {pairs.map((i) => (
              <div key={i} className="oui-box mc-lsplit-pair">
                <ActionButton button={data.actions[i]} />
                {data.actions[i + 1] ? <ActionButton button={data.actions[i + 1]} /> : null}
              </div>
            ))}
          </Scroll>
        </div>
      </div>
    </Window>
  );
}

/** InputScreen is a label over an input box and the button that sends it. */
export function InputScreen(props: { data: InputData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <div className="oui-box mc-board">
          {data.label ? (
            <>
              <Rich text={data.label} className="oui-text mc-body-text" />
              <Gap size={3} />
            </>
          ) : null}
          <div className="oui-box mc-search">
            <Field id={data.field} value={data.value} enter={data.submit.id} />
          </div>
          <Gap size={6} />
          <ListButton item={data.submit} />
        </div>
      </Scroll>
    </Window>
  );
}

/** ConsoleScreen is the log scrolled to its end over an input box and the run button. */
export function ConsoleScreen(props: { data: ConsoleData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <div className="oui-box mc-console">
        <Scroll end className="mc-console-log">
          {data.lines.map((l, i) => (
            <Rich key={i} text={l} className="oui-text mc-console-line" />
          ))}
        </Scroll>
        <div className="oui-box mc-console-input">
          <div className="oui-box mc-search">
            <Field key={data.field} id={data.field} value="" placeholder={data.hint} enter={data.submit.id} />
          </div>
          <ActionButton button={data.submit} />
        </div>
      </div>
    </Window>
  );
}
