# Precursor

Precursor is a desktop app for planning work as **chains of tasks**. Each task can
have a single *precursor* (the task that must finish before it), and the
*decisions* you record on a task drive its derived status — scheduled, in
progress, done, or redundant. Chains fan out radially from their endpoint task on
a pannable, zoomable canvas, with a live completion percentage per chain.

It is a [Wails](https://wails.io) application: a Go backend compiled into a native
window, with a Svelte 5 + `@xyflow/svelte` frontend.

## Architecture

```
main.go            Wails bootstrap and window options
app.go             Wails-bound API surface (methods callable from the frontend)
internal/
  model/           Core domain types (Node, Project, Graph, statuses) — no behaviour
  storage/         Per-project SQLite persistence (one database file per project)
  config/          User settings persisted as a human-editable TOML file
  graph/           Pure logic: chains, derived status, deletion/healing
  layout/          Turns chains into canvas coordinates (radial layout)
  export/          JSON backups and the completed-tasks Markdown table
  service/         Orchestration layer that ties the packages above together
frontend/          Svelte 5 UI, xyflow canvas, Vite build
```

Data lives per-user under the OS config directory (`os.UserConfigDir()/precursor`):
each project is its own SQLite database; settings are a single TOML file.

## Key functions

### Backend API (`app.go`) — callable from the frontend

These are exposed to the UI through generated TypeScript bindings.

| Method | Purpose |
| --- | --- |
| `ListProjects` / `CreateProject` / `UpdateProject` / `DeleteProject` | Project CRUD |
| `OpenProject` / `CurrentView` | Activate a project and return its laid-out view |
| `CreateTask` | Add a new endpoint task (the root of a chain) |
| `CreatePrecursor` | Add the single precursor that must finish before a task |
| `CreateDecision` / `CreateDecisionAfter` | Record a typed/plain decision on a task, or chain a decision after another |
| `UpdateNode` / `DeleteNode` | Edit a node's content, or remove a node and heal its chain |
| `SetDecisionsCollapsed` | Hide or show the decisions on a task's link to its parent |
| `CreateProximity` / `CreateProximityGroup` / `DeleteProximity` | Bond chains so they cluster in the layout |
| `GetSettings` / `SaveSettings` | Read and persist user settings |
| `GetCompletedMarkdown` / `SaveCompletedMarkdown` | Completed-tasks Markdown table (clipboard or file) |
| `ExportProject` / `ImportProject` | JSON backup to / from a user-chosen file |

### Domain logic (`internal/graph`)

| Function | Purpose |
| --- | --- |
| `DeriveStatus` | Compute a task's status from its decisions (endpoints auto-complete when all precursors resolve) |
| `DecisionsFor` / `NextDecisionOrderIndex` | Read and order the decisions documenting a task |
| `PrecursorOf` / `HasPrecursor` | Enforce the one-precursor-per-task rule |
| `Endpoints` / `EndpointID` / `Chain` | Identify chain roots and walk a task's full chain |
| `DeleteNode` | Remove a node and produce the `ChangeSet` that heals the surrounding chain and bonds |

### Layout (`internal/layout`)

| Function | Purpose |
| --- | --- |
| `Compute` | Place every chain's nodes into canvas coordinates radially, honouring proximity bonds |
| `DefaultConfig` | The spacing / sizing constants used by `Compute` |

### Service (`internal/service`)

`service.New` builds the orchestrator; its methods mirror the `app.go` API and are
where storage, graph, layout, and export are coordinated into a `ProjectView`.

### Frontend (`frontend/src/lib`)

| Module | Purpose |
| --- | --- |
| `store.ts` | Central Svelte stores and actions; the bridge to the Go API |
| `api.ts` | Thin wrapper over the generated Wails bindings |
| `chains.ts` | `computeChainAreas` — the translucent background + completion % per chain |
| `markdown.ts` | `renderMarkdown` for node bodies and the detail modal |
| `Canvas.svelte` | The xyflow canvas: nodes, edges, zoom, controls |

## Prerequisites

- **Go** 1.25+
- **Node.js** 18+ (frontend build, bundled npm)
- **Wails CLI** v2.12: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Platform toolchain that Wails needs:
  - **macOS:** Xcode Command Line Tools (`xcode-select --install`)
  - **Windows:** WebView2 runtime (preinstalled on Windows 11) and a C compiler (MSVC or MinGW)

Run `wails doctor` to confirm your environment is ready.

## Running (development)

From the project root:

```bash
wails dev
```

This starts a Vite dev server with hot reload for the frontend and rebuilds the Go
backend on change. A browser devtools endpoint is also served at
`http://localhost:34115` if you want to call Go methods from the browser.

## Building (production)

### macOS

```bash
wails build
```

Produces `build/bin/precursor.app`. To build a universal (Intel + Apple Silicon)
binary:

```bash
wails build -platform darwin/universal
```

### Windows

Run on a Windows machine (or cross-compile from another host with the right
toolchain):

```bash
wails build -platform windows/amd64
```

Produces `build/bin/precursor.exe`. To also generate an NSIS installer:

```bash
wails build -platform windows/amd64 -nsis
```

The build is configured by `wails.json`; the app icon is generated by
`build/make_appicon.py` into `build/appicon.png`.

## Tests

```bash
go test ./...                 # backend
cd frontend && npm run check  # frontend type/lint checks
```
