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
const slot = (id: string, icon = ''): InvSlot => ({ id, icon, count: '', tip: 'Dirt', rare: 0, empty: '' });

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
const clickOn = (el: Element, button = 0, shiftKey = false) => {
  mouse(el, 'mousedown', button, shiftKey);
  mouse(el, 'mouseup', button, shiftKey);
};
const key = (type: string, k: string) => document.dispatchEvent(new KeyboardEvent(type, { key: k }));
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
    clickOn(slotEl(container, 1));
    vi.advanceTimersByTime(200);
    clickOn(slotEl(container, 1));
    expect(send).toHaveBeenLastCalledWith('inv:0', 'slot', click(0, false, true));
  });
  it('a right press between breaks the double click', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    clickOn(slotEl(container, 1));
    clickOn(slotEl(container, 1), 2);
    clickOn(slotEl(container, 1));
    expect(send).toHaveBeenLastCalledWith('inv:0', 'slot', click(0));
  });
  it('a shift double click keeps shift', () => {
    const { container } = render(<InventoryScreen data={data(null)} />);
    clickOn(slotEl(container, 1), 0, true);
    clickOn(slotEl(container, 1), 0, true);
    expect(send).toHaveBeenLastCalledWith('inv:0', 'slot', click(0, true, true));
  });
  it('a press of a button still down is the drag reaching that slot', () => {
    const { container } = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    act(() => {
      mouse(slotEl(container, 1), 'mousedown');
    });
    act(() => {
      mouse(slotEl(container, 2), 'mousedown');
    });
    expect(send).toHaveBeenLastCalledWith('inv:0', 'drag', { b: 0, ids: ['inv:0', 'inv:1'] });
    expect(send).toHaveBeenCalledTimes(2);
  });
  it('a drag that placed the whole stack keeps it held at 0 until release', () => {
    const r = render(<InventoryScreen data={data(slot('', 'dirt'))} />);
    act(() => {
      mouse(slotEl(r.container, 1), 'mousedown');
    });
    r.rerender(<InventoryScreen data={data(null)} />);
    expect(r.container.querySelector('.mc-inv-cursor .mc-count')!.textContent).toBe('0');
    act(() => {
      mouse(slotEl(r.container, 2), 'mouseenter');
    });
    expect(send).toHaveBeenLastCalledWith('inv:0', 'drag', { b: 0, ids: ['inv:0', 'inv:1'] });
    act(() => {
      document.dispatchEvent(new MouseEvent('mouseup'));
    });
    expect(r.container.querySelector('.mc-inv-cursor')).toBeNull();
  });
  it('a hold screen hears buttons and shift held and let go', () => {
    const { container } = render(<InventoryScreen data={{ ...data(null), hold: true }} />);
    key('keydown', 'Shift');
    clickOn(slotEl(container, 1), 0, true);
    clickOn(slotEl(container, 1), 0, true);
    key('keyup', 'Shift');
    expect(send.mock.calls.filter((c) => c[1] === 'hold')).toEqual([['hold', 'hold', true], ['hold', 'hold', false]]);
    expect(send.mock.calls[0][1]).toBe('hold');
  });
  it('a click on a hold screen is held from before its press to the release', () => {
    const { container } = render(<InventoryScreen data={{ ...data(null), hold: true }} />);
    clickOn(slotEl(container, 1));
    expect(send.mock.calls.map((c) => c[1])).toEqual(['hold', 'slot', 'hold']);
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

describe('inventory buttons', () => {
  const btn = (id: string) => ({ id, tip: id, icon: '', glyph: '', quick: true });
  it('draws the tools left and each row bar level with its row', () => {
    const d = data(null);
    d.tools = [btn('t1'), btn('t2')];
    d.rows[0].bar = [btn('b1')];
    d.rows[1].bar = [btn('b2'), btn('b3')];
    d.rows[1].barTop = true;
    const { container } = render(<InventoryScreen data={d} />);
    expect(container.querySelectorAll('.mc-tools > .mc-square-button').length).toBe(2);
    expect(container.querySelector('.mc-inv-growbox > .mc-row-bar:not(.top)')).not.toBeNull();
    expect(container.querySelectorAll('.mc-inv-row > .mc-row-bar.top > .mc-square-button').length).toBe(2);
    mouse(container.querySelectorAll('.mc-square-button')[0], 'mousedown');
    vi.advanceTimersByTime(50);
    expect(send).toHaveBeenCalledWith('t1', 'click');
  });
  it('draws a gauge filled from the bottom and the progress icon', () => {
    const d = data(null);
    d.rows[1].parts = [
      { area: '', cols: 0, slots: [], big: false, arrow: false, button: null, gauge: { kind: 'energy', fill: 40, tip: '40 RE' } },
      { area: '', cols: 0, slots: [], big: false, arrow: false, button: null, progress: { icon: 'tex:textures/items/progress_5', tip: '50%' } },
    ];
    const { container } = render(<InventoryScreen data={d} />);
    const fill = container.querySelector('.mc-gauge.energy .mc-gauge-fill') as HTMLElement;
    expect(fill.style.height).toBe('40%');
    expect(container.querySelector('.mc-progress img')).not.toBeNull();
  });
  it('a centred row keeps its gauge left and balances it with a hidden one', () => {
    const d = data(null);
    d.rows[1].center = true;
    d.rows[1].parts = [{ area: '', cols: 0, slots: [], big: false, arrow: false, button: null, gauge: { kind: 'energy', fill: 0, tip: '' } }, ...d.rows[1].parts];
    const { container } = render(<InventoryScreen data={d} />);
    const row = container.querySelectorAll('.mc-inv-row')[1];
    expect(row.children[0].className).toContain('energy');
    expect(row.children[1].className).toContain('mc-inv-center');
    expect(row.children[1].querySelector('.mc-inv-slot')).not.toBeNull();
    expect(row.children[2].className).toContain('mc-gauge-spacer');
  });
  it('a filler slot is dark and takes no presses', () => {
    const d = data(null);
    d.rows[1].parts[0].slots[2] = { ...slot('inv:2'), off: true };
    const { container } = render(<InventoryScreen data={d} />);
    const filler = container.querySelectorAll('.mc-inv-part')[1].querySelectorAll('.mc-inv-slot')[2];
    expect(filler.className).toContain('mc-inv-off');
    mouse(filler, 'mousedown');
    expect(send).not.toHaveBeenCalled();
  });
  it('off slots are dark and take no presses', () => {
    const d = data(null);
    d.rows[0].parts[0].off = true;
    const { container } = render(<InventoryScreen data={d} />);
    const off = container.querySelector('.mc-inv-part.off .mc-inv-slot')!;
    expect(off.className).toContain('mc-inv-off');
    mouse(off, 'mousedown');
    expect(send).not.toHaveBeenCalled();
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
