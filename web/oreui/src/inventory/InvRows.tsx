import type { BarButton, InvPart, InvRow } from '../protocol';
import { usePressable } from '../interact/press';
import { Rich } from '../text/Rich';
import { Icon } from '../components/Icon';
import { Scroll } from '../components/Scroll';
import { InvSlotView } from './InvSlot';
import { ButtonColumn } from './ButtonColumn';
import { Gauge, Progress } from './Gauge';

function SmallButton(props: { button: BarButton }): JSX.Element {
  const b = props.button;
  const press = usePressable({ id: b.id, tip: b.tip, quick: true });
  return (
    <div className="oui-box mc-icon-button mc-inv-button" {...press}>
      <Icon icon={b.icon} />
    </div>
  );
}

function Part(props: { part: InvPart }): JSX.Element {
  const { part } = props;
  if (part.arrow) {
    return (
      <div className="oui-box mc-arrow">
        <div className="oui-box mc-arrow-img" />
      </div>
    );
  }
  if (part.button) return <SmallButton button={part.button} />;
  if (part.gauge) return <Gauge gauge={part.gauge} />;
  if (part.progress) return <Progress progress={part.progress} />;
  const rows: JSX.Element[] = [];
  const cols = Math.max(1, part.cols);
  for (let i = 0; i < part.slots.length; i += cols) {
    rows.push(
      <div key={i} className="oui-box mc-inv-line">
        {part.slots.slice(i, i + cols).map((s) => (
          <InvSlotView key={s.id} slot={s} big={part.big} off={part.off || s.off} />
        ))}
      </div>,
    );
  }
  return <div className={part.off ? 'oui-box mc-inv-part off' : 'oui-box mc-inv-part'}>{rows}</div>;
}

/** CenteredParts keeps the gauges at the row's left edge and sets the other parts in the middle of the row; hidden gauges on the right balance the left ones. */
function CenteredParts(props: { parts: InvPart[] }): JSX.Element {
  const gauges = props.parts.filter((p) => p.gauge);
  const rest = props.parts.filter((p) => !p.gauge);
  return (
    <>
      {gauges.map((p, i) => (
        <Part key={'g' + i} part={p} />
      ))}
      <div className="oui-box mc-inv-center">
        {rest.map((p, i) => (
          <Part key={p.area || 'p' + i} part={p} />
        ))}
      </div>
      {gauges.map((_, i) => (
        <div key={'s' + i} className="oui-box mc-gauge mc-gauge-spacer" />
      ))}
    </>
  );
}

function Row(props: { row: InvRow; info?: string }): JSX.Element {
  const { row, info } = props;
  const parts = row.parts.filter((p) => p.arrow || p.button || p.gauge || p.progress || p.slots.length > 0);
  const bar = row.bar && row.bar.length > 0 ? <ButtonColumn buttons={row.bar} place="bar" top={row.barTop} /> : null;
  const body =
    parts.length === 0 && row.note ? (
      <Rich text={row.note} className="oui-text mc-body-text mc-inv-note" />
    ) : (
      <div className="oui-box mc-inv-row">
        {row.center ? <CenteredParts parts={parts} /> : parts.map((p, i) => <Part key={p.area || 'p' + i} part={p} />)}
        {row.grow ? null : bar}
      </div>
    );
  return (
    <>
      {row.heading || info ? (
        <div className="oui-box mc-inv-heading-row">
          <Rich text={row.heading} className="oui-text mc-inv-heading" />
          {info ? <Rich text={info} className="oui-text mc-inv-info" /> : null}
        </div>
      ) : null}
      {row.grow ? (
        <div className="oui-box mc-inv-growbox">
          <Scroll className="mc-inv-grow">{body}</Scroll>
          {bar}
        </div>
      ) : (
        body
      )}
    </>
  );
}

/** InvRows draws the rows of an inventory screen top to bottom; a grow row fills the height left and scrolls. The info line goes right of the first heading under the grow row, which is about the rows above it. */
export function InvRows(props: { rows: InvRow[]; info?: string }): JSX.Element {
  const grow = props.rows.findIndex((r) => r.grow);
  const infoAt = props.info ? props.rows.findIndex((r, i) => i > grow && !!r.heading) : -1;
  return (
    <>
      {infoAt < 0 && props.info ? <Rich text={props.info} className="oui-text mc-inv-info" /> : null}
      {props.rows.map((r, i) => (
        <Row key={i} row={r} info={i === infoAt ? props.info : undefined} />
      ))}
    </>
  );
}
