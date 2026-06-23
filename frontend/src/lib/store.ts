// Central application state and the actions that mutate it. State is held in
// Svelte stores so any component can subscribe; actions wrap the API and keep the
// stores consistent after every backend call.
import { derived, get, writable } from 'svelte/store'
import { api, type Project, type ProjectView, type Settings } from './api'
import type { EditorMode } from './types'

// projects holds the metadata of every project for the sidebar.
export const projects = writable<Project[]>([])

// view holds the currently open project's render-ready view, or null when none.
export const view = writable<ProjectView | null>(null)

// settings holds the loaded user settings, or null before they load.
export const settings = writable<Settings | null>(null)

// selectedNodeIds holds every selected node id; shift-click builds a multi-selection
// used for grouped actions like proximity and delete.
export const selectedNodeIds = writable<string[]>([])

// selectedNodeId is the single selected node (or null when none or many are
// selected), used by the actions that act on exactly one node.
export const selectedNodeId = derived(selectedNodeIds, (ids) => (ids.length === 1 ? ids[0] : null))

// editorMode is the active behaviour of the bottom editor for the selection.
export const editorMode = writable<EditorMode>('precursor')

// editorOpen controls the floating compose popup (new task / precursor / decision /
// proximity). The canvas is full height, so this popup replaces the old panel.
export const editorOpen = writable<boolean>(false)

// editorAnchor is the viewport point of the last canvas click, used to spawn the
// compose popup near where the user clicked.
export const editorAnchor = writable<{ x: number; y: number }>({ x: 0, y: 0 })

// setEditorAnchor records where the next compose popup should appear.
export function setEditorAnchor(x: number, y: number): void {
  editorAnchor.set({ x, y })
}

// showSettings controls visibility of the settings panel.
export const showSettings = writable<boolean>(false)

// sidebarCollapsed controls whether the sidebar shows as a narrow icon rail.
export const sidebarCollapsed = writable<boolean>(false)

// projectModalOpen controls the centered project modal, opened from the sidebar's
// add button (cmd+n, create mode) or the cmd+e shortcut (edit mode).
export const projectModalOpen = writable<boolean>(false)

// projectEditId is the project being edited in the modal, or null when the modal is
// creating a new project. It drives the modal's heading, fields, and submit action.
export const projectEditId = writable<string | null>(null)

// modalNodeId is the node whose detail modal is open, or null when none is shown.
export const modalNodeId = writable<string | null>(null)

// editNodeId is the node the editor modal is editing; it is independent of the
// detail modal so the editor can be opened directly with the "e" shortcut.
export const editNodeId = writable<string | null>(null)

// editModalOpen controls the editor modal.
export const editModalOpen = writable<boolean>(false)

// errorMessage surfaces the most recent operation failure to the user.
export const errorMessage = writable<string>('')

// ConfirmRequest is a pending confirmation: the prompt and the resolver awaiting
// the user's choice. It backs an in-app dialog because the webview's native
// window.confirm does not reliably return a result.
type ConfirmRequest = { message: string; resolve: (confirmed: boolean) => void }

// confirmRequest holds the active confirmation, or null when none is pending.
export const confirmRequest = writable<ConfirmRequest | null>(null)

// requestConfirm shows the in-app confirm dialog and resolves to the user's choice.
export function requestConfirm(message: string): Promise<boolean> {
  return new Promise((resolve) => {
    confirmRequest.set({ message, resolve })
  })
}

// resolveConfirm settles the pending confirmation with the user's answer.
export function resolveConfirm(confirmed: boolean): void {
  const pending = get(confirmRequest)
  if (pending) {
    pending.resolve(confirmed)
    confirmRequest.set(null)
  }
}

// run executes an async action, recording any failure in errorMessage.
async function run<T>(action: () => Promise<T>): Promise<T | undefined> {
  try {
    errorMessage.set('')
    return await action()
  } catch (failure) {
    errorMessage.set(String(failure))
    return undefined
  }
}

// loadInitial loads the project list and settings when the app starts.
export async function loadInitial(): Promise<void> {
  await run(async () => {
    projects.set(await api.listProjects())
    settings.set(await api.getSettings())
  })
}

// refreshProjects reloads the project list, used after create, delete, or import.
async function refreshProjects(): Promise<void> {
  projects.set(await api.listProjects())
}

// applyView stores a new view and clears any selection or open modal whose node has
// vanished, keeping the editor and modals consistent after a mutation.
function applyView(next: ProjectView): void {
  view.set(next)
  const present = new Set(next.nodes.map((node) => node.id))
  selectedNodeIds.update((ids) => ids.filter((id) => present.has(id)))
  const currentModal = get(modalNodeId)
  if (currentModal && !present.has(currentModal)) {
    modalNodeId.set(null)
  }
  const currentEdit = get(editNodeId)
  if (currentEdit && !present.has(currentEdit)) {
    editNodeId.set(null)
    editModalOpen.set(false)
  }
}

