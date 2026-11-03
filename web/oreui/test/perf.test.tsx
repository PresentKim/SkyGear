import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { startDoc, docFacet } from '../src/runtime/doc';
import { Scroll } from '../src/components/Scroll';
import { Canvas } from '../src/components/Canvas';

type Fn = (v: unknown) => void;
const w = window as unknown as Record<string, unknown>;
const DOC = '{"form":"a","root":{"screen":"menu","data":{}}}';

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'setTimeout', 'clearTimeout', 'requestAnimationFrame', 'cancelAnimationFrame'] });
});
afterEach(() => {
  vi.useRealTimers();
  delete w.engine;
});

it('same value publishes nothing', () => {
  const handlers: Record<string, Fn> = {};
  const names: Record<string, string> = {};
  w.engine = {
    on: (e: string, cb: Fn) => (handlers[e] = cb),
    trigger: (e: string, id: number, ...p: unknown[]) => (names[id] = e.replace('query:subscribe/', '') + (e.endsWith('string') ? ':' + p[1] : '')),
  };
  startDoc();
  const emit = (name: string, v: unknown) => handlers['query:updated/' + Object.keys(names).find((k) => names[k] === name)!](v);
  let n = 0;
  const stop = docFacet.observe(() => n++);
  const base = n;
  emit('vanilla.ddui.screenId', { screenId: 'minecraft:custom_form' });
  emit('vanilla.ddui.string:custom_form_data_1', DOC);
  const first = n - base;
  emit('vanilla.ddui.screenId', { screenId: 'minecraft:custom_form' });
  emit('vanilla.ddui.string:custom_form_data_1', DOC);
  emit('vanilla.ddui.string:custom_form_data_1', DOC);
  stop();
  expect(first).toBe(1);
  expect(n - base).toBe(first);
});

it('Scroll writes styles only on change and polls at 250 ms', () => {
  const r = render(<Scroll><i /></Scroll>);
  const sc = r.container.querySelector('.mc-scroll') as HTMLElement;
  const bar = r.container.querySelector('.mc-scrollbar') as HTMLElement;
  const h = r.container.querySelector('.mc-scroll-handle') as HTMLElement;
  let reads = 0;
  Object.defineProperty(sc, 'scrollHeight', { configurable: true, value: 500 });
  Object.defineProperty(sc, 'clientHeight', { configurable: true, get: () => (reads++, 100) });
  Object.defineProperty(bar, 'clientHeight', { configurable: true, value: 100 });
  const writes: string[] = [];
  for (const [el, p] of [[bar, 'display'], [h, 'height'], [h, 'top']] as const) {
    let v = el.style[p];
    Object.defineProperty(el.style, p, { configurable: true, get: () => v, set: (x: string) => (writes.push(p), (v = x)) });
  }
  vi.advanceTimersByTime(250);
  const first = writes.length;
  expect(first).toBeGreaterThan(0);
  vi.advanceTimersByTime(1000);
  expect(writes.length).toBe(first);
  reads = 0;
  vi.advanceTimersByTime(240);
  expect(reads).toBe(0);
});

it('Canvas drag uses transform and reads no layout', () => {
  const r = render(<Canvas planeKey="perf-drag" focusX={0} focusY={0} />);
  vi.advanceTimersByTime(10);
  const view = r.container.querySelector('.oui-canvas') as HTMLElement;
  const plane = r.container.querySelector('.oui-canvas-plane') as HTMLElement;
  const tx = () => /translate\((-?[\d.]+)px, (-?[\d.]+)px\)/.exec(plane.style.transform)!;
  const x0 = parseFloat(tx()[1]);
  let layout = 0;
  for (const el of [view, plane]) {
    for (const p of ['clientWidth', 'clientHeight']) Object.defineProperty(el, p, { configurable: true, get: () => (layout++, 400) });
  }
  plane.getBoundingClientRect = () => (layout++, new DOMRect());
  view.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0, clientX: 10, clientY: 10 }));
  for (let i = 1; i <= 5; i++) document.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX: 10 + i * 10, clientY: 10 }));
  document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }));
  vi.advanceTimersByTime(50);
  expect(parseFloat(tx()[1])).toBeCloseTo(x0 + 50);
  expect(plane.style.left).not.toContain('50');
  expect(layout).toBe(0);
});
