import * as engine from '../runtime/engine';

// The Bedrock inventory's hands on the slots of an inventory screen, for the mouse and for touch.
//
// Mouse (as the Bedrock client does): every press acts at once. A press with a stack held is sent
// as a split press; each further slot the pointer reaches before the release sends the whole drag
// so far, which the server spreads again (evenly for the left button, one each for the right). A
// second left press on the same slot within DOUBLE_MS, with the same shift state and no right
// press between, is a double click (the server gathers when a stack is held, and with shift moves
// every stack of the clicked item). A press outside the window drops the held stack (one item with
// the right button).
//
// Touch: a tap is the left click and a touch held for HOLD_MS the right click (with nothing held it
// fires while the finger is still down). With a stack held, a finger moved across slots drags, one
// each when it was held first. A finger that moves with nothing held scrolls and clicks nothing.
// The quick move switch makes every click a shift click.
//
// A screen that asks for it (InventoryData.hold) hears a 'hold' event when the player starts holding
// a mouse button, a finger or shift, and another when every one is let go; it is sent before the
// press that starts it.

export const HOLD_MS = 400;
export const DOUBLE_MS = 250;
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
  split: boolean; // the press was sent as a split press
  timer: ReturnType<typeof setTimeout> | null;
}

let g: Gesture | null = null;
let last: { id: string; at: number; shift: boolean } | null = null;
let touchAt = -Infinity;
let carrying = false;
let quick = false;
let touchTip = '';
const buttons = new Set<number>();
let shiftDown = false;
let reportHold = false;
let holdSent = false;
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

/** setReportHold tells whether the screen hears the hold events. */
export function setReportHold(v: boolean): void {
  reportHold = v;
}

function sendHold(): void {
  const on = buttons.size > 0 || shiftDown || (!!g && g.touch);
  if (!reportHold || on === holdSent) return;
  holdSent = on;
  engine.send('hold', 'hold', on);
}

/** setShift follows the shift key. */
export function setShift(v: boolean): void {
  shiftDown = v;
  sendHold();
}

/** splitHeld tells whether a drag of a held stack is under way, which keeps the stack held at 0. */
export function splitHeld(): boolean {
  return !!g && g.split;
}

function onButtonUp(ev: MouseEvent): void {
  buttons.delete(ev.button);
  if (buttons.size === 0) document.removeEventListener('mouseup', onButtonUp);
  sendHold();
}

export function isQuick(): boolean {
  return quick;
}

export function toggleQuick(): void {
  quick = !quick;
  notify();
}

/** touchedRecently tells whether a touch began within the time a tap's mouse events follow it. */
export function touchedRecently(): boolean {
  return Date.now() - touchAt < TOUCH_WINDOW;
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

// press sends one press on a slot; it reports whether it was a double click.
function press(id: string, b: 0 | 1, shiftKey: boolean, split: boolean): boolean {
  const shift = shiftKey || quick;
  const now = Date.now();
  if (b === 1) {
    last = null;
    engine.send(id, 'slot', { b, shift, dbl: false, split: split && !shift });
    return false;
  }
  const dbl = !!last && last.id === id && last.shift === shift && now - last.at <= DOUBLE_MS;
  last = dbl ? null : { id, at: now, shift };
  engine.send(id, 'slot', { b, shift, dbl, split: split && !shift && !dbl });
  return dbl;
}

function sendDrag(cur: Gesture): void {
  engine.send(cur.ids[0], 'drag', { b: cur.b, ids: cur.ids });
}

/** reset forgets any gesture under way (a new screen, a test). */
export function reset(): void {
  if (g && g.timer) clearTimeout(g.timer);
  g = null;
  last = null;
  touchAt = -Infinity;
  quick = false;
  touchTip = '';
  buttons.clear();
  document.removeEventListener('mouseup', onButtonUp);
  shiftDown = false;
  reportHold = false;
  holdSent = false;
  notify();
}

function onMouseUp(ev: MouseEvent): void {
  if (!g || g.touch || (ev.button === 2 ? 1 : 0) !== g.b) return;
  document.removeEventListener('mouseup', onMouseUp);
  g = null;
  notify();
}

export function mouseDown(id: string, ev: MouseEvent): void {
  if (Date.now() - touchAt < TOUCH_WINDOW) return;
  if (ev.button !== 0 && ev.button !== 2) return;
  if (buttons.has(ev.button)) {
    mouseEnter(id);
    return;
  }
  if (buttons.size === 0) document.addEventListener('mouseup', onButtonUp);
  buttons.add(ev.button);
  shiftDown = ev.shiftKey;
  sendHold();
  const b: 0 | 1 = ev.button === 2 ? 1 : 0;
  const dbl = press(id, b, ev.shiftKey, carrying);
  if (g) document.removeEventListener('mouseup', onMouseUp);
  g = null;
  if (carrying && !ev.shiftKey && !quick && !dbl) {
    g = { b, ids: [id], touch: false, start: Date.now(), x: ev.clientX, y: ev.clientY, moved: false, fired: false, split: true, timer: null };
    document.addEventListener('mouseup', onMouseUp);
  }
  notify();
}

export function mouseEnter(id: string): void {
  if (g && !g.touch && !g.ids.includes(id)) {
    g.ids.push(id);
    sendDrag(g);
    notify();
  }
}

/** pressOutside is a press outside the window: a held stack is dropped, one item with the right button. */
export function pressOutside(button: number): void {
  if (Date.now() - touchAt < TOUCH_WINDOW || !carrying) return;
  engine.send('drop', 'drop', { b: button === 2 ? 1 : 0 });
}

/** touchOutside is a tap outside the window: a held stack is dropped. */
export function touchOutside(): void {
  touchAt = Date.now();
  if (carrying) engine.send('drop', 'drop', { b: 0 });
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
  if (!(carrying || cur.split) || quick) return;
  const hit = slotAt(at.x, at.y);
  if (!hit || cur.ids.includes(hit)) return;
  if (!cur.split) {
    cur.b = Date.now() - cur.start >= HOLD_MS ? 1 : 0;
    cur.split = true;
    press(cur.ids[0], cur.b, false, true);
  }
  cur.ids.push(hit);
  sendDrag(cur);
  notify();
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
  if (!(cur.fired || cur.split || cur.moved)) press(cur.ids[0], Date.now() - cur.start >= HOLD_MS ? 1 : 0, false, carrying);
  sendHold();
}

export function touchStart(id: string, tip: string, ev: TouchEvent): void {
  touchAt = Date.now();
  const at = point(ev) || { x: 0, y: 0 };
  touchTip = tip;
  const cur: Gesture = { b: 0, ids: [id], touch: true, start: Date.now(), x: at.x, y: at.y, moved: false, fired: false, split: false, timer: null };
  g = cur;
  sendHold();
  if (!carrying && !quick) {
    cur.timer = setTimeout(() => {
      if (g === cur && !cur.moved) {
        cur.fired = true;
        press(id, 1, false, false);
      }
    }, HOLD_MS);
  }
  document.addEventListener('touchmove', onTouchMove);
  document.addEventListener('touchend', onTouchEnd);
  document.addEventListener('touchcancel', onTouchEnd);
  notify();
}
