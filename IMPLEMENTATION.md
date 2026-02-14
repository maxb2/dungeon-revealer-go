# Dungeon Revealer Go — Initial Implementation

Single-binary Go rewrite of Dungeon Revealer. All dependencies are pure Go (no CGo) for trivial cross-compilation to ARM/Raspberry Pi.

## Tech Stack

- **Router:** `net/http` (Go 1.22+ pattern matching)
- **Templates:** `a-h/templ` (type-safe, compiles to Go)
- **Dynamic UI:** HTMX (vendored JS) + SSE for real-time updates
- **Database:** `zombiezen.com/go/sqlite` (pure Go SQLite via `modernc.org/sqlite`)
- **Sessions:** `gorilla/sessions` cookie store
- **Markdown:** `yuin/goldmark`
- **IDs:** `rs/xid`

## What Was Built

### Phase 1 — Skeleton + Auth
- `internal/config/config.go` — CLI flags + env vars (`DR_DM_PASSWORD`, `DR_PLAYER_PASSWORD`, `DR_PORT`, `DR_DATA_DIR`, `DR_SESSION_SECRET`)
- `internal/auth/auth.go` — Cookie sessions, three roles (admin/player/unauthenticated), `RequireRole` middleware
- `internal/store/db.go` — SQLite connection pool, numbered migration runner (3 migrations: notes table, notes FTS5, media table)
- `templates/layout.templ` — Base HTML with HTMX + SSE extension
- `templates/login.templ` — Password form, error display
- `main.go` — Wires config → store → auth → handlers → routes, embeds `static/` via `embed.FS`
- `Makefile` — `generate`, `build`, `dev`, `build-arm64`, `build-all` targets

### Phase 2 — Map Upload + Display
- `internal/store/maps.go` — Filesystem-based map storage. Each map gets a directory under `data/maps/{id}/` containing `map.{ext}` and `map.json` metadata. Supports list, get, create, delete, set active.
- `internal/handler/maps.go` — Upload (multipart), list, activate, delete, serve image, active map view (DM and player variants)
- `templates/maps.templ` — Map list with HTMX upload form, map area views for DM and player
- DM selects active map → player view polls `/maps/active` and displays it

### Phase 3 — Fog-of-War
- `internal/realtime/broker.go` — SSE pub/sub broker. Clients subscribe with role (DM/player). Events carry a name and data string. DM-only events are filtered.
- `internal/handler/fog.go` — Save fog progress PNG (PUT), push progress → live (POST, notifies players via SSE), serve progress/live images
- `static/canvas.js` — DM fog canvas: brush and rectangle tools for reveal/shroud, auto-saves to server on mouseup. Player fog canvas: loads fog-live PNG as overlay. Both canvases scale to match map image dimensions.
- Fog workflow: DM draws on semi-transparent overlay → saves as `fog-progress.png` → clicks "Push to Players" → server copies to `fog-live.png` → SSE `fogUpdate` event → player reloads fog

### Phase 4 — Tokens
- Token struct added to `map.json`: id, x, y, radius, label, color, visible, moveable, imageId
- `internal/handler/tokens.go` — CRUD endpoints. DM can create/update/delete all tokens. Players can only move tokens marked `moveable`. Token changes broadcast `tokenUpdate` SSE event.
- `static/canvas.js` token layer — Separate canvas above fog. Renders colored circles with labels. DM double-clicks to place, drag to move. Player drags moveable tokens.

### Phase 5 — Chat & Dice
- `internal/store/chat.go` — In-memory ring buffer (200 messages)
- `internal/dice/dice.go` — Regex-based parser for `[NdS]`, `[NdS+M]`, `[NdSdlN]` (drop lowest), `[NdSkhN]` (keep highest), etc. Replaces expressions inline with rolled results.
- `internal/handler/chat.go` — Post message (processes dice), renders HTML fragment, broadcasts via SSE
- `templates/chat.templ` — Chat panel with message list, input form, auto-scroll. SSE `chatMessage` events append to `#chat-messages`.

### Phase 6 — Notes
- `internal/store/notes.go` — SQLite CRUD with FTS5 full-text search. Notes have title, markdown content, `isPublic` and `isEntryPoint` flags.
- `internal/handler/notes.go` — List, view (renders markdown via goldmark), edit form, save, delete, search. Players see only public notes.
- `templates/notes.templ` — Notes panel with search input, note list, note viewer (rendered markdown), note editor (textarea + checkboxes)

### Phase 7 — Media Library
- `internal/store/media.go` — File upload to `data/files/{id}.{ext}`, SHA256 dedup (returns existing record if hash matches), SQLite metadata
- `internal/handler/media.go` — Upload, list, serve (with content-type), delete
- `templates/media.templ` — Upload form, thumbnail grid with delete buttons

## Project Layout

```
main.go                          # Entry point, route wiring, embed
Makefile                         # Build targets
internal/
  config/config.go               # CLI + env config
  auth/auth.go                   # Sessions, roles, middleware
  store/
    db.go                        # SQLite pool + migrations
    maps.go                      # Map + Token filesystem store
    chat.go                      # In-memory chat ring buffer
    notes.go                     # Notes SQLite CRUD + FTS5
    media.go                     # Media file store + dedup
  realtime/broker.go             # SSE pub/sub
  dice/dice.go                   # Dice expression parser
  handler/
    auth.go home.go dm.go        # Page handlers
    maps.go fog.go tokens.go     # Map feature handlers
    chat.go notes.go media.go    # Content feature handlers
templates/
  layout.templ login.templ       # Shell templates
  home.templ dm.templ            # Page templates
  maps.templ chat.templ          # Feature templates
  notes.templ media.templ
static/
  htmx.min.js htmx-sse.js       # Vendored HTMX
  canvas.js                      # Fog + token canvas
  style.css                      # Dark theme CSS
```

## Running

```bash
# Development
make dev

# Production
DR_DM_PASSWORD=secret ./dungeon-revealer --data-dir=/path/to/data --port=3000

# Cross-compile for Raspberry Pi
make build-arm64
```

## Verified

- `CGO_ENABLED=0 go build` — compiles with no CGo
- `GOOS=linux GOARCH=arm64 go build` — cross-compiles successfully
- Auth flow: login → session → role-based access → logout
- Map flow: upload → list → activate → player sees map
- Fog flow: draw → save progress → push → player sees revealed areas
- Token flow: create → move → visibility toggle → player sees visible tokens
- Chat flow: send message → dice parsed → SSE broadcast → all clients see it
- Notes flow: create → edit → search (FTS5) → public notes visible to players
- Media flow: upload → dedup → serve → delete
