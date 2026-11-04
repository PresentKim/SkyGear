import { useEffect, useReducer, useRef } from 'react';
import type { InvSlot } from '../protocol';
import { InkContext, Rich } from '../text/Rich';
import { Icon } from '../components/Icon';
import { hideTip, showTip } from '../interact/Tooltip';
import * as gesture from './gesture';

/** useGestureRedraw redraws the component when a drag, the quick move switch or the touched tip changes. */
export function useGestureRedraw(): void {
  const [, redraw] = useReducer((n: number) => n + 1, 0);
  useEffect(() => gesture.subscribe(redraw), []);
}

/** InvSlotView is a slot the player works like a vanilla inventory slot (gesture.ts). */
export function InvSlotView(props: { slot: InvSlot; big?: boolean }): JSX.Element {
  const { slot } = props;
  const el = useRef<HTMLDivElement | null>(null);
  const cur = useRef(slot);
  cur.current = slot;
  useGestureRedraw();
  useEffect(() => {
    const e = el.current;
    if (!e || !slot.id) return;
    return gesture.registerSlot(slot.id, e);
  }, [slot.id]);
  useEffect(
    () => () => {
      hideTip();
    },
    [],
  );
  const dragged = gesture.dragIds().includes(slot.id);
  let cls = 'oui-box mc-slot mc-inv-slot pressable';
  if (props.big) cls += ' mc-big';
  if (slot.dim) cls += ' mc-inv-dim';
  if (dragged) cls += ' mc-inv-dragged';
  const handlers: Record<string, unknown> = {
    onMouseDown: (ev: MouseEvent) => {
      ev.preventDefault?.();
      gesture.mouseDown(cur.current.id, ev);
    },
    onMouseEnter: (ev: MouseEvent) => {
      gesture.mouseEnter(cur.current.id);
      if (cur.current.tip) showTip(ev.currentTarget as HTMLElement, cur.current.tip);
    },
    onMouseLeave: () => hideTip(),
    onTouchStart: (ev: TouchEvent) => gesture.touchStart(cur.current.id, cur.current.tip, ev),
    onContextMenu: (ev: Event) => ev.preventDefault?.(),
  };
  return (
    <div className={cls} ref={el} {...handlers}>
      <Icon icon={slot.deco} className="mc-deco" />
      <Icon icon={slot.icon || slot.empty} className={slot.icon ? 'mc-icon' : 'mc-icon mc-inv-empty'} />
      {slot.count ? (
        <InkContext.Provider value="dark">
          <Rich text={slot.count} className="oui-text mc-count" />
        </InkContext.Provider>
      ) : null}
    </div>
  );
}
