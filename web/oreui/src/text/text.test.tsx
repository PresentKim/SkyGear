import { afterEach, describe, expect, it } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { localize } from './localize';
import { InkContext, Rich } from './Rich';

const w = window as any;
afterEach(() => {
  delete w.__commands__;
});

// The legacy rich() of oui-ddui.js, the oracle.
const COLOURS: Record<string, string> = {
  '0': '#000000', '1': '#0000aa', '2': '#00aa00', '3': '#00aaaa', '4': '#aa0000', '5': '#aa00aa', '6': '#ffaa00',
  '7': '#aaaaaa', '8': '#555555', '9': '#5555ff', a: '#55ff55', b: '#55ffff', c: '#ff5555', d: '#ff55ff',
  e: '#ffff55', f: '#ffffff', g: '#ddd605',
};
const ON_PANEL: Record<string, string> = { f: '0', '7': '8', '8': '7', e: '1', '6': '4', a: '2', b: '3', c: '4', d: '5' };
function legacy(cls: string, s: string, panel: boolean): HTMLElement {
  const e = document.createElement('div');
  e.className = cls;
  s.split('\n').forEach((line) => {
    const lineEl = document.createElement('div');
    lineEl.className = 'oui-line';
    if (/[^\x00-\x7f§]/.test(line)) lineEl.className += ' oui-wide';
    let colour = '', bold = false, italic = false, run = '';
    const flush = () => {
      if (!run) return;
      const span = document.createElement('span');
      span.textContent = run;
      if (colour) span.style.color = colour;
      if (bold) span.style.fontWeight = 'bold';
      if (italic) span.style.fontStyle = 'italic';
      lineEl.appendChild(span);
      run = '';
    };
    for (let j = 0; j < line.length; j++) {
      const c = line.charAt(j);
      if (c === '§' && j + 1 < line.length) {
        flush();
        let code = line.charAt(++j).toLowerCase();
        if (panel && ON_PANEL[code]) code = ON_PANEL[code];
        if (code === 'k') continue;
        if (COLOURS[code]) colour = COLOURS[code];
        else if (code === 'l') bold = true;
        else if (code === 'o') italic = true;
        else if (code === 'r') {
          colour = '';
          bold = italic = false;
        }
        continue;
      }
      run += c;
    }
    flush();
    if (!lineEl.firstChild) lineEl.textContent = ' ';
    e.appendChild(lineEl);
  });
  return e;
}

const draw = (text: string, ink: 'dark' | 'panel', className = 'oui-text mc-count') =>
  render(
    <InkContext.Provider value={ink}>
      <Rich text={text} className={className} />
    </InkContext.Provider>,
  ).container.firstElementChild as HTMLElement;

describe('localize', () => {
  it('falls back without the translate command', () => {
    expect(localize('﷐k:item.stone.name﷑Stone﷒')).toBe('Stone');
    expect(localize('﷐d:100﷑날짜﷒ ﷐a:100﷑전﷒')).toBe('날짜 전');
  });
  it('uses the engine answers', () => {
    w.__commands__ = {
      coreTranslateCommandGroup: {
        translate: { callable: (k: string) => ({ value: k === 'a.b' ? 'Client' : k }) },
        formatDate: { callable: (n: number) => 'D' + n },
        getHowLongAgoAsString: { callable: () => '' },
      },
    };
    expect(localize('﷐k:a.b﷑x﷒|﷐k:c﷑y﷒|﷐d:5﷑f﷒|﷐a:5﷑ago﷒')).toBe('Client|y|D5|ago');
  });
  it('span_unclosed', () => {
    const s = '﷐k:a﷑Stone';
    expect(localize(s)).toBe(s);
  });
});

describe('Rich', () => {
  it('bold_reset', () => {
    const spans = draw('§lA§rB', 'dark').querySelectorAll('span');
    expect(spans[0].style.fontWeight).toBe('bold');
    expect(spans[1].style.fontWeight).toBe('');
  });
  it('panel_ink', () => {
    const dark = draw('§fA', 'dark').querySelector('span')!;
    const panel = draw('§fA', 'panel').querySelector('span')!;
    expect(dark.style.color).not.toBe(panel.style.color);
    expect(panel.style.color).toBe('rgb(0, 0, 0)');
  });
  it('empty text keeps the class', () => {
    expect(draw('', 'dark', 'mc-title').className).toBe('mc-title');
  });
  it('equals the legacy DOM in both inks', () => {
    for (const ink of ['dark', 'panel'] as const) {
      const text = '§6Gold §7grey §lbold\n한글';
      expect(draw(text, ink).outerHTML).toBe(legacy('oui-text mc-count', text, ink === 'panel').outerHTML);
    }
  });
});

describe('Rich style', () => {
  it('applies style to its own element', () => {
    const e = render(<Rich text="a" className="oui-text" style={{ width: '5px', flexShrink: 0 }} />).container.firstElementChild as HTMLElement;
    expect(e.style.width).toBe('5px');
  });
});
