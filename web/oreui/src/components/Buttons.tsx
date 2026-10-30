import { useEffect, useState } from 'react';
import { ToggleButton } from '@ore-ui-react/oreui/ToggleButton/ToggleButton';
import type React from 'react';
import type { ReactNode } from 'react';
import type { Button, Id, ListItem, Text } from '../protocol';
import { usePressable } from '../interact/press';
import { InkContext, Rich } from '../text/Rich';
import { Icon } from './Icon';

/** McButton is a pressable Minecraft button box; the text in it is dark. */
export function McButton(props: { id: Id; className: string; tip?: Text; quick?: boolean; children?: ReactNode }): JSX.Element {
  const press = usePressable({ id: props.id, tip: props.tip, quick: props.quick });
  return (
    <div className={'oui-box ' + props.className} {...press}>
      <InkContext.Provider value="dark">{props.children}</InkContext.Provider>
    </div>
  );
}

function Face(props: { button: Button }): JSX.Element {
  return (
    <>
      <Icon icon={props.button.icon} />
      <Rich text={props.button.label} className="oui-text mc-btn-text" />
    </>
  );
}

// ToggleRow is a setting's row: the label at the left, ore-ui-react's ToggleButton at the right. The
// whole row presses; the row flips the switch at the press, so the knob slides at once, and the screen
// the server sends back sets it as it is.
function ToggleRow(props: { item: ListItem }): JSX.Element {
  const { item } = props;
  const [flipped, setFlipped] = useState(false);
  useEffect(() => setFlipped(false), [item]);
  const press = usePressable({ id: item.id });
  const down = press.onMouseDown as ((ev: React.MouseEvent<HTMLDivElement>) => void) | undefined;
  const on = flipped ? !item.on : !!item.on;
  return (
    <div
      className="oui-box mc-btn mc-list-row mc-toggle-row"
      {...press}
      onMouseDown={(ev: React.MouseEvent<HTMLDivElement>) => {
        if (item.id && ev.button === 0) setFlipped((f) => !f);
        if (down) down(ev);
      }}
    >
      <InkContext.Provider value="dark">
        <Rich text={item.label} className="oui-text mc-btn-text" />
      </InkContext.Provider>
      <div className="oui-box mc-switch">
        <ToggleButton value={on} />
      </div>
    </div>
  );
}

export function ListButton(props: { item: ListItem }): JSX.Element {
  if (props.item.kind === 'heading') return <Rich text={props.item.label} className="oui-text mc-head mc-list-heading" />;
  if (props.item.kind === 'divider') return <div className="oui-box mc-divider" />;
  if (props.item.toggle) return <ToggleRow item={props.item} />;
  return (
    <McButton id={props.item.id} className="mc-btn mc-list-row">
      <Face button={props.item} />
    </McButton>
  );
}

export function ActionButton(props: { button: Button }): JSX.Element {
  return (
    <McButton id={props.button.id} className="mc-btn mc-action">
      <Face button={props.button} />
    </McButton>
  );
}

export function Card(props: { button: Button }): JSX.Element {
  return (
    <div className="oui-box mc-card-cell">
      <McButton id={props.button.id} className="mc-btn mc-card">
        <Face button={props.button} />
      </McButton>
    </div>
  );
}
