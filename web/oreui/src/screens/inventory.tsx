import { useEffect, useRef } from 'react';
import type { BarButton, InventoryData } from '../protocol';
import { usePressable } from '../interact/press';
import { hideTip, showTip } from '../interact/Tooltip';
import { Rich } from '../text/Rich';
import { Icon } from '../components/Icon';
import { Field } from '../components/Field';
import { Window } from '../components/Window';
import { InvRows } from '../inventory/InvRows';
import { CursorStack } from '../inventory/Cursor';
import { useGestureRedraw } from '../inventory/InvSlot';
import * as gesture from '../inventory/gesture';

// The inventory window is nine slots wide and about as tall as the vanilla double chest, capped so
// the control bar above it stays on screen; the grow row takes what the other rows leave.
const INV_W = 190;
const INV_H = 240;

function SideButton(props: { button: BarButton }): JSX.Element {
  const b = props.button;
  const press = usePressable({ id: b.id, tip: b.tip, quick: true });
  return (
    <div className="oui-box mc-icon-button" {...press}>
      <Icon icon={b.icon} />
    </div>
  );
}

function QuickSwitch(props: { tip: string }): JSX.Element {
  useGestureRedraw();
  const on = gesture.isQuick();
  const handlers: Record<string, unknown> = {
    onMouseDown: () => gesture.toggleQuick(),
    onMouseEnter: (ev: MouseEvent) => showTip(ev.currentTarget as HTMLElement, props.tip),
    onMouseLeave: () => hideTip(),
    onTouchStart: (ev: TouchEvent) => {
      ev.preventDefault?.();
      gesture.toggleQuick();
    },
  };
  return (
    <div className={'oui-box mc-icon-button mc-inv-quick' + (on ? ' on' : '')} {...handlers}>
      <Icon icon="tex:textures/ui/sg/arrow_right" />
    </div>
  );
}

function TouchTip(): JSX.Element | null {
  useGestureRedraw();
  const tip = gesture.lastTouchTip();
  return tip ? <Rich text={tip.split('\n')[0]} className="oui-text mc-inv-touch-tip" /> : null;
}

/** InventoryScreen draws an inventory screen: the search box, the rows, the side buttons and the cursor stack. */
export function InventoryScreen(props: { data: InventoryData }): JSX.Element {
  const { data } = props;
  gesture.setCarrying(!!data.cursor);
  const typed = useRef(data.query);
  useEffect(() => () => gesture.reset(), []);
  return (
    <Window chrome={data.chrome} className="mc-inventory" size={[INV_W, INV_H]}>
      <div className="oui-box mc-inv-side">
        {data.side.map((b) => (
          <SideButton key={b.id} button={b} />
        ))}
        {data.quick ? <QuickSwitch tip={data.quick} /> : null}
      </div>
      {data.field ? (
        <div className="oui-box mc-search mc-inv-search">
          <Icon icon="tex:textures/ui/magnifyingGlass" />
          <Field id={data.field} value={typed.current} placeholder={data.placeholder} delay={250} />
        </div>
      ) : null}
      {data.info ? <Rich text={data.info} className="oui-text mc-body-text mc-inv-info" /> : null}
      <TouchTip />
      <InvRows rows={data.rows} />
      <CursorStack slot={data.cursor} />
    </Window>
  );
}
