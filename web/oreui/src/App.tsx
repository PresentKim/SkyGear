import { NO_VALUE, useFacetUnwrap } from '@react-facet/core';
import type { Root } from './protocol';
import { docFacet } from './runtime/doc';
import { useKeyboard } from './interact/keys';
import { TooltipHost } from './interact/Tooltip';
import { LobbyContext } from './components/Window';
import { screens } from './screens/registry';
import { Status } from './Status';

function Screen(props: { root: Root }): JSX.Element | null {
  const { root } = props;
  useKeyboard(root.keys);
  const Component = screens[root.screen];
  if (!Component) return null;
  return (
    <LobbyContext.Provider value={root.lobby ?? null}>
      <Component data={root.data} />
    </LobbyContext.Provider>
  );
}

/** App draws the form the server shows; a new form id remounts the screen, the same form keeps its state. */
export function App(): JSX.Element {
  const unwrapped = useFacetUnwrap(docFacet);
  const doc = unwrapped === NO_VALUE ? null : unwrapped;
  const root = doc && doc.active ? doc.root : null;
  return (
    <>
      <TooltipHost />
      {root ? <Screen key={doc!.form} root={root} /> : null}
      {doc ? <Status doc={doc} on={!!(root && root.debug)} /> : null}
    </>
  );
}
