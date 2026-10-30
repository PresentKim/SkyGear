import { describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { createContext } from 'react';
import type { HomeData } from '../protocol';

vi.mock('../text/Rich', () => ({
  InkContext: createContext('dark'),
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
}));
vi.mock('../interact/press', () => ({
  usePressable: (p: { id?: string }) => ({ ref: (e: Element | null) => e && p.id && e.setAttribute('data-oui-press', '') }),
}));

import { HomeScreen } from './home';

const face = Array.from({ length: 64 }, (_, i) => (i % 2 ? 'ff0000' : '00ff00'));
const base = (): HomeData => ({
  profile: { id: 'p', face: [], icon: 'tex:a', name: 'N', title: 'T', fill: 0.4, line: 'L' },
  corner: [{ id: 'c1', icon: 'tex:a', label: 'Mail', badge: 3 }, { id: 'c2', icon: 'tex:b', label: 'Opt', badge: 0 }],
  side: [{ id: 's1', icon: 'tex:a', label: 'Shop', badge: 0 }],
  feed: ['f1'],
  save: { head: 'H', name: 'S', icon: 'tex:a', lines: ['l1'], arrows: true, prev: 'pv', next: 'nx' },
  play: { id: 'play', label: 'Play', sub: 'Go' },
  manage: { id: 'mg', label: 'Manage' },
  overlay: null,
});

describe('home', () => {
  it('draws the structure', () => {
    const { container } = render(<HomeScreen data={base()} />);
    for (const c of ['mc-home-profile', 'mc-home-corner', 'mc-home-side', 'mc-home-feed', 'mc-home-start', 'mc-home-save', 'mc-home-play', 'mc-home-manage', 'mc-home-arrows']) {
      expect(container.querySelector('.' + c), c).not.toBeNull();
    }
    expect(container.querySelector('.mc-home-bar-fill')!.getAttribute('style')).toContain('40%');
    expect(container.querySelector('.mc-home-ov')).toBeNull();
  });
  it('draws the overlay', () => {
    const d = base();
    d.overlay = { title: 'Sys', body: 'B', buttons: [{ id: 'a', label: 'A' }, { id: 'b', label: 'B2' }] };
    const { container } = render(<HomeScreen data={d} />);
    expect(container.querySelectorAll('.mc-home-ov-panel .mc-home-ov-btn').length).toBe(2);
  });
  it('face_64', () => {
    const d = base();
    d.profile.face = face;
    const { container } = render(<HomeScreen data={d} />);
    const px = container.querySelector('.oui-pixels.mc-home-head')!;
    expect(px.querySelectorAll('.oui-pixels-row').length).toBe(8);
    expect(px.querySelectorAll('.oui-pixel').length).toBe(64);
    const r = render(<HomeScreen data={base()} />);
    expect(r.container.querySelector('.oui-pixels')).toBeNull();
    expect(r.container.querySelector('img.mc-home-head')).not.toBeNull();
  });
  it('badge_zero_hidden', () => {
    const { container } = render(<HomeScreen data={base()} />);
    const badges = container.querySelectorAll('.mc-home-badge');
    expect(badges.length).toBe(1);
    expect(badges[0].textContent).toBe('3');
  });
  it('arrow_off', () => {
    const d = base();
    d.save.next = '';
    const { container } = render(<HomeScreen data={d} />);
    const arrows = container.querySelectorAll('.mc-home-arrow');
    expect(arrows[0].className).not.toContain('off');
    expect(arrows[1].className).toContain('mc-home-arrow off');
  });
});
