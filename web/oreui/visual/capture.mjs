// Draws Go-dumped fixtures with the React page and compares them with the baseline.
// node visual/capture.mjs --renderer app --fixtures DIR --out DIR [--compare DIR] [--max PCT]
import { createRequire } from "node:module";
import fs from "node:fs";
import http from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { PNG } from "pngjs";
import pixelmatch from "pixelmatch";

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_PATH || "playwright");

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "../../..");

const args = {};
for (let i = 2; i < process.argv.length; i += 2) args[process.argv[i].replace(/^--/, "")] = process.argv[i + 1];
const renderer = args.renderer;
if (renderer !== "app" || !args.fixtures || !args.out) {
  console.error("usage: capture.mjs --renderer app --fixtures DIR --out DIR [--compare DIR] [--max PCT]");
  process.exit(2);
}
const max = args.max === undefined ? 0.5 : Number(args.max);
fs.mkdirSync(args.out, { recursive: true });

const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".png": "image/png" };
let fixture = "{}";
const server = http.createServer((req, res) => {
  const url = new URL(req.url, "http://x");
  if (url.pathname === "/web/oreui/visual/fixture.js") {
    res.writeHead(200, { "content-type": "text/javascript" });
    return res.end("window.__FIXTURE__=" + fixture + ";");
  }
  const file = path.join(repo, path.normalize(decodeURIComponent(url.pathname)));
  if (!file.startsWith(repo) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
    res.writeHead(404);
    return res.end();
  }
  res.writeHead(200, { "content-type": types[path.extname(file)] || "application/octet-stream" });
  res.end(fs.readFileSync(file));
});
await new Promise((r) => server.listen(0, "127.0.0.1", r));
const port = server.address().port;

const browser = await chromium.launch();
const names = fs.readdirSync(args.fixtures).filter((f) => f.endsWith(".json")).map((f) => f.slice(0, -5)).sort();
let failed = false;
for (const name of names) {
  fixture = fs.readFileSync(path.join(args.fixtures, name + ".json"), "utf8");
  const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
  await page.goto(`http://127.0.0.1:${port}/web/oreui/visual/harness.html`);
  await page.addStyleTag({ content: "*{animation:none!important;transition:none!important}" });
  await page.waitForTimeout(600);
  const png = path.join(args.out, name + ".png");
  await page.screenshot({ path: png });
  const html = await page.evaluate(() => (document.getElementById("oui-ddui") || { innerHTML: "" }).innerHTML);
  fs.writeFileSync(path.join(args.out, name + ".html"), html);
  await page.close();
  if (args.compare) {
    const a = PNG.sync.read(fs.readFileSync(png));
    const b = PNG.sync.read(fs.readFileSync(path.join(args.compare, name + ".png")));
    const pct = a.width !== b.width || a.height !== b.height ? 100 : (pixelmatch(a.data, b.data, null, a.width, a.height, { threshold: 0.1 }) * 100) / (a.width * a.height);
    console.log(`${name} ${pct.toFixed(2)}%`);
    if (pct > max) failed = true;
  } else {
    console.log(name);
  }
}
await browser.close();
server.close();
process.exit(failed ? 1 : 0);
