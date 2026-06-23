// Shared frontend types and constants describing the editor's interaction model.

// EditorMode is the active behaviour of the bottom editor. When a node is selected
// the user cycles through these with the Tab key; with no selection only a new
// endpoint task can be created.
export type EditorMode = 'edit' | 'precursor' | 'decision' | 'proximity'

// editorModeOrder is the fixed cycle order used when pressing Tab on a selection.
// Editing and proximity are keyboard actions, so the compose popup only adds a
// precursor or a decision.
export const editorModeOrder: EditorMode[] = ['precursor', 'decision']

// editorModeLabels are the human-readable names shown on the editor tabs.
export const editorModeLabels: Record<EditorMode, string> = {
  edit: 'Edit',
  precursor: 'New Precursor',
  decision: 'New Decision',
  proximity: 'Proximity'
}

// DecisionTypeOption pairs a decision type value with its display label and the
// glyph shown on its selector button.
export type DecisionTypeOption = { value: string; label: string; icon: string }

// decisionTypeOptions lists the decision types offered in New Decision mode.
// Scheduled is omitted because it is the default derived status of a task with no
// decisions, so it is never chosen explicitly.
export const decisionTypeOptions: DecisionTypeOption[] = [
  { value: 'in_progress', label: 'In Progress', icon: '⏳' },
  { value: 'done', label: 'Done', icon: '✅' },
  { value: 'redundant', label: 'Redundant', icon: '🚫' },
  { value: 'plain', label: 'Design intent', icon: '💡' }
]

// iconGlyphs is the curated icon palette shared by the icon picker and project
// creation, so both draw from the same set.
export const iconGlyphs = ['🚩', '🪼', '🐛', '🍀', '🌏', '🔥', '💥', '🎲', '🔷', '🛠️', '⚙️', '📡']

// randomGlyph returns a random icon from the palette, used to seed new projects.
export function randomGlyph(): string {
  return iconGlyphs[Math.floor(Math.random() * iconGlyphs.length)]
}
