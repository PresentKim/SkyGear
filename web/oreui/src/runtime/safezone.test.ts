import { expect, it } from 'vitest';
import { safeMargins, watchSafeZone } from './safezone';

it('full safe area leaves no margins', () => {
  expect(safeMargins({ safeAreaX: 1, safeAreaY: 1, screenPositionX: 0, screenPositionY: 0 }, 1000, 500)).toEqual({ top: 0, right: 0, bottom: 0, left: 0 });
});

it('shrunk and shifted safe area', () => {
  expect(safeMargins({ safeAreaX: 0.9, safeAreaY: 0.9, screenPositionX: 0.04, screenPositionY: -0.02 }, 1000, 500)).toEqual({ top: 20, right: 30, bottom: 30, left: 70 });
});

it('clamps out of range settings', () => {
  expect(safeMargins({ safeAreaX: 0.5, safeAreaY: 2, screenPositionX: 1, screenPositionY: 1 }, 1000, 500)).toEqual({ top: 25, right: 0, bottom: 0, left: 100 });
});

it('requests the facet and writes the margins on the body', () => {
  const on: Record<string, (v: unknown) => void> = {};
  const triggers: unknown[][] = [];
  (window as unknown as Record<string, unknown>).engine = {
    on: (e: string, cb: (v: unknown) => void) => (on[e] = cb),
    trigger: (...a: unknown[]) => triggers.push(a),
  };
  watchSafeZone();
  expect(triggers).toEqual([['facet:request', 'core.safeZone', 'core.safeZone', {}]]);
  on['facet:updated:core.safeZone']({ safeAreaX: 0.9, safeAreaY: 1, screenPositionX: 0, screenPositionY: 0 });
  const s = document.body.style;
  expect([s.getPropertyValue('--oui-safe-left'), s.getPropertyValue('--oui-safe-right'), s.getPropertyValue('--oui-safe-top')]).toEqual([
    Math.round(0.05 * window.innerWidth) + 'px',
    Math.round(0.05 * window.innerWidth) + 'px',
    '0px',
  ]);
});
