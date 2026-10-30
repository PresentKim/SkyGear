import { useEffect } from 'react';
import type { Keys } from '../protocol';
import { send } from '../runtime/engine';

const KEY_NAMES: Record<number, keyof Keys> = {
  38: 'up', 87: 'up', 40: 'down', 83: 'down', 37: 'left', 65: 'left', 39: 'right', 68: 'right', 13: 'ok', 32: 'ok',
};
// Gameface may not mark the repeats of a held key.
const KEY_REPEAT_MS = 120;
const SELECTOR = '[data-oui-press]';

/** isBack reports whether a key event is Esc or a phone's back button (KEYCODE_BACK 4, browser back 166). */
export function isBack(ev: KeyboardEvent): boolean {
  return ev.keyCode === 27 || ev.keyCode === 4 || ev.keyCode === 166 ||
    ev.key === 'Escape' || ev.key === 'GoBack' || ev.key === 'BrowserBack';
}

function rootEl(): HTMLElement | null {
  return document.getElementById('oui-ddui');
}

function navigable(e: HTMLElement): boolean {
  const root = rootEl();
  if (!root || !root.contains(e)) return false;
  const r = e.getBoundingClientRect();
  return r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0 &&
    r.top < window.innerHeight && r.left < window.innerWidth;
}

function list(): HTMLElement[] {
  const root = rootEl();
  if (!root) return [];
  return Array.from(root.querySelectorAll<HTMLElement>(SELECTOR)).filter(navigable);
}

function middle(e: HTMLElement) {
  const r = e.getBoundingClientRect();
  return { x: (r.left + r.right) / 2, y: (r.top + r.bottom) / 2, r };
}

// nearest is the element nearest to the point at, or the top left one without a point.
function nearest(els: HTMLElement[], at: { x: number; y: number } | null): HTMLElement | null {
  let best: HTMLElement | null = null;
  let bestScore = Infinity;
  for (const e of els) {
    const m = middle(e);
    const score = at ? Math.abs(m.x - at.x) + Math.abs(m.y - at.y) : m.y * 4 + m.x;
    if (score < bestScore) {
      bestScore = score;
      best = e;
    }
  }
  return best;
}

// revealIn scrolls the scrolling boxes around e so that e shows.
function revealIn(e: HTMLElement): void {
  const root = rootEl();
  for (let a = e.parentElement; a && a !== root; a = a.parentElement) {
    if (!/(^| )(oui-scroll|mc-scroll)( |$)/.test(a.className) || a.scrollHeight <= a.clientHeight) continue;
    const r = e.getBoundingClientRect();
    const box = a.getBoundingClientRect();
    if (r.top < box.top) a.scrollTop -= box.top - r.top;
    else if (r.bottom > box.bottom) a.scrollTop += r.bottom - box.bottom;
  }
}

// The selection shows the tooltip the way the pointer does: the element's own hover handlers draw it.
function hover(e: HTMLElement, on: boolean): void {
  for (const t of on ? ['mouseover', 'mouseenter'] : ['mouseout', 'mouseleave']) {
    e.dispatchEvent(new MouseEvent(t, { bubbles: t === 'mouseover' || t === 'mouseout' }));
  }
}

