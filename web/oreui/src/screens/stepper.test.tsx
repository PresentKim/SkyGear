import { describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import type { ReactNode } from 'react';
import type { StepperData } from '../protocol';

vi.mock('../text/Rich', () => ({
  Rich: (p: { text: string; className?: string }) => <div className={p.className}>{p.text}</div>,
}));
vi.mock('../components/Window', () => ({
  Window: (p: { children?: ReactNode }) => <div className="mc-screen"><div className="mc-window">{p.children}</div></div>,
}));

import { StepperScreen } from './stepper';

const btn = (id: string) => ({ id, tip: '', icon: '', glyph: '', quick: true });

describe('stepper', () => {
  it('draws a column per stepper with a dim arrow where there is no run', () => {
    const data: StepperData = {
      chrome: { title: 'Collector', tabs: [], bar: [] },
      line: 'Range (11×5×7)',
      steppers: [
        { label: 'x', value: '5', up: btn('u'), down: btn('d') },
        { label: 'y', value: '1', up: btn('u2'), down: btn('') },
      ],
      side: [btn('box')],
    };
    const { container } = render(<StepperScreen data={data} />);
    expect(container.querySelectorAll('.mc-step-col').length).toBe(2);
    expect(container.querySelector('.mc-step-line')?.textContent).toContain('11×5×7');
    expect(container.querySelectorAll('.mc-step-arrow.off').length).toBe(1);
    expect(container.querySelectorAll('.mc-step-side .mc-square-button').length).toBe(1);
  });
});
