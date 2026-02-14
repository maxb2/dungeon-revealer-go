# Dungeon Revealer — Architectural Summary

## What It Is

A self-hosted web app for tabletop RPG sessions (D&D, etc.). A **Dungeon Master (DM)** uploads maps, controls fog-of-war, places tokens, and manages game notes. **Players** connect via browser and see only what the DM reveals. Everything updates in real time.

---

## User Roles

| Role | Access | Capabilities |
|---|---|---|
| **DM (admin)** | `/dm` path, password-protected | Full control: upload maps, draw fog, manage tokens/notes, share content |
| **Player** | Root `/`, optionally password-protected | View revealed map, move allowed tokens, chat, roll dice |
| **Unauthenticated** | Limited | Splash screen only (chat if no player password set) |

---

## Core Features

### Map & Fog-of-War
- DM uploads a map image and draws fog over it (hiding areas from players)
- Two fog layers: **progress** (DM's working state) and **live** (what players see) — DM explicitly pushes fog to players
- Optional grid overlay with configurable cell size, color, and offset

### Tokens
- Placed on maps with position, rotation, color, label, and radius
- Per-token visibility (`isVisibleForPlayers`) and moveability (`isMovableByPlayers`, `isLocked`)
- Can have an attached image (cropped from media library) and a linked note

### DM Map Tools (keyboard shortcuts 1–5)
1. **Move/Drag** — pan the map
2. **Area select** — rectangle reveal/shroud
3. **Brush** — freehand reveal/shroud
4. **Mark/Ping** — temporary circle indicator visible to all
5. **Token placement**

### Notes
- Markdown-based game notes created by the DM
- Two access types: **admin** (DM-only) or **public** (players can view)
- Entry-point notes appear in a main list; others are linked from content
- Full-text search

### Chat & Dice
- Real-time chat between all connected users
- Inline dice rolling via notation like `[1d20]`
- Dice results show individual rolls with min/max categorization (for drop-lowest, etc.)
- System messages for join/leave events
- DM can share notes and images directly into chat

### Media Library
- DM uploads images for sharing or cropping into token portraits
- Token images are SHA256-deduplicated

---

## Old Architecture

```
┌──────────────┐         WebSocket (GraphQL live queries)        ┌──────────────┐
│  DM Client   │◄──────────────────────────────────────────────►│              │
│  /dm         │         REST (image upload/download)            │   Server     │
└──────────────┘                                                 │              │
                                                                 │  ┌────────┐  │
┌──────────────┐         WebSocket (GraphQL live queries)        │  │ SQLite  │  │
│ Player Client│◄──────────────────────────────────────────────►│  │   DB    │  │
│  /           │         REST (image download)                   │  └────────┘  │
└──────────────┘                                                 │  ┌────────┐  │
                                                                 │  │  File  │  │
                                                                 │  │ System │  │
                                                                 │  └────────┘  │
                                                                 └──────────────┘
```

### Communication
- **Primary:** GraphQL queries/mutations/subscriptions over WebSocket — a `@live` directive automatically pushes updated data to all connected clients when server state changes
- **Secondary:** REST endpoints for binary data (map image upload/download, fog images, media files)

### Data Storage

**SQLite database** for structured data:
- `file_uploads` — uploaded media metadata
- `notes` — game notes with full-text search index
- `tokenImages` — token portrait metadata

**Filesystem** for binary assets:
```
data/
├── db.sqlite
├── maps/{mapId}/
│   ├── map.{ext}           # original map image
│   ├── fog-progress.png    # DM's working fog
│   ├── fog-live.png        # player-visible fog
│   └── map.json            # map metadata (title, grid config, tokens array)
├── files/{fileId}.{ext}    # uploaded media
└── token-images/{id}.{ext} # cropped token portraits
```

---

## Old Data Model (Key Entities)

- **Map** — id, title, grid config, list of tokens, fog layers (progress + live), image
- **MapToken** — x, y, rotation, color, label, radius, isVisibleForPlayers, isMovableByPlayers, isLocked, optional tokenImageId, optional noteId
- **MapGrid** — showGrid, showGridToPlayers, color, offsetX/Y, columnWidth/Height
- **Note** — id, title, content (markdown), type (admin|public), isEntryPoint, timestamps
- **TokenImage** — id, title, sha256 hash, file extension
- **User** — session-based (socket ID), name, role (admin|user|unauthenticated)
- **ChatMessage** — three subtypes: user message (with optional dice rolls), system event (join/leave), shared resource (note/image embed)
- **DiceRoll** — result value, expression tree, individual roll outcomes

---

## Permission Model

- DM and player passwords are configured at startup (environment variables)
- Session role is assigned on WebSocket authentication
- **DM sees everything**; players only see: revealed fog, visible tokens, public notes
- Fog progress image is only served to DM; fog live image is served to players
- Tokens have per-token visibility and moveability flags

## New Architecture

For Dungeon Revealer specifically:

Best option:

✅ Go backend
✅ Go-generated HTML (templ)
✅ HTMX
✅ small vanilla JS for canvas

This gives you:

single executable

pure Go server

minimal JS

zero frontend frameworks

easy distribution

still powerful UI
