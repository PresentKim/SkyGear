import { useEffect, useRef, useState } from 'react';
import type { ProbeData, ProbeNode } from '../protocol';
import { send } from '../runtime/engine';
import { Rich } from '../text/Rich';
import { Field } from '../components/Field';
import { Scroll } from '../components/Scroll';
import { Window } from '../components/Window';

const PROBE_TYPES = ['mousedown', 'mouseup', 'click', 'dblclick', 'contextmenu', 'wheel', 'keydown', 'keyup', 'keypress',
  'touchstart', 'touchend', 'touchcancel', 'mousemove', 'touchmove', 'pointerdown', 'pointerup', 'pointercancel', 'pointermove',
  'focus', 'blur', 'focusin', 'focusout', 'input', 'change', 'compositionstart', 'compositionupdate', 'compositionend',
  'scroll', 'visibilitychange'];
const PROBE_WINDOW_TYPES = ['resize', 'gamepadconnected', 'gamepaddisconnected'];
const PROBE_MOVES: Record<string, boolean> = { mousemove: true, touchmove: true, pointermove: true, scroll: true };

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Any = any;

function commandList(): string[] {
  const c = (window as Any).__commands__ || {};
  const out: string[] = [];
  for (const g in c) {
    const names: string[] = [];
    for (const k in c[g]) names.push(k);
    out.push(g + ': ' + names.join(' '));
  }
  return out;
}

/** probeLineOf builds the legacy log line of an input event. */
function probeLineOf(ev: Any, downAt: { v: number }): string {
  const t: string = ev.type;
  const parts = [t];
  if (/mouse|click|contextmenu|wheel|pointer/.test(t)) parts.push('button=' + ev.button, 'buttons=' + ev.buttons, 'detail=' + ev.detail);
  if (/down$|start$/.test(t) && ev.clientX !== undefined) parts.push('x=' + Math.round(ev.clientX), 'y=' + Math.round(ev.clientY));
  if (/^pointer/.test(t)) parts.push('pointerType=' + ev.pointerType, 'pointerId=' + ev.pointerId);
  if (t === 'wheel') parts.push('dx=' + Math.round(ev.deltaX || 0), 'dy=' + Math.round(ev.deltaY || 0), 'mode=' + ev.deltaMode);
  if (/^key/.test(t)) parts.push('key=' + ev.key, 'code=' + ev.code, 'keyCode=' + ev.keyCode, 'repeat=' + ev.repeat);
  if (/^composition|^input$/.test(t)) parts.push('data=' + ev.data);
  if (t === 'change' || t === 'input') parts.push('value=' + (ev.target && ev.target.value));
  if (t === 'resize') parts.push('w=' + window.innerWidth, 'h=' + window.innerHeight);
  if (/^gamepad/.test(t) && ev.gamepad) parts.push('index=' + ev.gamepad.index, 'id=' + ev.gamepad.id);
  if (ev.touches) parts.push('touches=' + ev.touches.length);
  if (t === 'mousedown' || t === 'touchstart') downAt.v = Date.now();
  if ((t === 'mouseup' || t === 'touchend') && downAt.v) parts.push('held=' + (Date.now() - downAt.v) + 'ms');
  const mods: string[] = [];
  if (ev.shiftKey) mods.push('shift');
  if (ev.ctrlKey) mods.push('ctrl');
  if (ev.altKey) mods.push('alt');
  if (ev.metaKey) mods.push('meta');
  if (mods.length) parts.push(mods.join('+'));
  if (ev.target && ev.target.className) parts.push('on=' + String(ev.target.className).split(' ')[0]);
  return parts.join(' ');
}

