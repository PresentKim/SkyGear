import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { act } from 'react';
import type { ReactNode } from 'react';

const sent: unknown[][] = [];
vi.mock('../runtime/engine', async () => {
  const real = await vi.importActual<typeof import('../runtime/engine')>('../runtime/engine');
  return { ...real, send: (...a: unknown[]) => void sent.push(a) };
});
vi.mock('../text/Rich', () => ({
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
}));
vi.mock('../components/Window', () => ({ Window: (p: { children?: ReactNode }) => <div className="mc-window">{p.children}</div> }));
vi.mock('../components/Scroll', () => ({ Scroll: (p: { children?: ReactNode }) => <div className="mc-scroll">{p.children}</div> }));
vi.mock('../components/Field', () => ({ Field: (p: { id: string; placeholder?: string }) => <input id={p.id} placeholder={p.placeholder} /> }));

import { ProbeScreen } from './probe';

const chrome = { title: 'T', tabs: [], bar: [] };
const data = (nodes: unknown[]) => ({ chrome, info: 'info', nodes }) as never;

beforeEach(() => {
  sent.length = 0;
});
afterEach(() => {
  vi.useRealTimers();
});

describe('probe', () => {
  it('probe_lines: a keydown is sent once in the legacy format', () => {
    const { container } = render(<ProbeScreen data={data([{ t: 'probe' }])} />);
    sent.length = 0;
    act(() => {
      document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', code: 'KeyA', keyCode: 65, bubbles: true }));
    });
    expect(sent).toEqual([['probe-0', 'probe', 'keydown key=a code=KeyA keyCode=65 repeat=false on=']].map((r) => [r[0], r[1], expect.stringMatching(/^keydown key=a code=KeyA keyCode=65 repeat=false/)]));
    expect(container.querySelector('.oui-probe')!.textContent).toContain('keydown key=a');
    act(() => {
      document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', code: 'KeyA', keyCode: 65, bubbles: true }));
    });
    expect(sent.length).toBe(1);
    expect(container.querySelector('.oui-probe')!.textContent).toContain('§7x2');
  });

  it('renders input and box nodes', () => {
    const { container } = render(<ProbeScreen data={data([
      { t: 'input', id: 'probe-input', class: 'mc-input', placeholder: 'keys' },
      { t: 'box', class: 'mc-gap-3' },
    ])} />);
    expect(container.querySelector('input#probe-input')).not.toBeNull();
    expect(container.querySelector('div.oui-box.mc-gap-3')).not.toBeNull();
  });

  it('restest_result: an image that loads reports ok', async () => {
    vi.useFakeTimers();
    const desc = Object.getOwnPropertyDescriptor(HTMLImageElement.prototype, 'src');
    Object.defineProperty(HTMLImageElement.prototype, 'src', {
      configurable: true,
      set(this: HTMLImageElement) {
        setTimeout(() => this.onload?.(new Event('load')), 1);
      },
    });
    try {
      render(<ProbeScreen data={data([{ t: 'restest', js: '', css: '', html: '', http: [{ name: 'h', base: 'http://x' }] }])} />);
      await act(async () => {
        await vi.advanceTimersByTimeAsync(120000);
      });
      const res = sent.filter((s) => s[2] && String(s[2]).startsWith('res h img'));
      expect(res).toEqual([['probe-0', 'probe', 'res h img ok onload']]);
    } finally {
      if (desc) Object.defineProperty(HTMLImageElement.prototype, 'src', desc);
    }
  });
});
