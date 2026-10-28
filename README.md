# Sky Gear

A multi-game island server for **Minecraft: Bedrock Edition**, built on
[Dragonfly](https://github.com/df-mc/dragonfly) (Go).

Every player gets their own save slots; each save is a separate world (an
*instance*) that runs while its host is online and can be shared with friends.
The first game is **Sky Gear** (스카이기어), a SkyFactory / ATM10 Sky
style factory-building game: start bare-handed on a 3×3×3 sky island, craft
resources by hand, unlock recipes through research, and work your way up to
powered machines, conduits, alloys, monster waves and magic.

The game's text and design documents are in Korean; the server also ships an
English translation that follows each player's client language. A Korean
feature list is in [docs/features.md](docs/features.md).

## Requirements

- Go 1.26 or newer (see `go.mod`).
- A Minecraft: Bedrock Edition client (Windows, mobile or console) that matches
  the protocol of the bundled Dragonfly version.
- No add-ons to install: the server builds its resource packs into the binary
  and sends them to the client on join (the client must accept them).

## Build and run

```sh
go run .
# or
go build -o skygear . && ./skygear
```

On the first run the server writes its configuration files in the working
directory and stores its data under `data/`:

| File | Contents |
|---|---|
| `config.toml` | Dragonfly server settings: network address (default `:19132`), server name, Xbox Live authentication, … |
| `platform.toml` | Platform settings: data folder, instance day length, load radius, how many instances may run at once, autosave and world save intervals, backups, resource pack options (`Minify`, `Encrypt`, `SlotDeco`), operators (`Admin.Operators`) |
| `data/platform.db` | SQLite database: accounts, friends, saves, per-save player state, game state |
| `data/saves/<save id>/world` | One world per save |
| `data/backups/` | Automatic database and world backups ([docs/tech/backup.md](docs/tech/backup.md)) |
| `data/resourcepack.secret` | Key material for resource pack encryption; keep it, or clients re-download the packs |

Stop the server with Ctrl+C; running instances are saved before it exits.

## Connecting

Add the server in the Bedrock client (Play → Servers → Add Server) with the
host's address and port `19132`, then join. On first join you get a short
server guide and land in the lobby, where you can create a save and enter your
island.

## Becoming an operator

Operators can use the admin and debug commands (`/sgadmin`, `/sgdebug`,
`/copysave`). Add your Xbox **XUID** to `platform.toml` and restart:

```toml
[Admin]
Operators = ["2535412345678901"]
```

Names also work, but Xbox gamertags can change hands, so XUIDs are
recommended; when someone becomes operator by name only, the server logs a
warning with their XUID. With authentication disabled (test servers) XUIDs are
empty and only names match. See [docs/tech/admin.md](docs/tech/admin.md) for
the commands.

## Repository layout

```
main.go                         Server bootstrap; saves instances on shutdown
assets.go                       Embeds the resource packs into the binary
internal/db/                    SQLite access and schema migrations
internal/instance/              Instance manager (activate/close, lobby ↔ instance moves, clock, base handlers),
                                Game/Session interfaces between the platform and a game
internal/platform/              Lobby, server guide, save selection, friends, invites, members, titles, commands
internal/account/               Achievements, titles, name colours, per-account settings
internal/locales/               This server's text catalogs (ko/en, vanilla names), merged into common/i18n
internal/games/skygear/   Sky Gear: session-bound code (game, state, handlers, machine/drawer/barrel/
                                conduit blocks, research, advancement screens, codex, island expansion, waves, …)
  sgitem/                       Session-free items and simple blocks (pebbles, dust, ores, metals, meshes, hammers, …)
  naming/                       Item and block name keys and names (catalog, pack texts, vanillakeys.json)
  textfmt/                      Number and duration formatting, pack font glyphs
  blockmodel/                   Custom block materials and cube properties
  formicon/                     Form button icons
  wintext/                      Texts of machine window items, localised per viewer
  sievetable/                   Sieve drop tables and magic sieve themes
  machinerecipe/                Alloy smelter, essence extractor and auto crafter recipes
  tinker/                       Tinkers'-style tool materials, levels, experience and modifiers
  hopper/                       Wooden and reverse hoppers
  advancement/                  Advancement tree, chapters and layout
  achievement/                  Account achievement catalog and IDs
internal/itemio/                Item input/output interface used by hoppers and conduits
internal/energy, itemnet, fluidnet/  Energy, item and fluid networks
common/                         Game-independent packages and packs (forms, DDUI screens, entity chests, WAILA,
                                HUD, i18n, resource pack loader, backups, …; see common/README.md)
assets/resourcepack/skygear/  The game's resource pack: textures, models, names, entities, JSON UI
tools/                          Generators for textures, models and data files used by the packs
spike/scenarios/, spike/cmd/platformbot/  Connection scenarios and the scenario bot
third_party/dragonfly/          Dragonfly (1.26.60) with this project's patches (third_party/README.md)
docs/                           Design (docs/design) and technical notes (docs/tech), in Korean
```

## Testing

```sh
go vet ./...
go test ./...
gofmt -l .
```

`spike/scenarios/run.sh` checks real connection flows end to end: it builds the
server, runs it with authentication off, drives a bot through a scenario and
shuts everything down, failing if the server log has an `ERROR` or a panic.

```sh
spike/scenarios/run.sh spike/scenarios/advanced.sh
```

Each `spike/scenarios/*.sh` file is one scenario.

## Documentation

- [docs/design](docs/design/README.md): game design, the source of truth for features
- [docs/tech](docs/tech): technical notes (resource packs, JSON UI, Ore UI, chests, WAILA, conduits, performance, …)
- [docs/TODO.md](docs/TODO.md): open work and checks to do in a real client

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Licence

MIT, see [LICENSE](LICENSE). Third-party code and assets (the patched
Dragonfly, resource pack parts and mod textures) keep their own licences; see
[NOTICE.md](NOTICE.md) and [docs/tech/asset-credits.md](docs/tech/asset-credits.md).

Minecraft is a trademark of Mojang Synergies AB / Microsoft. This project is
not affiliated with Mojang or Microsoft.
