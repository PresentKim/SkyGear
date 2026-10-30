import { createContext, useContext, useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import type { BarButton, Chrome, Lobby, Tab } from '../protocol';
import { usePressable } from '../interact/press';
import { InkContext, Rich } from '../text/Rich';
import { Icon } from './Icon';

/** LobbyContext holds the lobby header while the player is in the lobby (null elsewhere). */
export const LobbyContext = createContext<Lobby | null>(null);

// The window size in GUI pixels: the vanilla inventory's. gu converts to rem like legacy `gu`.
const WINDOW_W = 326;
const WINDOW_H = 166;
const gu = (n: number): string => n / 5 + 'rem';

const fitted = new WeakSet<Element>();

// fitLobbyHead keeps the game's name at the top middle of a lobby screen clear of the window: where
// the window, its tabs or its control bar would cover the logo and the name, the window moves down
// when the screen has room below it, and the name is hidden when it has not. Once for each window.
export function fitLobbyHead(win: HTMLElement): void {
  const root = win.parentElement;
  const head = root && (root.querySelector('.mc-lobby-head') as HTMLElement | null);
  if (!root || !head || fitted.has(win)) return;
  fitted.add(win);
  let left = Infinity;
  let right = -Infinity;
  let bottom = -Infinity;
  for (let i = 0; i < head.children.length; i++) {
    const h = head.children[i].getBoundingClientRect();
    left = Math.min(left, h.left);
    right = Math.max(right, h.right);
    bottom = Math.max(bottom, h.bottom);
  }
  if (bottom < 0) return;
  let top = Infinity;
  [win, root.querySelector('.mc-tabs'), root.querySelector('.mc-toolbar')].forEach((e) => {
    if (!e) return;
    const r = e.getBoundingClientRect();
    if (r.width > 0 && r.left < right && r.right > left) top = Math.min(top, r.top);
  });
  const need = bottom - top;
  if (need <= 0) return;
  if (need <= window.innerHeight - win.getBoundingClientRect().bottom) {
    // the window is centred, so a margin moves it down by half of itself
    win.style.marginTop = Math.ceil(2 * need) + 'px';
  } else {
    head.style.visibility = 'hidden';
  }
}

function TabView(props: { tab: Tab }): JSX.Element {
  const { tab } = props;
  const press = usePressable({ id: tab.on ? undefined : tab.id, tip: tab.tip, quick: true });
  return (
    <div className={'oui-box' + (tab.on ? ' on' : '') + ' mc-tab'} {...press}>
      <div className="oui-box mc-tab-bg" />
      <div className="oui-box mc-tab-front" />
      <Icon icon={tab.icon} />
    </div>
  );
}

function BarItem(props: { button: BarButton }): JSX.Element {
  const b = props.button;
  const press = usePressable({ id: b.id, tip: b.tip, quick: b.quick });
  return (
    <div className="oui-box mc-icon-button" {...press}>
      {b.icon ? (
        <Icon icon={b.icon} />
      ) : (
        <InkContext.Provider value="dark">
          <Rich text={b.glyph} />
        </InkContext.Provider>
      )}
    </div>
  );
}

/** BarRow draws the buttons of the control bar. */
export function BarRow(props: { bar: BarButton[] }): JSX.Element {
  return (
    <>
      {props.bar.map((b) => (
        <BarItem key={b.id} button={b} />
      ))}
    </>
  );
}

function Close(): JSX.Element {
  const press = usePressable({ id: 'close' });
  return (
    <div className="oui-box mc-close" {...press}>
      <Icon icon="tex:textures/ui/close_button_default_light" />
    </div>
  );
}

/** ScreenRoot is the root box of a screen; in the lobby it adds the lobby skin and the dark ink. */
export function ScreenRoot(props: { className: string; children?: ReactNode }): JSX.Element {
  const lobby = useContext(LobbyContext);
  if (!lobby) return <div className={'oui-box ' + props.className}>{props.children}</div>;
  const cls = 'oui-box ' + props.className + ' mc-lobby' + (lobby.touch ? ' mc-touch' : '');
  return (
    <InkContext.Provider value="dark">
      <div className={cls}>
        <div className="oui-box mc-lobby-preload mc-lobby-preload-hover" />
        <div className="oui-box mc-lobby-preload mc-lobby-preload-pressed" />
        <div className="oui-box mc-lobby-shade" />
        <div className="oui-box mc-lobby-head">
          <Icon icon={lobby.logo} className="mc-lobby-logo" />
          <Rich text={lobby.title} className="mc-lobby-title" />
        </div>
        {props.children}
      </div>
    </InkContext.Provider>
  );
}

/** Window is the window of a screen: tabs over its top left, the control bar with the close button over its top right, the title and the content. */
export function Window(props: { chrome: Chrome; className?: string; children?: ReactNode }): JSX.Element {
  const { chrome } = props;
  const lobby = useContext(LobbyContext);
  const win = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!lobby) return;
    const t = setTimeout(() => win.current && fitLobbyHead(win.current), 100);
    return () => clearTimeout(t);
  }, [lobby]);
  return (
    <ScreenRoot className={props.className ? 'mc-screen ' + props.className : 'mc-screen'}>
      <InkContext.Provider value={lobby ? 'dark' : 'panel'}>
        <div ref={win} className="oui-box mc-window" style={{ width: gu(WINDOW_W), height: gu(WINDOW_H) }}>
          {chrome.tabs.length > 0 && (
            <div className="oui-box mc-tabs">
              {chrome.tabs.map((t) => (
                <TabView key={t.id} tab={t} />
              ))}
            </div>
          )}
          <div className="oui-box mc-toolbar">
            <div className="oui-box mc-toolbar-bg" />
            <div className="oui-box mc-toolbar-row">
              <BarRow bar={chrome.bar} />
              <Close />
            </div>
          </div>
          <Rich text={chrome.title} className="mc-title" />
          <div className="oui-box mc-content">{props.children}</div>
        </div>
      </InkContext.Provider>
    </ScreenRoot>
  );
}
