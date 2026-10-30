import { useEffect, useLayoutEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import { hideTip } from '../interact/Tooltip';

type Pan = { x: number; y: number; z: number };
type Finger = { clientX: number; clientY: number; at: number };

const ZOOM_MIN = 0.5;
const ZOOM_MAX = 3;
const ZOOM_STEP = 1.25;
const FINGER_STALE = 3000;
const MARGIN_REM = 2.4;

const pans = new Map<string, Pan>();
let lastZoom = 0;

/**
 * clampPan keeps the plane from being dragged off: the children's box (unzoomed offset x, y, default
 * 0, size w, h, already scaled by the zoom) stays in the view; when it is larger than the view its
 * edges cannot come inside the view's edges, less the margin.
 */
export function clampPan(
  pan: { x: number; y: number },
  box: { w: number; h: number; x?: number; y?: number },
  view: { w: number; h: number },
  margin: number,
): { x: number; y: number } {
  const axis = (p: number, o: number, size: number, vsize: number) => {
    const a = margin - o;
    const b = vsize - margin - o - size;
    return Math.max(Math.min(a, b), Math.min(Math.max(a, b), p));
  };
  return {
    x: axis(pan.x, box.x ?? 0, box.w, view.w),
    y: axis(pan.y, box.y ?? 0, box.h, view.h),
  };
}

// Fingers on the screen by touch id, kept from the whole page's touch events before any other
// listener: Gameface may list only the touch that changed, so a later finger is not in the first's list.
let fingers: Record<string, Finger> = {};
let pinching: { zoomTo: (z: number, x: number, y: number) => void; inView: (x: number, y: number) => { x: number; y: number }; d: number; z: number } | null = null;
let panning: { pan: Pan; place: () => void; x: number; y: number; px: number; py: number } | null = null;

function fingerPoints(): Finger[] {
  const now = Date.now();
  const out: Finger[] = [];
  for (const id of Object.keys(fingers)) {
    if (now - fingers[id].at > FINGER_STALE && !pinching) delete fingers[id];
    else out.push(fingers[id]);
  }
  return out;
}

function fingerGap(f: Finger[]): number {
  return Math.hypot(f[1].clientX - f[0].clientX, f[1].clientY - f[0].clientY);
}

function trackFingers(ev: Event): void {
  const te = ev as TouchEvent;
  const ending = ev.type === 'touchend' || ev.type === 'touchcancel';
  const lists = ending ? [te.changedTouches || te.touches] : [te.touches, te.changedTouches];
  const now = Date.now();
  for (const ts of lists) {
    for (let i = 0; ts && i < ts.length; i++) {
      const t = ts[i];
      const id = String(t.identifier === undefined ? i : t.identifier);
      if (ending) delete fingers[id];
      else fingers[id] = { clientX: t.clientX, clientY: t.clientY, at: now };
    }
  }
  if (ending && (!te.touches || te.touches.length === 0) && !te.changedTouches) fingers = {};
  if (ending && fingerPoints().length < 2) pinching = null;
}

function pointOf(ev: Event): { clientX: number; clientY: number } {
  const te = ev as TouchEvent;
  const t = te.touches && te.touches.length ? te.touches[0] : te.changedTouches && te.changedTouches[0];
  return t || (ev as MouseEvent);
}

function dragCanvas(ev: Event): void {
  if (pinching) {
    const f = fingerPoints();
    if (f.length < 2) return;
    const mid = pinching.inView((f[0].clientX + f[1].clientX) / 2, (f[0].clientY + f[1].clientY) / 2);
    pinching.zoomTo((pinching.z * fingerGap(f)) / pinching.d, mid.x, mid.y);
    return;
  }
  if (!panning) return;
  const at = pointOf(ev);
  panning.pan.x = panning.px + at.clientX - panning.x;
  panning.pan.y = panning.py + at.clientY - panning.y;
  panning.place();
}

function drop(): void {
  panning = null;
  if (fingerPoints().length < 2) pinching = null;
}

let users = 0;
const TRACK = ['touchstart', 'touchmove', 'touchend', 'touchcancel'];

function listen(): () => void {
  if (users++ === 0) {
    for (const t of TRACK) document.addEventListener(t, trackFingers, true);
    for (const t of ['mousemove', 'touchmove']) document.addEventListener(t, dragCanvas);
    for (const t of ['mouseup', 'touchend', 'touchcancel']) document.addEventListener(t, drop);
  }
  return () => {
    if (--users > 0) return;
    for (const t of TRACK) document.removeEventListener(t, trackFingers, true);
    for (const t of ['mousemove', 'touchmove']) document.removeEventListener(t, dragCanvas);
    for (const t of ['mouseup', 'touchend', 'touchcancel']) document.removeEventListener(t, drop);
    fingers = {};
    pinching = null;
    panning = null;
  };
}

function remPx(): number {
  return parseFloat(window.getComputedStyle(document.documentElement).fontSize) || 16;
}

/** A pannable, zoomable plane; pan and zoom are kept per planeKey across remounts. */
export function Canvas(props: { planeKey: string; focusX: number; focusY: number; children?: ReactNode }): JSX.Element {
  const viewRef = useRef<HTMLDivElement>(null);
  const planeRef = useRef<HTMLDivElement>(null);
  const boundsRef = useRef<{ x1: number; y1: number; x2: number; y2: number } | null>(null);
  const panRef = useRef<Pan | null>(null);
  const measureRef = useRef<() => void>(() => {});
  const place = useRef<() => void>(() => {});
  const zoomTo = useRef<(z: number, cx?: number, cy?: number) => void>(() => {});
  const inView = useRef<(x: number, y: number) => { x: number; y: number }>((x, y) => ({ x, y }));
  const { planeKey, focusX, focusY } = props;

  useLayoutEffect(() => {
    const view = viewRef.current!;
    const plane = planeRef.current!;
    boundsRef.current = null;
    let pan: Pan | null = pans.get(planeKey) ?? null;
    panRef.current = pan;

    const measure = () => {
      if (!panRef.current) return;
      const pan = panRef.current;
      const pr = plane.getBoundingClientRect();
      const kids = plane.children;
      let b: { x1: number; y1: number; x2: number; y2: number } | null = null;
      for (let i = 0; i < kids.length; i++) {
        if (/oui-line-seg/.test(kids[i].className)) continue;
        const r = kids[i].getBoundingClientRect();
        const x1 = (r.left - pr.left) / pan.z;
        const y1 = (r.top - pr.top) / pan.z;
        const x2 = (r.right - pr.left) / pan.z;
        const y2 = (r.bottom - pr.top) / pan.z;
        b = b ? { x1: Math.min(b.x1, x1), y1: Math.min(b.y1, y1), x2: Math.max(b.x2, x2), y2: Math.max(b.y2, y2) } : { x1, y1, x2, y2 };
      }
      boundsRef.current = b;
    };
    measureRef.current = measure;
    place.current = () => {
      if (!pan) return;
      const b = boundsRef.current;
      if (b) {
        const c = clampPan(
          pan,
          { x: b.x1 * pan.z, y: b.y1 * pan.z, w: (b.x2 - b.x1) * pan.z, h: (b.y2 - b.y1) * pan.z },
          { w: view.clientWidth, h: view.clientHeight },
          MARGIN_REM * remPx(),
        );
        pan.x = c.x;
        pan.y = c.y;
      }
      plane.style.left = `${pan.x}px`;
      plane.style.top = `${pan.y}px`;
      plane.style.transform = `scale(${pan.z})`;
    };
    zoomTo.current = (z, cx, cy) => {
      if (!pan) return;
      z = Math.max(ZOOM_MIN, Math.min(ZOOM_MAX, z));
      if (cx === undefined || cy === undefined) {
        cx = view.clientWidth / 2;
        cy = view.clientHeight / 2;
      }
      pan.x = cx - ((cx - pan.x) * z) / pan.z;
      pan.y = cy - ((cy - pan.y) * z) / pan.z;
      pan.z = lastZoom = z;
      place.current();
    };
    inView.current = (x, y) => {
      const r = view.getBoundingClientRect();
      return { x: x - r.left - (view.clientLeft || 0), y: y - r.top - (view.clientTop || 0) };
    };

    if (!pan) {
      const touch = !!view.closest('.mc-touch') || document.body.classList.contains('mc-touch');
      const z = lastZoom || (touch ? ZOOM_STEP : 1);
      const rem = remPx();
      pan = { x: view.clientWidth / 2 - (focusX || 0) * rem * z, y: view.clientHeight / 2 - (focusY || 0) * rem * z, z };
      pans.set(planeKey, pan);
      panRef.current = pan;
    }
    place.current();
    measure();
    place.current();
    // The children may lay out after this effect (fonts, images): measure again.
    const timer = setTimeout(() => {
      measure();
      place.current();
    }, 0);
    const startDrag = (ev: Event) => {
      const te = ev as TouchEvent;
      if (te.touches && te.touches.length >= 2) return;
      const t = ev.target as HTMLElement;
      if (t !== view && t !== plane && !/oui-line-seg|oui-canvas-deco/.test(t.className)) return;
      if (!pan) return;
      hideTip();
      const at = pointOf(ev);
      panning = { pan, place: place.current, x: at.clientX, y: at.clientY, px: pan.x, py: pan.y };
    };
    const onMouseDown = (ev: Event) => {
      if ((ev as MouseEvent).button === 0) startDrag(ev);
    };
    const onTouchStart = (ev: Event) => {
      const f = fingerPoints();
      if (pan && f.length >= 2) {
        // A second finger, wherever and whenever it lands, turns the drag into a pinch.
        panning = null;
        pinching = { zoomTo: zoomTo.current, inView: inView.current, d: fingerGap(f) || 1, z: pan.z };
        return;
      }
      startDrag(ev);
    };
    const onWheel = (ev: Event) => {
      const w = ev as WheelEvent;
      if (!pan || !w.deltaY) return;
      const at = inView.current(w.clientX, w.clientY);
      zoomTo.current(w.deltaY < 0 ? pan.z * ZOOM_STEP : pan.z / ZOOM_STEP, at.x, at.y);
      w.preventDefault();
    };
    view.addEventListener('mousedown', onMouseDown);
    view.addEventListener('touchstart', onTouchStart);
    view.addEventListener('wheel', onWheel);
    const stop = listen();
    return () => {
      clearTimeout(timer);
      view.removeEventListener('mousedown', onMouseDown);
      view.removeEventListener('touchstart', onTouchStart);
      view.removeEventListener('wheel', onWheel);
      stop();
    };
    // focusX/focusY only place a new key; new data with the same key keeps the pan.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [planeKey]);

  // New children under the same key: measure again, keep the pan.
  useEffect(() => {
    measureRef.current();
    place.current();
  }, [props.children]);

  return (
    <div className="oui-canvas" ref={viewRef}>
      <div className="oui-canvas-plane" ref={planeRef}>
        {props.children}
      </div>
      <div className="oui-canvas-zoom">
        {([['+', ZOOM_STEP], ['-', 1 / ZOOM_STEP]] as const).map(([label, k]) => (
          <div
            key={label}
            className="oui-canvas-zoom-button"
            onMouseDown={(ev) => {
              if (ev.button !== 0) return;
              if (panRef.current) zoomTo.current(panRef.current.z * k);
            }}
          >
            {label}
          </div>
        ))}
      </div>
    </div>
  );
}
