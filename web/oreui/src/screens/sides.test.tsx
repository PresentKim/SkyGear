import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { Chrome, SideCard, SidesData } from '../protocol';

vi.mock('../text/Rich', async (orig) => ({
  ...(await orig<object>()),
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
}));
vi.mock('../components/Window', () => ({
  Window: (p: { children?: ReactNode }) => <div className="mc-screen"><div className="mc-window">{p.children}</div></div>,
}));

const sent: unknown[][] = [];
vi.mock('../runtime/engine', async (orig) => ({
  ...(await orig<object>()),
  send: (...a: unknown[]) => sent.push(a),
  notePress: () => {},
}));

import { SidesScreen } from './sides';

const chrome: Chrome = { title: 'T', tabs: [], bar: [] };
const card = (over: Partial<SideCard> = {}): SideCard => ({
  name: 'West', icon: '', tip: '', focus: false, ports: [], none: 'none', ...over,
});
const data = (cards: SideCard[], picker: SidesData['picker'] = null): SidesData => ({ chrome, cards, guide: 'guide', picker });
const six = (over: Partial<SideCard> = {}): SideCard[] => Array.from({ length: 6 }, () => card(over));

describe('sides', () => {
  it('draws two rows of three cards and the guide', () => {
    const cards = six();
    cards[4] = card({ focus: true, icon: 'tex:b', tip: 'Chest' });
    const { container } = render(<SidesScreen data={data(cards)} />);
    const rows = container.querySelectorAll('.mc-sides > .mc-side-row');
    expect(rows.length).toBe(2);
    expect(rows[1].querySelectorAll('.mc-side-card').length).toBe(3);
    const focused = rows[1].querySelectorAll('.mc-side-card')[1];
    expect(focused.className).toContain('on');
    expect(focused.querySelector('.mc-side-block img')).not.toBeNull();
    expect(focused.querySelector('.mc-side-tip')!.textContent).toBe('Chest');
    expect(rows[1].querySelectorAll('.mc-side-card')[2].className).toContain('last');
    expect(container.querySelector('.mc-side-guide')!.textContent).toBe('guide');
  });

  it('swatch_alt', () => {
    sent.length = 0;
    const port = { label: 'In', swatch: { id: 'o1', alt: 'o2', colour: '#30c030', icon: '', tip: 'Green' }, plus: { id: 'o3', on: false } };
    const { container } = render(<SidesScreen data={data(six({ ports: [port] }))} />);
    const sw = container.querySelector('.mc-port-swatch') as HTMLElement;
    expect((sw.querySelector('.mc-port-colour') as HTMLElement).style.backgroundColor).not.toBe('');
    expect(container.querySelector('.mc-port-name')!.textContent).toBe('Green');
    fireEvent.mouseDown(sw, { button: 2 });
    expect(sent[sent.length - 1]).toEqual(['o2', 'click']);
  });

  it('no_ports', () => {
    const { container } = render(<SidesScreen data={data(six())} />);
    expect(container.querySelector('.mc-side-ports > .mc-port-none')!.textContent).toBe('none');
    expect(container.querySelector('.mc-port')).toBeNull();
  });

  it('picker_null', () => {
    const { container } = render(<SidesScreen data={data(six())} />);
    expect(container.querySelector('.mc-port-picker')).toBeNull();
    const last = container.querySelector('.mc-sides')!.lastElementChild as HTMLElement;
    expect(last.className).toBe('oui-box');
    expect(last.children.length).toBe(0);
  });

  it('open picker and plus', () => {
    const port = { label: 'In', swatch: null, plus: { id: 'o3', on: true } };
    const picker = {
      choices: [
        { id: 'c1', colour: '#ff3030', icon: '', tip: '', on: false },
        { id: 'c2', colour: '', icon: 'tex:i', tip: '', on: true },
      ],
    };
    const { container } = render(<SidesScreen data={data(six({ ports: [port] }), picker)} />);
    expect(container.querySelector('.mc-port-plus')!.className).toContain('on');
    expect(container.querySelector('.mc-port-colour')).toBeNull();
    const ch = container.querySelectorAll('.mc-port-picker .mc-port-choice');
    expect(ch[1].className).toContain('on');
    expect(ch[1].querySelector('.mc-choice-swatch.icon img')).not.toBeNull();
    expect((ch[0].firstElementChild as HTMLElement).style.backgroundColor).not.toBe('');
  });
});
