import { useEffect, useRef, useState } from 'react';
import type { EquipData, EquipSlot, PickCard, PickData, ResultData } from '../protocol';
import { ScreenRoot, Window } from '../components/Window';
import { Scroll } from '../components/Scroll';
import { ActionButton, McButton } from '../components/Buttons';
import { Gap, SlotView } from '../components/Slot';
import { Icon } from '../components/Icon';
import { InkContext, Rich } from '../text/Rich';
import { send } from '../runtime/engine';
import { Body, ListButtons } from './menus';

// Screens that show something off: big cards to pick one of, a result that comes in piece by piece
// (CSS animations, staggered by an inline delay) and equipment laid out like the vanilla inventory.

/** pickKey is the card a key picks: 1 to 9 on the row or the keypad, -1 for another key. */
export function pickKey(ev: KeyboardEvent): number {
  const c = ev.keyCode;
  if (c >= 49 && c <= 57) return c - 49;
  if (c >= 97 && c <= 105) return c - 97;
  if (ev.key && ev.key.length === 1 && ev.key >= '1' && ev.key <= '9') return ev.key.charCodeAt(0) - 49;
  return -1;
}

/**
 * PickScreen is the cards alone over the dimmed game, no window: the title and the body over them,
 * the list buttons (if any) under them. A card rises and glows while it is chosen: under the pointer,
 * or by its number key (1, 2, 3...). The chosen card's number again, Enter or Space picks it.
 */
export function PickScreen(props: { data: PickData }): JSX.Element {
  const { data } = props;
  const [on, setOn] = useState(-1);
  const onRef = useRef(on);
  onRef.current = on;
  useEffect(() => setOn(-1), [data]);
  useEffect(() => {
    const onKey = (ev: KeyboardEvent) => {
      const cur = onRef.current;
      const k = pickKey(ev);
      const ok = ev.keyCode === 13 || ev.keyCode === 32 || ev.key === 'Enter' || ev.key === ' ';
      if (k >= 0 && k < data.cards.length) {
        ev.preventDefault();
        if (k === cur) send(data.cards[k].id, 'click');
        else setOn(k);
      } else if (ok && cur >= 0 && cur < data.cards.length) {
        ev.preventDefault();
        ev.stopImmediatePropagation();
        send(data.cards[cur].id, 'click');
      }
    };
    // capturing, so Enter and Space pick the chosen card before the keyboard selection sees them
    window.addEventListener('keydown', onKey, true);
    return () => window.removeEventListener('keydown', onKey, true);
  }, [data]);
  return (
    <ScreenRoot className="mc-screen mc-cardpick-screen">
      <InkContext.Provider value="dark">
        <div className="oui-box mc-cardpick-wrap">
          <Rich text={data.chrome.title} className="oui-text mc-cardpick-head" />
          {data.body ? <Rich text={data.body} className="oui-text mc-cardpick-body" /> : null}
          <div className="oui-box mc-cardpick-row">
            {data.cards.map((c, i) => (
              <PickCardView key={c.id || 'c' + i} card={c} index={i} on={i === on} choose={() => setOn(i)} />
            ))}
          </div>
          {data.items.length > 0 ? (
            <div className="oui-box mc-cardpick-items">
              <ListButtons items={data.items} gap />
            </div>
          ) : null}
        </div>
      </InkContext.Provider>
    </ScreenRoot>
  );
}

function PickCardView(props: { card: PickCard; index: number; on: boolean; choose: () => void }): JSX.Element {
  const { card } = props;
  return (
    <div
      className="oui-box mc-cardpick-cell"
      style={{ animationDelay: props.index * 0.08 + 's' }}
      onMouseEnter={props.choose}
      onMouseOver={props.choose}
    >
      <McButton
        id={card.id}
        className={'mc-cardpick-card mc-cardpick-' + (card.rarity || 'common') + (props.on ? ' mc-cardpick-on' : '')}
      >
        <Rich text={String(props.index + 1)} className="oui-text mc-cardpick-key" />
        <Rich text={card.tag} className="oui-text mc-cardpick-tag" />
        <div className="oui-box mc-cardpick-icon">
          <Icon icon={card.icon} />
        </div>
        <Rich text={card.title} className="oui-text mc-cardpick-title" />
        <Rich text={card.text} className="oui-text mc-cardpick-text" />
      </McButton>
    </div>
  );
}

