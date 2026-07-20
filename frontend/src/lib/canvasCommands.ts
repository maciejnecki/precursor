// Canvas command registry. The shortcut dispatcher in shortcuts.ts and the search
// store live outside the <SvelteFlow> context, so they cannot use the flow hooks
// directly. Components inside the flow register their viewport commands here, and
// outside callers invoke them imperatively through the accessor.

import { createCommandRegistry } from './commandRegistry'

// CanvasCommands describes the viewport actions the canvas exposes to the rest of
// the app.
export type CanvasCommands = {
  home(): void
  fitAll(): void
  centerOnNode(identifier: string): void
}

// registry holds the currently mounted canvas's commands, or null when no project
// (and therefore no canvas) is open.
const registry = createCommandRegistry<CanvasCommands>()

// registerCanvasCommands stores the given commands and returns an unregister
// function.
export function registerCanvasCommands(commands: CanvasCommands): () => void {
  return registry.register(commands)
}

// canvasCommands returns the currently registered commands, or null when no canvas
// is mounted; callers treat null as a safe no-op.
export function canvasCommands(): CanvasCommands | null {
  return registry.current()
}
