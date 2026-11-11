# Contributing

Thanks for your interest. These are the rules the project follows.

## Branches and commits

- Work on the `dev` branch and open pull requests against `dev`. `main` is the
  branch the live server runs.
- AI agents (Claude Code and the like) commit and push straight to `dev`
  unless the request names another branch.
- Before committing, these must be clean:

  ```sh
  go vet ./...
  go test ./...
  gofmt -l .
  ```

- Write commit messages in English.
- One commit per change: when a piece of work covers several requests or
  fixes, commit each one separately.

## Versions and releases

- The server is versioned `year.month.build` (`26.10.1`): the two-digit year,
  the month, and a build number that starts again at 1 each month.
- `main` moves only by merging `dev` into it. Each merge is a release: raise
  `Version` in `cmd/skygear/version.go`, write what changed since the last
  release in `change_logs/<version>.md` (first line `# <version> (<date>)`,
  in English), and tag the merge `v<version>`. A test checks that the newest
  change log is `Version`.
- Pushing a `v<version>` tag runs `.github/workflows/release.yml`: it checks
  that the tag matches `Version`, builds the server for Windows and Linux
  (amd64), and creates a GitHub release with the archives and the body of
  `change_logs/<version>.md` as its notes.

## Design and documentation

- `docs/design/` is the source of truth for game design. Read the relevant
  document before implementing a feature, and update it when the design
  changes.
- Documents under `docs/` are written in Korean and describe the current design
  and future direction only: no change history, and no "this is not
  implemented" statements. Rewrite a document to the current state instead.
- Postponed work and checks to do in a real client go in `docs/TODO.md`.
  Undecided questions go in `docs/design/open-questions.md`; balancing numbers
  are drafts, and the balancing table there is updated when they change.

## Code structure

- Game-independent code lives in `common/` (same Go module). Platform and game
  rules (lobby, saves, account storage, Sky Gear content) do not go there.
- Player moves and state saving always go through `instance.Manager`
  (`Enter`/`ToLobby`). The instance package reaches Sky Gear only
  through the `instance.Game`/`Session` interfaces (`content/skygear`
  imports `platform/instance`, not the other way round).
- Sky Gear session state belongs to its instance world: read and write it
  only inside that world's transactions (handlers, form submits, block ticks,
  `w.Do`).
- Database schema changes add a new entry to `migrations` in
  `platform/db/db.go`; existing entries are never edited.
- Saves record the release their data is in (`save_state` key
  `skygear.version`, `data_versions` for account data). A change that
  existing data has to follow (a renamed block, item, counter or key) adds a
  patch at the end of `Patches` in `content/skygear/migrate/patches.go` with
  the release it ships in (`Rename` for renamed strings, `Plain` for retired
  items that become a plain stack of another item, `Blocks` for blocks that
  change into another block with its states); at startup the server applies the patches newer
  than each save's record after a backup. Released patches are never edited.

## Text and translations

- Text shown to players goes in a catalog and is read with `i18n.T`:
  `content/skygear/locales/` for the game, `platform/locales/` for the
  platform, `common/i18n/locales/` for `common/` packages. Do not put Korean string literals in `common/` code (a test checks
  this).
- Block and item names are client translation keys in the resource pack, in
  both `texts/ko_KR.lang` and `texts/en_US.lang`.
- New research entries need a name, short description and description under
  `research.<id>.*` in `content/skygear/locales/ko.json`.
- Keep player-facing text short. Do not announce unfinished features
  ("coming soon"), and do not send messages for results the player can already
  see.

## Resource packs

- Everything visible lives in the resource pack
  (`assets/resourcepack/skygear/`): textures, atlases, block models,
  names, animations, JSON UI. Go code only names the model identifier and the
  material key; Dragonfly's automatic pack generation is turned off.
- Do not edit the version in a pack's `manifest.json`;
  `common/resourcepack.LoadFS` derives it from the content hash.
- Do not change the UUIDs of the packs under `assets/resourcepack/` (clients have them
  cached).
- New craftable items and blocks must also be registered in
  `content/skygear/catalog/creative.go` (the client recipe book only lists
  results that are in the creative inventory; a test checks this).

## Ore UI page

- The DDUI screens are drawn by the React page in `web/oreui` (see
  `docs/tech/oreui.md`). After changing anything under `web/oreui/src` or its
  `package-lock.json`, run `npm ci --ignore-scripts` once and `npm run build`
  in `web/oreui`, and commit the rebuilt `oui-app.js` and `oui-app.css` in
  `assets/resourcepack/oreui/hbui/`. `TestOreUIBundleFresh` fails when the
  bundle does not match the sources.
- Always install with `--ignore-scripts`. Run the page's tests with
  `npx vitest run` and `npm run typecheck`.

## Patched Dragonfly

The server uses the patched copy in `third_party/dragonfly`. Keep the patch
small and extend Dragonfly from the outside first, in this order:

- `common/clientdef`: components a client gets in the item and block entries
  (digger, fuel, tool tags, collision boxes, bone visibility, render methods).
  A custom item or block implements the `clientdef` interface; the game installs
  it once in `core.EditEntries` (`content/skygear/core`) through
  `sghook.EditCustomItems` and `sghook.EditCustomBlocks`. Client components do
  not belong in a patch.
- `common/sghook`: packet hooks, `go:linkname` to session internals and
  reflection. Only this package reaches Dragonfly internals.
- A patch, only when Dragonfly's behaviour itself has to change (storage,
  ticking, chunk handling, hooks it lacks). Update
  `third_party/dragonfly.patch` and the row in `third_party/README.md`, and
  follow its steps to regenerate and check the patch.

See `docs/tech/dragonfly.md`.

`content/skygear/wire` pins what a joining player is sent (block entries,
item registry, crafting data, creative content) as golden files, hashed per
block, item, recipe and creative entry. A mismatch names the entries that
changed. A change that moves client data or edits the patch must leave the
goldens as they are. Re-record only in a commit meant to change what clients
get, with `SKYGEAR_GOLDEN=update go test ./content/skygear/wire/`, and give the
reason in the commit message. To see what changed inside an entry, run once per
tree with `SKYGEAR_GOLDEN_DUMP=<dir>` and `diff -r` the two folders.

## Licence

By contributing you agree that your contribution is licensed under the MIT
License of this project.
