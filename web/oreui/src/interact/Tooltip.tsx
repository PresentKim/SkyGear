import { createFiberRoot, createReconciler } from '@react-facet/dom-fiber';
import type { Text } from '../protocol';
import { Rich } from '../text/Rich';

let tip: HTMLElement | null = null;
let touch = false;

// setTipTouch makes the tooltip a step larger (#oui-tip.touch) while the presses come from a finger.
export function setTipTouch(on: boolean): void {
  touch = on;
}

const reconciler = createReconciler();
const roots = new WeakMap<HTMLElement, ReturnType<ReturnType<typeof createFiberRoot>>>();

function ensureTip(): HTMLElement {
  if (!tip) {
    tip = document.createElement('div');
    tip.id = 'oui-tip';
    document.body.appendChild(tip);
  }
  return tip;
}

// TooltipHost is the element the tooltip is drawn in.
export function TooltipHost(): JSX.Element {
  return (
    <div
      id="oui-tip"
      ref={(e: HTMLElement | null) => {
        if (e) tip = e;
      }}
    />
  );
}

// showTip shows text as the tooltip of el, right of it, or left of it when there is no room at the
// right, kept inside the screen.
export function showTip(el: HTMLElement, text: Text): void {
  const t = ensureTip();
  t.className = touch ? 'touch' : '';
  let root = roots.get(t);
  if (!root) {
    root = createFiberRoot(reconciler)(t);
    roots.set(t, root);
  }
  const fiber = root;
  // flushSync: the tip is measured right below, so its content must be in the DOM by then.
  reconciler.flushSync(() => reconciler.updateContainer(<Rich text={text} />, fiber, null, () => {}));
  t.style.left = '0px';
  t.style.top = '0px';
  t.style.display = 'flex';
  const r = el.getBoundingClientRect();
  const b = t.getBoundingClientRect();
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  const gap = 6;
  let x = r.right + gap;
  if (x + b.width > vw - gap) x = r.left - gap - b.width;
  x = Math.max(gap, Math.min(x, vw - gap - b.width));
  const y = Math.max(gap, Math.min(r.top, vh - gap - b.height));
  t.style.left = `${Math.round(x)}px`;
  t.style.top = `${Math.round(y)}px`;
}

// hideTip puts the tooltip away.
export function hideTip(): void {
  if (tip) tip.style.display = 'none';
}
