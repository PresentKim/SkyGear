import { createFacet, type Facet } from '@react-facet/core';
import type { Root } from '../protocol';
import { FORM_PROPERTY, STORE_NAME, hasEngine, onError, subscribe, unwrap } from './engine';
import { keepSettings } from './settings';

export type DocState = { active: boolean; form: string; root: Root | null; error: string; updates: number; bytes: number };

const SCREEN = 'minecraft:custom_form';

const empty: DocState = { active: false, form: '', root: null, error: '', updates: 0, bytes: 0 };
const facet = createFacet<DocState>({ initialValue: empty });
const setDoc = (v: DocState) => facet.set(v);

/** docFacet is the form the server shows: its newest root and the state of the link. */
export const docFacet: Facet<DocState> = facet;

let screen = '';
let text = '';
let cur: DocState = empty;

/**
 * publish sends a new state only when a field changed. `updates` is a debug counter: it is
 * bumped on the current object without publishing, and only the Status line reads it.
 */
function publish(patch: Partial<DocState>): void {
  const next = { ...cur, ...patch, updates: cur.updates };
  next.active = screen === SCREEN && next.root !== null;
  if (
    next.active === cur.active && next.form === cur.form && next.root === cur.root &&
    next.error === cur.error && next.bytes === cur.bytes
  ) {
    return;
  }
  cur = next;
  if (document.body) document.body.classList.toggle('oui-active', next.active);
  setDoc(next);
}

/** startDoc subscribes to the screen id and the server's document. */
export function startDoc(): void {
  screen = '';
  text = '';
  cur = { ...empty };
  setDoc(cur);
  onError((message) => publish({ error: message }));
  if (!hasEngine()) {
    publish({ error: 'no window.engine' });
    return;
  }
  subscribe('vanilla.ddui.screenId', [], (v) => {
    v = unwrap(v);
    screen = v && typeof v === 'object' ? String((v as { screenId?: string }).screenId || '') : String(v || '');
    publish({});
  });
  subscribe('vanilla.ddui.string', [STORE_NAME, FORM_PROPERTY, 'oui', false], (v) => {
    v = unwrap(v);
    cur.updates++;
    if (typeof v !== 'string' || v === '' || v === text) return;
    try {
      const doc = JSON.parse(v) as { form: string; root: Root; settings?: Parameters<typeof keepSettings>[0] };
      text = v;
      keepSettings(doc.settings);
      publish({ form: doc.form, root: doc.root, error: '', bytes: v.length });
    } catch (e) {
      publish({ error: 'layout: ' + e });
    }
  });
}
