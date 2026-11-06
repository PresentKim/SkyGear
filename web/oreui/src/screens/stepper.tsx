import type { BarButton, StepperData, StepperItem } from '../protocol';
import { usePressable } from '../interact/press';
import { Rich } from '../text/Rich';
import { Icon } from '../components/Icon';
import { Window } from '../components/Window';
import { SquareButton } from '../inventory/ButtonColumn';

// The window: 176 GUI pixels wide, as the Java Edition mods' small setting windows.
const STEPPER_W = 176;
const STEPPER_H = 84;

function Arrow(props: { button: BarButton; down?: boolean }): JSX.Element {
  const b = props.button;
  const press = usePressable({ id: b.id || undefined, tip: b.tip, quick: true });
  return (
    <div className={'oui-box mc-step-arrow' + (b.id ? '' : ' off')} {...press}>
      <Icon icon={props.down ? 'tex:textures/ui/sg/step_down' : 'tex:textures/ui/sg/step_up'} />
    </div>
  );
}

function Column(props: { item: StepperItem }): JSX.Element {
  const { item } = props;
  return (
    <div className="oui-box mc-step-col">
      <Arrow button={item.up} />
      <div className="oui-box mc-step-value">
        <Rich text={item.label + ': ' + item.value} className="oui-text" />
      </div>
      <Arrow button={item.down} down />
    </div>
  );
}

/** StepperScreen is a line over a row of steppers (an up arrow, the label and value, a down arrow) and square buttons right of them. */
export function StepperScreen(props: { data: StepperData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome} className="mc-stepper" size={[STEPPER_W, STEPPER_H]}>
      <Rich text={data.line} className="oui-text mc-step-line" />
      <div className="oui-box mc-step-row">
        {data.steppers.map((s, i) => (
          <Column key={i} item={s} />
        ))}
        <div className="oui-box mc-step-side">
          {data.side.map((b) => (
            <SquareButton key={b.id} button={b} />
          ))}
        </div>
      </div>
    </Window>
  );
}
