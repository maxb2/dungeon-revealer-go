# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```sh
make build          # Generate templ + build binary (CGO_ENABLED=0)
make dev            # Run locally with --dm-password=admin
make build-all      # Cross-compile to dist/ (linux/amd64, linux/arm64, linux/armv7, darwin/arm64, windows/amd64)
make clean          # Remove binaries and dist/

CGO_ENABLED=0 go test ./...   # Run all tests
CGO_ENABLED=0 go test ./internal/dice/  # Run a single package's tests
~/go/bin/templ generate        # Regenerate templ files (templ is not on PATH)
```

`CGO_ENABLED=0` is required for all go build/test commands (pure-Go SQLite via zombiezen).

## Architecture

Single-binary Go web app: Go stdlib HTTP server + templ templates + HTMX + SSE for real-time.

### Data Flow

`main.go` wires everything: config → stores → handlers → mux. Auth middleware wraps the entire mux, then per-route `RequireRole` middleware enforces admin vs player access.

### Storage: Two Models

- **Filesystem (maps)**: Each map is a directory under `data/maps/{id}/` containing `map.json` (metadata + tokens), `image.{ext}`, `fog-progress.png`, and `fog-live.png`. The `MapStore` is in-memory with filesystem persistence.
- **SQLite (notes, media)**: `zombiezen.com/go/sqlite` with a connection pool. Migrations are an ordered slice of SQL strings in `internal/store/db.go` — append new migrations to the `migrations` slice.

### Real-time Updates

`internal/realtime/broker.go` implements SSE pub/sub. Handlers publish `Event{Name, Data, DMOnly}` and the broker fans out to connected clients. The client connects to `GET /events?role=dm|player`. HTMX listens for SSE events to swap HTML fragments.

### Auth Model

Three roles: `admin` > `player` > `unauthenticated`. Stored in a gorilla/sessions cookie. If no DM password is set, all users get admin access. `RequireRole(minRole)` is middleware that wraps individual routes.

### Templates

`.templ` files in `templates/` generate `_templ.go` files. Always run `templ generate` before building. The generated files are checked into git.

### Static Assets

`static/` is embedded via `//go:embed static` in `main.go`. Key files: `canvas.js` (fog drawing, token rendering, pan/zoom), `htmx.min.js`, `style.css`.

## Configuration

All via flags or env vars: `--port`/`DR_PORT`, `--data-dir`/`DR_DATA_DIR`, `--dm-password`/`DR_DM_PASSWORD`, `--player-password`/`DR_PLAYER_PASSWORD`, `--session-secret`/`DR_SESSION_SECRET`.
