import { describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { Button, Chrome, ConsoleData, InputData, ListSplitData, Slot, SplitData } from '../protocol';

vi.mock('../text/Rich', () => ({
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
}));
vi.mock('../components/Icon', () => ({ Icon: (p: { icon: string }) => <img src={p.icon} /> }));
vi.mock('../components/Window', () => ({
  Window: (p: { children?: ReactNode }) => <div className="mc-screen"><div className="mc-window">{p.children}</div></div>,
}));
vi.mock('../components/Scroll', () => ({
  Scroll: (p: { className?: string; end?: boolean; children?: ReactNode }) => (
    <div className={'mc-scroll mc-col ' + (p.className ?? '') + (p.end ? ' oui-end' : '')}>{p.children}</div>
  ),
}));
vi.mock('../components/Field', () => ({
  Field: (p: { id: string; value: string; placeholder?: string }) => (
    <div className="mc-input">{p.id + '|' + p.value + '|' + (p.placeholder ?? '')}</div>
  ),
}));
vi.mock('../components/Slot', () => ({
  SlotView: (p: { slot: { id: string }; variant?: string }) => (
    <div className={'mc-slot' + (p.variant ? ' mc-' + p.variant : '')}>{p.slot.id}</div>
  ),
  Frame: () => <div className="mc-frame" />,
  Gap: (p: { size?: number }) => <div className={p.size ? 'mc-gap-' + p.size : 'mc-gap'} />,
}));
vi.mock('../components/Buttons', () => ({
  ListButton: (p: { item: { label: string } }) => <div className="mc-btn mc-list-row">{p.item.label}</div>,
  ActionButton: (p: { button: { label: string } }) => <div className="mc-btn mc-action">{p.button.label}</div>,
  McButton: (p: { id: string; className: string; children?: ReactNode }) => (
    <div className={'oui-box ' + p.className} id={p.id}>{p.children}</div>
  ),
}));

import { ConsoleScreen, InputScreen, ListSplitScreen, SplitScreen } from './splits';

const chrome: Chrome = { title: 'T', tabs: [], bar: [] };
const slot = (id: string): Slot => ({ id, icon: 'tex:a', count: '', tip: '', deco: '', selected: false });
const btn = (id: string): Button => ({ id, label: id, icon: '' });

const split = (over: Partial<SplitData> = {}): SplitData => ({
  chrome, slots: [], prev: '', next: '', detail: { icon: 'tex:d', name: 'A\nB', status: 'ok', body: 'body' },
  actions: [], rows: [], ...over,
});

describe('split', () => {
  it('split_48_cells: 5 slots draw 5 slots and 43 frames', () => {
    const { container } = render(<SplitScreen data={split({ slots: [1, 2, 3, 4, 5].map((n) => slot('s' + n)) })} />);
    const grid = container.querySelector('.mc-split-grid')!;
    expect(grid.querySelectorAll('.mc-slot').length).toBe(5);
    expect(grid.querySelectorAll('.mc-frame').length).toBe(43);
  });

  it('pager_off: an empty id draws mc-btn off', () => {
    const { container } = render(<SplitScreen data={split({ prev: '', next: 'n1' })} />);
    const btns = container.querySelectorAll('.mc-pager > div');
    expect(btns[0].className).toBe('oui-box mc-btn off');
    expect(btns[1].className).toBe('oui-box mc-btn');
    expect(btns[1].id).toBe('n1');
  });

  it('draws the detail', () => {
    const { container } = render(
      <SplitScreen data={split({ actions: [btn('a1')], rows: [{ heading: 'H', slots: [slot('r1')] }] })} />,
    );
    const right = container.querySelector('.mc-split-right')!;
    expect(right.querySelector('.mc-big')).not.toBeNull();
    expect(right.querySelectorAll('.mc-small-text').length).toBe(4);
    expect(right.querySelector('.mc-action')).not.toBeNull();
    expect(right.querySelector('.mc-head')?.textContent).toBe('H');
    expect(right.querySelectorAll('.mc-grid .mc-slot').length).toBe(1);
  });
});

describe('listsplit', () => {
  const data = (n: number): ListSplitData => ({
    chrome,
    rows: [
      { id: 'r1', icon: 'tex:a', name: 'N', info: 'I', selected: true },
      { id: 'r2', icon: '', name: 'M', info: '', selected: false },
    ],
    body: 'B',
    actions: Array.from({ length: n }, (_, i) => btn('a' + i)),
  });

  it('draws rows', () => {
    const { container } = render(<ListSplitScreen data={data(0)} />);
    const rows = container.querySelectorAll('.mc-lsplit-left .mc-lrow');
    expect(rows.length).toBe(2);
    expect(rows[0].className).toBe('oui-box mc-btn mc-lrow on');
    expect(rows[0].querySelector('.mc-lrow-info')?.textContent).toBe('I');
    expect(rows[1].querySelector('.mc-lrow-info')).toBeNull();
  });

  it('pairs: 4 actions -> one full and two pairs plus one', () => {
    const { container } = render(<ListSplitScreen data={data(4)} />);
    const right = container.querySelector('.mc-lsplit-right .mc-scroll')!;
    expect(right.querySelectorAll(':scope > .mc-action').length).toBe(1);
    const pairs = right.querySelectorAll('.mc-lsplit-pair');
    expect(pairs.length).toBe(2);
    expect(pairs[0].children.length).toBe(2);
    expect(pairs[1].children.length).toBe(1);
  });
});

describe('input', () => {
  it('draws label, field and submit', () => {
    const d: InputData = { chrome, label: 'L', field: 'text', value: 'v', submit: btn('ok') };
    const { container } = render(<InputScreen data={d} />);
    expect(container.querySelector('.mc-body-text')?.textContent).toBe('L');
    expect(container.querySelector('.mc-search .mc-input')?.textContent).toBe('text|v|');
    expect(container.querySelector('.mc-list-row')?.textContent).toBe('ok');
  });
});

describe('console', () => {
  it('console_end: the log is an end scroll', () => {
    const d: ConsoleData = { chrome, lines: ['> a', 'b'], field: 'cmd2', hint: 'cmd', submit: btn('run') };
    const { container } = render(<ConsoleScreen data={d} />);
    const log = container.querySelector('.mc-console-log')!;
    expect(log.className).toContain('oui-end');
    expect(log.querySelectorAll('.mc-console-line').length).toBe(2);
    expect(container.querySelector('.mc-console-input .mc-input')?.textContent).toBe('cmd2||cmd');
    expect(container.querySelector('.mc-console-input .mc-action')).not.toBeNull();
  });
});
