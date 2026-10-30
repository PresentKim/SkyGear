# Parity harness

Draws the screens the Go side sends with the React page in a real browser and compares the pictures with `baseline/`.

```
cd web/oreui
OREUI_FIXTURES=$PWD/visual/out/fx go test ../../common/ui -run TestDumpFixtures
node visual/capture.mjs --renderer app --fixtures visual/out/fx --out visual/out --compare visual/baseline [--max PCT]
```

- `TestDumpFixtures` (common/ui/fixtures_test.go) writes `<name>.json` = `{"root": <the Root sent>, ...}` per screen.
- `capture.mjs` serves the repo, opens `harness.html` (the pack's CSS, `oui-app.js`, and `mock-engine.js`, which answers the page's engine queries with the fixture), screenshots 1280x720 and prints `<name> <diff %>` per fixture. It exits 1 when a fixture is over `--max` (default 0.5). Playwright must be resolvable (`NODE_PATH`, or `PLAYWRIGHT_PATH` for its location). `visual/out/` is git-ignored output.
- `baseline/<name>.png` and `.html` (the `#oui-ddui` innerHTML) were drawn by the legacy renderer `oui-ddui.js` at commit 5d91720, before it was deleted. They are not regenerated: when a screen is meant to look different, replace the baseline from the app renderer's output after checking it by eye.
