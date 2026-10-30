import { useContext } from 'react';
import type { Chapter, QuestData, QuestNode, QuestPanel } from '../protocol';
import { usePressable } from '../interact/press';
import { Canvas } from '../components/Canvas';
import { Line } from '../components/Line';
import { Scroll } from '../components/Scroll';
import { Icon } from '../components/Icon';
import { SlotView } from '../components/Slot';
import { ActionButton } from '../components/Buttons';
import { BarRow, LobbyContext, ScreenRoot } from '../components/Window';
import { InkContext, Rich } from '../text/Rich';
import { Gap } from '../components/Slot';

// The window size in GUI pixels, as the other windows; gu converts to rem.
const WINDOW_W = 326;
const WINDOW_H = 166;
const gu = (n: number): string => n / 5 + 'rem';

const CLOSE_ICON = 'tex:textures/ui/close_button_default_light';

function ChapterView(props: { chapter: Chapter }): JSX.Element {
  const c = props.chapter;
  // The chapter shown is not pressable.
  const press = usePressable({ id: c.on ? undefined : c.id });
  return (
    <InkContext.Provider value="dark">
      <div className={'oui-box' + (c.on ? ' on' : '') + ' mc-chapter'} {...press}>
        <Icon icon={c.icon} />
        <div className="oui-box mc-chapter-text">
          <Rich text={c.name} className="oui-text mc-chapter-name" />
          <Rich text={c.count} className="oui-text mc-chapter-count" />
        </div>
      </div>
    </InkContext.Provider>
  );
}

function NodeView(props: { node: QuestNode }): JSX.Element {
  const n = props.node;
  const press = usePressable({ id: n.id, tip: n.tip });
  const cls = ['oui-box mc-quest', n.shape || 'square', n.state, n.tracked ? 'tracked' : ''].filter(Boolean).join(' ');
  return (
    <div className={cls} style={{ left: n.x + 'rem', top: n.y + 'rem' }} {...press}>
      <div className="oui-box mc-quest-shape" />
      <Icon icon={n.icon} />
    </div>
  );
}

function PanelView(props: { panel: QuestPanel }): JSX.Element {
  const pn = props.panel;
  const close = usePressable({ id: pn.close });
  return (
    <div className="oui-box mc-quest-panel">
      <div className="oui-box mc-quest-panel-head">
        <SlotView slot={{ id: '', icon: pn.icon, count: '', tip: '', deco: '', selected: false }} variant="big" />
        <div className="oui-box mc-col">
          <Rich text={pn.title} className="oui-text mc-quest-title" />
          <Rich text={pn.status} className="oui-text mc-small-text" />
        </div>
        <div className="oui-box mc-close" {...close}>
          <Icon icon={CLOSE_ICON} />
        </div>
      </div>
      <Scroll>
        <div className="oui-box mc-col">
          {pn.body && <Rich text={pn.body} className="oui-text mc-small-text" />}
          {pn.rows.map((r, i) => (
            <div key={i} style={{ display: 'contents' }}>
              <Gap size={3} />
              {r.heading && <Rich text={r.heading} className="oui-text mc-head" />}
              <div className="oui-box mc-grid">
                {r.slots.map((s, k) => (
                  <SlotView key={k} slot={s} />
                ))}
              </div>
            </div>
          ))}
          {pn.actions.map((a) => (
            <div key={a.id} style={{ display: 'contents' }}>
              <Gap size={3} />
              <ActionButton button={a} />
            </div>
          ))}
        </div>
      </Scroll>
    </div>
  );
}

function Close(): JSX.Element {
  const press = usePressable({ id: 'close' });
  return (
    <div className="oui-box mc-close" {...press}>
      <Icon icon={CLOSE_ICON} />
    </div>
  );
}

/** QuestScreen is the quest book: chapters at the left, the chapter's quests on a plane, and an open quest's panel over it. */
export function QuestScreen(props: { data: QuestData }): JSX.Element {
  const d = props.data;
  const lobby = useContext(LobbyContext);
  return (
    <ScreenRoot className="mc-screen">
      <InkContext.Provider value={lobby ? 'dark' : 'panel'}>
        <div className="oui-box mc-quest-window" style={{ width: gu(WINDOW_W), height: gu(WINDOW_H) }}>
          <div className="oui-box mc-toolbar">
            <div className="oui-box mc-toolbar-bg" />
            <div className="oui-box mc-toolbar-row">
              <BarRow bar={d.bar} />
              <Close />
            </div>
          </div>
          <div className="oui-box mc-chapters">
            <Scroll>
              <div className="oui-box mc-col">
                {d.chapters.map((c) => (
                  <ChapterView key={c.id} chapter={c} />
                ))}
              </div>
            </Scroll>
          </div>
          <div className="oui-box mc-quest-main">
            <div className="oui-box mc-quest-head">
              <Rich text={d.heading} className="oui-text" />
              <Rich text={d.progress} className="oui-text mc-quest-progress" />
            </div>
            <Canvas planeKey={d.key} focusX={d.focusX} focusY={d.focusY}>
              {d.lines.map((l, i) => (
                <Line key={i} x1={l.x1} y1={l.y1} x2={l.x2} y2={l.y2} className={l.cls} />
              ))}
              {d.nodes.map((n) => (
                <NodeView key={n.id} node={n} />
              ))}
            </Canvas>
            {d.panel && <PanelView panel={d.panel} />}
          </div>
        </div>
      </InkContext.Provider>
    </ScreenRoot>
  );
}
