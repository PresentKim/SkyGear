import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act } from 'react';
import { render } from '@react-facet/dom-fiber-testing-library';
import type { Chrome, InventoryData, InvSlot } from '../protocol';
import { InventoryScreen } from '../screens/inventory';
import * as gesture from './gesture';

vi.mock('../runtime/engine', () => ({ send: vi.fn(), notePress: vi.fn(), registerInput: vi.fn(() => () => {}) }));
import * as engine from '../runtime/engine';

const send = engine.send as unknown as ReturnType<typeof vi.fn>;
const chrome: Chrome = { title: 'Terminal', tabs: [], bar: [] };
const slot = (id: string, icon = ''): InvSlot => ({ id, icon, count: '', tip: 'Dirt', deco: '', empty: '' });

function data(cursor: InvSlot | null): InventoryData {
  return {
    chrome, field: '', query: '', placeholder: '', info: '', side: [], quick: 'Quick', cursor,
    rows: [
      { heading: '', grow: true, note: '', parts: [{ area: 'net', cols: 9, slots: [slot('net:0', 'dirt')], big: false, arrow: false, button: null }] },
      { heading: 'Inventory', grow: false, note: '', parts: [{ area: 'inv', cols: 9, slots: [slot('inv:0'), slot('inv:1'), slot('inv:2')], big: false, arrow: false, button: null }] },
    ],
  };
}

function slotEl(container: HTMLElement, i: number): Element {
  return container.querySelectorAll('.mc-inv-slot')[i];
}

const mouse = (el: Element, type: string, button = 0, shiftKey = false) =>
  el.dispatchEvent(new MouseEvent(type, { bubbles: true, button, shiftKey }));
const touch = (el: Element, type: string) => el.dispatchEvent(new Event(type, { bubbles: true }));

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(10_000);
  send.mockClear();
  gesture.reset();
});
afterEach(() => {
  vi.useRealTimers();
});

describe('inventory mouse', () => {
  it('clicks at once with nothing carried', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 0), 'mousedown', 2);
    expect(send).toHaveBeenCalledWith('net:0', 'slot', { b: 1, shift: false, dbl: false });
  });
  it('sends shift', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 1), 'mousedown', 0, true);
    expect(send).toHaveBeenCalledWith('inv:0', 'slot', { b: 0, shift: true, dbl: false });
  });
  it('a second click soon after is a double click', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 1), 'mousedown');
    vi.advanceTimersByTime(100);
    mouse(slotEl(container, 1), 'mousedown');
    expect(send).toHaveBeenLastCalledWith('inv:0', 'slot', { b: 0, shift: false, dbl: true });
  });
  it('drags across slots with a carried stack', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    act(() => {
      mouse(slotEl(container, 1), 'mousedown');
      mouse(slotEl(container, 2), 'mouseenter');
      mouse(slotEl(container, 3), 'mouseenter');
    });
    expect(container.querySelectorAll('.mc-inv-dragged').length).toBe(3);
    document.dispatchEvent(new MouseEvent('mouseup'));
    expect(send).toHaveBeenCalledWith('inv:0', 'drag', { b: 0, ids: ['inv:0', 'inv:1', 'inv:2'] });
  });
  it('a press and release on one slot with a carried stack is a click', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    mouse(slotEl(container, 2), 'mousedown', 2);
    expect(send).not.toHaveBeenCalled();
    document.dispatchEvent(new MouseEvent('mouseup'));
    expect(send).toHaveBeenCalledWith('inv:1', 'slot', { b: 1, shift: false, dbl: false });
  });
  it('draws the cursor stack', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    expect(container.querySelector('.mc-inv-cursor')).not.toBeNull();
  });
});

describe('inventory touch', () => {
  it('a tap is a left click', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    touch(slotEl(container, 1), 'touchstart');
    vi.advanceTimersByTime(100);
    touch(document.body, 'touchend');
    expect(send).toHaveBeenCalledWith('inv:0', 'slot', { b: 0, shift: false, dbl: false });
  });
  it('a held touch is a right click at once', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    touch(slotEl(container, 0), 'touchstart');
    vi.advanceTimersByTime(gesture.HOLD_MS + 10);
    expect(send).toHaveBeenCalledWith('net:0', 'slot', { b: 1, shift: false, dbl: false });
    touch(document.body, 'touchend');
    expect(send).toHaveBeenCalledTimes(1);
  });
  it('the mousedown of a tap is not a second click', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    touch(slotEl(container, 1), 'touchstart');
    touch(document.body, 'touchend');
    mouse(slotEl(container, 1), 'mousedown');
    expect(send).toHaveBeenCalledTimes(1);
  });
  it('the quick move switch makes taps shift clicks', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    touch(container.querySelector('.mc-inv-quick')!, 'touchstart');
    touch(slotEl(container, 1), 'touchstart');
    touch(document.body, 'touchend');
    expect(send).toHaveBeenCalledWith('inv:0', 'slot', { b: 0, shift: true, dbl: false });
  });
});