function gamepadLine(): string {
  if (!navigator.getGamepads) return '';
  let pads: Any;
  try {
    pads = navigator.getGamepads() || [];
  } catch (e) {
    return '';
  }
  const out: string[] = [];
  for (let i = 0; i < pads.length; i++) {
    const g = pads[i];
    if (!g) continue;
    const held: number[] = [];
    const axes: string[] = [];
    (g.buttons || []).forEach((b: Any, j: number) => {
      if (b && (b.pressed || b.value > 0.5)) held.push(j);
    });
    (g.axes || []).forEach((a: number, j: number) => {
      if (Math.abs(a) > 0.3) axes.push(j + ':' + a.toFixed(1));
    });
    out.push('gamepad ' + g.index + ' buttons=' + held.join(',') + ' axes=' + axes.join(','));
  }
  return out.join(' | ');
}

function ProbeLog(props: { id: string }): JSX.Element {
  const st = useRef({
    log: [] as string[], moves: { mousemove: 0, touchmove: 0, pointermove: 0, scroll: 0 } as Record<string, number>,
    downAt: { v: 0 }, pads: '', last: '', repeats: 0, envSent: false,
  });
  const [, redraw] = useState(0);
  useEffect(() => {
    const s = st.current;
    const draw = () => redraw((n) => n + 1);
    const line = (text: string) => {
      if (text === s.last) {
        s.repeats++;
        s.log[0] = text + ' §7x' + (s.repeats + 1);
        return draw();
      }
      s.last = text;
      s.repeats = 0;
      s.log.unshift(text);
      if (s.log.length > 16) s.log.length = 16;
      draw();
      send(props.id, 'probe', text);
    };
    const onEvent = (ev: Event) => {
      if (PROBE_MOVES[ev.type]) {
        s.moves[ev.type]++;
        return draw();
      }
      line(probeLineOf(ev, s.downAt));
    };
    if (!s.envSent) {
      s.envSent = true;
      send(props.id, 'probe', 'env engine=' + typeof (window as Any).engine + ' size=' + window.innerWidth + 'x' + window.innerHeight +
        ' getGamepads=' + typeof navigator.getGamepads + ' PointerEvent=' + typeof (window as Any).PointerEvent +
        ' href=' + location.href + ' | ' + commandList().join(' | '));
    }
    PROBE_TYPES.forEach((t) => document.addEventListener(t, onEvent, true));
    PROBE_WINDOW_TYPES.forEach((t) => window.addEventListener(t, onEvent));
    const timer = setInterval(() => {
      const p = gamepadLine();
      if (p === s.pads) return;
      s.pads = p;
      if (p) line(p);
    }, 200);
    return () => {
      PROBE_TYPES.forEach((t) => document.removeEventListener(t, onEvent, true));
      PROBE_WINDOW_TYPES.forEach((t) => window.removeEventListener(t, onEvent));
      clearInterval(timer);
    };
  }, [props.id]);
  const s = st.current;
  const cmds = commandList();
  const lines = ['§emoves§r mouse ' + s.moves.mousemove + ', touch ' + s.moves.touchmove + ', pointer ' + s.moves.pointermove +
    ', scroll ' + s.moves.scroll, '§eevents§r'].concat(s.log);
  lines.push('§ecommand groups§r ' + cmds.length);
  return <Rich text={lines.concat(cmds).join('\n')} className="oui-probe" />;
}

const RES_TIMEOUT = 4000;
const RES_SETTLE_MS = 300;

interface ResTest {
  check?: () => boolean;
  run: (finish: (ok: boolean, detail?: string) => void) => void;
}

/** ResRunner holds the state the resource tests share: the tag a script reported and the styled element. */
class ResRunner {
  ran: string | null = null;
  probe: HTMLElement | null = null;

  styled = (): boolean => {
    if (!this.probe) {
      this.probe = document.createElement('div');
      this.probe.className = 'oui-res-probe';
      this.probe.style.position = 'absolute';
      this.probe.style.left = '-100px';
      this.probe.style.top = '0px';
      document.body.appendChild(this.probe);
    }
    return Math.round(this.probe.getBoundingClientRect().width) === 13;
  };

  styleNote = (): string => {
    if (!this.probe) return '';
    let cs = '?';
    try {
      cs = window.getComputedStyle(this.probe).width;
    } catch (e) {
      /* keep ? */
    }
    return ' w=' + this.probe.getBoundingClientRect().width + ' computed=' + cs;
  };

