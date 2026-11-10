import { useEffect, useRef, useState } from 'react';
import type { MouseEvent as ReactMouseEvent } from 'react';
import { RadialMenu } from '@ore-ui-react/oreui/RadialMenu/RadialMenu';
import { sectorAtPointer, sectorPlacement } from '@ore-ui-react/core';
import type { RadialData, RadialItem } from '../protocol';
import { ScreenRoot } from '../components/Window';
import { McButton } from '../components/Buttons';
import { Icon } from '../components/Icon';
import { InkContext, Rich } from '../text/Rich';
import { send } from '../runtime/engine';
import { pickKey, pickStep } from './showcase';

// ore-ui-react's RadialMenu is drawn 24rem wide and only writes a label per sector; the screen scales
// it up and lays the icons, labels and values over it at the same places. The pointer and touch pick
// through the RadialMenu itself; the keys move a chosen sector around the ring.
const RING_SCALE = 1.3;
// The ring in the RadialMenu's rem: the middle of its 24rem box, the dead zone (RADIAL_INNER) and the
// disc's edge inside its border.
const MID = 12;
const R_IN = 4;
const R_OUT = 11.85;
// The page has no canvas and SVG is not known to draw, so a sector is a fan of thin boxes turned
// around the middle of the ring, each FAN_STEP degrees wide.
const FAN_STEP = 3;

/** fanBars are the boxes that fill the ring between radii r1 and r2 from angle a1 to a2 (degrees clockwise from the top). */
export function fanBars(a1: number, a2: number, r1: number, r2: number): { angle: number; width: number }[] {
  const n = Math.max(1, Math.ceil((a2 - a1) / FAN_STEP));
  const step = (a2 - a1) / n;
  // a little wider than the arc at r2, so the neighbours overlap with no seam
  const width = 2 * r2 * Math.tan((step * Math.PI) / 360) * 1.15;
  const out = [];
  for (let i = 0; i < n; i++) out.push({ angle: a1 + step * (i + 0.5), width });
  return out;
}

function Bar(props: { angle: number; width: number; r1: number; r2: number; className: string }): JSX.Element {
  const { width, r1, r2 } = props;
  return (
    <div
      className={'oui-box ' + props.className}
      style={{
        left: MID - width / 2 + 'rem',
        top: MID - r2 + 'rem',
        width: width + 'rem',
        height: r2 - r1 + 'rem',
        transformOrigin: '50% ' + r2 + 'rem',
        transform: 'rotate(' + props.angle + 'deg)',
      }}
    />
  );
}

/** Fan is the sector of the ring from a1 to a2 between r1 and r2. */
function Fan(props: { a1: number; a2: number; r1: number; r2: number; className: string }): JSX.Element {
  return (
    <>
      {fanBars(props.a1, props.a2, props.r1, props.r2).map((b, i) => (
        <Bar key={i} angle={b.angle} width={b.width} r1={props.r1} r2={props.r2} className={props.className} />
      ))}
    </>
  );
}

/** Sectors draws the ring's sectors: the chosen one filled, a green rim on the current ones, a spoke between each two. */
function Sectors(props: { items: RadialItem[]; chosen: number }): JSX.Element {
  const n = props.items.length;
  const span = 360 / n;
  return (
    <div className="oui-box mc-radial-sectors">
      {props.items.map((it, i) => (
        <div key={i} className="oui-box mc-radial-sector">
          {i === props.chosen ? <Fan a1={i * span - span / 2} a2={i * span + span / 2} r1={R_IN} r2={R_OUT} className="mc-radial-fan chosen" /> : null}
          {it.on ? <Fan a1={i * span - span / 2} a2={i * span + span / 2} r1={R_OUT - 0.6} r2={R_OUT} className="mc-radial-fan cur" /> : null}
        </div>
      ))}
      {n > 1 ? props.items.map((_, i) => <Bar key={'s' + i} angle={i * span + span / 2} width={0.2} r1={R_IN} r2={R_OUT} className="mc-radial-spoke" />) : null}
    </div>
  );
}

