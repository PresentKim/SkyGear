import { expect, it } from 'vitest';
import type { Root } from '../src/protocol';

it('loads the protocol', () => {
  const r: Root = { screen: 'menu', data: {} };
  expect(r.screen).toBe('menu');
});