  html(text: unknown): boolean {
    return typeof text === 'string' && text.indexOf('oui-res-html') >= 0;
  }

  // element adds e to the head and passes when ok() holds RES_SETTLE_MS after it loads (or is added), then takes it away.
  element(e: HTMLElement, ok: () => boolean, now: boolean, note?: () => string): ResTest {
    return {
      check: ok,
      run: (finish) => {
        let over = false;
        const done = (label: string) => {
          if (over) return;
          over = true;
          setTimeout(() => {
            const pass = ok();
            const detail = label + (note ? note() : '');
            if (e.parentNode) e.parentNode.removeChild(e);
            finish(pass, detail);
          }, RES_SETTLE_MS);
        };
        e.onload = () => done('onload');
        e.onerror = () => {
          if (over) return;
          over = true;
          if (e.parentNode) e.parentNode.removeChild(e);
          finish(false, 'onerror');
        };
        this.styled();
        document.head.appendChild(e);
        if (now) done('added');
      },
    };
  }

  script(src: string | null, text?: string): ResTest {
    const s = document.createElement('script');
    if (src) s.src = src;
    else s.textContent = text || '';
    return this.element(s, () => this.ran !== null, !src);
  }

  sheet(href: string): ResTest {
    const l = document.createElement('link');
    l.rel = 'stylesheet';
    l.href = href;
    return this.element(l, this.styled, false, this.styleNote);
  }

  image(src: string): ResTest {
    const i = document.createElement('img');
    return {
      run: (finish) => {
        i.onload = () => finish(true, 'onload');
        i.onerror = () => finish(false, 'onerror');
        i.src = src;
      },
    };
  }

  fetch(url: string): ResTest {
    return {
      run: (finish) => {
        if (typeof window.fetch !== 'function') return finish(false, 'no fetch');
        window.fetch(url)
          .then((r) => r.text().then((t) => finish(this.html(t), 'status=' + r.status + ' ' + t.length + 'B')))
          .catch((err) => finish(false, 'catch ' + err));
      },
    };
  }

  xhr(url: string): ResTest {
    return {
      run: (finish) => {
        if (typeof window.XMLHttpRequest === 'undefined') return finish(false, 'no XMLHttpRequest');
        const x = new XMLHttpRequest();
        x.onload = () => finish(this.html(x.responseText), 'status=' + x.status + ' ' + String(x.responseText || '').length + 'B');
        x.onerror = () => finish(false, 'onerror status=' + x.status);
        x.open('GET', url);
        x.send();
      },
    };
  }

  tests(n: ProbeNode): [string, ResTest][] {
    const js = (n.js as string) || '';
    const css = (n.css as string) || '';
    const html = (n.html as string) || '';
    const now = (f: (finish: (ok: boolean, detail?: string) => void) => void): ResTest => ({ run: (finish) => f(finish) });
    const style = document.createElement('style');
    style.textContent = css;
    const tests: [string, ResTest][] = [
      ['packet eval', now((finish) => {
        (0, eval)(js);
        finish(this.ran !== null);
      })],
      ['packet Function', now((finish) => {
        new Function(js)();
        finish(this.ran !== null);
      })],
      ['packet inline script', this.script(null, js)],
      ['packet data: script', this.script('data:text/javascript;charset=utf-8,' + encodeURIComponent(js))],
      ['packet blob: script', now((finish) => {
        if (typeof window.Blob === 'undefined' || !window.URL || !URL.createObjectURL) return finish(false, 'no Blob');
        this.script(URL.createObjectURL(new Blob([js]))).run(finish);
      })],
      ['packet style', this.element(style, this.styled, true, this.styleNote)],
      ['packet data: link', this.sheet('data:text/css;charset=utf-8,' + encodeURIComponent(css))],
      ['packet innerHTML', now((finish) => {
        const d = document.createElement('div');
        d.innerHTML = html;
        finish(!!d.querySelector('.oui-res-html'));
      })],
      ['pack script', this.script('/hbui/oui-probe.js')],
      ['pack link', this.sheet('/hbui/oui-probe.css')],
      ['pack fetch', this.fetch('/hbui/oui-probe.html')],
      ['pack xhr', this.xhr('/hbui/oui-probe.html')],
      ['pack:// xhr', this.xhr('pack://hbui/oui-probe.html')],
    ];
    ((n.http as { name: string; base: string }[]) || []).forEach((h) => {
      const q = '?t=' + encodeURIComponent(h.name);
      tests.push(
        [h.name + ' script', this.script(h.base + '/probe.js' + q)],
        [h.name + ' link', this.sheet(h.base + '/probe.css' + q)],
        [h.name + ' img', this.image(h.base + '/probe.png' + q)],
        [h.name + ' fetch', this.fetch(h.base + '/probe.html' + q)],
        [h.name + ' xhr', this.xhr(h.base + '/probe.html' + q)],
      );
    });
    return tests;
  }
}

