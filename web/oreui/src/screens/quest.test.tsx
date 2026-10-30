import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { act } from 'react';
import type { ReactNode } from 'react';
import type { QuestData } from '../protocol';

const sent = vi.hoisted(() => [] as unknown[][]);
vi.mock('../runtime/engine', async (orig) => ({
  ...(await orig<object>()),
  send: (...a: unknown[]) => sent.push(a),
  notePress: () => {},
}));
vi.mock('../text/Rich', () => ({
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
  InkContext: { Provider: (p: { children?: ReactNode }) => <>{p.children}</> },
}));
vi.mock('../components/Icon', () => ({ Icon: (p: { icon: string }) => <img src={p.icon} /> }));
vi.mock('../components/Scroll', () => ({
  Scroll: (p: { children?: ReactNode }) => <div className="mc-scroll mc-col">{p.children}</div>,
}));
vi.mock('../components/Canvas', () => ({
  Canvas: (p: { children?: ReactNode }) => <div className="oui-canvas">{p.children}</div>,
}));
vi.mock('../components/Slot', () => ({
  SlotView: (p: { slot: { id: string }; variant?: string }) => (
    <div className={'mc-slot' + (p.variant ? ' mc-' + p.variant : '')}>{p.slot.id}</div>
  ),
  Gap: (p: { size?: number }) => <div className={'mc-gap-' + p.size} />,
}));
vi.mock('../components/Buttons', () => ({
  ActionButton: (p: { button: { label: string } }) => <div className="mc-btn mc-action">{p.button.label}</div>,
}));
vi.mock('../components/Window', async () => ({
  LobbyContext: (await import('react')).createContext(null),
  ScreenRoot: (p: { className: string; children?: ReactNode }) => <div className={'oui-box ' + p.className}>{p.children}</div>,
  BarRow: () => <div className="mc-icon-button" />,
}));

import { QuestScreen } from './quest';

const data: QuestData = {
  bar: [],
  chapters: [
    { id: 'c1', name: 'One', icon: 'tex:a', count: '3/5', on: true, new: false },
    { id: 'c2', name: 'Two', icon: 'tex:b', count: '0/8', on: false, new: false },
  ],
  heading: 'One',
  progress: '3/5',
  key: 'k',
  focusX: 0,
  focusY: 0,
  recenter: false,
  lines: [{ x1: 0, y1: 7.2, x2: 3.6, y2: 7.2, cls: 'done' }],
  nodes: [
    { id: 'n1', x: 0, y: 7.2, shape: 'diamond', state: 'open', tracked: true, new: false, seen: '', tip: 't', icon: 'tex:a' },
    { id: 'n2', x: 7.2, y: 0, shape: '', state: 'locked', tracked: false, new: true, seen: 'o9', tip: 'u', icon: 'tex:a' },
  ],
  panel: null,
};
const withPanel: QuestData = {
  ...data,
  panel: {
    icon: 'tex:a', title: 'Q', status: 'S', body: 'B',
    rows: [{ heading: 'Bring', slots: [{ id: 's1', icon: 'tex:a', count: '1', tip: '', deco: '', selected: false }] }],
    actions: [{ id: 'a1', label: 'Go', icon: '' }],
    close: 'pc',
  },
};

function down(el: Element) {
  el.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));
  vi.advanceTimersByTime(1000);
}

describe('quest', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('draws window, chapters, plane', () => {
    const { container } = render(<QuestScreen data={data} />);
    expect(container.querySelector('.mc-screen > .mc-quest-window')).not.toBeNull();
    expect(container.querySelectorAll('.mc-chapter').length).toBe(2);
    expect(container.querySelector('.mc-quest-head .mc-quest-progress')!.textContent).toBe('3/5');
    expect(container.querySelectorAll('.oui-line-seg').length).toBe(1);
    expect(container.querySelector('.mc-quest-panel')).toBeNull();
  });

  it('node_classes', () => {
    const { container } = render(<QuestScreen data={data} />);
    const n = container.querySelectorAll('.mc-quest');
    expect(n[0].className).toBe('oui-box mc-quest diamond open tracked');
    expect(n[1].className).toBe('oui-box mc-quest square locked');
    expect((n[0] as HTMLElement).style.left).toBe('0rem');
    expect((n[0] as HTMLElement).style.top).toBe('7.2rem');
    expect(n[0].querySelector('.mc-quest-shape')).not.toBeNull();
  });

  it('chapter_on_not_pressable', () => {
    sent.length = 0;
    const { container } = render(<QuestScreen data={data} />);
    const [on, off] = Array.from(container.querySelectorAll('.mc-chapter'));
    expect(on.className).toContain(' on');
    expect(on.hasAttribute('data-oui-press')).toBe(false);
    expect(off.hasAttribute('data-oui-press')).toBe(true);
    down(on);
    expect(sent.length).toBe(0);
    down(off);
    expect(sent).toEqual([['c2', 'click']]);
  });

  it('new_tag_goes_at_the_first_look', () => {
    sent.length = 0;
    const { container } = render(<QuestScreen data={data} />);
    const n = container.querySelectorAll('.mc-quest');
    expect(n[0].querySelector('.mc-new-tag')).toBeNull();
    expect(n[1].querySelector('.mc-new-tag')!.textContent).toBe('NEW');
    act(() => {
      n[1].dispatchEvent(new MouseEvent('mouseenter'));
    });
    expect(n[1].querySelector('.mc-new-tag')).toBeNull();
    expect(sent).toEqual([['o9', 'seen']]);
    act(() => {
      n[1].dispatchEvent(new MouseEvent('mouseenter'));
    });
    expect(sent).toEqual([['o9', 'seen']]);
  });

  it('chapter_new_dot', () => {
    const { container } = render(<QuestScreen data={{ ...data, chapters: [data.chapters[0], { ...data.chapters[1], new: true }] }} />);
    const [one, two] = Array.from(container.querySelectorAll('.mc-chapter'));
    expect(one.querySelector('.mc-new-dot')).toBeNull();
    expect(two.querySelector('.mc-new-dot')).not.toBeNull();
  });

  it('panel_close', () => {
    sent.length = 0;
    const { container } = render(<QuestScreen data={withPanel} />);
    const panel = container.querySelector('.mc-quest-panel')!;
    expect(panel.querySelector('.mc-slot.mc-big')).not.toBeNull();
    expect(panel.querySelector('.mc-grid .mc-slot')).not.toBeNull();
    expect(panel.querySelector('.mc-btn.mc-action')!.textContent).toBe('Go');
    down(panel.querySelector('.mc-quest-panel-head .mc-close')!);
    expect(sent).toEqual([['pc', 'click']]);
  });
});
