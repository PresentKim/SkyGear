import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { BoardData, Cell, Chrome, ListItem, SearchData, SieveData, Slot } from '../protocol';

vi.mock('../text/Rich', () => ({
  Rich: (p: { text: string; className?: string; style?: object }) => <div className={p.className} style={p.style}>{p.text}</div>,
}));
vi.mock('../components/Icon', () => ({ Icon: (p: { icon: string }) => <img src={p.icon} /> }));
vi.mock('../components/Window', () => ({
  Window: (p: { children?: ReactNode }) => <div className="mc-screen"><div className="mc-window">{p.children}</div></div>,
}));
vi.mock('../components/Scroll', () => ({
  Scroll: (p: { children?: ReactNode }) => <div className="mc-scroll mc-col">{p.children}</div>,
}));
vi.mock('../components/Field', () => ({
  Field: (p: { id: string; value: string; placeholder?: string }) => (
    <input className="mc-input" id={p.id} defaultValue={p.value} placeholder={p.placeholder} />
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
}));

import { BoardScreen, SearchScreen, SieveScreen } from './boards';

const chrome: Chrome = { title: 'T', tabs: [], bar: [] };
const slot = (id: string): Slot => ({ id, icon: 'tex:a', count: '2', tip: '', deco: '', selected: false });
const items: ListItem[] = [{ id: 'b1', label: 'Back', icon: '' }];

describe('board', () => {
  it('draws body, rows and buttons', () => {
    const rows: Cell[][] = [[{ k: 'slot', slot: slot('s1') }, { k: 'arrow' }, { k: 'frame' }, { k: 'mini', slot: slot('s2') }]];
    const data: BoardData = { chrome, body: 'Body', rows, items };
    const { container } = render(<BoardScreen data={data} />);
    const board = container.querySelector('.mc-board')!;
    expect(board.querySelector('.mc-body-text')!.textContent).toBe('Body');
    const row = board.querySelector('.mc-board-row')!;
    expect(row.children.length).toBe(11);
    expect(row.querySelector('.mc-arrow > .mc-arrow-img')).not.toBeNull();
    expect(row.querySelector('.mc-slot.mc-mini')).not.toBeNull();
    expect(container.querySelector('.mc-list-row')!.textContent).toBe('Back');
  });

  it('text_span_width', () => {
    const rows: Cell[][] = [[{ k: 'text', text: 'Hi' }, { k: 'blank' }, { k: 'blank' }, { k: 'slot', slot: slot('s1') }]];
    const { container } = render(<BoardScreen data={{ chrome, body: '', rows, items: [] }} />);
    const row = container.querySelector('.mc-board-row')!;
    const text = row.querySelector('.mc-cell-text') as HTMLElement;
    expect(text.parentElement).toBe(row);
    expect(text.style.width).toBe('10.8rem');
    expect(text.style.flexShrink).toBe('0');
    // text covers the cells up to the slot (3), the slot, then 7 blanks
    expect(row.children.length).toBe(1 + 1 + 7);
  });
});

describe('sieve', () => {
  const data = (drops: number, lines: number): SieveData => ({
    chrome,
    top: [[{ k: 'slot', slot: slot('t1') }]],
    groups: [{ head: slot('h'), name: 'Sieve', recipes: [{ input: slot('i'), mesh: slot('m'), drops: Array.from({ length: drops }, (_, i) => slot('d' + i)) }] }],
    lines,
    items,
  });
  it('drops_padded', () => {
    const { container } = render(<SieveScreen data={data(2, 2)} />);
    const drops = container.querySelector('.mc-jei-drops')!;
    expect(drops.querySelectorAll('.mc-frame').length).toBe(16);
    expect(drops.querySelectorAll('.mc-slot').length).toBe(2);
  });
  it('draws the recipe and heading', () => {
    const { container } = render(<SieveScreen data={data(1, 1)} />);
    expect(container.querySelector('.mc-jei-top .mc-slot.mc-jei-mesh')).not.toBeNull();
    expect(container.textContent).toContain('§lSieve');
  });
});

describe('search', () => {
  const data = (query: string, id: string): SearchData => ({
    chrome, field: 'search', query, placeholder: 'Name', info: 'Found', items,
    sections: [{ heading: 'Weapons', slots: [slot(id)] }],
  });
  it('draws the field, info and sections', () => {
    const { container } = render(<SearchScreen data={data('a', 'x1')} />);
    expect(container.querySelector('.mc-search .mc-input')).not.toBeNull();
    expect(container.querySelector('.mc-body-text')!.textContent).toBe('Found');
    expect(container.querySelector('.mc-head')!.textContent).toBe('Weapons');
    expect(container.querySelectorAll('.mc-grid .mc-slot').length).toBe(1);
  });
  it('search_keeps_typing', () => {
    const { container, rerender } = render(<SearchScreen data={data('a', 'x1')} />);
    const input = container.querySelector('input') as HTMLInputElement;
    fireEvent.input(input, { target: { value: 'abc' } });
    rerender(<SearchScreen data={data('a', 'x2')} />);
    const after = container.querySelector('input') as HTMLInputElement;
    expect(after).toBe(input);
    expect(after.value).toBe('abc');
    expect(container.querySelector('.mc-grid .mc-slot')!.textContent).toBe('x2');
  });
});
