import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { ChoiceData, PlaneData } from '../protocol';

vi.mock('../text/Rich', async () => {
  const { createContext } = await import('react');
  return {
    InkContext: createContext('dark'),
    Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
  };
});
vi.mock('../components/Icon', () => ({ Icon: (p: { icon: string; className?: string }) => <img className={p.className} src={p.icon} /> }));
vi.mock('../components/Window', () => ({
  Window: (p: { children?: ReactNode }) => <div className="mc-screen"><div className="mc-window">{p.children}</div></div>,
  ScreenRoot: (p: { className: string; children?: ReactNode }) => <div className={'oui-box ' + p.className}>{p.children}</div>,
}));
vi.mock('../components/Scroll', () => ({
  Scroll: (p: { children?: ReactNode }) => <div className="mc-scroll mc-col">{p.children}</div>,
}));
vi.mock('../components/Slot', () => ({ Gap: (p: { size?: number }) => <div className={'mc-gap-' + p.size} /> }));
vi.mock('../components/Buttons', () => ({
  ListButton: (p: { item: { label: string } }) => <div className="mc-btn mc-list-row">{p.item.label}</div>,
  ActionButton: (p: { button: { label: string } }) => <div className="mc-btn mc-action">{p.button.label}</div>,
}));
vi.mock('../runtime/engine', () => ({ send: vi.fn(), notePress: vi.fn() }));

import { ChoiceScreen, LoadingScreen, PlaneScreen } from './boards2';

const chrome = { title: 'T', tabs: [], bar: [] };

describe('choice', () => {
  const data: ChoiceData = {
    chrome,
    stats: [{ icon: 'tex:a', label: 'L', value: '12' }, { icon: '', label: 'M', value: '3' }],
    note: 'Note',
    groups: [{ title: 'G', choices: [
      { id: 'c1', label: 'A', icon: 'tex:a', tip: 't', on: true, dim: false },
      { id: 'c2', label: 'B', icon: '', tip: '', on: false, dim: false },
      { id: 'c3', label: 'C', icon: '', tip: '', on: false, dim: true },
    ] }],
    items: [{ id: 'r', label: 'Reset', icon: '' }],
  };
  it('draws stats, groups and buttons', () => {
    const { container } = render(<ChoiceScreen data={data} />);
    expect(container.querySelectorAll('.mc-stats > .mc-stat > .mc-stat-body').length).toBe(2);
    expect(container.querySelectorAll('.mc-stat-icon').length).toBe(1);
    expect(container.querySelector('.mc-pick-note')!.textContent).toBe('Note');
    expect(container.querySelector('.mc-pick-title')!.textContent).toBe('G');
    const tiles = container.querySelectorAll('.mc-picks > .mc-pick');
    expect(tiles[0].className).toBe('oui-box mc-btn mc-pick on');
    expect(tiles[2].className).toBe('oui-box mc-btn mc-pick dim');
    expect(tiles[0].querySelector('.mc-pick-ring')).not.toBeNull();
    expect(container.querySelector('.mc-gap-6 + .mc-list-row')).not.toBeNull();
  });
  it('choice_on_not_pressable', () => {
    const { container } = render(<ChoiceScreen data={data} />);
    const tiles = container.querySelectorAll('.mc-picks > .mc-pick');
    expect(tiles[0].hasAttribute('data-oui-press')).toBe(false);
    expect(tiles[1].hasAttribute('data-oui-press')).toBe(true);
    expect(tiles[2].hasAttribute('data-oui-press')).toBe(true);
  });
});

describe('loading', () => {
  it('draws the sky, clouds and texts', () => {
    const { container } = render(<LoadingScreen data={{ icon: 'tex:a', title: 'N', sub: '', line: 'Wait', tip: 'Tip' }} />);
    expect(container.querySelector('.mc-screen.mc-load')).not.toBeNull();
    expect(container.querySelectorAll('.mc-load-cloud').length).toBe(5);
    expect(container.querySelectorAll('.mc-load-dot').length).toBe(3);
    expect(container.querySelector('.mc-load-sub')).toBeNull();
    expect(container.querySelector('.mc-load-title')!.textContent).toBe('§lN');
    expect(container.querySelector('.mc-load-tip')!.textContent).toBe('Tip');
  });
});

// Oracle: the legacy layoutPlanes projection math (formerly oui-ddui.js), copied here.
function legacyPlace(n: { fov: number; dist: number }, at: number[], w: number, h: number): Record<string, string> {
  const half = (n.dist || 1) * Math.tan((((n.fov || 60) / 2) * Math.PI) / 180);
  const scale = h / 2 / half;
  const cx = w / 2;
  const cy = h / 2;
  return {
    left: Math.round(cx + at[0] * scale) + 'px',
    top: Math.round(cy + at[1] * scale) + 'px',
    width: Math.round(at[2] * scale) + 'px',
    height: Math.round(at[3] * scale) + 'px',
  };
}

describe('plane', () => {
  const data: PlaneData = {
    fov: 70, dist: 10, info: 'Info', corner: '',
    spots: [{ id: 's1', at: [0, 0, 1, 1], on: true, tip: 'tip' }, { id: 's2', at: [1, -2, 2, 1], on: false, tip: '' }],
    actions: [{ id: 'a1', label: 'OK', icon: '' }],
  };
  const place = (c: Element, i: number) => {
    const s = (c.querySelectorAll('.mc-spot')[i] as HTMLElement).style;
    return { left: s.left, top: s.top, width: s.width, height: s.height };
  };
  it('plane_projection', () => {
    window.innerWidth = 1280;
    window.innerHeight = 720;
    const { container } = render(<PlaneScreen data={data} />);
    expect(place(container, 0)).toEqual(legacyPlace(data, [0, 0, 1, 1], 1280, 720));
    expect(place(container, 0)).toEqual({ left: '640px', top: '360px', width: '51px', height: '51px' });
    expect(place(container, 1)).toEqual(legacyPlace(data, [1, -2, 2, 1], 1280, 720));
    expect(container.querySelector('.mc-spot.on')).not.toBeNull();
    expect(container.querySelector('.mc-plane-corner')).toBeNull();
    expect(container.querySelector('.mc-plane-panel .mc-action')!.textContent).toBe('OK');
  });
  it('plane_resize', () => {
    window.innerWidth = 1280;
    window.innerHeight = 720;
    const { container } = render(<PlaneScreen data={data} />);
    window.innerWidth = 800;
    window.innerHeight = 600;
    fireEvent(window, new Event('resize'));
    expect(place(container, 0)).toEqual(legacyPlace(data, [0, 0, 1, 1], 800, 600));
    expect(place(container, 0).left).toBe('400px');
  });
});
