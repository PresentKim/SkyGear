# Contributing

Thanks for your interest. These are the rules the project follows; the full

## Branches and commits

- Work on the `dev` branch and open pull requests against `dev`. `main` is the
  branch the live server runs.
- Before committing, these must be clean:

  ```sh
  go vet ./...
  go test ./...
  gofmt -l .
  ```

- If you change a connection flow, run the matching scenario
  (`spike/scenarios/run.sh spike/scenarios/<name>.sh`), and add a scenario for
  a new feature.
- Write commit messages in English.
- One commit per change: when a piece of work covers several requests or
  fixes, commit each one separately.

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
  rules (lobby, saves, account storage, DragonFactory content) do not go there.
- Player moves and state saving always go through `instance.Manager`
  (`Enter`/`ToLobby`). Games connect to the platform only through the
  `instance.Game`/`Session` interfaces.
- DragonFactory session state belongs to its instance world: read and write it
  only inside that world's transactions (handlers, form submits, block ticks,
  `w.Do`).
- Database schema changes add a new entry to `migrations` in
  `internal/db/db.go`; existing entries are never edited.

## Text and translations

- Text shown to players goes in a catalog and is read with `i18n.T`:
  `internal/locales/` for this server, `common/i18n/locales/` for `common/`
  packages. Do not put Korean string literals in `common/` code (a test checks
  this).
- Block and item names are client translation keys in the resource pack, in
  both `texts/ko_KR.lang` and `texts/en_US.lang`.
- New research entries need a name, short description and description under
  `research.<id>.*` in `internal/locales/ko.json`.
- Keep player-facing text short. Do not announce unfinished features
  ("coming soon"), and do not send messages for results the player can already
  see.

## Resource packs

- Everything visible lives in the resource pack
  (`assets/resourcepack/dragonfactory/`): textures, atlases, block models,
  names, animations, JSON UI. Go code only names the model identifier and the
  material key; Dragonfly's automatic pack generation is turned off.
- Do not edit the version in a pack's `manifest.json`;
  `common/resourcepack.Load` derives it from the content hash.
- Do not change the UUIDs of the packs under `common/packs/` (clients have them
  cached).
- New craftable items and blocks must also be registered in `creative.go`
  (the client recipe book only lists results that are in the creative
  inventory; a test checks this).
- Before JSON UI or modelling work, check the reference repositories listed in

## Patched Dragonfly

The server uses the patched copy in `third_party/dragonfly`. Keep the patch
small: anything that can be done by intercepting packets or reading and
writing session internals belongs in `common/dfhook` (packet hooks,
`go:linkname`, reflection). Patch Dragonfly only when its behaviour itself has
to change, and then update `third_party/dragonfly.patch` and
`third_party/README.md`.

## Licence

By contributing you agree that your contribution is licensed under the MIT
License of this project.
