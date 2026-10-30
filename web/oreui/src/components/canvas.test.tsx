import { afterEach, describe, expect, it, vi } from 'vitest';
import { showTip } from '../interact/Tooltip';
import { render } from '@react-facet/dom-fiber-testing-library';
import { Canvas, clampPan } from './Canvas';
import { Line } from './Line';

const plane = (c: Element) => c.querySelector('.oui-canvas-plane') as HTMLElement;
const zoomOf = (c: Element) => parseFloat(/scale\(([\d.]+)\)/.exec(plane(c).style.transform)![1]);

function touch(type: string, touches: { identifier: number; clientX: number; clientY: number }[], target: Element) {
  const ev = new Event(type, { bubbles: true, cancelable: true }) as any;
  ev.touches = touches;
  ev.changedTouches = touches;
  target.dispatchEvent(ev);
}
function wheel(el: Element, deltaY: number) {
  el.dispatchEvent(new WheelEvent('wheel', { bubbles: true, cancelable: true, deltaY, clientX: 0, clientY: 0 }));
}

afterEach(() => {
  document.body.className = '';
  document.dispatchEvent(new Event('touchend'));
});

describe('clampPan', () => {
  it('keeps the box in the view less the margin', () => {
    expect(clampPan({ x: -500, y: 0 }, { w: 100, h: 100 }, { w: 300, h: 200 }, 24).x).toBe(24);
    expect(clampPan({ x: 999, y: 0 }, { w: 100, h: 100 }, { w: 300, h: 200 }, 24).x).toBe(176);
  });
  it('a box larger than the view cannot show its inside edges', () => {
    expect(clampPan({ x: 100, y: 0 }, { w: 500, h: 10 }, { w: 300, h: 200 }, 24).x).toBe(24);
    expect(clampPan({ x: -900, y: 0 }, { w: 500, h: 10 }, { w: 300, h: 200 }, 24).x).toBe(-224);
  });
});

describe('Line', () => {
  it('draws a turned box', () => {
    const { container } = render(<Line x1={1} y1={2} x2={4} y2={6} className="" />);
    const e = container.firstElementChild as HTMLElement;
    expect(e.className).toBe('oui-line-seg');
    expect(e.style.width).toBe('5rem');
    expect(e.style.left).toBe('1rem');
  });
});

describe('Canvas', () => {
  it('zoom_bounds: zoom stays in 0.5..3 and wheel zooms', () => {
    const { container } = render(<Canvas planeKey="zb" focusX={0} focusY={0} />);
    const view = container.querySelector('.oui-canvas')!;
    wheel(view, -1);
    expect(zoomOf(container)).toBeCloseTo(1.25);
    for (let i = 0; i < 20; i++) wheel(view, -1);
    expect(zoomOf(container)).toBe(3);
    for (let i = 0; i < 40; i++) wheel(view, 1);
    expect(zoomOf(container)).toBe(0.5);
  });

  it('pinch_late_finger: a second finger landing later pinches', () => {
    const { container } = render(<Canvas planeKey="pl" focusX={0} focusY={0} />);
    const view = container.querySelector('.oui-canvas')!;
    const a = { identifier: 1, clientX: 100, clientY: 100 };
    touch('touchstart', [a], view);
    // the second touchstart lists only its own touch
    const b = { identifier: 2, clientX: 200, clientY: 100 };
    touch('touchstart', [b], view);
    const z0 = zoomOf(container);
    touch('touchmove', [{ ...a, clientX: 50 }], document.body);
    touch('touchmove', [{ ...b, clientX: 250 }], document.body);
    expect(zoomOf(container)).toBeGreaterThan(z0);
  });

  it('pan_kept_per_key: same key keeps pan and zoom, a new key starts at the last zoom', () => {
    const first = render(<Canvas planeKey="pk" focusX={0} focusY={0} />);
    wheel(first.container.querySelector('.oui-canvas')!, -1);
    const left = plane(first.container).style.left;
    first.unmount();
    const again = render(<Canvas planeKey="pk" focusX={5} focusY={5} />);
    expect(plane(again.container).style.left).toBe(left);
    expect(zoomOf(again.container)).toBeCloseTo(1.25);
    again.unmount();
    const other = render(<Canvas planeKey="pk2" focusX={0} focusY={0} />);
    expect(zoomOf(other.container)).toBeCloseTo(1.25);
  });

  it('touch_starts_zoomed: a touch screen starts one step in', async () => {
    vi.resetModules();
    const fresh = await import('./Canvas');
    document.body.classList.add('mc-touch');
    const { container } = render(<fresh.Canvas planeKey="ts" focusX={0} focusY={0} />);
    expect(zoomOf(container)).toBeCloseTo(1.25);
  });
});

describe('tooltip while panning', () => {
  it('a drag start hides the tooltip', () => {
    const { container } = render(<Canvas planeKey="tip" focusX={0} focusY={0}><div /></Canvas>);
    const view = container.querySelector('.oui-canvas') as HTMLElement;
    showTip(view, 'T');
    const t = document.getElementById('oui-tip') as HTMLElement;
    expect(t.style.display).toBe('flex');
    view.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));
    expect(t.style.display).toBe('none');
  });
});
