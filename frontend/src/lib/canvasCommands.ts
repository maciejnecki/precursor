// Canvas command registry. The window-level keyboard handler in App.svelte and the
// search store live outside the <SvelteFlow> context, so they cannot use the flow
// hooks directly. Components inside the flow register their viewport commands here,
// and outside callers invoke them imperatively through the accessor.

// CanvasCommands describes the viewport actions the canvas exposes to the rest of
// the app.
export type CanvasCommands = {
  home(): void
  fitAll(): void
  centerOnNode(identifier: string): void
}

// registeredCommands holds the currently mounted canvas's commands, or null when no
// project (and therefore no canvas) is open. A plain module variable suffices
// because nothing renders from it; it is only invoked imperatively.
let registeredCommands: CanvasCommands | null = null

// registerCanvasCommands stores the given commands and returns an unregister
// function. Unregistering only clears the registry if it still holds the same
// registration, so a newly mounted canvas is never clobbered by a stale cleanup.
export function registerCanvasCommands(commands: CanvasCommands): () => void {
  registeredCommands = commands
  return () => {
    if (registeredCommands === commands) {
      registeredCommands = null
    }
  }
}

// canvasCommands returns the currently registered commands, or null when no canvas
// is mounted; callers treat null as a safe no-op.
export function canvasCommands(): CanvasCommands | null {
  return registeredCommands
}
