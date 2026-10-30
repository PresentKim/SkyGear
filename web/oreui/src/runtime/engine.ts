interface Engine {
  on(event: string, cb: (v: unknown) => void): void;
  trigger(event: string, ...args: unknown[]): void;
}

interface Command {
  callable?: (...a: unknown[]) => unknown;
}

type Win = Window & { engine?: Engine; __commands__?: Record<string, Record<string, Command>> };

const STORE = 'minecraft';
const PROPERTY = 'custom_form_data_1';

let nextQuery = 0x7e000000;
let seq = 0;
let lastPress: { x: number; y: number; w: number; h: number } | null = null;
let errorSink: (message: string) => void = () => {};
const inputs = new Map<string, () => string>();

/** onError sets where send and the settings keeper report a failure. */
export function onError(cb: (message: string) => void): void {
  errorSink = cb;
}

/** reportError tells the error sink about a failure. */
export function reportError(message: string): void {
  errorSink(message);
}

/** hasEngine tells whether the game's engine object exists. */
export function hasEngine(): boolean {
  return !!(window as Win).engine;
}

/** unwrap takes the value out of a query answer that wraps it as { value }. */
export function unwrap(v: unknown): unknown {
  if (v !== null && typeof v === 'object' && 'value' in (v as object)) return (v as { value: unknown }).value;
  return v;
}

/** subscribe asks the engine for a query and calls cb with every answer. */
export function subscribe(name: string, params: unknown[], cb: (v: unknown) => void): void {
  const engine = (window as Win).engine;
  if (!engine) return;
  const id = nextQuery++;
  engine.on('query:subscribed/' + id, cb);
  engine.on('query:updated/' + id, cb);
  engine.trigger('query:subscribe/' + name, id, ...params);
}

/** command finds a callable engine command, or null. */
export function command(group: string, name: string): ((...a: unknown[]) => unknown) | null {
  const g = (window as Win).__commands__?.[group];
  const c = g?.[name];
  if (!c || !c.callable) return null;
  return (...a: unknown[]) => c.callable!(...a);
}

function inputValues(): Record<string, string> | null {
  let out: Record<string, string> | null = null;
  inputs.forEach((read, id) => {
    out = out || {};
    out[id] = read();
  });
  return out;
}

/** send writes one event for the server into the form's data store. */
export function send(id: string, kind: string, value?: unknown): void {
  const set = command('vanillaDduiDataStoreSetCommandGroup', 'setDataStorePathString');
  seq++;
  const msg: Record<string, unknown> = { seq, id, kind, value: value === undefined ? null : value, at: kind === 'click' ? lastPress : null };
  // A press carries the text of the input boxes, so a button that submits them reads what was typed
  // even before the box's own change event.
  if (kind === 'click') msg.inputs = inputValues();
  if (!set) {
    errorSink('setDataStorePathString not found');
    return;
  }
  try {
    set(STORE, PROPERTY, 'oui_event', JSON.stringify(msg));
  } catch (e) {
    errorSink('send: ' + e);
  }
}

/** registerInput lists an input box whose text every click carries; it returns the function that removes it. */
export function registerInput(id: string, read: () => string): () => void {
  inputs.set(id, read);
  return () => {
    if (inputs.get(id) === read) inputs.delete(id);
  };
}

/** notePress remembers where the last press was, in pixels, with the window size. */
export function notePress(x: number, y: number): void {
  lastPress = { x: Math.round(x), y: Math.round(y), w: window.innerWidth, h: window.innerHeight };
}

export const STORE_NAME = STORE;
export const FORM_PROPERTY = PROPERTY;