function Sector(props: { item: RadialItem; index: number; count: number; chosen: boolean }): JSX.Element {
  const { item } = props;
  const cls = 'oui-box mc-radial-item' + (props.chosen ? ' chosen' : '') + (item.on ? ' cur' : '') + (item.id ? '' : ' off');
  return (
    <div className={cls} style={sectorPlacement(props.index, props.count)}>
      <Icon icon={item.icon} className="mc-radial-ic" />
      <Rich text={item.label} className="oui-text mc-radial-label" />
      {item.value ? <Rich text={item.value} className="oui-text mc-radial-value" /> : null}
    </div>
  );
}

/**
 * RadialScreen is a ring of choices over the dimmed game, no window: the title and a line over it, the
 * chosen sector's line under it, and in the middle a button that goes back (or closes). A sector is
 * chosen under the pointer, by the arrows and WASD (clockwise from the top) or by its number; a press,
 * a tap, Enter or Space (or the number again) picks it. Backspace goes back, Esc closes.
 */
export function RadialScreen(props: { data: RadialData }): JSX.Element {
  const { data } = props;
  const n = data.items.length;
  const [on, setOn] = useState(-1);
  const onRef = useRef(on);
  onRef.current = on;
  const ring = useRef<HTMLDivElement>(null);
  useEffect(() => setOn(-1), [data.title]);
  const pick = (i: number) => {
    const it = data.items[i];
    if (it && it.id) send(it.id, 'click');
  };
  useEffect(() => {
    const onKey = (ev: KeyboardEvent) => {
      const cur = onRef.current;
      const k = pickKey(ev);
      const step = pickStep(ev);
      const ok = ev.keyCode === 13 || ev.keyCode === 32 || ev.key === 'Enter' || ev.key === ' ';
      if (step !== 0 && n > 0) {
        ev.preventDefault();
        ev.stopImmediatePropagation();
        setOn(cur < 0 ? (step > 0 ? 0 : n - 1) : (cur + step + n) % n);
      } else if (k >= 0 && k < n) {
        ev.preventDefault();
        if (k === cur) pick(k);
        else setOn(k);
      } else if (ok && cur >= 0 && cur < n) {
        ev.preventDefault();
        ev.stopImmediatePropagation();
        pick(cur);
      } else if (ev.keyCode === 8 && data.back !== 'close') {
        ev.preventDefault();
        send(data.back, 'click');
      }
    };
    // capturing, so the keys move the ring's choice before the keyboard selection sees them
    window.addEventListener('keydown', onKey, true);
    return () => window.removeEventListener('keydown', onKey, true);
  }, [data]);
  const hover = (ev: ReactMouseEvent) => {
    const root = ring.current && (ring.current.querySelector('.oui-radialmenu') as HTMLElement | null);
    if (!root) return;
    setOn(sectorAtPointer(ev.clientX, ev.clientY, root.getBoundingClientRect(), n));
  };
  const info = on >= 0 && on < n && data.items[on].info ? data.items[on].info : data.info;
  return (
    <ScreenRoot className="mc-screen mc-radial-screen">
      <InkContext.Provider value="dark">
        <div className="oui-box mc-radial-wrap">
          <Rich text={data.title} className="oui-text mc-radial-head" />
          {data.line ? <Rich text={data.line} className="oui-text mc-radial-line" /> : null}
          <div className="oui-box mc-radial-box">
            <div
              ref={ring}
              className="oui-box mc-radial-ring"
              style={{ transform: 'scale(' + RING_SCALE + ')' }}
              onMouseMove={hover}
              onMouseLeave={() => setOn(-1)}
            >
              <RadialMenu items={data.items.map(() => '')} onChange={pick} />
              <Sectors items={data.items} chosen={on} />
              <div className="oui-box mc-radial-items">
                {data.items.map((it, i) => (
                  <Sector key={i} item={it} index={i} count={n} chosen={i === on} />
                ))}
              </div>
              <McButton id={data.back} quick className="mc-radial-hub">
                <Icon icon={data.back === 'close' ? 'tex:textures/ui/close_button_default_light' : 'tex:textures/ui/sg/chevron_left'} />
              </McButton>
            </div>
          </div>
          <Rich text={info} className="oui-text mc-radial-info" />
        </div>
      </InkContext.Provider>
    </ScreenRoot>
  );
}
