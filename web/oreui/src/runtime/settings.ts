import { command, reportError, subscribe, unwrap } from './engine';

export type Setting = { id: string; kind: 'bool' | 'option' | 'number' | 'string'; value: unknown };

const QUERIES = { bool: 'settingsBooleanQuery', option: 'settingsOptionQuery', number: 'settingsNumberQuery', string: 'settingsStringQuery' };
const WRITES = { bool: 'updateBoolean', option: 'updateOption', number: 'updateNumber', string: 'updateString' };

interface Kept {
  kind: Setting['kind'];
  want: unknown;
  has: boolean;
  value: unknown;
}
const kept = new Map<string, Kept>();

/**
 * keepSettings holds the client settings the server asks for at their values: it watches each through the
 * settings query of its kind and writes it back when the player changes it.
 */
export function keepSettings(list: Setting[] | undefined): void {
  (list || []).forEach((s) => {
    if (!(s.kind in QUERIES)) return;
    const k = kept.get(s.id);
    if (k) {
      k.want = s.value;
      fixSetting(s.id);
      return;
    }
    kept.set(s.id, { kind: s.kind, want: s.value, has: false, value: undefined });
    subscribe('vanilla.menus.' + QUERIES[s.kind], [s.id], (v) => {
      v = unwrap(v);
      const k = kept.get(s.id)!;
      k.has = true;
      k.value = v;
      fixSetting(s.id);
    });
  });
}

function fixSetting(id: string): void {
  const k = kept.get(id)!;
  if (!k.has || k.value === k.want) return;
  const write = command('vanilla_menus_update_settings', WRITES[k.kind]);
  if (!write) return;
  try {
    write(id, k.want);
  } catch (e) {
    reportError('setting ' + id + ': ' + e);
  }
}

/** resetSettings forgets the watched settings (tests). */
export function resetSettings(): void {
  kept.clear();
}
