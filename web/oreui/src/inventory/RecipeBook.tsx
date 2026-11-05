import type { InvBook, InvTab } from '../protocol';
import { usePressable } from '../interact/press';
import { Rich } from '../text/Rich';
import { Icon } from '../components/Icon';
import { Field } from '../components/Field';
import { Scroll } from '../components/Scroll';
import { InvSlotView } from './InvSlot';

// The recipe book's 18u cells across the panel's width, beside the scrollbar.
const BOOK_COLS = 7;

function BookTab(props: { tab: InvTab }): JSX.Element {
  const { tab } = props;
  const press = usePressable({ id: tab.on ? undefined : tab.id, tip: tab.tip, quick: true });
  return (
    <div className={'oui-box mc-tab' + (tab.on ? ' on' : '')} {...press}>
      <div className="oui-box mc-tab-bg" />
      <div className="oui-box mc-tab-front" />
      <Icon icon={tab.icon} />
    </div>
  );
}

function Filter(props: { button: NonNullable<InvBook['filter']> }): JSX.Element {
  const b = props.button;
  const press = usePressable({ id: b.id, tip: b.tip, quick: true });
  return (
    <div className="oui-box mc-inv-book-filter" {...press}>
      <Icon icon={b.icon} />
    </div>
  );
}

/** RecipeBook is the vanilla recipe book left of the window: the tabs over it, the open tab's label or the search box with the filter switch, and the scrolling grid of recipes. */
export function RecipeBook(props: { book: InvBook }): JSX.Element {
  const { book } = props;
  const lines: JSX.Element[] = [];
  for (let i = 0; i < book.slots.length; i += BOOK_COLS) {
    lines.push(
      <div key={i} className="oui-box mc-inv-line">
        {book.slots.slice(i, i + BOOK_COLS).map((s) => (
          <InvSlotView key={s.id} slot={s} />
        ))}
      </div>,
    );
  }
  return (
    <div className="oui-box mc-inv-book">
      <div className="oui-box mc-inv-book-tabs">
        {book.tabs.filter((t) => !t.end).map((t, i) => (
          <BookTab key={'l' + i} tab={t} />
        ))}
        <div className="oui-box mc-inv-book-fill" />
        {book.tabs.filter((t) => t.end).map((t, i) => (
          <BookTab key={'r' + i} tab={t} />
        ))}
      </div>
      <div className="oui-box mc-inv-book-head">
        {book.field ? (
          <div className="oui-box mc-search mc-inv-book-search">
            <Icon icon="tex:textures/ui/magnifying_glass" />
            <Field id={book.field} value={book.query} placeholder={book.placeholder} delay={250} />
          </div>
        ) : (
          <Rich text={book.title} className="oui-text mc-inv-book-label" />
        )}
        {book.filter ? <Filter button={book.filter} /> : null}
      </div>
      <Scroll className="mc-inv-grow">
        {lines.length > 0 ? lines : <Rich text={book.note} className="oui-text mc-body-text mc-inv-note" />}
      </Scroll>
    </div>
  );
}