/** useKeyboard answers the screen's keys and otherwise moves a keyboard selection over the pressable elements. */
export function useKeyboard(keys: Keys | undefined): void {
  useEffect(() => {
    let focus: HTMLElement | null = null;
    let kbOn = false;
    let kbAt: { x: number; y: number } | null = null;
    const keyAt: Record<string, number> = {};
    let mouseAt: string | null = null;

    const unmark = (e: HTMLElement) => {
      e.classList.remove('oui-kb');
      hover(e, false);
    };
    const select = (e: HTMLElement | null) => {
      if (!e) return;
      if (focus && focus !== e) unmark(focus);
      focus = e;
      e.classList.add('oui-kb');
      revealIn(e);
      const m = middle(e);
      kbAt = { x: m.x, y: m.y };
      hover(e, true);
    };
    const press = (e: HTMLElement) => {
      const m = middle(e);
      e.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, button: 0, clientX: m.x, clientY: m.y }));
    };

    const navKey = (name: keyof Keys) => {
      if (name === 'ok') {
        if (focus && navigable(focus)) press(focus);
        return;
      }
      const els = list();
      if (!els.length) return;
      kbOn = true;
      if (!focus || !navigable(focus)) return select(nearest(els, kbAt));
      const from = middle(focus);
      let best: HTMLElement | null = null;
      let bestScore = Infinity;
      for (const p of els) {
        if (p === focus) continue;
        const to = middle(p);
        let along: number;
        let across: number;
        if (name === 'left' || name === 'right') {
          along = name === 'right' ? to.x - from.x : from.x - to.x;
          const overlap = to.r.top < from.r.bottom && from.r.top < to.r.bottom;
          across = overlap ? 0 : Math.abs(to.y - from.y);
        } else {
          along = name === 'down' ? to.y - from.y : from.y - to.y;
          const overlap = to.r.left < from.r.right && from.r.left < to.r.right;
          across = overlap ? 0 : Math.abs(to.x - from.x);
        }
        if (along <= 1) continue;
        const score = along + 2 * across;
        if (score < bestScore) {
          bestScore = score;
          best = p;
        }
      }
      if (best) select(best);
    };

    const onKey = (ev: KeyboardEvent) => {
      const name = KEY_NAMES[ev.keyCode];
      if (!name) return;
      const t = ev.target instanceof Element ? ev.target.tagName : '';
      if (t === 'INPUT' || t === 'TEXTAREA') return;
      const now = Date.now();
      if (now - (keyAt[name] || 0) < KEY_REPEAT_MS) return;
      keyAt[name] = now;
      ev.preventDefault();
      const id = keys && keys[name];
      if (id) send(id, 'click');
      else navKey(name);
    };
    const onBack = (ev: KeyboardEvent) => {
      if (isBack(ev)) send('close', 'key');
    };
    // The vanilla inventory takes half a stack with the right button, which must not open a menu.
    const onMenu = (ev: Event) => ev.preventDefault();
    // A mouse moving takes the keyboard selection away; a mousemove where the pointer already was does not.
    const dropFocus = (ev: Event) => {
      const m = ev as MouseEvent;
      const at = m.clientX !== undefined ? m.clientX + ',' + m.clientY : '';
      const moved = ev.type !== 'mousemove' || (mouseAt !== null && at !== mouseAt);
      mouseAt = at;
      if (!kbOn || !moved) return;
      kbOn = false;
      if (focus) unmark(focus);
      focus = null;
    };

    // keepFocus selects again, once the screen is laid out, after a patch or a new screen.
    let timer: ReturnType<typeof setTimeout> | undefined;
    const keepFocus = () => {
      if (!kbOn || timer !== undefined) return;
      timer = setTimeout(() => {
        timer = undefined;
        if (!kbOn) return;
        if (focus && navigable(focus)) {
          focus.classList.add('oui-kb');
          return;
        }
        focus = null;
        const els = list();
        if (els.length) select(nearest(els, kbAt));
      }, 30);
    };
    const root = rootEl();
    const observer = root && typeof MutationObserver !== 'undefined' ? new MutationObserver(keepFocus) : null;
    if (root && observer) observer.observe(root, { childList: true, subtree: true, attributes: true, attributeFilter: ['class'] });

    document.addEventListener('keydown', onBack);
    document.addEventListener('keydown', onKey);
    document.addEventListener('contextmenu', onMenu);
    document.addEventListener('mousemove', dropFocus);
    document.addEventListener('touchstart', dropFocus);
    return () => {
      document.removeEventListener('keydown', onBack);
      document.removeEventListener('keydown', onKey);
      document.removeEventListener('contextmenu', onMenu);
      document.removeEventListener('mousemove', dropFocus);
      document.removeEventListener('touchstart', dropFocus);
      observer?.disconnect();
      if (timer !== undefined) clearTimeout(timer);
      if (focus) unmark(focus);
    };
  }, [keys && keys.up, keys && keys.down, keys && keys.left, keys && keys.right, keys && keys.ok]);
}
