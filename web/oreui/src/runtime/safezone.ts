const FACET = 'core.safeZone';

type SafeZone = { safeAreaX: number; safeAreaY: number; screenPositionX: number; screenPositionY: number };
type FacetEngine = { on(event: string, cb: (v: unknown) => void): void; trigger(event: string, ...args: unknown[]): void };

let zone: SafeZone | null = null;

const clamp = (v: number, lo: number, hi: number): number => Math.min(Math.max(Number(v) || 0, lo), hi);
const margin = (part: number, size: number): number => Math.max(0, Math.round((part / 2) * size));

/** safeMargins turns the game's safe zone settings into the screen margins in pixels, as the vanilla screens do. */
export function safeMargins(z: SafeZone, w: number, h: number): { top: number; right: number; bottom: number; left: number } {
  const x = clamp(z.safeAreaX, 0.9, 1);
  const y = clamp(z.safeAreaY, 0.9, 1);
  const px = clamp(z.screenPositionX, -0.1, 0.1);
  const py = clamp(z.screenPositionY, -0.1, 0.1);
  return {
    top: margin(1 - y + py, h),
    bottom: margin(1 - y - py, h),
    left: margin(1 - x + px, w),
    right: margin(1 - x - px, w),
  };
}

function apply(): void {
  if (!zone || !document.body) return;
  const m = safeMargins(zone, window.innerWidth, window.innerHeight);
  const s = document.body.style;
  s.setProperty('--oui-safe-top', m.top + 'px');
  s.setProperty('--oui-safe-right', m.right + 'px');
  s.setProperty('--oui-safe-bottom', m.bottom + 'px');
  s.setProperty('--oui-safe-left', m.left + 'px');
}

/** watchSafeZone keeps the safe zone margins of the game's video settings in the body's --oui-safe-* properties. */
export function watchSafeZone(): void {
  const engine = (window as Window & { engine?: FacetEngine }).engine;
  if (!engine) return;
  engine.on('facet:updated:' + FACET, (v) => {
    if (!v || typeof v !== 'object') return;
    zone = v as SafeZone;
    apply();
  });
  engine.trigger('facet:request', FACET, FACET, {});
  window.addEventListener('resize', apply);
}
