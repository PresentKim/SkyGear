import { useEffect, useRef } from 'react';
import type { Id, Text } from '../protocol';
import { registerInput, send } from '../runtime/engine';
import { localize } from '../text/localize';

/** Field is a text input that sends its text a moment after the typing stops, or at Enter; with `enter` Enter presses that button instead. */
export function Field(props: { id: Id; value: Text; placeholder?: Text; enter?: Id; delay?: number }): JSX.Element {
  const ref = useRef<HTMLInputElement>(null);
  const latest = useRef(props);
  latest.current = props;

  // The typed text stays through a re-render with the same id; a new id starts from the server's value.
  useEffect(() => {
    const input = ref.current;
    if (!input) return;
    input.value = latest.current.value || '';
    const unregister = registerInput(props.id, () => input.value);
    let timer: ReturnType<typeof setTimeout> | null = null;
    const flush = () => {
      if (timer) clearTimeout(timer);
      timer = null;
      send(props.id, 'change', input.value);
    };
    const onInput = () => {
      if (timer) clearTimeout(timer);
      timer = setTimeout(flush, latest.current.delay || 400);
    };
    const onKey = (ev: KeyboardEvent) => {
      if (ev.keyCode !== 13 && ev.key !== 'Enter') return;
      const enter = latest.current.enter;
      if (enter) {
        if (timer) clearTimeout(timer);
        timer = null;
        send(enter, 'click', null);
      } else flush();
    };
    input.addEventListener('input', onInput);
    input.addEventListener('keydown', onKey);
    return () => {
      input.removeEventListener('input', onInput);
      input.removeEventListener('keydown', onKey);
      if (timer) clearTimeout(timer);
      unregister();
    };
  }, [props.id]);

  return <input className="oui-input mc-input" type="text" placeholder={props.placeholder ? localize(props.placeholder) : undefined} ref={ref} />;
}
