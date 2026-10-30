import { useEffect, useState } from 'react';
import type { ChoiceData, Choice, LoadingData, PlaneData, Spot, Stat } from '../protocol';
import { Window, ScreenRoot } from '../components/Window';
import { Scroll } from '../components/Scroll';
import { Gap } from '../components/Slot';
import { ActionButton, ListButton } from '../components/Buttons';
import { Icon } from '../components/Icon';
import { usePressable } from '../interact/press';
import { InkContext, Rich } from '../text/Rich';

function StatView(props: { stat: Stat }): JSX.Element {
  const { stat } = props;
  const press = usePressable({ tip: stat.label });
  return (
    <div className="oui-box mc-stat">
      <div className="oui-box mc-stat-body" {...press}>
        {stat.icon ? <Icon icon={stat.icon} className="mc-stat-icon" /> : null}
        <Rich text={stat.value} className="oui-text mc-stat-value" />
      </div>
    </div>
  );
}

// A tile is pressable unless it is the current choice.
function Tile(props: { choice: Choice }): JSX.Element {
  const { choice: c } = props;
  const press = usePressable({ id: c.on ? undefined : c.id, tip: c.tip });
  const cls = 'oui-box mc-btn mc-pick' + (c.on ? ' on' : c.dim ? ' dim' : '');
  return (
    <div className={cls} {...press}>
      <InkContext.Provider value="dark">
        <Icon icon={c.icon} />
        <Rich text={c.label} className="oui-text mc-pick-name" />
        {c.on ? <div className="oui-box mc-pick-ring" /> : null}
      </InkContext.Provider>
    </div>
  );
}

/** ChoiceScreen is the figures strip, a row of tiles per group and the buttons after them. */
export function ChoiceScreen(props: { data: ChoiceData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        {data.stats.length > 0 ? (
          <div className="oui-box mc-panel mc-stats">
            {data.stats.map((s, i) => (
              <StatView key={i} stat={s} />
            ))}
          </div>
        ) : null}
        {data.note ? <Rich text={data.note} className="oui-text mc-pick-note" /> : null}
        {data.groups.map((g, gi) => (
          <div className="oui-box mc-pick-group" key={gi}>
            <Rich text={g.title} className="oui-text mc-pick-title" />
            <div className="oui-box mc-picks">
              {g.choices.map((c, ci) => (
                <Tile key={ci} choice={c} />
              ))}
            </div>
          </div>
        ))}
        {data.items.length > 0 ? <Gap size={6} /> : null}
        {data.items.map((item) => (
          <ListButton key={item.id} item={item} />
        ))}
      </Scroll>
    </Window>
  );
}

/** LoadingScreen is the sky with clouds, the bobbing picture, the texts and the moving bar. */
export function LoadingScreen(props: { data: LoadingData }): JSX.Element {
  const { data } = props;
  return (
    <ScreenRoot className="mc-screen mc-load">
      <div className="oui-box mc-load-sky" />
      <div className="oui-box mc-load-clouds">
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className={'oui-box mc-load-cloud mc-load-cloud-' + i} />
        ))}
      </div>
      <div className="oui-box mc-load-mid">
        <div className="oui-box mc-load-isle">
          <div className="oui-box mc-load-glow" />
          <Icon icon={data.icon} className="mc-load-icon" />
        </div>
        <Rich text={'§l' + data.title} className="oui-text mc-load-title" />
        {data.sub ? <Rich text={data.sub} className="oui-text mc-load-sub" /> : null}
        <div className="oui-box mc-load-bar">
          <div className="oui-box mc-load-bar-fill" />
        </div>
        <div className="oui-box mc-load-line">
          <Rich text={data.line} className="oui-text mc-load-line-text" />
          {[0, 1, 2].map((i) => (
            <Rich key={i} text="." className={'oui-text mc-load-dot mc-load-dot-' + i} />
          ))}
        </div>
      </div>
      {data.tip ? <Rich text={data.tip} className="oui-text mc-load-tip" /> : null}
    </ScreenRoot>
  );
}

/** projectSpot places a spot of the ground a straight-down camera at height dist with vertical field of view fov shows. */
export function projectSpot(
  at: [number, number, number, number],
  fov: number,
  dist: number,
  w: number,
  h: number,
): { left: string; top: string; width: string; height: string } {
  const half = (dist || 1) * Math.tan((((fov || 60) / 2) * Math.PI) / 180);
  const scale = h / 2 / half;
  return {
    left: Math.round(w / 2 + at[0] * scale) + 'px',
    top: Math.round(h / 2 + at[1] * scale) + 'px',
    width: Math.round(at[2] * scale) + 'px',
    height: Math.round(at[3] * scale) + 'px',
  };
}

function SpotView(props: { spot: Spot; fov: number; dist: number; size: { w: number; h: number } }): JSX.Element {
  const { spot, size } = props;
  const press = usePressable({ id: spot.id, tip: spot.tip, quick: !!spot.tip });
  return (
    <div
      className={'oui-box' + (spot.on ? ' on' : '') + ' mc-spot'}
      style={projectSpot(spot.at, props.fov, props.dist, size.w, size.h)}
      {...press}
    >
      {spot.on ? <div className="oui-box mc-spot-frame" /> : null}
    </div>
  );
}

/** PlaneScreen is buttons over the ground a straight-down camera shows, with a panel of text and buttons. */
export function PlaneScreen(props: { data: PlaneData }): JSX.Element {
  const { data } = props;
  const [size, setSize] = useState({ w: window.innerWidth, h: window.innerHeight });
  useEffect(() => {
    const on = (): void => setSize({ w: window.innerWidth, h: window.innerHeight });
    window.addEventListener('resize', on);
    on();
    return () => window.removeEventListener('resize', on);
  }, []);
  return (
    <ScreenRoot className="mc-screen mc-plane-screen">
      <div className="oui-plane">
        {data.spots.map((s) => (
          <SpotView key={s.id} spot={s} fov={data.fov} dist={data.dist} size={size} />
        ))}
      </div>
      <InkContext.Provider value="dark">
        <div className="oui-box mc-plane-panel">
          <Rich text={data.info} className="oui-text mc-plane-info" />
          {data.actions.map((b) => (
            <ActionButton key={b.id} button={b} />
          ))}
        </div>
        {data.corner ? (
          <div className="oui-box mc-plane-panel mc-plane-corner">
            <Rich text={data.corner} className="oui-text mc-plane-info" />
          </div>
        ) : null}
      </InkContext.Provider>
    </ScreenRoot>
  );
}
