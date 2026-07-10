// Shared frontend types and constants describing the editor's interaction model.

// EditorMode is the active behaviour of the bottom editor. With a node selected
// the compose popup adds a precursor or a decision; with no selection only a new
// endpoint task can be created.
export type EditorMode = 'precursor' | 'decision'

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
