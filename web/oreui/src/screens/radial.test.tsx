import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render } from '@react-facet/dom-fiber-testing-library';
import type { RadialData } from '../protocol';

const sent: [string, string][] = [];
vi.mock('../runtime/engine', () => ({ send: (id: string, kind: string) => sent.push([id, kind]) }));
vi.mock('../text/Rich', () => ({
  InkContext: { Provider: (p: { children?: unknown }) => p.children },
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
}));

import { RadialScreen, fanBars } from './radial';

const item = (id: string, label: string, extra: Partial<RadialData['items'][number]> = {}) => ({ id, label, value: '', info: label + ' info', icon: '', on: false, ...extra });

const data = (back = 'close'): RadialData => ({
  title: 'Stone wand',
  line: 'Blocks at once 9',
  info: 'pick one',
  items: [item('a', 'Lock', { value: 'None' }), item('b', 'Match', { on: true }), item('', 'Undo'), item('d', 'Random')],
  back,
  backTip: 'Back',
});

const key = (keyCode: number) => act(() => {
  window.dispatchEvent(new KeyboardEvent('keydown', { keyCode, bubbles: true } as KeyboardEventInit));
});

describe('radial', () => {
  beforeEach(() => {
    sent.length = 0;
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue({ left: 0, top: 0, width: 240, height: 240, right: 240, bottom: 240, x: 0, y: 0, toJSON: () => ({}) });
  });
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('draws a sector per item over the RadialMenu with the value, the current one and the dim one', () => {
    const { container } = render(<RadialScreen data={data()} />);
    expect(container.querySelector('.oui-radialmenu')).not.toBeNull();
    expect(container.querySelectorAll('.oui-radialmenu-item').length).toBe(4);
    const sectors = container.querySelectorAll('.mc-radial-item');
    expect(sectors.length).toBe(4);
    expect(sectors[0].querySelector('.mc-radial-value')?.textContent).toBe('None');
    expect(sectors[1].className).toContain('cur');
    expect(sectors[2].className).toContain('off');
    expect(container.querySelector('.mc-radial-info')?.textContent).toBe('pick one');
  });

  it('picks the sector under the pointer and shows its line while it is chosen', () => {
    const { container } = render(<RadialScreen data={data()} />);
    const ring = container.querySelector('.mc-radial-ring') as HTMLElement;
    // right of the middle: the second of four sectors, clockwise from the top
    fireEvent.mouseMove(ring, { clientX: 220, clientY: 120 });
    expect(container.querySelectorAll('.mc-radial-item')[1].className).toContain('chosen');
    expect(container.querySelector('.mc-radial-info')?.textContent).toBe('Match info');
    fireEvent.mouseDown(container.querySelector('.oui-radialmenu') as HTMLElement, { button: 0, clientX: 220, clientY: 120 });
    expect(sent).toEqual([['b', 'click']]);
  });

  it('moves the choice with the arrows, picks with Enter, skips a dim one and goes back with Backspace', () => {
    const { container } = render(<RadialScreen data={data('back1')} />);
    key(39); // right: the first
    key(39);
    expect(container.querySelectorAll('.mc-radial-item')[1].className).toContain('chosen');
    key(13);
    expect(sent).toEqual([['b', 'click']]);
    key(37); // left, back to the first
    key(37); // left again: wraps to the last
    expect(container.querySelectorAll('.mc-radial-item')[3].className).toContain('chosen');
    key(51); // 3 chooses the dim undo, 3 again does nothing
    key(51);
    expect(sent.length).toBe(1);
    key(8);
    expect(sent[1]).toEqual(['back1', 'click']);
  });

  it('draws the sectors as ring sectors: the chosen one filled, a rim on the current one, a spoke between each two', () => {
    const { container } = render(<RadialScreen data={data()} />);
    expect(container.querySelectorAll('.mc-radial-spoke').length).toBe(4);
    expect(container.querySelectorAll('.mc-radial-fan.chosen').length).toBe(0);
    // the current one (Match, the second) has its rim
    const rim = container.querySelectorAll('.mc-radial-sector')[1].querySelectorAll('.mc-radial-fan.cur');
    expect(rim.length).toBe(fanBars(45, 135, 11.25, 11.85).length);
    fireEvent.mouseMove(container.querySelector('.mc-radial-ring') as HTMLElement, { clientX: 120, clientY: 20 });
    expect(container.querySelectorAll('.mc-radial-sector')[0].querySelectorAll('.mc-radial-fan.chosen').length).toBeGreaterThan(0);
  });

  it('fanBars cover the sector edge to edge', () => {
    const bars = fanBars(-45, 45, 4, 11.85);
    expect(bars.length).toBe(30);
    expect(bars[0].angle).toBeCloseTo(-43.5);
    expect(bars[bars.length - 1].angle).toBeCloseTo(43.5);
    // each bar at least as wide as its arc at the outer radius
    expect(bars[0].width).toBeGreaterThan(2 * 11.85 * Math.tan((3 * Math.PI) / 360));
  });

  it('the middle button closes the menu at the top', () => {
    const { container } = render(<RadialScreen data={data()} />);
    const hub = container.querySelector('.mc-radial-hub') as HTMLElement;
    expect(hub).not.toBeNull();
    expect(hub.querySelector('img')?.getAttribute('src')).toContain('close_button');
  });
});
