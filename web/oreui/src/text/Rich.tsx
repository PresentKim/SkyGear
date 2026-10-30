import { createContext, useContext } from 'react';
import type { CSSProperties } from 'react';
import type { Text } from '../protocol';
import { localize } from './localize';

export type Ink = 'panel' | 'dark';

/** InkContext is the ink of the text being built: on the grey window the § colours become their dark variants. */
export const InkContext = createContext<Ink>('dark');

const COLOURS: Record<string, string> = {
  '0': '#000000', '1': '#0000aa', '2': '#00aa00', '3': '#00aaaa', '4': '#aa0000', '5': '#aa00aa', '6': '#ffaa00',
  '7': '#aaaaaa', '8': '#555555', '9': '#5555ff', a: '#55ff55', b: '#55ffff', c: '#ff5555', d: '#ff55ff',
  e: '#ffff55', f: '#ffffff', g: '#ddd605',
};

const ON_PANEL: Record<string, string> = { f: '0', '7': '8', '8': '7', e: '1', '6': '4', a: '2', b: '3', c: '4', d: '5' };

type Run = { text: string; style: CSSProperties };

function runsOf(line: string, panel: boolean): Run[] {
  const runs: Run[] = [];
  let colour = '', bold = false, italic = false, run = '';
  const flush = () => {
    if (!run) return;
    const style: CSSProperties = {};
    if (colour) style.color = colour;
    if (bold) style.fontWeight = 'bold';
    if (italic) style.fontStyle = 'italic';
    runs.push({ text: run, style });
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
  return runs;
}

/** Rich draws text with § colour and format codes as spans and newlines as lines, in the legacy text element. */
export function Rich(props: { text: Text; className?: string; style?: CSSProperties }): JSX.Element {
  const panel = useContext(InkContext) === 'panel';
  const s = props.text === undefined || props.text === null ? '' : localize(String(props.text));
  return (
    <div className={props.className} style={props.style}>
      {s.split('\n').map((line, i) => {
        const runs = runsOf(line, panel);
        // Gameface lays the colour runs of a line out as flex items aligned at their tops, so a run the
        // Minecraft font draws sits higher than one in the locale's Noto font: a line with any non-ASCII
        // letter is drawn in the Noto font throughout.
        const cls = /[^\x00-\x7f§]/.test(line) ? 'oui-line oui-wide' : 'oui-line';
        return (
          <div className={cls} key={i}>
            {runs.length
              ? runs.map((r, k) => (
                  <span key={k} style={Object.keys(r.style).length ? r.style : undefined}>
                    {r.text}
                  </span>
                ))
              : ' '}
          </div>
        );
      })}
    </div>
  );
}