function ResTestView(props: { id: string; node: ProbeNode }): JSX.Element {
  const [lines, setLines] = useState<string[]>([]);
  useEffect(() => {
    const r = new ResRunner();
    (window as Any).__ouiRes = (tag: unknown) => {
      r.ran = String(tag);
    };
    const env = 'href=' + location.href + ' origin=' + location.origin + ' fetch=' + typeof window.fetch + ' XMLHttpRequest=' +
      typeof window.XMLHttpRequest + ' Blob=' + typeof window.Blob + ' createObjectURL=' + typeof (window.URL && URL.createObjectURL);
    const out = ['§eenv§r ' + env];
    setLines(out.slice());
    const line = (s: string) => {
      out.push(s);
      setLines(out.slice());
    };
    send(props.id, 'probe', 'res-env ' + env);
    const tests = r.tests(props.node);
    let i = 0;
    let alive = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let next0: ReturnType<typeof setTimeout> | undefined;
    const next = () => {
      if (!alive) return;
      if (i >= tests.length) return line('§edone§r');
      const [name, t] = tests[i++];
      let over = false;
      r.ran = null;
      const finish = (ok: boolean, detail?: string) => {
        if (over || !alive) return;
        over = true;
        clearTimeout(timer);
        const s = name + ' ' + (ok ? 'ok' : 'fail') + (detail ? ' ' + detail : '') + (r.ran !== null ? ' ran=' + r.ran : '');
        line((ok ? '§a' : '§c') + s);
        send(props.id, 'probe', 'res ' + s);
        next0 = setTimeout(next, 50);
      };
      timer = setTimeout(() => finish(t.check ? t.check() : false, 'timeout'), RES_TIMEOUT);
      try {
        t.run(finish);
      } catch (err) {
        finish(false, 'throw ' + err);
      }
    };
    next0 = setTimeout(next, 100);
    return () => {
      alive = false;
      clearTimeout(timer);
      clearTimeout(next0);
      if (r.probe && r.probe.parentNode) r.probe.parentNode.removeChild(r.probe);
    };
  }, [props.id]);
  return <Rich text={lines.join('\n')} className="oui-probe" />;
}

function NodeView(props: { node: ProbeNode; index: number }): JSX.Element {
  const n = props.node;
  const id = 'probe-' + props.index;
  switch (n.t) {
    case 'input':
      return <Field id={(n.id as string) || id} value="" placeholder={n.placeholder as string | undefined} />;
    case 'box':
      return <div className={'oui-box ' + ((n.class as string) || '')} />;
    case 'probe':
      return <ProbeLog id={id} />;
    case 'restest':
      return <ResTestView id={id} node={n} />;
    default:
      return <div className="oui-text">{'unknown node ' + n.t}</div>;
  }
}

/** ProbeScreen draws the input and resource probes the debug commands open. */
export function ProbeScreen(props: { data: ProbeData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <Rich text={data.info} className="oui-text mc-small-text" />
        {data.nodes.map((n, i) => (
          <NodeView key={i} node={n} index={i} />
        ))}
      </Scroll>
    </Window>
  );
}
