import type { Icon as IconName } from '../protocol';

/** iconSrc is the URL the engine loads an icon from. */
export function iconSrc(icon: IconName, size: number): string {
  icon = String(icon || '');
  if (icon.indexOf('tex:') === 0) return 'pack://' + icon.slice(4) + '.png?filtering=point';
  // geo-item:// looks the host up by item name, then draws "#item_id_aux" from the bag instead when
  // the bag has it: any valid name works as the host, and the id draws custom items and blocks.
  if (icon.indexOf('aux:') === 0) {
    const bag = encodeURIComponent('{"#item_id_aux":' + icon.slice(4) + '}');
    return 'geo-item://stone?width=' + size + '&height=' + size + '&bag=' + bag;
  }
  if (icon.indexOf('geo:') === 0) icon = icon.slice(4);
  return 'geo-item://' + icon + '?width=' + size + '&height=' + size;
}

export function Icon(props: { icon: IconName; className?: string; size?: number }): JSX.Element | null {
  if (!props.icon) return null;
  const cls = props.className ? 'oui-img ' + props.className : 'oui-img';
  return <img className={cls} src={iconSrc(props.icon, props.size || 64)} />;
}
