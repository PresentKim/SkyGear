import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import * as esbuild from 'esbuild';

const web = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const out = resolve(web, '../../assets/resourcepack/oreui/hbui');

function walk(dir) {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    return statSync(p).isDirectory() ? walk(p) : [p];
  });
}

function sourceHash() {
  const files = [...walk(join(web, 'src')), join(web, 'package-lock.json')]
    .map((p) => relative(web, p).split(sep).join('/'))
    .sort();
  const h = createHash('sha256');
  for (const rel of files) {
    h.update(`${rel}\n`);
    h.update(readFileSync(join(web, rel)));
    h.update('\n');
  }
  return h.digest('hex');
}

const oreUi = {
  name: 'ore-ui-react',
  setup(build) {
    // A single component of the oreui package (its own file, so only its CSS comes along).
    build.onResolve({ filter: /^@ore-ui-react\/oreui\/./ }, (args) => ({
      path: resolve(web, 'node_modules/ore-ui-react/packages/oreui/src', args.path.slice('@ore-ui-react/oreui/'.length) + '.tsx'),
    }));
    build.onResolve({ filter: /^@ore-ui-react\/[^/]+$/ }, (args) => ({
      path: resolve(web, 'node_modules/ore-ui-react/packages', args.path.slice('@ore-ui-react/'.length), 'src/index.ts'),
    }));
  },
};

await esbuild.build({
  entryPoints: [join(web, 'src/main.tsx')],
  outfile: join(out, 'oui-app.js'),
  bundle: true,
  format: 'iife',
  target: 'es2017',
  minify: true,
  jsx: 'automatic',
  define: { 'process.env.NODE_ENV': '"production"' },
  banner: { js: `/* oui-app src ${sourceHash()} */` },
  plugins: [oreUi],
  logLevel: 'warning',
});

for (const file of ['oui-app.js', 'oui-app.css']) {
  console.log(`${file} ${statSync(join(out, file)).size} bytes`);
}
