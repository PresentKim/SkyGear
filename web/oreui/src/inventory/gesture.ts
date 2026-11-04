import * as engine from '../runtime/engine';

// The vanilla inventory's hands on the slots of an inventory screen, for the mouse and for touch.
//
// Mouse: a press on a slot with nothing on the cursor (or with shift) is a click at once; with a stack
// on the cursor the press waits for the release, and the slots the pointer crosses meanwhile make a
// drag. The right button is the right click; a second left click on the same slot soon after is a
// double click.
//
// Touch: a tap is the left click and a touch held for HOLD_MS the right click (with nothing on the
// cursor it fires while the finger is still down, like the game's split). With a stack on the cursor,
// a finger moved across slots drags (one each when it was held first). A finger that moves with
// nothing on the cursor scrolls and clicks nothing. The quick move switch makes every click a shift
// click.

export const HOLD_MS = 400;
export const DOUBLE_MS = 300;
const TOUCH_WINDOW = 700; // ms in which a mousedown belongs to the tap just made
const MOVE_PX = 8;

interface Gesture {
  b: 0 | 1;
  ids: string[];
  touch: boolean;
  start: number;
  x: number;
  y: number;
  moved: boolean;
  fired: boolean;
  timer: ReturnType<typeof setTimeout> | null;
}

let g: Gesture | null = null;
let last = { id: '', at: -Infinity };
let touchAt = -Infinity;
let carrying = false;
let quick = false;
let touchTip = '';
const slots = new Map<string, Element>();
const listeners = new Set<() => void>();

function notify(): void {
  listeners.forEach((f) => f());
}

/** subscribe calls f whenever the drag, the quick move switch or the touched slot's tip changes; it returns the unsubscribe. */
export function subscribe(f: () => void): () => void {
  listeners.add(f);
  return () => listeners.delete(f);
}

/** setCarrying tells the gestures whether the server's cursor holds a stack. */
export function setCarrying(v: boolean): void {
  carrying = v;
}

export function isQuick(): boolean {
  return quick;
}

export function toggleQuick(): void {
  quick = !quick;
  notify();
}

/** lastTouchTip is the tooltip of the slot touched last (a touch screen shows no tooltips over slots). */
export function lastTouchTip(): string {
  return touchTip;
}

/** dragIds are the slots of the drag under way. */
export function dragIds(): string[] {
  return g && g.ids.length > 1 ? g.ids : [];
}

/** registerSlot lets a touch drag find the slot under the finger; it returns the unregister. */
export function registerSlot(id: string, el: Element): () => void {
  slots.set(id, el);
  return () => {
    if (slots.get(id) === el) slots.delete(id);
  };
}

function slotAt(x: number, y: number): string {
  let hit = '';
  slots.forEach((el, id) => {
    const r = el.getBoundingClientRect();
    if (x >= r.left && x < r.right && y >= r.top && y < r.bottom) hit = id;
  });
  return hit;
}

function click(id: string, b: 0 | 1, shift: boolean): void {
  const now = Date.now();
  const dbl = b === 0 && !shift && last.id === id && now - last.at < DOUBLE_MS;
  last = dbl ? { id: '', at: -Infinity } : { id, at: now };
  engine.send(id, 'slot', { b, shift: shift || quick, dbl });
}

function end(cur: Gesture, b: 0 | 1): void {
  if (cur.ids.length > 1) engine.send(cur.ids[0], 'drag', { b, ids: cur.ids });
  else click(cur.ids[0], b, false);
}

/** reset forgets any gesture under way (a new screen, a test). */
export function reset(): void {
  if (g && g.timer) clearTimeout(g.timer);
  g = null;
  last = { id: '', at: -Infinity };
  touchAt = -Infinity;
  quick = false;
  touchTip = '';
  notify();
}

function onMouseUp(): void {
  document.removeEventListener('mouseup', onMouseUp);
  const cur = g;
  if (!cur || cur.touch) return;
  g = null;
  notify();
  end(cur, cur.b);
}

export function mouseDown(id: string, ev: MouseEvent): void {
  if (Date.now() - touchAt < TOUCH_WINDOW) return;
  if (ev.button !== 0 && ev.button !== 2) return;
  const b: 0 | 1 = ev.button === 2 ? 1 : 0;
  if (!carrying || ev.shiftKey || quick) {
    click(id, b, ev.shiftKey);
    return;
  }
  g = { b, ids: [id], touch: false, start: Date.now(), x: ev.clientX, y: ev.clientY, moved: false, fired: false, timer: null };
  document.addEventListener('mouseup', onMouseUp);
}

export function mouseEnter(id: string): void {
  if (g && !g.touch && !g.ids.includes(id)) {
    g.ids.push(id);
    notify();
  }
}

function point(ev: TouchEvent): { x: number; y: number } | null {
  const t = (ev.touches && ev.touches[0]) || (ev.changedTouches && ev.changedTouches[0]);
  return t ? { x: t.clientX, y: t.clientY } : null;
}

function onTouchMove(ev: TouchEvent): void {
  const cur = g;
  const at = point(ev);
  if (!cur || !cur.touch || !at) return;
  if (Math.abs(at.x - cur.x) > MOVE_PX || Math.abs(at.y - cur.y) > MOVE_PX) cur.moved = true;
  if (!carrying || quick) return;
  const hit = slotAt(at.x, at.y);
  if (hit && !cur.ids.includes(hit)) {
    cur.ids.push(hit);
    notify();
  }
}

function onTouchEnd(): void {
  document.removeEventListener('touchmove', onTouchMove);
  document.removeEventListener('touchend', onTouchEnd);
  document.removeEventListener('touchcancel', onTouchEnd);
  const cur = g;
  if (!cur || !cur.touch) return;
  g = null;
  touchAt = Date.now();
  if (cur.timer) clearTimeout(cur.timer);
  notify();
  if (cur.fired) return;
  const long = Date.now() - cur.start >= HOLD_MS;
  if (carrying && !quick && cur.ids.length > 1) {
    end(cur, long ? 1 : 0);
    return;
  }
  if (cur.moved) return;
  click(cur.ids[0], long ? 1 : 0, false);
}

export function touchStart(id: string, tip: string, ev: TouchEvent): void {
  touchAt = Date.now();
  const at = point(ev) || { x: 0, y: 0 };
  touchTip = tip;
  const cur: Gesture = { b: 0, ids: [id], touch: true, start: Date.now(), x: at.x, y: at.y, moved: false, fired: false, timer: null };
  g = cur;
  if (!carrying && !quick) {
    cur.timer = setTimeout(() => {
      if (g === cur && !cur.moved) {
        cur.fired = true;
        click(id, 1, false);
      }
    }, HOLD_MS);
  }
  document.addEventListener('touchmove', onTouchMove);
  document.addEventListener('touchend', onTouchEnd);
  document.addEventListener('touchcancel', onTouchEnd);
  notify();
}
