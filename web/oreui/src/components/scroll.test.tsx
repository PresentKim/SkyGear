import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';

const REAL = vi.hoisted(() => !!(globalThis as any).process?.env?.SUPERLITE_REAL);
vi.mock('../text/localize', () => ({ localize: (s: string) => s }));
vi.mock('../runtime/engine', async (orig) => {
  const real: any = await orig();
  return REAL ? real : { ...real, send: vi.fn(), registerInput: vi.fn(() => () => {}) };
});

import * as engine from '../runtime/engine';
import { Field } from './Field';
import { Scroll } from './Scroll';

const sent = () => (engine.send as any).mock?.calls ?? [];
const w = window as any;
let set: ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.useFakeTimers();
  set = vi.fn();
  w.__commands__ = { vanillaDduiDataStoreSetCommandGroup: { setDataStorePathString: { callable: set } } };
});
afterEach(() => {
  vi.useRealTimers();
  delete w.__commands__;
  vi.clearAllMocks();
});

// The events written to the store, whichever engine answers.
function events(): any[] {
  if (REAL) return set.mock.calls.map((c) => JSON.parse(c[3]));
  return sent().map((c: any[]) => ({ id: c[0], kind: c[1], value: c[2] ?? null }));
}
const type = (input: HTMLInputElement, v: string) => {
  input.value = v;
  input.dispatchEvent(new Event('input', { bubbles: true }));
};

describe('Field', () => {
  it('sends the text after the delay', () => {
    const { container } = render(<Field id="search" value="" delay={250} />);
    const input = container.querySelector('input.oui-input.mc-input') as HTMLInputElement;
    type(input, 'a');
    vi.advanceTimersByTime(200);
    type(input, 'ab');
    vi.advanceTimersByTime(249);
    expect(events()).toHaveLength(0);
    vi.advanceTimersByTime(1);
    expect(events()).toMatchObject([{ id: 'search', kind: 'change', value: 'ab' }]);
  });
  it('field_keeps_text', () => {
    const r = render(<Field id="a" value="x" />);
    const input = r.container.querySelector('input') as HTMLInputElement;
    expect(input.value).toBe('x');
    type(input, 'typed');
    r.rerender(<Field id="a" value="y" />);
    expect(r.container.querySelector('input')!.value).toBe('typed');
  });
  it('field_new_id_clears', () => {
    const r = render(<Field id="a" value="x" />);
    const input = r.container.querySelector('input') as HTMLInputElement;
    type(input, 'typed');
    r.rerender(<Field id="b" value="" />);
    expect(r.container.querySelector('input')!.value).toBe('');
  });
  it('enter_presses', () => {
    const { container } = render(<Field id="a" value="" enter="go" />);
    const input = container.querySelector('input') as HTMLInputElement;
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', keyCode: 13, bubbles: true }));
    expect(events()).toMatchObject([{ id: 'go', kind: 'click' }]);
  });
});

describe('Scroll', () => {
  it('draws the legacy elements', () => {
    const { container } = render(
      <Scroll>
        <i />
      </Scroll>,
    );
    expect(container.innerHTML).toContain('<div class="mc-scroll-wrap"><div class="oui-box mc-scroll mc-col"><i></i></div><div class="mc-scrollbar"');
    expect(container.querySelector('.mc-scroll-rail')).not.toBeNull();
    expect(container.querySelector('.mc-scroll-handle')).not.toBeNull();
  });
  it('end_scrolls', () => {
    const r = render(<Scroll end><i /></Scroll>);
    const sc = r.container.querySelector('.mc-scroll') as HTMLElement;
    Object.defineProperty(sc, 'scrollHeight', { configurable: true, value: 500 });
    Object.defineProperty(sc, 'clientHeight', { configurable: true, value: 100 });
    r.rerender(<Scroll end><i /><i /></Scroll>);
    expect(sc.scrollTop).toBe(500);
    expect(sc.className).toContain('oui-end');
  });
});
