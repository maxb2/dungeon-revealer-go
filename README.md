# Dungeon Revealer

A web-based map sharing tool for tabletop RPGs. The DM uploads maps, draws fog of war, and reveals areas to players in real time. Built as a single Go binary with no external dependencies.

## Features

- **Fog of War** — DM draws to hide/reveal areas of the map, pushes updates live to players
- **Tokens** — Place and move tokens on the map, visible to all players in real time (including custom image tokens)
- **Chat & Dice** — Built-in chat with dice rolling (`/roll 2d6+3`)
- **Notes** — Markdown notes with full-text search, shareable between DM and players
- **Media Library** — Upload images for use in notes and handouts
- **Real-time sync** — All updates pushed via Server-Sent Events
- **Single binary** — No runtime dependencies, no database server, no Node.js
- **Raspberry Pi ready** — Cross-compiles to ARM with no CGo

## Quick Start

Download a binary from the [releases page](../../releases), or build from source:

```sh
make build
./dungeon-revealer --dm-password=secretDM
```

Then open `http://localhost:3000` in your browser. Players connect to the same address.

### Docker

A Docker image is published to the GitHub Container Registry on every release:

```sh
docker run -d \
  -p 3000:3000 \
  -v dungeon-data:/data \
  -e DR_DM_PASSWORD=secretDM \
  ghcr.io/maxb2/dungeon-revealer-go:latest
```

See all available tags on the [packages page](https://github.com/maxb2/dungeon-revealer-go/pkgs/container/dungeon-revealer-go).

## Configuration

All options can be set via flags or environment variables:

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--port` | `DR_PORT` | `3000` | HTTP port |
| `--data-dir` | `DR_DATA_DIR` | `./data` | Directory for maps, media, and database |
| `--dm-password` | `DR_DM_PASSWORD` | *(none)* | Password for DM access (if unset, all users are admins) |
| `--player-password` | `DR_PLAYER_PASSWORD` | *(none)* | Password for player access (if unset, players can join freely) |
| `--session-secret` | `DR_SESSION_SECRET` | *(random)* | Secret for session cookies |

## Building from Source

Requires Go 1.25+. If you don't have Go installed, follow the official instructions at [go.dev/doc/install](https://go.dev/doc/install).

```sh
# Build for current platform
make build

# Build for all platforms
make build-all

# Run in dev mode (dm-password=admin)
make dev
```

Cross-compiled binaries are output to `dist/`:
- `linux/amd64`, `linux/arm64`, `linux/armv7`
- `darwin/arm64`
- `windows/amd64`

## Releases

Every push to `main` automatically creates a new version tag and GitHub release. The version bump is determined by [Conventional Commits](https://www.conventionalcommits.org/):

- **Patch** — `fix:`, `docs:`, `chore:`, etc.: `v1.0.0` → `v1.0.1`
- **Minor** — `feat:`: `v1.0.1` → `v1.1.0`
- **Major** — `feat!:` or `BREAKING CHANGE` in the commit body: `v1.1.0` → `v2.0.0`

Each release builds cross-platform binaries and publishes a Docker image to GHCR.

## How It Works

1. The DM logs in with the DM password and uploads a map image
2. The DM uses brush tools to paint fog of war over the map, then pushes the revealed state to players
3. Players see the map with fog applied in real time
4. Tokens can be placed on the map and moved by the DM or players
5. Chat, dice rolls, and notes are available in the sidebar

## Tech Stack

- **Go** standard library HTTP server and router
- **[templ](https://templ.guide)** for type-safe HTML templates
- **[HTMX](https://htmx.org)** for dynamic page updates
- **[zombiezen/sqlite](https://github.com/nicholasgasior/zombiezen-sqlite)** for pure-Go SQLite (notes + media metadata)
- **Canvas API** for fog-of-war drawing and token rendering
- **Server-Sent Events** for real-time updates

## License

MIT
