#!/usr/bin/env node
// Writes the second dimension's weapon effects (content/skygear/dimension/fx.go)
// from pixelfx (https://github.com/patch-gg/pixelfx, MIT): each effect is
// rendered, cut to the part that is kept, shrunk by k so one pixel is about
// a sixteenth of a block in game, the size of a vanilla texel, and laid out
// as a vertical strip of frames, the layout a particle's flipbook steps
// through. Shrinking keeps the strongest pixel of each k by k cell and makes
// every pixel either clear or opaque; pixels fainter than cut are dropped.
//
//	node tools/pixelfx/export.mjs <pixelfx checkout> [assets/resourcepack/skygear]

import { mkdirSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

const effects = [
  { name: "fx_flame", id: "fire/torch", from: 0, to: 2, fps: 15, crop: [0, 0, 72, 78], k: 2, cut: 0.7 },
  { name: "fx_disc", id: "projectiles/energy-disc", from: 0, to: 0.5, fps: 30, crop: [75, 5, 38, 42], k: 2 },
  { name: "fx_lightning", id: "lightning/lightning-strike", from: 0.8, to: 1.9, fps: 12, crop: [0, 0, 240, 250], k: 2, cut: 0.6 },
  { name: "fx_hit", id: "impacts/hit-spark", from: 0, to: 0.34, fps: 30, opts: { aim: -1.2 }, crop: [0, 0, 136, 62], k: 3 },
  { name: "fx_poof", id: "spawn/poof", from: 0, to: 1.2, fps: 20, k: 3 },
  { name: "fx_slash", id: "melee/sword-slash", from: 0.06, to: 0.4, fps: 30, crop: [0, 0, 136, 86], k: 2 },
  { name: "fx_tracer", id: "projectiles/bullet-tracer", from: 0, to: 0.6, fps: 30, k: 2 },
  { name: "fx_blast", id: "explosions/small-explosion", from: 0.1, to: 2.4, fps: 10, k: 3 },
  { name: "fx_blink", id: "arcane/blink-out", from: 0.7, to: 1.45, fps: 20, k: 4, cut: 0.5 },
  { name: "fx_mend", id: "holy/heal-aoe", from: 0.4, to: 2.4, fps: 10, k: 6 },
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

function shrink(px, w, h, n, k, cut) {
  const ow = Math.ceil(w / k), oh = Math.ceil(h / k);
  const out = new Uint8ClampedArray(ow * oh * n * 4);
  for (let i = 0; i < n; i++)
    for (let y = 0; y < oh; y++)
      for (let x = 0; x < ow; x++) {
        let best = -1, at = -1;
        for (let sy = y * k; sy < Math.min(y * k + k, h); sy++)
          for (let sx = x * k; sx < Math.min(x * k + k, w); sx++) {
            const s = ((i * h + sy) * w + sx) * 4;
            const score = (px[s + 3] / 255) * (0.3 + (px[s] + px[s + 1] + px[s + 2]) / 765);
            if (score > best) (best = score, at = s);
          }
        if (px[at + 3] / 255 >= cut) out.set([px[at], px[at + 1], px[at + 2], 255], ((i * oh + y) * ow + x) * 4);
      }
  return { w: ow, h: oh, px: out };
}

for (const e of effects) {
  const fx = await fxlib.loadEffect(e.id, join(resolve(lib), "src", "fx"));
  const r = fxlib.renderFrames(fx, { fps: e.fps, options: e.opts });
  const frames = r.frames.filter((f) => f.t >= e.from - 1e-6 && f.t < e.to - 1e-6).map((f) => f.clear);
  const b = bounds(frames, r.w, e.crop ?? [0, 0, r.w, r.h]);
  const [ox, oy] = fx.meta.origin;
  const s = shrink(strip(frames, r.w, b), b.w, b.h, frames.length, e.k, e.cut ?? 0.3);
  writeFileSync(join(outDir, `${e.name}.png`), encodePNG(s.w, s.h * frames.length, s.px));
  console.log(JSON.stringify({ name: e.name, frame: [s.w, s.h], frames: frames.length, fps: e.fps, origin: [(ox - b.x) / e.k, (oy - b.y) / e.k] }));
}
