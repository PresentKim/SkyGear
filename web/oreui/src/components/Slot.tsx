import type { Slot } from '../protocol';
import { usePressable } from '../interact/press';
import { InkContext, Rich } from '../text/Rich';
import { Icon } from './Icon';

const VARIANTS = { big: ' mc-big', mini: ' mc-mini', 'jei-mesh': ' mc-jei-mesh' };

export function SlotView(props: { slot: Slot; variant?: 'big' | 'mini' | 'jei-mesh' }): JSX.Element {
  const { slot } = props;
  // A selected slot is not pressable.
  const id = slot.selected ? '' : slot.id;
  const press = usePressable({ id, alt: slot.selected ? '' : slot.alt, tip: slot.tip });
  let cls = 'oui-box mc-slot' + (props.variant ? VARIANTS[props.variant] : '');
  if (slot.selected) cls += ' on';
  else if (slot.id) cls += ' pressable';
  return (
    <div className={cls} {...press}>
      <Icon icon={slot.deco} className="mc-deco" />
      <Icon icon="tex:textures/ui/sg/slot_chosen" className="mc-chosen" />
      <Icon icon={slot.icon} className="mc-icon" />
      {slot.count ? (
        <InkContext.Provider value="dark">
          <Rich text={slot.count} className="oui-text mc-count" />
        </InkContext.Provider>
      ) : null}
    </div>
  );
}

/** Frame is an empty slot frame. */
export function Frame(): JSX.Element {
  return <div className="oui-box mc-frame" />;
}

/** Gap is empty space; the default one has no size suffix. */
export function Gap(props: { size?: 3 | 6 }): JSX.Element {
  return <div className={props.size ? 'oui-box mc-gap-' + props.size : 'oui-box mc-gap'} />;
}
