import type { EquipData, EquipSlot, PickCard, PickData, ResultData } from '../protocol';
import { ScreenRoot, Window } from '../components/Window';
import { Scroll } from '../components/Scroll';
import { ActionButton, McButton } from '../components/Buttons';
import { Gap, SlotView } from '../components/Slot';
import { Icon } from '../components/Icon';
import { InkContext, Rich } from '../text/Rich';
import { Body, ListButtons } from './menus';

// Screens that show something off: big cards to pick one of, a result that comes in piece by piece
// (CSS animations, staggered by an inline delay) and equipment laid out like the vanilla inventory.

/** PickScreen is the cards alone over the dimmed game, no window: the title and the body over them,
 * the list buttons (if any) under them. */
export function PickScreen(props: { data: PickData }): JSX.Element {
  const { data } = props;
  return (
    <ScreenRoot className="mc-screen mc-cardpick-screen">
      <InkContext.Provider value="dark">
        <div className="oui-box mc-cardpick-wrap">
          <Rich text={data.chrome.title} className="oui-text mc-cardpick-head" />
          {data.body ? <Rich text={data.body} className="oui-text mc-cardpick-body" /> : null}
          <div className="oui-box mc-cardpick-row">
            {data.cards.map((c, i) => (
              <PickCardView key={c.id || 'c' + i} card={c} index={i} />
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

function PickCardView(props: { card: PickCard; index: number }): JSX.Element {
  const { card } = props;
  return (
    <div className="oui-box mc-cardpick-cell" style={{ animationDelay: props.index * 0.08 + 's' }}>
      <McButton id={card.id} className={'mc-cardpick-card mc-cardpick-' + (card.rarity || 'common')}>
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
