import { useRef } from 'react';
import { isPrimaryPress } from '@ore-ui-react/core';
import type { Id, Text } from '../protocol';
import * as engine from '../runtime/engine';
import { hideTip, setTipTouch, showTip } from './Tooltip';

export type PressProps = Record<string, unknown>;

const TOUCH_WINDOW = 700; // ms between a touch event and the mousedown of the same tap

let touchAt = -Infinity; // when the last touch began
let touchSeen = false; // whether the presses come from a touch screen (until the mouse moves alone)
let armed: Element | null = null; // the element whose tooltip the last tap showed; the next tap on it presses it
let claimed: Event | null = null; // the mousedown an element answered
let listening = false;

function listen(): void {
  if (listening) return;
  listening = true;
  document.addEventListener('touchstart', (ev) => {
    touchAt = Date.now();
    touchSeen = true;
    setTipTouch(true);
    if (armed && !armed.contains(ev.target as Node)) {
      armed = null;
      hideTip();
    }
  });
  // A key is no touch: the mousedown the keyboard selection dispatches for Enter presses at once.
  document.addEventListener(
    'keydown',
    () => {
      touchSeen = false;
      touchAt = -Infinity;
      armed = null;
      setTipTouch(false);
    },
    true,
  );
  // A mouse that moves with no touch about is a mouse: presses go through at once again.
  document.addEventListener('mousemove', () => {
    if (touchSeen && Date.now() - touchAt > TOUCH_WINDOW) {
      touchSeen = false;
      armed = null;
      setTipTouch(false);
    }
  });
}

// usePressable returns the props that make an element answer a press with `id` (the left button) or
// `alt` (the right button). Gameface does not deliver click to plain divs, so presses fire on mousedown.
// On a touch screen an element with a tip shows it at the first tap and presses at the second, unless
// it is `quick`. Gameface sends a mousedown for a tap too, in an order the devices do not agree on, so
// the decision waits a moment.
export function usePressable(p: { id?: Id; alt?: Id; tip?: Text; quick?: boolean }): PressProps {
  listen();
  const cur = useRef(p);
  cur.current = p;
  const props: PressProps = {
    onMouseDown: (ev: MouseEvent) => {
      const { id, alt } = cur.current;
      if (claimed === ev || (!isPrimaryPress(ev) && ev.button !== 2)) return;
      const el = ev.currentTarget as HTMLElement;
      if (ev.button === 2) {
        if (!alt) return;
        claimed = ev;
        engine.notePress(Math.round(ev.clientX), Math.round(ev.clientY));
        engine.send(alt, 'click');
        return;
      }
      if (!id) return;
      claimed = ev;
      engine.notePress(Math.round(ev.clientX), Math.round(ev.clientY));
      const at = Date.now();
      setTimeout(
        () => {
          const t = touchSeen || Math.abs(touchAt - at) < TOUCH_WINDOW;
          const now = cur.current;
          if (t && now.tip && !now.quick && armed !== el) {
            armed = el;
            showTip(el, now.tip);
            return;
          }
          armed = null;
          if (t) hideTip();
          engine.send(now.id ?? id, 'click');
        },
        touchSeen ? 0 : 40,
      );
    },
    onMouseEnter: (ev: MouseEvent) => {
      const tip = cur.current.tip;
      if (tip) showTip(ev.currentTarget as HTMLElement, tip);
    },
    onMouseLeave: (ev: MouseEvent) => {
      // The tooltip a tap showed stays until the next tap.
      if (armed !== ev.currentTarget && cur.current.tip) hideTip();
    },
  };
  // dom-fiber only passes a fixed set of data-* props, so the attribute goes on through a ref.
  props.ref = (e: Element | null) => {
    if (!e) return;
    if (cur.current.id) e.setAttribute('data-oui-press', '');
    else e.removeAttribute('data-oui-press');
  };
  return props;
}
