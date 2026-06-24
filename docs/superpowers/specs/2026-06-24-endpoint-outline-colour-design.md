# Endpoint outline colour — design

Date: 2026-06-24

## Goal

Pending endpoints (chain-root tasks that are not yet resolved) should visually
stand out so an incomplete chain is obvious at a glance. They get a distinct
outline colour, `rgb(252, 78, 38)` (`#fc4e26`), which is itself user-customisable
in Settings.

## Behaviour

A node card draws the standout outline when it is a *pending endpoint*:

- `kind === 'task'`
- no `parentId` (it is a chain root — the same test already used in
  `CanvasControls.svelte`)
- derived `status` is `scheduled` or `in_progress`

Once the endpoint resolves to `done` or `redundant`, it reverts to its normal
status colour. The selected-state border (accent blue) is unchanged and still
overrides the resting outline.

## Architecture choice

`colourForNode()` in `Canvas.svelte` is shared by node cards and edges
(`colourForEdge` delegates to it), and a decision attached to an endpoint
produces an edge owned by that endpoint. To avoid recolouring those decision
links, `colourForNode()` stays purely status-based. A thin wrapper
`outlineForNode()` adds the pending-endpoint override and is used only by the
node-card mapping. Edges are untouched.

## Changes

1. **`internal/config/config.go`**
   - Add `EndpointColour string` to `Settings`
     (`toml:"endpointColour" json:"endpointColour"`).
   - Add `fallbackEndpoint = "#fc4e26"`.
   - Wire into `DefaultSettings()` and `withFallbacks()`.

2. **`frontend/wailsjs/go/models.ts`**
   - Add `endpointColour: string` to the generated `config.Settings` class and
     its constructor (regenerate via the wails CLI, or hand-edit to the identical
     result).

3. **`frontend/src/lib/Canvas.svelte`**
   - `isPendingEndpoint(node)` — the test above.
   - `outlineForNode(node, settings)` — standout colour for pending endpoints,
     else `colourForNode(node, settings)`.
   - Node-card mapping `colour:` uses `outlineForNode`. Edges unchanged.

4. **`frontend/src/lib/Settings.svelte`**
   - New `Endpoint colour` section with one `<input type="color">` bound to
     `draft.endpointColour`, labelled "Pending endpoint", placed after the Status
     colours section.

## Default

`rgb(252, 78, 38)` = `#fc4e26`, stored as hex because `<input type="color">` is
hex-only.

## Out of scope

Edge / decision-link recolouring, the selected-state border, the chain
completion label.
