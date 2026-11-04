import { useEffect, useRef } from 'react';
import type { InvSlot } from '../protocol';
import { InkContext, Rich } from '../text/Rich';
import { Icon } from '../components/Icon';

let px = -1000;
let py = -1000;
let tracking = false;
const followers = new Set<() => void>();

function track(): void {
  if (tracking) return;
  tracking = true;
  const move = (x: number, y: number) => {
    px = x;
    py = y;
    followers.forEach((f) => f());
  };
  document.addEventListener('mousemove', (ev) => move(ev.clientX, ev.clientY));
  document.addEventListener('mousedown', (ev) => move(ev.clientX, ev.clientY));
  const touch = (ev: TouchEvent) => {
    const t = (ev.touches && ev.touches[0]) || (ev.changedTouches && ev.changedTouches[0]);
    if (t) move(t.clientX, t.clientY);
  };
  document.addEventListener('touchstart', touch);
  document.addEventListener('touchmove', touch);
}

/** CursorStack draws the stack the server's cursor holds beside the pointer (or where the finger was). */
export function CursorStack(props: { slot: InvSlot | null }): JSX.Element | null {
  const box = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    track();
    const place = () => {
      const b = box.current;
      if (b) b.style.transform = 'translate(' + Math.round(px) + 'px, ' + Math.round(py) + 'px)';
    };
    place();
    followers.add(place);
    return () => {
      followers.delete(place);
    };
  }, [props.slot]);
  if (!props.slot) return null;
  return (
    <div className="oui-box mc-inv-cursor" ref={box}>
      <Icon icon={props.slot.icon} className="mc-icon" />
      {props.slot.count ? (
        <InkContext.Provider value="dark">
          <Rich text={props.slot.count} className="oui-text mc-count" />
        </InkContext.Provider>
      ) : null}
    </div>
  );
}
