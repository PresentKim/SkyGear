# Sky Gear

The **Sky Gear** (스카이기어) server for **Minecraft: Bedrock Edition**, built on
[Dragonfly](https://github.com/df-mc/dragonfly) (Go).

Every player gets their own save slots; each save is a separate world (an
*instance*) that runs while its host is online and can be shared with friends.
Sky Gear is a SkyFactory / ATM10 Sky style factory-building game: start bare-handed on a 3×3×3 sky island, craft
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
go run ./cmd/skygear
# or
go build -o skygear ./cmd/skygear && ./skygear
```

On the first run the server writes its configuration files in the working
directory and stores its data under `data/`:

| File | Contents |
|---|---|
| `config.toml` | Dragonfly server settings: network address (default `:19132`), server name, Xbox Live authentication, … |
| `platform.toml` | Platform settings: data folder, instance day length, load radius, how many instances may run at once, autosave and world save intervals, backups, resource pack options (`Minify`, `Encrypt`, `SlotDeco`; an unencrypted pack carries an empty `contents.zip`, so the game still takes the Ore UI page the menus are drawn on), operators (`Admin.Operators`) |
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

The code is in three layers, each importing only the ones above it:

| Layer | Folder | What it holds |
|---|---|---|
| Common | `common/` | Game-independent packages any Dragonfly server can use (menus as DDUI screens, entity chests, WAILA, HUD, i18n, resource pack loader, backups, item I/O, energy/item/fluid networks; see [common/README.md](common/README.md)) |
| Platform | `platform/` | The server around the games: lobby, accounts, saves, friends, instances (see [platform/README.md](platform/README.md)) |
| Content | `content/skygear/` | Sky Gear itself: blocks, items, machines, research, codex, waves, … (see [content/skygear/README.md](content/skygear/README.md)) |

```
cmd/skygear/                    The server command: wires the layers together, loads the packs, saves on shutdown
common/                         Common layer
platform/                       Platform layer: lobby, server guide, save selection, friends, invites, members, commands
  account/                      Achievements, titles, name colours, per-account settings
  db/                           SQLite access and schema migrations
  locales/                      The platform's text catalogs (ko/en)
  instance/                     Instance manager (activate/close, lobby ↔ instance moves, clock, base handlers),
                                Game/Session interfaces the manager calls a game through
content/skygear/                Content layer: Sky Gear (sub-packages in content/skygear/README.md), its text
                                catalogs in content/skygear/locales
assets/                         The resource packs, embedded in the binary (assets.Pack): resourcepack/skygear
                                is the game's pack (textures, models, names, entities, JSON UI), the others
                                (inventoryui, waila, slotdeco, sgui, oreui) are drawn on by common/
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

Each `spike/scenarios/*.sh` file is one scenario. The bot reads JSON UI forms only, and the menus are now
DDUI screens, so scenarios that go through a menu need a bot that drives DDUI screens first
([docs/TODO.md](docs/TODO.md)).

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
