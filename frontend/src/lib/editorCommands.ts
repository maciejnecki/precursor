// Editor command registry. The save shortcut is dispatched centrally in shortcuts.ts,
// but the fields being saved belong to whichever editor is currently open — the
// compose popup or the node edit modal. Each registers its save action while it is
// visible, mirroring how the canvas exposes its viewport commands.

import { createCommandRegistry } from './commandRegistry'

// EditorCommands describes the actions the open editor exposes to the shortcut
// dispatcher.
export type EditorCommands = {
  save(): void
}

// registry holds the open editor's commands, or null when no editor is open.
const registry = createCommandRegistry<EditorCommands>()

// registerEditorCommands stores the given commands and returns an unregister
// function.
export function registerEditorCommands(commands: EditorCommands): () => void {
  return registry.register(commands)
}

// editorCommands returns the open editor's commands, or null when none is open;
// callers treat null as a safe no-op.
export function editorCommands(): EditorCommands | null {
  return registry.current()
}
