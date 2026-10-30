import { beforeEach, expect, it, vi } from 'vitest';
import { startDoc, docFacet, type DocState } from './doc';
import { registerInput, send } from './engine';
import { keepSettings, resetSettings } from './settings';

type W = Record<string, unknown>;
const w = window as unknown as W;
let handlers: Record<string, (v: unknown) => void>;
let names: Record<string, string>;
let sets: unknown[][];

function state(): DocState {
  let s!: DocState;
  docFacet.observe((v) => (s = v))();
  return s;
}

function emit(name: string, v: unknown) {
  const id = Object.keys(names).find((k) => names[k] === name)!;
  handlers['query:updated/' + id](v);
}

beforeEach(() => {
  handlers = {};
  names = {};
  sets = [];
  resetSettings();
  w.engine = {
    on: (e: string, cb: (v: unknown) => void) => (handlers[e] = cb),
    trigger: (e: string, id: number, ...p: unknown[]) => (names[id] = e.replace('query:subscribe/', '') + (e.endsWith('string') ? ':' + p[1] : '')),
  };
  w.__commands__ = {
    vanillaDduiDataStoreSetCommandGroup: { setDataStorePathString: { callable: (...a: unknown[]) => sets.push(a) } },
  };
});

const DOC = '{"form":"a","root":{"screen":"menu","data":{}}}';

it('shows the document', () => {
  startDoc();
  emit('vanilla.ddui.screenId', { screenId: 'minecraft:custom_form' });
  emit('vanilla.ddui.string:custom_form_data_1', DOC);
  expect(state()).toEqual({ active: true, form: 'a', root: { screen: 'menu', data: {} }, error: '', updates: 1, bytes: 47 });
});

it('same_text_skipped', () => {
  startDoc();
  emit('vanilla.ddui.string:custom_form_data_1', DOC);
  const root = state().root;
  emit('vanilla.ddui.string:custom_form_data_1', DOC);
  expect(state().updates).toBe(2);
  expect(state().root).toBe(root);
});

it('bad_json', () => {
  startDoc();
  emit('vanilla.ddui.string:custom_form_data_1', DOC);
  emit('vanilla.ddui.string:custom_form_data_1', '{nope');
  expect(state().error).toMatch(/^layout: /);
  expect(state().root).toEqual({ screen: 'menu', data: {} });
});

it('no_engine', () => {
  delete w.engine;
  startDoc();
  expect(state().error).toBe('no window.engine');
});

it('sends a click with the inputs', () => {
  startDoc();
  const off = registerInput('text', () => 'hi');
  send('o1', 'click');
  expect(sets[0].slice(0, 3)).toEqual(['minecraft', 'custom_form_data_1', 'oui_event']);
  const msg = JSON.parse(sets[0][3] as string);
  expect(msg).toMatchObject({ seq: expect.any(Number), id: 'o1', kind: 'click', value: null, inputs: { text: 'hi' } });
  off();
});

it('setting_reverted', () => {
  const update = vi.fn();
  w.__commands__ = { vanilla_menus_update_settings: { updateOption: { callable: update } } };
  keepSettings([{ id: 'video.ui_profile', kind: 'option', value: 0 }]);
  handlers['query:updated/' + Object.keys(names).pop()]({ value: 1 });
  expect(update).toHaveBeenCalledWith('video.ui_profile', 0);
});
