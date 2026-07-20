// Shortcut dispatch. Every shortcut the app offers is named by a stable identifier
// and performed by runShortcut, so the keyboard handler in App.svelte and the native
// menu bar drive exactly the same code. The identifiers are the contract with the Go
// side: buildAppMenu in main.go emits them on the "menu:action" event.

import { get } from 'svelte/store'
import { canvasCommands } from './canvasCommands'
import { editorCommands } from './editorCommands'
import {
  confirmAndDeleteActiveProject,
  confirmAndDeleteSelected,
  confirmRequest,
  createProximityGroup,
  editModalOpen,
  editorOpen,
  modalNodeId,
  openComposeForSelection,
  openEditModal,
  openNewTaskAtCenter,
  openNodeModal,
  openProjectDetail,
  openProjectEditModal,
  openProjectModal,
  openSearch,
  projectDetailOpen,
  projectModalOpen,
  selectedNodeId,
  selectedNodeIds,
  showSettings,
  view
} from './store'
import type { NodeView } from './api'

// ShortcutId names every action reachable from a shortcut or a menu item. Keep this
// union in step with the menu tables in main.go.
export type ShortcutId =
  | 'app.settings'
  | 'project.new'
  | 'project.edit'
  | 'project.delete'
  | 'editor.save'
  | 'node.precursor'
  | 'node.newTask'
  | 'node.decision'
  | 'node.edit'
  | 'details.show'
  | 'node.group'
  | 'node.delete'
  | 'view.home'
  | 'view.fit'
  | 'view.find'

// overlayOpen reports whether a modal, popup, or confirmation is on screen. Most
// shortcuts stay inert behind one so they cannot act on the canvas underneath.
export function overlayOpen(): boolean {
  return (
    get(editorOpen) ||
    get(modalNodeId) !== null ||
    get(projectDetailOpen) ||
    get(editModalOpen) ||
    get(projectModalOpen) ||
    get(showSettings) ||
    get(confirmRequest) !== null
  )
}

// singleSelectedNode returns the node when exactly one is selected, else undefined.
function singleSelectedNode(): NodeView | undefined {
  const identifier = get(selectedNodeId)
  if (!identifier) {
    return undefined
  }
  return get(view)?.nodes.find((node) => node.id === identifier)
}

// runShortcut performs the named action if the app is in a state where it applies.
// Every branch carries its own guard because menu items are always clickable: an
// action that does not apply right now is a silent no-op rather than an error.
export function runShortcut(identifier: ShortcutId): void {
  // Saving is the one action that stays available behind an overlay, since the
  // overlay in question is the editor doing the saving.
  if (identifier === 'editor.save') {
    editorCommands()?.save()
    return
  }
  if (overlayOpen()) {
    return
  }
  // Settings and New Project are the only actions that do not need an open project.
  if (identifier === 'app.settings') {
    showSettings.set(true)
    return
  }
  if (identifier === 'project.new') {
    openProjectModal()
    return
  }
  // Everything below acts on the open project, so it needs a view.
  const openView = get(view)
  if (!openView) {
    return
  }
  const node = singleSelectedNode()
  switch (identifier) {
    case 'project.edit':
      openProjectEditModal(openView.project.id)
      break
    case 'project.delete':
      void confirmAndDeleteActiveProject()
      break
    case 'node.precursor':
      if (node && node.kind === 'task') {
        openComposeForSelection('precursor')
      }
      break
    case 'node.newTask':
      openNewTaskAtCenter()
      break
    case 'node.decision':
      if (node && ((node.kind === 'task' && node.parentId) || node.kind === 'decision')) {
        openComposeForSelection('decision')
      }
      break
    case 'node.edit':
      if (node) {
        openEditModal(node.id)
      }
      break
    // Details is contextual: the selected node's card, or the project's own card
    // when the canvas selection is empty, since the project card is not selectable.
    case 'details.show':
      if (node) {
        openNodeModal(node.id)
      } else if (get(selectedNodeIds).length === 0) {
        openProjectDetail()
      }
      break
    case 'node.group':
      if (get(selectedNodeIds).length >= 2) {
        void createProximityGroup()
      }
      break
    case 'node.delete':
      if (get(selectedNodeIds).length > 0) {
        void confirmAndDeleteSelected()
      }
      break
    case 'view.home':
      canvasCommands()?.home()
      break
    case 'view.fit':
      canvasCommands()?.fitAll()
      break
    case 'view.find':
      openSearch()
      break
  }
}

// shortcutForEvent maps a keystroke to the action it triggers, or null when the
// keystroke is not a shortcut. Every combination carries a modifier: an unmodified
// letter would have to be registered as a bare menu accelerator, which macOS routes
// to the menu bar before the webview and would swallow the letter while typing.
export function shortcutForEvent(event: KeyboardEvent): ShortcutId | null {
  if (!(event.metaKey || event.ctrlKey)) {
    return null
  }
  const key = event.key.toLowerCase()
  if (event.shiftKey) {
    switch (key) {
      case 'e':
        return 'project.edit'
      case 'backspace':
      case 'delete':
        return 'project.delete'
      case 't':
        return 'node.newTask'
      // Shift+0 reports ")" on a US layout, so both spellings map to fit.
      case '0':
      case ')':
        return 'view.fit'
      default:
        return null
    }
  }
  switch (key) {
    case ',':
      return 'app.settings'
    case 'n':
      return 'project.new'
    case 's':
      return 'editor.save'
    case 't':
      return 'node.precursor'
    case 'd':
      return 'node.decision'
    case 'e':
      return 'node.edit'
    case 'i':
      return 'details.show'
    case 'g':
      return 'node.group'
    case 'backspace':
    case 'delete':
      return 'node.delete'
    case '0':
      return 'view.home'
    case 'f':
      return 'view.find'
    default:
      return null
  }
}
