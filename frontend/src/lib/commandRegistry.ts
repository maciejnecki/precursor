// Command registry factory. Several parts of the app need to invoke actions that
// only exist inside a mounted component — the canvas viewport lives inside the
// <SvelteFlow> context, and saving belongs to whichever editor is currently open.
// Callers outside those components register their actions here and invoke them
// imperatively through the accessor.

// CommandRegistry pairs a registration function with an accessor over the commands
// currently on offer.
export type CommandRegistry<Commands> = {
  register(commands: Commands): () => void
  current(): Commands | null
}

// createCommandRegistry builds a registry holding one set of commands at a time.
// Registering returns an unregister function that only clears the registry if it
// still holds the same registration, so a newly mounted owner is never clobbered by
// a stale cleanup. A plain closure variable suffices because nothing renders from
// the registry; it is only invoked imperatively.
export function createCommandRegistry<Commands>(): CommandRegistry<Commands> {
  let registeredCommands: Commands | null = null

  return {
    register(commands: Commands): () => void {
      registeredCommands = commands
      return () => {
        if (registeredCommands === commands) {
          registeredCommands = null
        }
      }
    },
    current(): Commands | null {
      return registeredCommands
    }
  }
}
