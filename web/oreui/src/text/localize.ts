import { command, unwrap } from '../runtime/engine';

const TRANSLATE = 'coreTranslateCommandGroup';
const translated: Record<string, string | null> = {}; // key -> the client's text, null: none

/** engineText is the text the engine command `name` of the translate group gives for arg; undefined without the command, null without an answer. */
function engineText(name: string, arg: unknown): string | null | undefined {
  const f = command(TRANSLATE, name);
  if (!f) return undefined;
  let v: unknown;
  try {
    v = unwrap(f(arg));
  } catch {
    return null;
  }
  return typeof v === 'string' && v !== '' ? v : null;
}

/** tr is the client's text for a translation key, or fallback when it has none. */
export function tr(key: string, fallback?: string): string {
  if (fallback === undefined) fallback = key;
  if (!(key in translated)) {
    const v = engineText('translate', key);
    if (v === undefined) return fallback; // not ready: ask again next time
    translated[key] = v === key ? null : v;
  }
  const t = translated[key];
  return t === null ? fallback : t;
}

/** fmtDate is a moment in Unix seconds as the device's date text. */
export function fmtDate(sec: string | number, fallback: string): string {
  const v = engineText('formatDate', Number(sec));
  return v ? v : fallback;
}

/** fmtAgo is a moment in Unix seconds as how long ago it was. */
export function fmtAgo(sec: string | number, fallback: string): string {
  const v = engineText('getHowLongAgoAsString', Number(sec));
  return v ? v : fallback;
}

const SPAN_OPEN = '﷐';
const SPAN = /﷐([a-z]):([^﷑]*)﷑([^﷒]*)﷒/g;

/** localize puts the client's own text in the place of the client spans of s. */
export function localize(s: string): string {
  if (typeof s !== 'string' || s.indexOf(SPAN_OPEN) < 0) return s;
  return s.replace(SPAN, (_, kind: string, value: string, fallback: string) => {
    if (kind === 'k') return tr(value, fallback);
    if (kind === 'd') return fmtDate(value, fallback);
    if (kind === 'a') return fmtAgo(value, fallback);
    return fallback;
  });
}
