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

function publish(patch: Partial<DocState>): void {
  const next = { ...cur, ...patch };
  next.active = screen === SCREEN && next.root !== null;
  cur = next;
  if (document.body) document.body.classList.toggle('oui-active', next.active);
  setDoc(next);
}

/** startDoc subscribes to the screen id and the server's document. */
export function startDoc(): void {
  screen = '';
  text = '';
  cur = empty;
  setDoc(empty);
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
    const updates = cur.updates + 1;
    if (typeof v !== 'string' || v === '' || v === text) {
      publish({ updates });
      return;
    }
    try {
      const doc = JSON.parse(v) as { form: string; root: Root; settings?: Parameters<typeof keepSettings>[0] };
      text = v;
      keepSettings(doc.settings);
      publish({ updates, form: doc.form, root: doc.root, error: '', bytes: v.length });
    } catch (e) {
      publish({ updates, error: 'layout: ' + e });
    }
  });
}
