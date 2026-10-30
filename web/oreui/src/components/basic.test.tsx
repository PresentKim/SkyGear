import { describe, expect, it, vi } from 'vitest';
import { render } from '@react-facet/dom-fiber-testing-library';
import { createContext } from 'react';

vi.mock('../text/Rich', () => ({
  InkContext: createContext('dark'),
  Rich: (p: { text: string; className?: string }) => <span className={p.className}>{p.text}</span>,
}));
vi.mock('../interact/press', () => ({
  usePressable: (p: { id?: string }) => ({ ref: (e: Element | null) => e && p.id && e.setAttribute('data-oui-press', '') }),
}));

import { Icon, iconSrc } from './Icon';
import { Frame, Gap, SlotView } from './Slot';
import { ActionButton, Card, ListButton, McButton } from './Buttons';

const slot = { id: 'o1', icon: 'tex:a', count: '3', tip: 't', deco: '', selected: false };

describe('icon', () => {
  it('builds sources', () => {
    expect(iconSrc('tex:textures/ui/confirm', 32)).toBe('pack://textures/ui/confirm.png?filtering=point');
    expect(iconSrc('aux:65536', 32)).toBe('geo-item://stone?width=32&height=32&bag=%7B%22%23item_id_aux%22%3A65536%7D');
    expect(iconSrc('geo:stone', 8)).toBe('geo-item://stone?width=8&height=8');
  });
  it('empty_icon', () => {
    const { container } = render(<Icon icon="" />);
    expect(container.innerHTML).toBe('');
  });
  it('draws an img', () => {
    const { container } = render(<Icon icon="tex:a" className="mc-icon" />);
    const img = container.querySelector('img.oui-img.mc-icon')!;
    expect(img.getAttribute('src')).toBe('pack://a.png?filtering=point');
  });
});

describe('slot', () => {
  it('draws the legacy children', () => {
    const { container } = render(<SlotView slot={slot} />);
    const s = container.firstElementChild!;
    expect(s.className).toBe('oui-box mc-slot pressable');
    expect(Array.from(s.children).map((c) => c.className)).toEqual(['oui-img mc-chosen', 'oui-img mc-icon', 'oui-text mc-count']);
    expect(s.hasAttribute('data-oui-press')).toBe(true);
  });
  it('deco goes first, variant after mc-slot', () => {
    const { container } = render(<SlotView slot={{ ...slot, deco: 'tex:d' }} variant="mini" />);
    const s = container.firstElementChild!;
    expect(s.className).toBe('oui-box mc-slot mc-mini pressable');
    expect(s.children[0].className).toBe('oui-img mc-deco');
  });
  it('selected_not_pressable', () => {
    const { container } = render(<SlotView slot={{ ...slot, selected: true }} />);
    const s = container.firstElementChild!;
    expect(s.className).toBe('oui-box mc-slot on');
    expect(s.hasAttribute('data-oui-press')).toBe(false);
  });
  it('frame and gap', () => {
    expect(render(<Frame />).container.innerHTML).toBe('<div class="oui-box mc-frame"></div>');
    expect(render(<Gap size={3} />).container.innerHTML).toBe('<div class="oui-box mc-gap-3"></div>');
    expect(render(<Gap />).container.innerHTML).toBe('<div class="oui-box mc-gap"></div>');
  });
});

describe('buttons', () => {
  it('list button', () => {
    const { container } = render(<ListButton item={{ id: 'o2', label: 'A', icon: '' }} />);
    const b = container.firstElementChild!;
    expect(b.className).toBe('oui-box mc-btn mc-list-row');
    expect(b.innerHTML).toBe('<span class="oui-text mc-btn-text">A</span>');
  });
  it('icon comes first', () => {
    const { container } = render(<ActionButton button={{ id: 'o2', label: 'A', icon: 'tex:i' }} />);
    const b = container.firstElementChild!;
    expect(b.className).toBe('oui-box mc-btn mc-action');
    expect(Array.from(b.children).map((c) => c.tagName)).toEqual(['IMG', 'SPAN']);
  });
  it('card', () => {
    const { container } = render(<Card button={{ id: 'o2', label: 'A', icon: '' }} />);
    expect(container.querySelector('div.mc-card-cell > div.mc-btn.mc-card')).not.toBeNull();
  });
  it('toggle_row', () => {
    const { container } = render(<ListButton item={{ id: 'o4', label: 'V', icon: '', toggle: true, on: true }} />);
    const b = container.firstElementChild!;
    expect(b.className).toBe('oui-box mc-btn mc-list-row mc-toggle-row');
    expect(b.querySelector('.mc-switch')!.classList.contains('on')).toBe(true);
    const off = render(<ListButton item={{ id: 'o4', label: 'V', icon: '', toggle: true, on: false }} />);
    expect(off.container.querySelector('.mc-switch')!.classList.contains('on')).toBe(false);
  });
  it('McButton holds its children', () => {
    const { container } = render(
      <McButton id="o3" className="mc-btn mc-home-sq">
        <i />
      </McButton>,
    );
    expect(container.firstElementChild!.className).toBe('oui-box mc-btn mc-home-sq');
    expect(container.querySelector('i')).not.toBeNull();
  });
});
