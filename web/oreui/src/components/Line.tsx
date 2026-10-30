/** A straight line from (x1, y1) to (x2, y2) in rem: a thin box turned around its start. */
export function Line(props: { x1: number; y1: number; x2: number; y2: number; className: string }): JSX.Element {
  const dx = props.x2 - props.x1;
  const dy = props.y2 - props.y1;
  return (
    <div
      className={props.className ? `oui-line-seg ${props.className}` : 'oui-line-seg'}
      style={{
        left: `${props.x1}rem`,
        top: `${props.y1}rem`,
        width: `${Math.sqrt(dx * dx + dy * dy)}rem`,
        transform: `rotate(${(Math.atan2(dy, dx) * 180) / Math.PI}deg)`,
      }}
    />
  );
}
