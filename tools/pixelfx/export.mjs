#!/usr/bin/env node
// Writes the second dimension's weapon effects (content/skygear/dimension/fx.go)
// from pixelfx (https://github.com/patch-gg/pixelfx, MIT): each effect is
// rendered, cut to the part that is kept and laid out as a vertical strip of
// frames, the layout a particle's flipbook steps through.
//
//	node tools/pixelfx/export.mjs <pixelfx checkout> [assets/resourcepack/skygear]

import { mkdirSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

const effects = [
  { name: "fx_flame", id: "fire/torch", from: 0, to: 2, fps: 15, crop: [0, 0, 72, 78] },
  { name: "fx_disc", id: "projectiles/energy-disc", from: 0, to: 0.5, fps: 30, crop: [75, 5, 38, 42] },
  { name: "fx_lightning", id: "lightning/lightning-strike", from: 0.8, to: 1.9, fps: 12, crop: [0, 0, 240, 250] },
  { name: "fx_hit", id: "impacts/hit-spark", from: 0, to: 0.34, fps: 30, opts: { aim: -1.2 }, crop: [0, 0, 136, 62] },
  { name: "fx_poof", id: "spawn/poof", from: 0, to: 1.2, fps: 20 },
];

const [lib, pack = "assets/resourcepack/skygear"] = process.argv.slice(2);
if (!lib) {
  console.error("usage: node tools/pixelfx/export.mjs <pixelfx checkout> [resource pack]");
  process.exit(1);
}
const fxlib = await import(pathToFileURL(join(resolve(lib), "tools/lib/fxlib.mjs")).href);
const { encodePNG } = await import(pathToFileURL(join(resolve(lib), "tools/lib/png.mjs")).href);
const outDir = join(pack, "textures", "particle");
mkdirSync(outDir, { recursive: true });

function bounds(frames, w, [cx, cy, cw, ch]) {
  let x0 = Infinity, y0 = Infinity, x1 = -1, y1 = -1;
  for (const f of frames)
    for (let y = cy; y < cy + ch; y++)
      for (let x = cx; x < cx + cw; x++)
        if (f[(y * w + x) * 4 + 3] > 8) {
          x0 = Math.min(x0, x), y0 = Math.min(y0, y), x1 = Math.max(x1, x), y1 = Math.max(y1, y);
        }
  return { x: x0, y: y0, w: x1 - x0 + 1, h: y1 - y0 + 1 };
}

function strip(frames, w, b) {
  const out = new Uint8ClampedArray(b.w * b.h * frames.length * 4);
  frames.forEach((f, i) => {
    for (let y = 0; y < b.h; y++) {
      const src = ((b.y + y) * w + b.x) * 4;
      out.set(f.subarray(src, src + b.w * 4), ((i * b.h + y) * b.w) * 4);
    }
  });
  return out;
}

for (const e of effects) {
  const fx = await fxlib.loadEffect(e.id, join(resolve(lib), "src", "fx"));
  const r = fxlib.renderFrames(fx, { fps: e.fps, options: e.opts });
  const frames = r.frames.filter((f) => f.t >= e.from - 1e-6 && f.t < e.to - 1e-6).map((f) => f.clear);
  const b = bounds(frames, r.w, e.crop ?? [0, 0, r.w, r.h]);
  const [ox, oy] = fx.meta.origin;
  writeFileSync(join(outDir, `${e.name}.png`), encodePNG(b.w, b.h * frames.length, strip(frames, r.w, b)));
  console.log(JSON.stringify({ name: e.name, frame: [b.w, b.h], frames: frames.length, fps: e.fps, origin: [ox - b.x, oy - b.y] }));
}
