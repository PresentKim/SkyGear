import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('react', () => ({ useEffect: (f: () => unknown) => { cleanups.push(f() as () => void); } }));
vi.mock('../runtime/engine', () => ({ send: vi.fn() }));

const cleanups: Array<() => void> = [];
import { send } from '../runtime/engine';
import { isBack, useKeyboard } from './keys';

function box(id: string, x: number, y: number): HTMLElement {
  const e = document.createElement('div');
  e.id = id;
  e.setAttribute('data-oui-press', '');
  e.getBoundingClientRect = () =>
    ({ left: x, top: y, right: x + 50, bottom: y + 50, width: 50, height: 50, x, y } as DOMRect);
  document.getElementById('oui-ddui')!.appendChild(e);
  return e;
}
function key(code: number, target: EventTarget = document, extra: KeyboardEventInit = {}) {
  const ev = new KeyboardEvent('keydown', { keyCode: code, bubbles: true, cancelable: true, ...extra } as KeyboardEventInit);
  target.dispatchEvent(ev);
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.mocked(send).mockClear();
  document.body.innerHTML = '<div id="oui-ddui"></div>';
});
afterEach(() => {
  cleanups.splice(0).forEach((c) => c && c());
  vi.useRealTimers();
});

describe('keys', () => {
  it('sends the answered key once per 120 ms', () => {
    useKeyboard({ up: 'o3' });
    key(38);
    expect(send).toHaveBeenCalledWith('o3', 'click');
    key(38);
    expect(send).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(130);
    key(38);
    expect(send).toHaveBeenCalledTimes(2);
  });

  it('sends close on Escape', () => {
    useKeyboard(undefined);
    key(27, document, { key: 'Escape' });
    expect(send).toHaveBeenCalledWith('close', 'key');
  });

  it('typing_no_keys', () => {
    useKeyboard({ up: 'o3', ok: 'o4' });
    const input = document.createElement('input');
    document.body.appendChild(input);
    key(38, input);
    key(13, input);
    expect(send).not.toHaveBeenCalled();
  });

  it('nav_first_key and enter_presses_selection', () => {
    useKeyboard(undefined);
    const a = box('a', 0, 0), b = box('b', 100, 0), c = box('c', 0, 100);
    box('d', 100, 100);
    const pressed: string[] = [];
    for (const e of [a, b, c]) e.addEventListener('mousedown', () => pressed.push(e.id));
    key(39);
    expect(a.classList.contains('oui-kb')).toBe(true);
    vi.advanceTimersByTime(130);
    key(39);
    expect(a.classList.contains('oui-kb')).toBe(false);
    expect(b.classList.contains('oui-kb')).toBe(true);
    vi.advanceTimersByTime(130);
    key(13);
    expect(pressed).toEqual(['b']);
    expect(send).not.toHaveBeenCalled();
  });

  it('back_codes', () => {
    const k = (o: object) => isBack(new KeyboardEvent('keydown', o as KeyboardEventInit));
    expect(k({ keyCode: 4 })).toBe(true);
    expect(k({ keyCode: 166 })).toBe(true);
    expect(k({ key: 'GoBack' })).toBe(true);
    expect(k({ key: 'BrowserBack' })).toBe(true);
    expect(k({ key: 'Escape' })).toBe(true);
    expect(k({ keyCode: 65 })).toBe(false);
  });
});
