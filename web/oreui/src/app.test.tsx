import { describe, expect, it, vi } from 'vitest';
import { act, useState } from 'react';
import { createFacet } from '@react-facet/core';
import { render } from '@react-facet/dom-fiber-testing-library';
import type { DocState } from './runtime/doc';

const doc = vi.hoisted(() => ({ facet: null as unknown }));
vi.mock('./runtime/doc', async () => {
  const { createFacet } = await import('@react-facet/core');
  const f = createFacet({ initialValue: { active: false, form: '', root: null, error: '', updates: 0, bytes: 0 } });
  doc.facet = f;
  return { docFacet: f, startDoc: () => {} };
});
vi.mock('./interact/keys', () => ({ useKeyboard: () => {} }));
vi.mock('./interact/Tooltip', () => ({ TooltipHost: () => <div id="oui-tip" /> }));
vi.mock('./screens/registry', () => {
  const Counter = (p: { data: { t: string } }) => {
    const [n, setN] = useState(0);
    return <button className="ctr" onClick={() => setN(n + 1)}>{p.data.t + n}</button>;
  };
  return { screens: { menu: Counter } };
});

import { App } from './App';

const set = (v: Partial<DocState>) =>
  act(() => {
    (doc.facet as ReturnType<typeof createFacet<DocState>>).set({
      active: true, form: 'a', root: { screen: 'menu', data: { t: 'x' } }, error: '', updates: 1, bytes: 1, ...v,
    });
  });

describe('app', () => {
  it('tooltip_renders_last', () => {
    const { container } = render(<App />);
    set({});
    const kids = Array.from(container.children);
    expect(kids[kids.length - 1].id).toBe('oui-tip');
    expect(kids.length).toBeGreaterThan(1);
  });

  it('form_change_remounts', () => {
    const { container } = render(<App />);
    set({});
    const btn = () => container.querySelector('.ctr') as HTMLElement;
    act(() => btn().click());
    expect(btn().textContent).toBe('x1');
    set({ updates: 2, root: { screen: 'menu', data: { t: 'y' } } });
    expect(btn().textContent).toBe('y1');
    set({ form: 'b', root: { screen: 'menu', data: { t: 'y' } } });
    expect(btn().textContent).toBe('y0');
  });

  it('unknown_screen', () => {
    const { container } = render(<App />);
    set({ root: { screen: 'nope' as never, data: {} } });
    expect(container.querySelector('.ctr')).toBeNull();
    expect(container.querySelector('#oui-ddui-status')!.textContent).toContain('unknown screen nope');
  });

  it('inactive_hides', () => {
    const { container } = render(<App />);
    set({});
    expect(container.querySelector('.ctr')).not.toBeNull();
    set({ active: false });
    expect(container.querySelector('.ctr')).toBeNull();
    expect(container.querySelector('#oui-ddui-status')).toBeNull();
  });

  it('shows the debug line only with root.debug', () => {
    const { container } = render(<App />);
    set({});
    expect(container.querySelector('#oui-ddui-status')).toBeNull();
    set({ root: { screen: 'menu', data: { t: 'x' }, debug: true } });
    expect(container.querySelector('#oui-ddui-status')!.textContent).toContain('screen=menu');
  });
});