/** ResultScreen is the headline, the stars, the figures and the rewards, coming in one after another. */
export function ResultScreen(props: { data: ResultData }): JSX.Element {
  const { data } = props;
  const starAt = 0.45;
  const rewardAt = starAt + data.stars.length * 0.25 + 0.15;
  return (
    <Window chrome={data.chrome}>
      <Scroll>
        <div className="oui-box mc-result">
          <Rich text={data.headline} className={'oui-text mc-result-head ' + (data.win ? 'win' : 'lose')} />
          <div className="oui-box mc-result-stars">
            {data.stars.map((s, i) => (
              <div
                key={i}
                className={'oui-box mc-result-star' + (s.on ? ' on' : '') + (s.new ? ' new' : '')}
                style={{ animationDelay: starAt + i * 0.25 + 's' }}
              >
                <Rich text="★" className="oui-text mc-result-star-glyph" />
              </div>
            ))}
          </div>
          <div className="oui-box mc-result-lines">
            {data.lines.map((l, i) => (
              <div key={i} className="oui-box mc-result-line" style={{ animationDelay: 0.2 + i * 0.1 + 's' }}>
                <Rich text={l.label} className="oui-text mc-result-label" />
                <Rich text={l.value} className="oui-text mc-result-value" />
              </div>
            ))}
          </div>
          {data.rewards.length > 0 ? (
            <div className="oui-box mc-result-rewards">
              {data.rewards.map((s, i) => (
                <div key={s.id || 'r' + i} className="oui-box mc-result-reward" style={{ animationDelay: rewardAt + i * 0.15 + 's' }}>
                  <SlotView slot={s} variant="big" />
                </div>
              ))}
            </div>
          ) : null}
          <Gap size={3} />
          <div className="oui-box mc-result-ok" style={{ animationDelay: rewardAt + data.rewards.length * 0.15 + 0.1 + 's' }}>
            <ActionButton button={data.ok} />
          </div>
        </div>
      </Scroll>
    </Window>
  );
}

/** EquipScreen is the equipment columns at the left, as in the vanilla inventory, and at the right
 * what can go in the chosen slot, the note and the list buttons. */
export function EquipScreen(props: { data: EquipData }): JSX.Element {
  const { data } = props;
  return (
    <Window chrome={data.chrome}>
      <div className="oui-box mc-equip">
        <div className="oui-box mc-equip-left">
          {data.groups.map((g, i) => (
            <div key={i} className="oui-box mc-equip-group">
              <Rich text={g.heading} className="oui-text mc-equip-head" />
              {g.slots.map((s, j) => (
                <EquipSlotView key={s.id || 'e' + j} slot={s} />
              ))}
            </div>
          ))}
        </div>
        <div className="oui-box mc-equip-right">
          <Scroll>
            {data.bagHeading ? <Rich text={data.bagHeading} className="oui-text mc-head" /> : null}
            <div className="oui-box mc-equip-bag">
              {data.bag.map((s, i) => (
                <SlotView key={s.id || 'b' + i} slot={s} />
              ))}
            </div>
            <Gap size={3} />
            <Body text={data.note} gap={3} />
            <ListButtons items={data.items} />
          </Scroll>
        </div>
      </div>
    </Window>
  );
}

function EquipSlotView(props: { slot: EquipSlot }): JSX.Element {
  const { slot } = props;
  return (
    <div className="oui-box mc-equip-slot">
      <SlotView slot={slot} />
      {slot.icon ? null : <Icon icon={slot.empty} className="mc-equip-empty" />}
    </div>
  );
}
