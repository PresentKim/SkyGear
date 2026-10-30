import type { Picker, Port, SideCard, SidesData, Swatch } from '../protocol';
import { Window } from '../components/Window';
import { Icon } from '../components/Icon';
import { Line } from '../components/Line';
import { Rich } from '../text/Rich';
import { usePressable } from '../interact/press';

// The layout in GUI pixels: the view, the card and the block in the middle. One pixel is 0.2rem.
const VIEW_W = 310;
const VIEW_H = 144;
const CARD_W = 96;
const CARD_H = 56;
const CENTRE = 20;
const REM = 0.2;

const rem = (px: number): string => `${Math.round(px * REM * 10000) / 10000}rem`;

/** Choice is the square of a choice: its colour, or its icon. */
function Choice(props: { className: string; colour: string; icon: string }): JSX.Element {
  if (props.colour) return <div className={`oui-box ${props.className}`} style={{ backgroundColor: props.colour }} />;
  return (
    <div className={`oui-box ${props.className} icon`}>
      <Icon icon={props.icon} />
    </div>
  );
}

/** PortSwatch is the choice a port has now: the left button picks the next, the right the one before. */
function PortSwatch(props: { swatch: Swatch }): JSX.Element {
  const { swatch } = props;
  const press = usePressable({ id: swatch.id, alt: swatch.alt, tip: swatch.tip });
  const style = swatch.colour ? { backgroundColor: swatch.colour } : undefined;
  return (
    <div className={swatch.colour ? 'oui-box mc-port-swatch' : 'oui-box mc-port-swatch icon'} style={style} {...press}>
      {swatch.colour ? null : <Icon icon={swatch.icon} />}
    </div>
  );
}

/** PortPlus opens the choices of a port, or closes them when open. */
function PortPlus(props: { id: string; on: boolean }): JSX.Element {
  const press = usePressable({ id: props.id });
  return (
    <div className={props.on ? 'oui-box mc-port-plus on' : 'oui-box mc-port-plus'} {...press}>
      <Rich text="+" className="oui-text" />
    </div>
  );
}

function PortRow(props: { port: Port }): JSX.Element {
  const { port } = props;
  return (
    <div className="oui-box mc-port">
      {port.swatch ? <PortSwatch swatch={port.swatch} /> : null}
      <Rich text={port.label} className="oui-text mc-port-label" />
      <PortPlus id={port.plus.id} on={port.plus.on} />
    </div>
  );
}

/** SideBlock is the neighbouring block of a side, with its tooltip. */
function SideBlock(props: { icon: string; tip: string }): JSX.Element {
  const press = usePressable({ tip: props.tip });
  if (!props.icon) {
    return (
      <div className="oui-box mc-side-block">
        <div className="oui-box mc-side-empty" />
      </div>
    );
  }
  return (
    <div className="oui-box mc-side-block" {...press}>
      <Icon icon={props.icon} />
    </div>
  );
}

function Card(props: { card: SideCard }): JSX.Element {
  const { card } = props;
  return (
    <div className={card.focus ? 'oui-box mc-side-card on' : 'oui-box mc-side-card'} style={{ left: rem(card.x), top: rem(card.y) }}>
      <div className="oui-box mc-side-head">
        <Rich text={card.arrow} className="oui-text mc-side-arrow" />
        <Rich text={card.name} className="oui-text mc-side-name" />
      </div>
      <div className="oui-box mc-side-body">
        <SideBlock icon={card.icon} tip={card.tip} />
        <div className="oui-box mc-side-ports">
          {card.ports.length === 0 ? (
            <div className="oui-box mc-port-none">
              <Rich text={card.none} className="oui-text" />
            </div>
          ) : (
            card.ports.map((port, i) => <PortRow key={i} port={port} />)
          )}
        </div>
      </div>
    </div>
  );
}

function PickChoice(props: { choice: Picker['choices'][number] }): JSX.Element {
  const { choice } = props;
  const press = usePressable({ id: choice.id, tip: choice.tip });
  return (
    <div className={choice.on ? 'oui-box mc-port-choice on' : 'oui-box mc-port-choice'} {...press}>
      <Choice className="mc-choice-swatch" colour={choice.colour} icon={choice.icon} />
    </div>
  );
}

/** SidesScreen shows a block with a card for each of its sides and the choices of one open port. */
export function SidesScreen(props: { data: SidesData }): JSX.Element {
  const { data } = props;
  const cx = VIEW_W / 2;
  const cy = VIEW_H / 2;
  const picker = data.picker;
  return (
    <Window chrome={data.chrome}>
      <div className="oui-box mc-sides" style={{ width: rem(VIEW_W), height: rem(VIEW_H) }}>
        {data.cards.map((c, i) => (
          <Line key={i} className="mc-side-line" x1={cx * REM} y1={cy * REM} x2={(c.x + CARD_W / 2) * REM} y2={(c.y + CARD_H / 2) * REM} />
        ))}
        <div className="oui-box mc-side-centre" style={{ left: rem(cx - CENTRE / 2), top: rem(cy - CENTRE / 2) }}>
          <Icon icon={data.center} />
        </div>
        {data.cards.map((c, i) => (
          <Card key={i} card={c} />
        ))}
        {picker ? (
          <div className="oui-box mc-port-picker" style={{ left: rem(picker.x), top: rem(picker.y), width: rem(picker.w) }}>
            {picker.choices.map((c, i) => (
              <PickChoice key={i} choice={c} />
            ))}
          </div>
        ) : (
          <div className="oui-box" />
        )}
      </div>
    </Window>
  );
}
