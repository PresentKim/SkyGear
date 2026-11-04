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
const click = (b: 0 | 1, shift = false, dbl = false, split = false) => ({ b, shift, dbl, split });

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
  it('presses at once', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 0), 'mousedown', 2);
    expect(send).toHaveBeenCalledWith('net:0', 'slot', click(1));
  });
  it('sends shift', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 1), 'mousedown', 0, true);
    expect(send).toHaveBeenCalledWith('inv:0', 'slot', click(0, true));
  });
  it('a second left press within 250ms is a double click', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 1), 'mousedown');
    vi.advanceTimersByTime(200);
    mouse(slotEl(container, 1), 'mousedown');
    expect(send).toHaveBeenLastCalledWith('inv:0', 'slot', click(0, false, true));
  });
  it('a right press between breaks the double click', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 1), 'mousedown');
    mouse(slotEl(container, 1), 'mousedown', 2);
    mouse(slotEl(container, 1), 'mousedown');
    expect(send).toHaveBeenLastCalledWith('inv:0', 'slot', click(0));
  });
  it('a shift double click keeps shift', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(slotEl(container, 1), 'mousedown', 0, true);
    mouse(slotEl(container, 1), 'mousedown', 0, true);
    expect(send).toHaveBeenLastCalledWith('inv:0', 'slot', click(0, true, true));
  });
  it('a held stack press places at once and the drag spreads live', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    act(() => {
      mouse(slotEl(container, 1), 'mousedown');
    });
    expect(send).toHaveBeenCalledWith('inv:0', 'slot', click(0, false, false, true));
    act(() => {
      mouse(slotEl(container, 2), 'mouseenter');
    });
    expect(send).toHaveBeenLastCalledWith('inv:0', 'drag', { b: 0, ids: ['inv:0', 'inv:1'] });
    act(() => {
      mouse(slotEl(container, 3), 'mouseenter');
    });
    expect(send).toHaveBeenLastCalledWith('inv:0', 'drag', { b: 0, ids: ['inv:0', 'inv:1', 'inv:2'] });
    expect(container.querySelectorAll('.mc-inv-dragged').length).toBe(3);
    document.dispatchEvent(new MouseEvent('mouseup'));
    expect(send).toHaveBeenCalledTimes(3);
  });
  it('a press outside the window drops the held stack', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    mouse(container.querySelector('.mc-screen')!, 'mousedown', 2);
    expect(send).toHaveBeenCalledWith('drop', 'drop', { b: 1 });
  });
  it('a press outside with nothing held drops nothing', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    mouse(container.querySelector('.mc-screen')!, 'mousedown');
    expect(send).not.toHaveBeenCalled();
  });
  it('a press inside the window drops nothing', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    mouse(container.querySelector('.mc-title')!, 'mousedown');
    expect(send).not.toHaveBeenCalled();
  });
  const book = (field = '') => ({
    tabs: [
      { id: '', tip: 'Construction', icon: '', on: true, end: false },
      { id: 't1', tip: 'Equipment', icon: '', on: false, end: false },
      { id: 't4', tip: 'All recipes', icon: '', on: false, end: true },
    ],
    title: 'Construction', field, query: '', placeholder: '',
    filter: { id: 'f', tip: 'All', icon: '', glyph: '', quick: true }, note: '',
    slots: [{ ...slot('book:0', 'stick'), dim: true }],
  });
  it('draws the recipe book and presses its recipes like slots', () => {
    const { container } = render(<InventoryScreen data={{ ...data(null), book: book() }} />);
    const recipe = container.querySelector('.mc-inv-book .mc-inv-slot')!;
    expect(recipe.className).toContain('mc-inv-dim');
    mouse(recipe, 'mousedown', 0, true);
    expect(send).toHaveBeenCalledWith('book:0', 'slot', click(0, true));
  });
  it('draws the book tabs with the search tab last and the open tab label', () => {
    const { container } = render(<InventoryScreen data={{ ...data(null), book: book() }} />);
    const tabs = container.querySelectorAll('.mc-inv-book-tabs > *');
    expect(tabs[0].className).toContain('on');
    expect(tabs[2].className).toContain('mc-inv-book-fill');
    expect(container.querySelector('.mc-inv-book-label')!.textContent).toBe('Construction');
    expect(container.querySelector('.mc-inv-book input')).toBeNull();
  });
  it('a search tab draws the book search box', () => {
    const { container } = render(<InventoryScreen data={{ ...data(null), book: book('booksearch') }} />);
    expect(container.querySelector('.mc-inv-book input')).not.toBeNull();
    expect(container.querySelector('.mc-inv-book-label')).toBeNull();
  });
  it('puts the info line beside the first heading under the grow row', () => {
    const { container } = render(<InventoryScreen data={{ ...data(null), info: '3 kinds' }} />);
    const row = container.querySelector('.mc-inv-heading-row')!;
    expect(row.textContent).toContain('Inventory');
    expect(row.querySelector('.mc-inv-info')!.textContent).toBe('3 kinds');
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
    expect(send).toHaveBeenCalledWith('inv:0', 'slot', click(0));
  });
  it('a held touch is a right click at once', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    touch(slotEl(container, 0), 'touchstart');
    vi.advanceTimersByTime(gesture.HOLD_MS + 10);
    expect(send).toHaveBeenCalledWith('net:0', 'slot', click(1));
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
    expect(send).toHaveBeenCalledWith('inv:0', 'slot', click(0, true));
  });
  it('a tap with a held stack places it', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    touch(slotEl(container, 2), 'touchstart');
    touch(document.body, 'touchend');
    expect(send).toHaveBeenCalledWith('inv:1', 'slot', click(0, false, false, true));
  });
});
