import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { usePressable } from './press';

vi.mock('../runtime/engine', () => ({ send: vi.fn(), notePress: vi.fn() }));
import * as engine from '../runtime/engine';

function Btn(p: { id?: string; alt?: string; tip?: string; quick?: boolean; name?: string }) {
  const props = usePressable(p);
  return <div id={p.name ?? 'b'} {...props} />;
}

const send = engine.send as unknown as ReturnType<typeof vi.fn>;
const tipEl = () => document.getElementById('oui-tip') as HTMLElement;

function down(el: Element, button = 0) {
  el.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button, clientX: 3, clientY: 4 }));
}
function touch(el: Element) {
  el.dispatchEvent(new Event('touchstart', { bubbles: true }));
}

beforeEach(() => {
  vi.useFakeTimers();
  send.mockClear();
});
afterEach(() => {
  // a mouse moving alone puts the page back to mouse mode
  vi.advanceTimersByTime(1000);
  document.dispatchEvent(new Event('mousemove'));
  if (tipEl()) tipEl().style.display = 'none';
  vi.useRealTimers();
});

describe('mouse', () => {
  it('presses with the left button, sets the attribute', () => {
    const { container } = render(<Btn id="o1" />);
    const b = container.querySelector('#b')!;
    expect(b.getAttribute('data-oui-press')).not.toBeNull();
    down(b);
    vi.advanceTimersByTime(50);
    expect(send).toHaveBeenCalledWith('o1', 'click');
    expect(engine.notePress).toHaveBeenCalledWith(3, 4);
  });
  it('presses alt with the right button', () => {
    const { container } = render(<Btn id="o1" alt="o2" />);
    down(container.querySelector('#b')!, 2);
    expect(send).toHaveBeenCalledWith('o2', 'click');
    expect(send).toHaveBeenCalledTimes(1);
  });
  it('has no attribute without id', () => {
    const { container } = render(<Btn />);
    expect(container.querySelector('#b')!.getAttribute('data-oui-press')).toBeNull();
  });
  it('hover shows the tip, leave hides it', () => {
    const { container } = render(<Btn id="o1" tip="Hello §aworld" />);
    const b = container.querySelector('#b')!;
    b.dispatchEvent(new MouseEvent('mouseenter'));
    expect(tipEl().style.display).toBe('flex');
    expect(tipEl().textContent).toBe('Hello world');
    b.dispatchEvent(new MouseEvent('mouseleave'));
    expect(tipEl().style.display).toBe('none');
  });
  it('each mousedown presses once', () => {
    const { container } = render(
      <div>
        <Btn id="x" name="x" />
      </div>,
    );
    const b = container.querySelector('#x')!;
    down(b);
    down(b);
    vi.advanceTimersByTime(50);
    expect(send).toHaveBeenCalledTimes(2);
  });
});

describe('placement', () => {
  it('goes left when no room right, clamped inside', () => {
    const { container } = render(<Btn id="o1" tip="tip" />);
    const b = container.querySelector('#b')!;
    b.getBoundingClientRect = () => ({ left: 1000, right: 1020, top: 0, bottom: 10, width: 20, height: 10 }) as DOMRect;
    Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true });
    b.dispatchEvent(new MouseEvent('mouseenter'));
    expect(tipEl().style.visibility).toBe('hidden');
    vi.advanceTimersToNextFrame();
    vi.advanceTimersToNextFrame();
    // tooltip is 0 wide in jsdom: right side x=1026 > 1024-6, so it goes left of the element
    expect(tipEl().style.left).toBe('994px');
    expect(tipEl().style.top).toBe('6px');
    expect(tipEl().style.visibility).toBe('');
  });
  it('measures the tip after the layout of its new text', () => {
    // Gameface: right after the text changes the tip still has the last size, here none
    const { container } = render(<Btn id="o1" tip="tip" />);
    const b = container.querySelector('#b')!;
    b.getBoundingClientRect = () => ({ left: 900, right: 920, top: 50, bottom: 60, width: 20, height: 10 }) as DOMRect;
    Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true });
    let w = 0;
    tipEl().getBoundingClientRect = () => ({ left: 0, right: w, top: 0, bottom: 20, width: w, height: 20 }) as DOMRect;
    b.dispatchEvent(new MouseEvent('mouseenter'));
    w = 300; // laid out
    vi.advanceTimersToNextFrame();
    vi.advanceTimersToNextFrame();
    expect(tipEl().style.left).toBe('594px');
    expect(tipEl().style.top).toBe('50px');
  });
  it('a placement left from an earlier tip does not move the next one', () => {
    const { container } = render(
      <div>
        <Btn id="a" name="a" tip="A" />
        <Btn id="c" name="c" tip="C" />
      </div>,
    );
    const a = container.querySelector('#a')!;
    const c = container.querySelector('#c')!;
    a.getBoundingClientRect = () => ({ left: 10, right: 30, top: 0, bottom: 10, width: 20, height: 10 }) as DOMRect;
    c.getBoundingClientRect = () => ({ left: 100, right: 120, top: 0, bottom: 10, width: 20, height: 10 }) as DOMRect;
    Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true });
    a.dispatchEvent(new MouseEvent('mouseenter'));
    vi.advanceTimersToNextFrame();
    a.dispatchEvent(new MouseEvent('mouseleave'));
    c.dispatchEvent(new MouseEvent('mouseenter'));
    vi.advanceTimersToNextFrame();
    expect(tipEl().style.visibility).toBe('hidden');
    vi.advanceTimersToNextFrame();
    expect(tipEl().style.left).toBe('126px');
    expect(tipEl().style.visibility).toBe('');
  });
});