// createProject creates a project and immediately opens it. Project colour is no
// longer surfaced in the UI, so an empty colour is passed for the stored field.
export async function createProject(name: string, description: string, icon: string): Promise<void> {
  await run(async () => {
    const created = await api.createProject(name, description, '', icon)
    await refreshProjects()
    await openProject(created.id)
  })
}

// updateProject saves edits to a project's metadata, keeping the sidebar list and
// the open view in sync. Project colour is no longer surfaced, so an empty colour
// is passed for the stored field.
export async function updateProject(identifier: string, name: string, description: string, icon: string): Promise<void> {
  await run(async () => {
    const updated = await api.updateProject(identifier, name, description, '', icon)
    await refreshProjects()
    const openView = get(view)
    if (openView && openView.project.id === identifier) {
      openView.project = updated
      view.set(openView)
    }
  })
}

// openProject opens a project and resets the selection.
export async function openProject(identifier: string): Promise<void> {
  await run(async () => {
    const opened = await api.openProject(identifier)
    selectedNodeIds.set([])
    modalNodeId.set(null)
    editNodeId.set(null)
    editModalOpen.set(false)
    editorOpen.set(false)
    editorMode.set('precursor')
    view.set(opened)
  })
}

// deleteProject removes a project and clears the view if it was open.
export async function deleteProject(identifier: string): Promise<void> {
  await run(async () => {
    await api.deleteProject(identifier)
    const openView = get(view)
    if (openView && openView.project.id === identifier) {
      view.set(null)
      selectedNodeIds.set([])
    }
    await refreshProjects()
  })
}

// openProjectModal shows the project modal in create mode, closing any open compose
// popup or node modal so the two do not overlap.
export function openProjectModal(): void {
  editorOpen.set(false)
  modalNodeId.set(null)
  editModalOpen.set(false)
  projectEditId.set(null)
  projectModalOpen.set(true)
}

// openProjectEditModal shows the project modal in edit mode for the given project.
export function openProjectEditModal(identifier: string): void {
  editorOpen.set(false)
  modalNodeId.set(null)
  editModalOpen.set(false)
  projectEditId.set(identifier)
  projectModalOpen.set(true)
}

// closeProjectModal hides the project modal.
export function closeProjectModal(): void {
  projectModalOpen.set(false)
  projectEditId.set(null)
}

// confirmAndDeleteActiveProject asks for confirmation, then deletes the currently
// open project. Used by the Backspace shortcut when focus is in the sidebar.
export async function confirmAndDeleteActiveProject(): Promise<void> {
  const openView = get(view)
  if (!openView) {
    return
  }
  const name = openView.project.name || 'Untitled'
  if (!(await requestConfirm(`Delete project "${name}"? This cannot be undone.`))) {
    return
  }
  await deleteProject(openView.project.id)
}

// selectNode replaces the selection with a single node (or clears it).
export function selectNode(identifier: string | null): void {
  selectedNodeIds.set(identifier ? [identifier] : [])
  editorMode.set('precursor')
}

// toggleNodeSelection adds or removes a node from the multi-selection (shift-click).
export function toggleNodeSelection(identifier: string): void {
  selectedNodeIds.update((ids) =>
    ids.includes(identifier) ? ids.filter((id) => id !== identifier) : [...ids, identifier]
  )
}

// openNewTaskEditor opens the compose popup with no selection, for creating a new
// central endpoint task.
export function openNewTaskEditor(): void {
  selectedNodeIds.set([])
  editorMode.set('precursor')
  modalNodeId.set(null)
  editModalOpen.set(false)
  editorOpen.set(true)
}

// openNewTaskAtCenter spawns the new-task popup in the middle of the window, used
// when the "t" shortcut fires with nothing selected.
export function openNewTaskAtCenter(): void {
  editorAnchor.set({ x: window.innerWidth / 2 - 280, y: window.innerHeight / 2 - 180 })
  openNewTaskEditor()
}

// openComposeForSelection opens the compose popup in the given mode for the single
// selected node (precursor via "t", decision via "d").
export function openComposeForSelection(mode: EditorMode): void {
  if (!get(selectedNodeId)) {
    return
  }
  editorMode.set(mode)
  modalNodeId.set(null)
  editModalOpen.set(false)
  editorOpen.set(true)
}

// closeEditor hides the floating compose popup.
export function closeEditor(): void {
  editorOpen.set(false)
}

// openNodeModal shows the detail modal for a node (its title and rendered body),
// closing the compose popup so the two do not overlap.
export function openNodeModal(identifier: string): void {
  modalNodeId.set(identifier)
  editorOpen.set(false)
}

// closeNodeModal hides the detail modal.
export function closeNodeModal(): void {
  modalNodeId.set(null)
}

// openEditModal opens the editor modal for a node, used by the "e" shortcut and the
// detail modal's Edit button.
export function openEditModal(identifier: string): void {
  editNodeId.set(identifier)
  editModalOpen.set(true)
  editorOpen.set(false)
}

