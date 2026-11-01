import { describe, expect, it, vi } from 'vitest';
import { createContext } from 'react';
import { render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { Chrome, Slot } from '../protocol';

vi.mock('../text/Rich', () => ({
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
  InkContext: createContext('dark'),
}));
vi.mock('../components/Window', () => ({
  Window: (p: { children?: ReactNode }) => <div className="mc-window">{p.children}</div>,
  ScreenRoot: (p: { className: string; children?: ReactNode }) => <div className={p.className}>{p.children}</div>,
}));
vi.mock('../components/Scroll', () => ({
  Scroll: (p: { children?: ReactNode }) => <div className="mc-scroll">{p.children}</div>,
}));

import { EquipScreen, PickScreen, ResultScreen } from './showcase';

const chrome: Chrome = { title: 'T', tabs: [], bar: [] };
const slot = (id: string, icon = 'tex:a'): Slot => ({ id, icon, count: '', tip: '', deco: '', selected: false });

describe('showcase', () => {
  it('pick: a card per choice, coloured by rarity, rising one after another', () => {
    const { container } = render(
      <PickScreen data={{ chrome, body: '', items: [], cards: [
        { id: 'a', icon: '', title: 'A', text: '', tag: '', rarity: 'rare' },
        { id: 'b', icon: '', title: 'B', text: '', tag: '', rarity: '' },
      ] }} />,
    );
    const cards = container.querySelectorAll('.mc-cardpick-card');
    expect(cards.length).toBe(2);
    expect(cards[0].className).toContain('mc-cardpick-rare');
    expect(cards[1].className).toContain('mc-cardpick-common');
    expect((container.querySelectorAll('.mc-cardpick-cell')[1] as HTMLElement).style.animationDelay).toBe('0.08s');
    expect(container.querySelector('.mc-window')).toBeNull();
    expect(container.querySelector('.mc-cardpick-head')!.textContent).toBe('T');
  });
  it('result: the headline, a star each (new ones marked) and the rewards after the stars', () => {
    const { container } = render(
      <ResultScreen data={{ chrome, headline: 'H', win: true, lines: [{ label: 'a', value: 'b' }],
        stars: [{ on: true, new: false }, { on: true, new: true }, { on: false, new: false }],
        rewards: [slot('r1')], ok: { id: 'ok', label: 'OK', icon: '' } }} />,
    );
    expect(container.querySelector('.mc-result-head')!.className).toContain('win');
    const stars = container.querySelectorAll('.mc-result-star');
    expect(stars.length).toBe(3);
    expect(stars[1].className).toContain('new');
    expect(stars[2].className).not.toContain('on');
    const reward = container.querySelector('.mc-result-reward') as HTMLElement;
    expect(parseFloat(reward.style.animationDelay)).toBeGreaterThan(parseFloat((stars[2] as HTMLElement).style.animationDelay));
  });
  it('equip: an empty slot shows its silhouette, a filled one does not', () => {
    const { container } = render(
      <EquipScreen data={{ chrome, note: '', bagHeading: '', bag: [slot('b1')], items: [], groups: [
        { heading: 'h', slots: [{ ...slot('e1', ''), empty: 'tex:e' }, { ...slot('e2'), empty: 'tex:e' }] },
      ] }} />,
    );
    const slots = container.querySelectorAll('.mc-equip-slot');
    expect(slots.length).toBe(2);
    expect(slots[0].querySelector('.mc-equip-empty')).not.toBeNull();
    expect(slots[1].querySelector('.mc-equip-empty')).toBeNull();
    expect(container.querySelectorAll('.mc-equip-bag .mc-slot').length).toBe(1);
  });
});
