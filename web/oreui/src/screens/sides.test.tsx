import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { Chrome, SideCard, SidesData } from '../protocol';

vi.mock('../text/Rich', () => ({
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
  x: 0, y: 0, arrow: '←', name: 'West', icon: '', tip: '', focus: false, ports: [], none: 'none', ...over,
});
const data = (cards: SideCard[], picker: SidesData['picker'] = null): SidesData => ({ chrome, center: 'tex:a', cards, picker });

describe('sides', () => {
  it('draws lines, centre and cards', () => {
    const cards = [card({ x: 0, y: 0 }), card({ x: 107, y: 88, focus: true, icon: 'tex:b', tip: 'tip' })];
    const { container } = render(<SidesScreen data={data(cards)} />);
    const view = container.querySelector('.mc-sides') as HTMLElement;
    expect(view.style.width).toBe('62rem');
    expect(view.style.height).toBe('28.8rem');
    expect(view.querySelectorAll('.mc-side-line').length).toBe(2);
    const centre = view.querySelector('.mc-side-centre') as HTMLElement;
    expect(centre.style.left).toBe('29rem');
    expect(centre.style.top).toBe('12.4rem');
    const cs = view.querySelectorAll('.mc-side-card');
    expect(cs[1].className).toContain('on');
    expect((cs[1] as HTMLElement).style.left).toBe('21.4rem');
    expect(cs[0].querySelector('.mc-side-empty')).not.toBeNull();
    expect(cs[1].querySelector('.mc-side-block img')).not.toBeNull();
  });

  it('swatch_alt', () => {
    sent.length = 0;
    const port = { label: 'In', swatch: { id: 'o1', alt: 'o2', colour: '#30c030', icon: '', tip: '' }, plus: { id: 'o3', on: false } };
    const { container } = render(<SidesScreen data={data([card({ ports: [port] })])} />);
    const sw = container.querySelector('.mc-port-swatch') as HTMLElement;
    expect(sw.style.backgroundColor).not.toBe('');
    fireEvent.mouseDown(sw, { button: 2 });
    expect(sent[sent.length - 1]).toEqual(['o2', 'click']);
  });

  it('no_ports', () => {
    const { container } = render(<SidesScreen data={data([card()])} />);
    const none = container.querySelector('.mc-side-ports > .mc-port-none') as HTMLElement;
    expect(none.textContent).toBe('none');
    expect(container.querySelector('.mc-port')).toBeNull();
  });

  it('picker_null', () => {
    const { container } = render(<SidesScreen data={data([card()])} />);
    expect(container.querySelector('.mc-port-picker')).toBeNull();
    const last = container.querySelector('.mc-sides')!.lastElementChild as HTMLElement;
    expect(last.className).toBe('oui-box');
    expect(last.children.length).toBe(0);
  });

  it('open picker and plus', () => {
    const port = { label: 'In', swatch: null, plus: { id: 'o3', on: true } };
    const picker = {
      x: 25, y: 68, w: 78,
      choices: [
        { id: 'c1', colour: '#ff3030', icon: '', tip: '', on: false },
        { id: 'c2', colour: '', icon: 'tex:i', tip: '', on: true },
      ],
    };
    const { container } = render(<SidesScreen data={data([card({ ports: [port] })], picker)} />);
    expect(container.querySelector('.mc-port-plus')!.className).toContain('on');
    expect(container.querySelector('.mc-port-swatch')).toBeNull();
    const p = container.querySelector('.mc-port-picker') as HTMLElement;
    expect(p.style.left).toBe('5rem');
    expect(p.style.width).toBe('15.6rem');
    const ch = p.querySelectorAll('.mc-port-choice');
    expect(ch[1].className).toContain('on');
    expect(ch[1].querySelector('.mc-choice-swatch.icon img')).not.toBeNull();
    expect((ch[0].firstElementChild as HTMLElement).style.backgroundColor).not.toBe('');
  });
});
