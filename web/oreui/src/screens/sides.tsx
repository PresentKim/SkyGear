import type { Picker, Port, SideCard, SidesData, Swatch } from '../protocol';
import { Window } from '../components/Window';
import { Icon } from '../components/Icon';
import { InkContext, Rich } from '../text/Rich';
import { usePressable } from '../interact/press';

// The window size in GUI pixels.
const SIDES_W = 300;
const SIDES_H = 172;

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
  return (
    <div className="oui-box mc-port-swatch" {...press}>
      <Choice className="mc-port-colour" colour={swatch.colour} icon={swatch.icon} />
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

/** PortColumn is a port: its label, the square of its choice with the plus button on its corner, and the choice's name. */
function PortColumn(props: { port: Port }): JSX.Element {
  const { port } = props;
  return (
    <div className="oui-box mc-port">
      <Rich text={port.label} className="oui-text mc-port-label" />
      <div className="oui-box mc-port-slot">
        {port.swatch ? <PortSwatch swatch={port.swatch} /> : <div className="oui-box mc-port-swatch" />}
        <PortPlus id={port.plus.id} on={port.plus.on} />
      </div>
      <Rich text={port.swatch ? port.swatch.tip : ''} className="oui-text mc-port-name" />
    </div>
  );
}

/** SideBlock is the neighbouring block of a side, with its tooltip. */
function SideBlock(props: { icon: string; tip: string }): JSX.Element {
  const press = usePressable({ tip: props.tip });
  if (!props.icon) return <div className="oui-box mc-side-block" />;
  return (
    <div className="oui-box mc-side-block" {...press}>
      <Icon icon={props.icon} />
    </div>
  );
}

function Card(props: { card: SideCard; last: boolean }): JSX.Element {
  const { card } = props;
  const cls = 'oui-box mc-side-card' + (card.focus ? ' on' : '') + (props.last ? ' last' : '');
  return (
    <div className={cls}>
      <div className="oui-box mc-side-head">
        <SideBlock icon={card.icon} tip={card.tip} />
        <Rich text={card.name} className="oui-text mc-side-name" />
      </div>
      <div className="oui-box mc-side-ports">
        {card.ports.length === 0 ? (
          <Rich text={card.none} className="oui-text mc-port-none" />
        ) : (
          card.ports.map((port, i) => <PortColumn key={i} port={port} />)
        )}
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

function CardRow(props: { cards: SideCard[] }): JSX.Element {
  return (
    <div className="oui-box mc-side-row">
      {props.cards.map((c, i) => (
        <Card key={i} card={c} last={i === props.cards.length - 1} />
      ))}
    </div>
  );
}

/** SidesScreen shows a card for each side of a block, three to a row, the guide under them and the choices of one open port. */
export function SidesScreen(props: { data: SidesData }): JSX.Element {
  const { data } = props;
  const picker = data.picker;
  return (
    <Window chrome={data.chrome} size={[SIDES_W, SIDES_H]}>
      <InkContext.Provider value="dark">
        <div className="oui-box mc-sides">
          <CardRow cards={data.cards.slice(0, 3)} />
          <CardRow cards={data.cards.slice(3)} />
          <div className="oui-box mc-side-guide">
            <Rich text={data.guide} className="oui-text" />
          </div>
          {picker ? (
            <div className="oui-box mc-port-picker">
              {picker.choices.map((c, i) => (
                <PickChoice key={i} choice={c} />
              ))}
            </div>
          ) : (
            <div className="oui-box" />
          )}
        </div>
      </InkContext.Provider>
    </Window>
  );
}
