import type { DocState } from './runtime/doc';
import { screens } from './screens/registry';

function cssVar(name: string): string {
  try {
    return window.getComputedStyle(document.documentElement).getPropertyValue(name).trim() || '-';
  } catch (e) {
    return '?';
  }
}

function fontOf(e: Element | null): string {
  try {
    return e ? window.getComputedStyle(e).fontFamily || '-' : '-';
  } catch (err) {
    return '?';
  }
}

function remPx(): string {
  try {
    return window.getComputedStyle(document.documentElement).fontSize;
  } catch (e) {
    return '?';
  }
}

/** fontFaces are the families of the @font-face rules of the page's style sheets. */
function fontFaces(): string {
  const out: string[] = [];
  try {
    for (let i = 0; i < document.styleSheets.length; i++) {
      const rules = document.styleSheets[i].cssRules || [];
      for (let j = 0; j < rules.length; j++) {
        const r = rules[j] as CSSFontFaceRule;
        if (r.type !== 5) continue;
        const f = ((r.style && (r.style.getPropertyValue('font-family') || r.style.fontFamily)) || '').replace(/["']/g, '');
        if (f && out.indexOf(f) < 0) out.push(f);
      }
    }
  } catch (e) {
    return '? ' + e;
  }
  return out.length ? out.join(', ') : '-';
}

/** Status is the debug line: shown when `on`, or when the screen name is unknown or the link has an error. */
export function Status(props: { doc: DocState; on: boolean }): JSX.Element | null {
  const { doc } = props;
  const name = doc.root ? doc.root.screen : '';
  const unknown = doc.root !== null && !(name in screens);
  if (!props.on && !unknown && !doc.error) return null;
  const error = unknown ? 'unknown screen ' + name : doc.error;
  const lines = [
    'oui-ddui screen=' + (name || '-') + ' form=' + (doc.form || '-') + ' layout=' + doc.bytes + 'B updates=' + doc.updates +
      (error ? ' error=' + error : ''),
    'localeFonts=' + cssVar('--localeSpecificNotoSansFonts') + ' html=' + (document.documentElement.lang || '-') + ' .' +
      (document.documentElement.className || '-') + ' body=' + (document.body.className || '-'),
    'rootFont=' + fontOf(document.getElementById('root')) + ' bodyFont=' + fontOf(document.body),
    'fontFaces=' + fontFaces(),
    'w=' + window.innerWidth + ' h=' + window.innerHeight + ' rem=' + remPx() + ' ua=' + (navigator.userAgent || '-'),
  ];
  return (
    <div id="oui-ddui-status" style={{ display: 'flex' }}>
      {lines.map((l, i) => (
        <div key={i}>{l}</div>
      ))}
    </div>
  );
}
