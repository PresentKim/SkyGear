import type { InvBook } from '../protocol';
import { usePressable } from '../interact/press';
import { Rich } from '../text/Rich';
import { Icon } from '../components/Icon';
import { Scroll } from '../components/Scroll';
import { InvSlotView } from './InvSlot';

function Filter(props: { button: NonNullable<InvBook['filter']> }): JSX.Element {
  const b = props.button;
  const press = usePressable({ id: b.id, tip: b.tip, quick: true });
  return (
    <div className="oui-box mc-icon-button mc-inv-button" {...press}>
      <Icon icon={b.icon} />
    </div>
  );
}

/** RecipeBook is the recipe panel beside the window's left edge: its title and filter, and a scrolling grid of recipe slots. */
export function RecipeBook(props: { book: InvBook }): JSX.Element {
  const { book } = props;
  const cols = Math.max(1, book.cols);
  const lines: JSX.Element[] = [];
  for (let i = 0; i < book.slots.length; i += cols) {
    lines.push(
      <div key={i} className="oui-box mc-inv-line">
        {book.slots.slice(i, i + cols).map((s) => (
          <InvSlotView key={s.id} slot={s} />
        ))}
      </div>,
    );
  }
  return (
    <div className="oui-box mc-inv-book">
      <div className="oui-box mc-inv-book-head">
        <Rich text={book.title} className="oui-text mc-inv-heading" />
        {book.filter ? <Filter button={book.filter} /> : null}
      </div>
      <Scroll className="mc-inv-grow">
        {lines.length > 0 ? lines : <Rich text={book.note} className="oui-text mc-body-text mc-inv-note" />}
      </Scroll>
    </div>
  );
}
