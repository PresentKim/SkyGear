import { useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import { thumbOf } from '@ore-ui-react/core';

function pointY(ev: Event): number {
  const e = ev as MouseEvent & TouchEvent;
  const t = (e.touches && e.touches[0]) || (e.changedTouches && e.changedTouches[0]);
  return t ? t.clientY : e.clientY;
}

/** Scroll is the scrolling column with the vanilla mouse scrollbar beside it; `end` keeps it at its bottom. */
export function Scroll(props: { className?: string; end?: boolean; children?: ReactNode }): JSX.Element {
  const sc = useRef<HTMLDivElement>(null);
  const bar = useRef<HTMLDivElement>(null);
  const handle = useRef<HTMLDivElement>(null);
  const stuck = useRef(!!props.end);

  // Gameface draws no scrollbar of its own; the bar shows only while the content is taller than the box.
  const update = () => {
    const c = sc.current, b = bar.current, h = handle.current;
    if (!c || !b || !h) return;
    const view = c.clientHeight, full = c.scrollHeight;
    if (full <= view + 1) {
      b.style.display = 'none';
      return;
    }
    b.style.display = 'flex';
    const track = b.clientHeight;
    const min = parseFloat(getComputedStyle(h).minHeight) || 0;
    h.style.height = Math.round(thumbOf(c.scrollTop, view, full, track, min).height) + 'px';
    const hh = h.offsetHeight;
    h.style.top = Math.round(((track - hh) * c.scrollTop) / (full - view)) + 'px';
  };

  useEffect(() => {
    const c = sc.current, b = bar.current, h = handle.current;
    if (!c || !b || !h) return;
    let drag: { y: number; top: number } | null = null;
    const onScroll = () => {
      if (props.end) stuck.current = c.scrollTop + c.clientHeight >= c.scrollHeight - 1;
      update();
    };
    const onBar = (ev: MouseEvent) => {
      const r = h.getBoundingClientRect();
      c.scrollTop += ev.clientY < r.top ? -c.clientHeight : c.clientHeight;
      update();
    };
    const onHandle = (ev: Event) => {
      drag = { y: pointY(ev), top: c.scrollTop };
      ev.stopPropagation();
    };
    const onMove = (ev: Event) => {
      if (!drag) return;
      const room = b.clientHeight - h.offsetHeight;
      if (room > 0) c.scrollTop = drag.top + ((pointY(ev) - drag.y) * (c.scrollHeight - c.clientHeight)) / room;
      update();
    };
    const onUp = () => {
      drag = null;
    };
    c.addEventListener('scroll', onScroll);
    b.addEventListener('mousedown', onBar);
    h.addEventListener('mousedown', onHandle);
    h.addEventListener('touchstart', onHandle);
    document.addEventListener('mousemove', onMove);
    document.addEventListener('touchmove', onMove);
    document.addEventListener('mouseup', onUp);
    document.addEventListener('touchend', onUp);
    // Content changes (images loading) fire no scroll event, so the bar is also checked on a timer.
    const timer = setInterval(update, 100);
    return () => {
      c.removeEventListener('scroll', onScroll);
      b.removeEventListener('mousedown', onBar);
      h.removeEventListener('mousedown', onHandle);
      h.removeEventListener('touchstart', onHandle);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('touchmove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.removeEventListener('touchend', onUp);
      clearInterval(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.end]);

  // Runs after the children change: an `end` box that was at its bottom stays there.
  useEffect(() => {
    const c = sc.current;
    if (c && props.end && stuck.current) c.scrollTop = c.scrollHeight;
    update();
  });

  return (
    <div className="mc-scroll-wrap">
      <div className={'oui-box mc-scroll mc-col' + (props.className ? ' ' + props.className : '') + (props.end ? ' oui-end' : '')} ref={sc}>
        {props.children}
      </div>
      <div className="mc-scrollbar" ref={bar}>
        <div className="mc-scroll-rail"></div>
        <div className="mc-scroll-handle" ref={handle}></div>
      </div>
    </div>
  );
}
