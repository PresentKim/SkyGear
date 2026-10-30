import { describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { Chrome, ListItem, Slot } from '../protocol';

vi.mock('../text/Rich', () => ({
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
}));
vi.mock('../components/Window', () => ({
  Window: (p: { children?: ReactNode }) => <div className="mc-screen"><div className="mc-window">{p.children}</div></div>,
}));
vi.mock('../components/Scroll', () => ({
  Scroll: (p: { children?: ReactNode }) => <div className="mc-scroll mc-col">{p.children}</div>,
}));
vi.mock('../components/Slot', () => ({
  SlotView: (p: { slot: { id: string }; variant?: string }) => (
    <div className={'mc-slot' + (p.variant ? ' mc-' + p.variant : '')}>{p.slot.id}</div>
  ),
  Gap: (p: { size?: number }) => <div className={p.size ? 'mc-gap-' + p.size : 'mc-gap'} />,
}));
vi.mock('../components/Buttons', () => ({
  ListButton: (p: { item: { label: string } }) => <div className="mc-btn mc-list-row">{p.item.label}</div>,
  Card: (p: { button: { label: string } }) => <div className="mc-card-cell"><div className="mc-btn mc-card">{p.button.label}</div></div>,
}));

import { CardsScreen, GridScreen, MenuScreen, SlotMenuScreen } from './menus';

const chrome: Chrome = { title: 'T', tabs: [], bar: [] };
const slot = (id: string): Slot => ({ id, icon: 'tex:a', count: '', tip: '', deco: '', selected: false });
const items: ListItem[] = [{ id: 'b1', label: 'One', icon: '' }, { id: 'b2', label: 'Two', icon: '' }];
const classes = (el: Element) => Array.from(el.children).map((c) => c.className.replace(/oui-(box|text) /, ''));

describe('menu', () => {
  it('draws body, gap and a list button per item', () => {
    const { container } = render(<MenuScreen data={{ chrome, body: 'Hi', items }} />);
    expect(classes(container.querySelector('.mc-scroll')!)).toEqual([
      'mc-body-text', 'mc-gap-6', 'mc-btn mc-list-row', 'mc-btn mc-list-row',
    ]);
  });
  it('empty_body_no_gap', () => {
    const { container } = render(<MenuScreen data={{ chrome, body: '', items }} />);
    expect(container.querySelector('.mc-body-text')).toBeNull();
    expect(container.querySelector('.mc-gap-6')).toBeNull();
    expect(container.querySelectorAll('.mc-list-row').length).toBe(2);
    const cards = render(<CardsScreen data={{ chrome, body: '', items }} />);
    expect(cards.container.querySelector('.mc-body-text')).toBeNull();
    expect(cards.container.querySelector('.mc-gap-6')).toBeNull();
    const grid = render(<GridScreen data={{ chrome, body: '', sections: [], items: [] }} />);
    expect(grid.container.querySelector('.mc-body-text')).toBeNull();
    expect(grid.container.querySelector('.mc-gap-3')).toBeNull();
  });
});

describe('cards', () => {
  it('draws the cards in mc-cards', () => {
    const { container } = render(<CardsScreen data={{ chrome, body: 'Pick', items }} />);
    expect(classes(container.querySelector('.mc-scroll')!)).toEqual(['mc-body-text', 'mc-gap-6', 'mc-cards']);
    expect(container.querySelectorAll('.mc-cards > .mc-card-cell').length).toBe(2);
  });
});

describe('grid', () => {
  it('draws headed sections then buttons', () => {
    const sections = [{ heading: 'Weapons', slots: [slot('s1'), slot('s2')] }, { heading: '', slots: [slot('s3')] }];
    const { container } = render(<GridScreen data={{ chrome, body: 'B', sections, items }} />);
    expect(classes(container.querySelector('.mc-scroll')!)).toEqual([
      'mc-body-text', 'mc-gap-3', 'mc-head', 'mc-grid', 'mc-gap-3', 'mc-grid', 'mc-gap-3',
      'mc-btn mc-list-row', 'mc-btn mc-list-row',
    ]);
    expect(container.querySelectorAll('.mc-grid')[0].children.length).toBe(2);
  });
});

describe('slotmenu', () => {
  it('draws store, info, slot columns and gapped buttons', () => {
    const data = { chrome, store: slot('st'), info: 'Info', left: [slot('l1')], right: [slot('r1'), slot('r2')], items };
    const { container } = render(<SlotMenuScreen data={data} />);
    const board = container.querySelector('.mc-board')!;
    expect(classes(board)).toEqual([
      'mc-row', 'mc-gap-6', 'mc-slotmenu-slots', 'mc-gap-6', 'mc-btn mc-list-row', 'mc-gap-6', 'mc-btn mc-list-row',
    ]);
    expect(board.querySelector('.mc-row > .mc-slot.mc-big')).not.toBeNull();
    expect(board.querySelector('.mc-slotmenu-info .mc-body-text')!.textContent).toBe('Info');
    const cols = board.querySelectorAll('.mc-slotmenu-slots > .mc-row');
    expect([cols[0].children.length, cols[1].children.length]).toEqual([1, 2]);
  });
});