// closeEditModal hides the editor modal.
export function closeEditModal(): void {
  editModalOpen.set(false)
}

// handleNodeClick selects a node on a plain click, or toggles it in the
// multi-selection on a shift-click. Any open compose popup is closed first.
export function handleNodeClick(identifier: string, additive: boolean): void {
  closeEditor()
  if (additive) {
    toggleNodeSelection(identifier)
  } else {
    selectNode(identifier)
  }
}

// createEndpointTask logs a new central endpoint task with no selection required.
export async function createEndpointTask(title: string, body: string, icon: string): Promise<boolean> {
  const result = await run(async () => {
    applyView(await api.createTask(title, body, icon))
    return true
  })
  return result === true
}

// saveSelected applies the editor content according to the current mode against the
// selected node, returning whether the editor should clear its fields afterwards.
export async function saveSelected(
  mode: EditorMode,
  title: string,
  body: string,
  icon: string,
  decisionType: string
): Promise<boolean> {
  const selection = get(selectedNodeId)
  if (!selection) {
    return false
  }
  const result = await run(async () => {
    if (mode === 'edit') {
      applyView(await api.updateNode(selection, title, body, icon))
      return false
    }
    if (mode === 'precursor') {
      applyView(await api.createPrecursor(selection, title, body, icon))
      return true
    }
    if (mode === 'decision') {
      // Adding a decision to a selected decision inserts it downstream; on a task it
      // appends to that task's decisions.
      const selectedNode = get(view)?.nodes.find((node) => node.id === selection)
      if (selectedNode?.kind === 'decision') {
        applyView(await api.createDecisionAfter(selection, decisionType, title, body, icon))
      } else {
        applyView(await api.createDecision(selection, decisionType, title, body, icon))
      }
      return true
    }
    return false
  })
  return result === true
}

// confirmAndDeleteSelected asks for confirmation, then deletes every selected node,
// healing each chain. Used by the Backspace shortcut for one or many tasks.
export async function confirmAndDeleteSelected(): Promise<void> {
  const ids = get(selectedNodeIds)
  if (ids.length === 0) {
    return
  }
  const message =
    ids.length > 1
      ? `Delete ${ids.length} selected nodes? Their chains will be healed.`
      : 'Delete this node? Its chain will be healed.'
  if (!(await requestConfirm(message))) {
    return
  }
  await run(async () => {
    let latest: ProjectView | undefined
    for (const id of ids) {
      latest = await api.deleteNode(id)
    }
    if (latest) {
      applyView(latest)
    }
    selectedNodeIds.set([])
  })
}

// createProximityGroup bonds the chains of all selected tasks so they cluster.
export async function createProximityGroup(): Promise<void> {
  const ids = get(selectedNodeIds)
  if (ids.length < 2) {
    return
  }
  await run(async () => {
    applyView(await api.createProximityGroup(ids))
    selectedNodeIds.set([])
  })
}

// saveNodeEdit updates a node's fields by id, used by the editor modal.
export async function saveNodeEdit(identifier: string, title: string, body: string, icon: string): Promise<void> {
  await run(async () => {
    applyView(await api.updateNode(identifier, title, body, icon))
    editModalOpen.set(false)
  })
}

// toggleDecisions collapses or expands the decisions on a task's parent link.
export async function toggleDecisions(identifier: string, collapsed: boolean): Promise<void> {
  await run(async () => {
    applyView(await api.setDecisionsCollapsed(identifier, collapsed))
  })
}

// deleteNodeById removes a node by id and heals its chain, closing any open modal.
export async function deleteNodeById(identifier: string): Promise<void> {
  await run(async () => {
    applyView(await api.deleteNode(identifier))
    selectedNodeIds.update((ids) => ids.filter((id) => id !== identifier))
    modalNodeId.set(null)
    editNodeId.set(null)
    editModalOpen.set(false)
  })
}

// saveSettings persists settings and updates the in-memory copy.
export async function saveSettings(next: Settings): Promise<void> {
  await run(async () => {
    await api.saveSettings(next)
    settings.set(await api.getSettings())
  })
}

// copyCompletedMarkdown writes the completed-tasks table to the system clipboard.
export async function copyCompletedMarkdown(): Promise<void> {
  const openView = get(view)
  if (!openView) {
    return
  }
  await run(async () => {
    const markdown = await api.getCompletedMarkdown(openView.project.id)
    await navigator.clipboard.writeText(markdown)
  })
}

// exportProjectFile prompts for a path and writes a JSON backup of the open project.
export async function exportProjectFile(): Promise<void> {
  const openView = get(view)
  if (!openView) {
    return
  }
  await run(async () => {
    await api.exportProject(openView.project.id)
  })
}

// importProjectFile imports a JSON backup as a new project and opens it.
export async function importProjectFile(): Promise<void> {
  await run(async () => {
    const imported = await api.importProject()
    await refreshProjects()
    if (imported && imported.id) {
      await openProject(imported.id)
    }
  })
}