describe('touch', () => {
  it('touch_two_taps: first shows the tip, second presses', () => {
    const { container } = render(<Btn id="o1" tip="T" />);
    const b = container.querySelector('#b')!;
    touch(b);
    down(b);
    vi.advanceTimersByTime(50);
    expect(send).not.toHaveBeenCalled();
    expect(tipEl().style.display).toBe('flex');
    touch(b);
    down(b);
    vi.advanceTimersByTime(50);
    expect(send).toHaveBeenCalledWith('o1', 'click');
    expect(tipEl().style.display).toBe('none');
  });
  it('quick_touch: presses at the first tap', () => {
    const { container } = render(<Btn id="o1" tip="T" quick />);
    const b = container.querySelector('#b')!;
    touch(b);
    down(b);
    vi.advanceTimersByTime(50);
    expect(send).toHaveBeenCalledWith('o1', 'click');
  });
  it('other_tap_hides: a tap elsewhere hides the tip and disarms', () => {
    const { container } = render(
      <div>
        <Btn id="o1" tip="T" name="a" />
        <Btn id="o2" name="c" />
      </div>,
    );
    const a = container.querySelector('#a')!;
    touch(a);
    down(a);
    vi.advanceTimersByTime(50);
    expect(tipEl().style.display).toBe('flex');
    touch(container.querySelector('#c')!);
    expect(tipEl().style.display).toBe('none');
    touch(a);
    down(a);
    vi.advanceTimersByTime(50);
    expect(send).not.toHaveBeenCalled();
  });
  it('mouse_after_touch: the mousedown after a touchstart is one press', () => {
    const { container } = render(<Btn id="o1" />);
    const b = container.querySelector('#b')!;
    touch(b);
    down(b);
    vi.advanceTimersByTime(50);
    expect(send).toHaveBeenCalledTimes(1);
  });
  it('a touch arriving after the mousedown still counts as that tap', () => {
    const { container } = render(<Btn id="o1" tip="T" />);
    const b = container.querySelector('#b')!;
    down(b);
    touch(b);
    vi.advanceTimersByTime(50);
    expect(send).not.toHaveBeenCalled();
  });
  it('a key press after a touch makes the next mousedown a plain press', () => {
    const { container } = render(<Btn id="o1" tip="T" />);
    const b = container.querySelector('#b')!;
    touch(b);
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    down(b);
    vi.advanceTimersByTime(50);
    expect(send).toHaveBeenCalledWith('o1', 'click');
  });
});

describe('tooltip text', () => {
  it('draws § codes as coloured Rich spans', () => {
    const { container } = render(<Btn id="o1" tip="§aGreen" />);
    const b = container.querySelector('#b')!;
    b.dispatchEvent(new MouseEvent('mouseenter'));
    const span = tipEl().querySelector('.oui-line span') as HTMLElement;
    expect(span.textContent).toBe('Green');
    expect(span.style.color).not.toBe('');
  });
});

describe('tooltip lifetime', () => {
  it('unmount hides the tip the element showed', () => {
    const r = render(<Btn id="o1" tip="T" />);
    r.container.querySelector('#b')!.dispatchEvent(new MouseEvent('mouseenter'));
    expect(tipEl().style.display).toBe('flex');
    r.unmount();
    expect(tipEl().style.display).toBe('none');
  });
  it('a changed tip redraws and a removed tip hides', () => {
    const r = render(<Btn id="o1" tip="One" />);
    r.container.querySelector('#b')!.dispatchEvent(new MouseEvent('mouseenter'));
    r.rerender(<Btn id="o1" tip="Two" />);
    expect(tipEl().textContent).toBe('Two');
    r.rerender(<Btn id="o1" />);
    expect(tipEl().style.display).toBe('none');
  });
  it('an id that went away by the delay sends nothing', () => {
    const r = render(<Btn id="o1" />);
    down(r.container.querySelector('#b')!);
    r.rerender(<Btn id="" />);
    vi.advanceTimersByTime(50);
    expect(send).not.toHaveBeenCalled();
  });
});
