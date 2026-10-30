import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { createContext, useContext } from 'react';

vi.mock('../text/Rich', () => {
  const InkContext = createContext('panel');
  return {
    InkContext,
    Rich: (p: { text: string; className?: string }) => {
      const ink = useContext(InkContext);
      return <span className={(p.className ? p.className + ' ' : '') + 'ink-' + ink}>{p.text}</span>;
    },
  };
});
vi.mock('./Icon', () => ({
  Icon: (p: { icon: string; className?: string }) =>
    p.icon ? <img className={p.className} src={p.icon} /> : null,
}));
const send = vi.fn();
vi.mock('../interact/press', () => ({
  usePressable: (p: { id?: string }) => ({
    onMouseDown: () => p.id && send(p.id, 'click'),
    ref: (e: Element | null) => {
      if (!e) return;
      if (p.id) e.setAttribute('data-oui-press', '');
      else e.removeAttribute('data-oui-press');
    },
  }),
}));

import { BarRow, LobbyContext, ScreenRoot, Window, fitLobbyHead } from './Window';

const chrome = { title: 'T', tabs: [], bar: [] };
const down = (e: Element) => e.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0 }));

beforeEach(() => send.mockClear());

describe('window', () => {
  it('ScreenRoot outside the lobby', () => {
    const { container } = render(<ScreenRoot className="mc-screen mc-home"><b /></ScreenRoot>);
    expect(container.innerHTML).toBe('<div class="oui-box mc-screen mc-home"><b></b></div>');
  });
  it('BarRow', () => {
    const bar = [{ id: 'o1', tip: 'Back', icon: 'tex:b', glyph: '', quick: true }];
    const { container } = render(<BarRow bar={bar} />);
    const b = container.querySelector('div.mc-icon-button')!;
    expect(b.querySelector('img')).not.toBeNull();
    down(b);
    expect(send).toHaveBeenCalledWith('o1', 'click');
  });
  it('glyph renders text', () => {
    const bar = [{ id: 'o1', tip: '', icon: '', glyph: '?', quick: false }];
    const { container } = render(<BarRow bar={bar} />);
    expect(container.querySelector('.mc-icon-button')!.textContent).toBe('?');
    expect(container.querySelector('span')!.classList.contains('ink-dark')).toBe(true);
  });
  it('draws the window', () => {
    const { container } = render(<Window chrome={chrome}><i className="x" /></Window>);
    const win = container.querySelector('.mc-screen > .mc-window') as HTMLElement;
    expect(win.style.width).toBe('65.2rem');
    expect(win.style.height).toBe('33.2rem');
    expect(win.querySelector('.mc-toolbar > .mc-toolbar-bg')).not.toBeNull();
    expect(win.querySelector('.mc-toolbar-row > .mc-close')).not.toBeNull();
    expect(win.querySelector('.mc-title')!.textContent).toBe('T');
    expect(win.querySelector('.mc-content > .x')).not.toBeNull();
    expect(win.querySelector('.mc-tabs')).toBeNull();
    expect(win.querySelector('.mc-content')!.parentElement).toBe(win);
    expect(container.querySelector('.mc-title')!.classList.contains('ink-panel')).toBe(true);
  });
  it('close_sends_close', () => {
    const { container } = render(<Window chrome={chrome} />);
    down(container.querySelector('.mc-close')!);
    expect(send).toHaveBeenCalledWith('close', 'click');
  });
  it('tab_on_not_pressable', () => {
    const tabs = [
      { id: 't1', tip: 'A', icon: 'tex:a', on: true },
      { id: 't2', tip: 'B', icon: 'tex:b', on: false },
    ];
    const { container } = render(<Window chrome={{ ...chrome, tabs }} />);
    const els = container.querySelectorAll('.mc-tabs > .mc-tab');
    expect(els.length).toBe(2);
    expect(els[0].querySelector('.mc-tab-bg')).not.toBeNull();
    expect(els[0].querySelector('.mc-tab-front')).not.toBeNull();
    expect(els[0].querySelector('img')).not.toBeNull();
    expect(els[0].classList.contains('on')).toBe(true);
    expect(els[0].hasAttribute('data-oui-press')).toBe(false);
    expect(els[1].hasAttribute('data-oui-press')).toBe(true);
  });
  it('lobby_dark_ink', () => {
    const lobby = { title: 'Sky', logo: 'tex:l', touch: true };
    const { container } = render(
      <LobbyContext.Provider value={lobby}>
        <Window chrome={chrome} className="mc-x" />
      </LobbyContext.Provider>,
    );
    const root = container.firstElementChild!;
    expect(root.className).toBe('oui-box mc-screen mc-x mc-lobby mc-touch');
    expect(root.children[0].className).toContain('mc-lobby-preload-hover');
    expect(root.children[1].className).toContain('mc-lobby-preload-pressed');
    expect(root.children[2].className).toContain('mc-lobby-shade');
    expect(root.children[3].className).toContain('mc-lobby-head');
    expect(container.querySelector('.mc-title')!.classList.contains('ink-dark')).toBe(true);
    expect(container.querySelector('.mc-lobby-title')!.classList.contains('ink-dark')).toBe(true);
  });
});

describe('fit_lobby_head', () => {
  const rect = (left: number, top: number, right: number, bottom: number) =>
    ({ left, top, right, bottom, width: right - left, height: bottom - top }) as DOMRect;
  function setup(room: number) {
    document.body.innerHTML =
      '<div><div class="mc-lobby-head"><img><div></div></div><div class="mc-window"></div></div>';
    const head = document.querySelector('.mc-lobby-head') as HTMLElement;
    const win = document.querySelector('.mc-window') as HTMLElement;
    for (const c of Array.from(head.children)) c.getBoundingClientRect = () => rect(100, 0, 200, 40);
    win.getBoundingClientRect = () => (rect(50, 20, 250, 100));
    Object.defineProperty(window, 'innerHeight', { value: 100 + room, configurable: true });
    return { head, win };
  }
  it('moves the window down when there is room', () => {
    const { head, win } = setup(100);
    fitLobbyHead(win);
    expect(win.style.marginTop).toBe('40px');
    expect(head.style.visibility).toBe('');
  });
  it('hides the head when there is no room', () => {
    const { head, win } = setup(10);
    fitLobbyHead(win);
    expect(head.style.visibility).toBe('hidden');
    expect(win.style.marginTop).toBe('');
  });
});
