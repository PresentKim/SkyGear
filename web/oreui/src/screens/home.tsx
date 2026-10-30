import type { HomeButton, HomeData, HomeProfile, HomeSave } from '../protocol';
import { ScreenRoot } from '../components/Window';
import { Icon } from '../components/Icon';
import { McButton } from '../components/Buttons';
import { Rich } from '../text/Rich';

/** Pixels is a small picture drawn as boxes (w*h colours 'rrggbb', '' for none): the page has no canvas. */
export function Pixels(props: { w: number; h: number; px: string[]; className?: string }): JSX.Element {
  const { w, h, px } = props;
  const rows = [];
  for (let y = 0; y < h; y++) {
    const cells = [];
    for (let x = 0; x < w; x++) {
      const colour = px[y * w + x];
      cells.push(<div key={x} className="oui-pixel" style={colour ? { width: 100 / w + '%', backgroundColor: '#' + colour } : { width: 100 / w + '%' }} />);
    }
    rows.push(
      <div key={y} className="oui-pixels-row" style={{ height: 100 / h + '%' }}>
        {cells}
      </div>,
    );
  }
  return <div className={'oui-pixels' + (props.className ? ' ' + props.className : '')}>{rows}</div>;
}

function Profile(props: { profile: HomeProfile }): JSX.Element {
  const pr = props.profile;
  const fill = Math.min(Math.max(pr.fill, 0), 1);
  return (
    <McButton id={pr.id} className="mc-btn mc-home-profile">
      {pr.face.length === 64 ? <Pixels w={8} h={8} px={pr.face} className="mc-home-head" /> : <Icon icon={pr.icon} className="mc-home-head" />}
      <div className="oui-box mc-home-info">
        <Rich text={pr.name} className="oui-text mc-home-name" />
        {pr.title ? <Rich text={pr.title} className="oui-text mc-home-title" /> : null}
        <div className="oui-box mc-home-bar">
          <div className="oui-box mc-home-bar-fill" style={{ width: Math.floor(fill * 100 + 0.5) + '%' }} />
        </div>
        <Rich text={pr.line} className="oui-text mc-home-bar-line" />
      </div>
    </McButton>
  );
}

function Buttons(props: { list: HomeButton[]; className: string; labelled: boolean }): JSX.Element {
  return (
    <>
      {props.list.map((b, i) => (
        <McButton key={i} id={b.id} className={'mc-btn ' + props.className} tip={props.labelled ? undefined : b.label} quick={props.labelled ? undefined : true}>
          <Icon icon={b.icon} className="mc-home-ic" />
          {props.labelled ? <Rich text={b.label} className="oui-text mc-home-tile-label" /> : null}
          {b.badge > 0 ? <Rich text={String(b.badge)} className="oui-text mc-home-badge" /> : null}
        </McButton>
      ))}
    </>
  );
}

function Arrow(props: { id: string; icon: string }): JSX.Element {
  return (
    <McButton id={props.id} className={'mc-btn mc-home-arrow' + (props.id ? '' : ' off')} quick>
      <Icon icon={props.icon} />
    </McButton>
  );
}

function Save(props: { save: HomeSave }): JSX.Element {
  const s = props.save;
  return (
    <div className="oui-box mc-panel mc-home-save">
      <Icon icon={s.icon} className="mc-home-save-pic" />
      <div className="oui-box mc-home-save-text">
        <Rich text={s.head} className="oui-text mc-home-save-head" />
        <Rich text={s.name} className="oui-text mc-home-save-name" />
        {s.lines.map((line, i) => (
          <Rich key={i} text={line} className="oui-text mc-home-save-line" />
        ))}
      </div>
      {s.arrows ? (
        <div className="oui-box mc-home-arrows">
          <Arrow id={s.prev} icon="tex:textures/ui/sg/chevron_left" />
          <Arrow id={s.next} icon="tex:textures/ui/sg/chevron_right" />
        </div>
      ) : null}
    </div>
  );
}

/** HomeScreen is the lobby home: profile, icon buttons, feed, the chosen save and the play button. */
export function HomeScreen(props: { data: HomeData }): JSX.Element {
  const d = props.data;
  const o = d.overlay;
  return (
    <ScreenRoot className="mc-screen mc-home">
      <Profile profile={d.profile} />
      <div className="oui-box mc-home-corner">
        <Buttons list={d.corner} className="mc-home-sq" labelled={false} />
      </div>
      <div className="oui-box mc-home-side">
        <Buttons list={d.side} className="mc-home-tile" labelled />
      </div>
      {d.feed.length > 0 ? (
        <div className="oui-box mc-home-feed">
          {d.feed.map((line, i) => (
            <Rich key={i} text={line} className="oui-text mc-home-feed-line" />
          ))}
        </div>
      ) : null}
      <div className="oui-box mc-home-start">
        <Save save={d.save} />
        <McButton id={d.play.id} className="mc-home-play">
          <Rich text={d.play.label} className="oui-text mc-home-play-label" />
          {d.play.sub ? <Rich text={d.play.sub} className="oui-text mc-home-play-sub" /> : null}
        </McButton>
        {d.manage ? (
          <McButton id={d.manage.id} className="mc-btn mc-home-manage">
            <Rich text={d.manage.label} className="oui-text mc-btn-text" />
          </McButton>
        ) : null}
      </div>
      {o ? (
        <div className="oui-box mc-home-ov">
          <div className="oui-box mc-home-ov-panel">
            <Rich text={o.title} className="oui-text mc-home-ov-title" />
            {o.body ? <Rich text={o.body} className="oui-text mc-home-ov-body" /> : null}
            {o.buttons.map((b, i) => (
              <McButton key={i} id={b.id} className="mc-btn mc-home-ov-btn">
                <Rich text={b.label} className="oui-text mc-btn-text" />
              </McButton>
            ))}
          </div>
        </div>
      ) : null}
    </ScreenRoot>
  );
}
