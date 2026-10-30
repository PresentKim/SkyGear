import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const here = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  resolve: {
    alias: [
      // A single component of the oreui package (its own file, so only its CSS comes along).
      { find: /^@ore-ui-react\/oreui\/(.+)$/, replacement: `${here}/node_modules/ore-ui-react/packages/oreui/src/$1` },
      { find: /^@ore-ui-react\/(.+)$/, replacement: `${here}/node_modules/ore-ui-react/packages/$1/src/index.ts` },
    ],
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}', 'test/**/*.test.ts'],
  },
});
