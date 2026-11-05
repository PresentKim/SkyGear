import type { InvGauge, InvProgress } from '../protocol';
import { usePressable } from '../interact/press';
import { Icon } from '../components/Icon';

/** Gauge is a tall sunken bar filled from the bottom (a machine's energy or fluid), its tooltip the amount. */
export function Gauge(props: { gauge: InvGauge }): JSX.Element {
  const { gauge } = props;
  const press = usePressable({ tip: gauge.tip });
  return (
    <div className={`oui-box mc-gauge ${gauge.kind}`} {...press}>
      <div className="oui-box mc-gauge-fill" style={{ height: `${gauge.fill}%` }} />
    </div>
  );
}

/** Progress is a machine's progress arrow or ring, its tooltip the percent. */
export function Progress(props: { progress: InvProgress }): JSX.Element {
  const press = usePressable({ tip: props.progress.tip });
  return (
    <div className="oui-box mc-progress" {...press}>
      <Icon icon={props.progress.icon} />
    </div>
  );
}
