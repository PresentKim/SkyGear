import type { BarButton } from '../protocol';
import { usePressable } from '../interact/press';
import { Icon } from '../components/Icon';

/** SquareButton is a raised square button with an icon, as Refined Storage's side buttons. */
export function SquareButton(props: { button: BarButton }): JSX.Element {
  const b = props.button;
  const press = usePressable({ id: b.id, tip: b.tip, quick: true });
  return (
    <div className="oui-box mc-square-button" {...press}>
      <Icon icon={b.icon} />
    </div>
  );
}

/**
 * ButtonColumn is a column of square buttons outside the window: `tools` at the window's left top,
 * `bar` at its right beside a row (level with the row's bottom, or its top with `top`).
 */
export function ButtonColumn(props: { buttons: BarButton[]; place: 'tools' | 'bar'; top?: boolean }): JSX.Element | null {
  if (props.buttons.length === 0) return null;
  const cls = props.place === 'tools' ? 'oui-box mc-button-column mc-tools' : 'oui-box mc-button-column mc-row-bar' + (props.top ? ' top' : '');
  return (
    <div className={cls}>
      {props.buttons.map((b) => (
        <SquareButton key={b.id} button={b} />
      ))}
    </div>
  );
}
