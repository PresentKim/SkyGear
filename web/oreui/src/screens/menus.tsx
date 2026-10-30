import { Fragment } from 'react';
import type { ReactNode } from 'react';
import type { Button, CardsData, GridData, ListItem, MenuData, SlotMenuData } from '../protocol';
import { Window } from '../components/Window';
import { Scroll } from '../components/Scroll';
import { Card, ListButton } from '../components/Buttons';
import { Gap, SlotView } from '../components/Slot';
import { Rich } from '../text/Rich';

/** Body is the body text and the gap under it; an empty body draws neither. */
function Body(props: { text: string; gap: 3 | 6 }): JSX.Element | null {
  if (!props.text) return null;
  return (
    <>
      <Rich text={props.text} className="oui-text mc-body-text" />
      <Gap size={props.gap} />
    </>
  );
}

function ListButtons(props: { items: ListItem[]; gap?: boolean }): JSX.Element {
  return (
    <>
      {props.items.map((item, i) => (
        <Fragment key={item.id || 'row' + i}>
          {props.gap ? <Gap size={6} /> : null}
          <ListButton item={item} />
        </Fragment>
      ))}
    </>
  );
}

/** MenuScreen is a plain menu: the body text and a list button per item. */
export function MenuScreen(props: { data: MenuData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <Body text={data.body} gap={6} />
        <ListButtons items={data.items} />
      </Scroll>
    </Window>
  );
}

/** CardsScreen is a card menu: the body text and the cards, four to a row. */
export function CardsScreen(props: { data: CardsData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <Body text={data.body} gap={6} />
        <div className="oui-box mc-cards">
          {data.items.map((b: Button) => (
            <Card key={b.id} button={b} />
          ))}
        </div>
      </Scroll>
    </Window>
  );
}

/** GridScreen is the body text, the headed sections of slots and the list buttons. */
export function GridScreen(props: { data: GridData }): JSX.Element {
  const { data } = props;
  const sections: ReactNode[] = data.sections.map((sec, i) => (
    <Fragment key={i}>
      {sec.heading ? <Rich text={sec.heading} className="oui-text mc-head" /> : null}
      <div className="oui-box mc-grid">
        {sec.slots.map((s) => (
          <SlotView key={s.id} slot={s} />
        ))}
      </div>
      <Gap size={3} />
    </Fragment>
  ));
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <Body text={data.body} gap={3} />
        {sections}
        <ListButtons items={data.items} />
      </Scroll>
    </Window>
  );
}

/** SlotMenuScreen is the storage slot with its info, the left and right slots and the list buttons. */
export function SlotMenuScreen(props: { data: SlotMenuData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <div className="oui-box mc-board">
          <div className="oui-box mc-row">
            <SlotView slot={data.store} variant="big" />
            <div className="oui-box mc-slotmenu-info">
              <Rich text={data.info} className="oui-text mc-body-text" />
            </div>
          </div>
          <Gap size={6} />
          <div className="oui-box mc-slotmenu-slots">
            <div className="oui-box mc-row">
              {data.left.map((s) => (
                <SlotView key={s.id} slot={s} />
              ))}
            </div>
            <div className="oui-box mc-row">
              {data.right.map((s) => (
                <SlotView key={s.id} slot={s} />
              ))}
            </div>
          </div>
          <ListButtons items={data.items} gap />
        </div>
      </Scroll>
    </Window>
  );
}
