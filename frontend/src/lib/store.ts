// Central application state and the actions that mutate it. State is held in
// Svelte stores so any component can subscribe; actions wrap the API and keep the
// stores consistent after every backend call.
import { derived, get, writable } from 'svelte/store'
import { api, type Project, type ProjectGroup, type ProjectView, type Settings, type SidebarState } from './api'
import { canvasCommands } from './canvasCommands'
import type { EditorMode } from './types'

// projects holds the metadata of every project for the sidebar, in stored order.
export const projects = writable<Project[]>([])

// projectGroups holds the sidebar's project groups, in the order they are stored.
export const projectGroups = writable<ProjectGroup[]>([])

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

// projectDetailOpen reports whether the open project's detail modal is showing. The
// modal reads the project straight from the view, so a flag is all it needs.
export const projectDetailOpen = writable<boolean>(false)

// editNodeId is the node the editor modal is editing; it is independent of the
// detail modal so the editor can be opened directly with the "e" shortcut.
export const editNodeId = writable<string | null>(null)

// editModalOpen controls the editor modal.
export const editModalOpen = writable<boolean>(false)

// Toast is a transient message shown at the bottom of the window: an operation
// failure that stays until dismissed, or a brief success confirmation.
export type Toast = { kind: 'error' | 'success'; text: string }

// toast holds the visible transient message, or null when none is shown.
export const toast = writable<Toast | null>(null)

// successTimer auto-dismisses success toasts; errors stay until clicked away.
let successTimer: ReturnType<typeof setTimeout> | undefined

// showError surfaces an operation failure until the user dismisses it.
export function showError(text: string): void {
  clearTimeout(successTimer)
  toast.set({ kind: 'error', text })
}

// showSuccess briefly confirms a completed action, clearing itself after a moment.
export function showSuccess(text: string): void {
  clearTimeout(successTimer)
  toast.set({ kind: 'success', text })
  successTimer = setTimeout(() => toast.set(null), 2500)
}

// dismissToast hides the current transient message.
export function dismissToast(): void {
  clearTimeout(successTimer)
  toast.set(null)
}

// searchOpen controls visibility of the canvas search bar (cmd+f).
export const searchOpen = writable<boolean>(false)

// searchQuery is the live text being searched across node titles and descriptions.
export const searchQuery = writable<string>('')

// searchFocusToken is bumped every time the search bar is (re)opened, so the bar
// refocuses and reselects its query text even when it is already showing.
export const searchFocusToken = writable<number>(0)

// searchActiveIndex is the position within the match list that Enter-cycling last
// visited, shown as the first half of the bar's "2 / 7" counter.
export const searchActiveIndex = writable<number>(0)

// searchMatchIds is the ordered list of node ids whose title or description contains
// the query, case-insensitively. It is empty for a blank query and recomputes from
// the view, so edited or deleted nodes drop out of the match set automatically.
export const searchMatchIds = derived([view, searchQuery], ([openView, query]) => {
  const needle = query.trim().toLowerCase()
  if (!openView || needle.length === 0) {
    return []
  }
  return openView.nodes
    .filter(
      (node) =>
        node.title.toLowerCase().includes(needle) || node.bodyMarkdown.toLowerCase().includes(needle)
    )
    .map((node) => node.id)
})

// searchNavigated tracks whether Enter has been pressed for the current query, so
// the first Enter pans to the first match instead of skipping past it.
let searchNavigated = false

// openSearch shows the search bar and bumps the focus token so the input grabs
// focus (and reselects its text) even if the bar was already open.
export function openSearch(): void {
  searchOpen.set(true)
  searchFocusToken.update((token) => token + 1)
}

// closeSearch hides the search bar and clears the query, which empties the match
// set and removes all dimming from the canvas.
export function closeSearch(): void {
  searchOpen.set(false)
  searchQuery.set('')
  searchActiveIndex.set(0)
  searchNavigated = false
}

// setSearchQuery updates the live query and restarts cycling from the first match.
export function setSearchQuery(query: string): void {
  searchQuery.set(query)
  searchActiveIndex.set(0)
  searchNavigated = false
}

// navigateSearch pans the canvas to a match: the first press lands on the current
// match, and subsequent presses step by delta with wrap-around in both directions.
export function navigateSearch(delta: number): void {
  const matches = get(searchMatchIds)
  if (matches.length === 0) {
    return
  }
  const current = get(searchActiveIndex)
  const clamped = current < matches.length ? current : 0
  const next = searchNavigated ? (clamped + delta + matches.length) % matches.length : clamped
  searchNavigated = true
  searchActiveIndex.set(next)
  canvasCommands()?.centerOnNode(matches[next])
}

// appVersion is the build version reported by the backend: the release tag, or
// "dev" when running an uninjected build. Shown in the settings footer.
export const appVersion = writable<string>('dev')

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

// readableMessage extracts a human-readable message from a thrown value, since
// Wails rejections and plain objects stringify unhelpfully.
function readableMessage(failure: unknown): string {
  if (failure instanceof Error) {
    return failure.message
  }
  if (typeof failure === 'string') {
    return failure
  }
  return String(failure)
}

// run executes an async action, surfacing any failure as an error toast.
async function run<T>(action: () => Promise<T>): Promise<T | undefined> {
  try {
    dismissToast()
    return await action()
  } catch (failure) {
    showError(readableMessage(failure))
    return undefined
  }
}

// loadInitial loads the project list, settings, and build version when the app
// starts.
export async function loadInitial(): Promise<void> {
  await run(async () => {
    applySidebar(await api.sidebar())
    settings.set(await api.getSettings())
    appVersion.set(await api.version())
  })
}

// applySidebar stores a sidebar state returned by the backend.
function applySidebar(state: SidebarState): void {
  projects.set(state.projects)
  projectGroups.set(state.groups)
}

// refreshProjects reloads the sidebar, used after create, delete, or import. The
// backend drops any group membership a deleted project left behind.
async function refreshProjects(): Promise<void> {
  applySidebar(await api.sidebar())
}

// saveSidebar persists a new project order and set of groups. Both are applied
// optimistically so a drop lands without a flicker, then replaced by what the
// backend stored, or rolled back when the write fails.
export async function saveSidebar(order: string[], groups: ProjectGroup[]): Promise<void> {
  const previousProjects = get(projects)
  const previousGroups = get(projectGroups)
  const byId = new Map(previousProjects.map((project) => [project.id, project]))
  const reordered = order
    .map((identifier) => byId.get(identifier))
    .filter((project): project is Project => project !== undefined)
  projects.set(reordered)
  projectGroups.set(groups)
  const stored = await run(() => api.saveSidebar(order, groups))
  if (stored) {
    applySidebar(stored)
  } else {
    projects.set(previousProjects)
    projectGroups.set(previousGroups)
  }
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

// selectCreatedNode makes the single node present in the new view but absent from
// the before-set the active selection, so a freshly created task or precursor is
// selected automatically and is ready for the next "t" press.
function selectCreatedNode(next: ProjectView, before: Set<string>): void {
  const created = next.nodes.find((node) => !before.has(node.id))
  if (created) {
    selectNode(created.id)
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
    closeSearch()
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

// confirmAndDeleteProject asks for confirmation, then deletes the given project. It
// backs both the sidebar Backspace shortcut and the right-click delete option.
export async function confirmAndDeleteProject(identifier: string, name: string): Promise<void> {
  if (!(await requestConfirm(`Delete project "${name || 'Untitled'}"? This cannot be undone.`))) {
    return
  }
  await deleteProject(identifier)
}

// confirmAndDeleteActiveProject deletes the currently open project, used by the
// Backspace shortcut when focus is in the sidebar.
export async function confirmAndDeleteActiveProject(): Promise<void> {
  const openView = get(view)
  if (!openView) {
    return
  }
  await confirmAndDeleteProject(openView.project.id, openView.project.name)
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

// openProjectDetail shows the open project's detail modal, which presents the full
// description the canvas card can only show a scrolling window onto. It closes the
// compose popup for the same reason openNodeModal does.
export function openProjectDetail(): void {
  projectDetailOpen.set(true)
  editorOpen.set(false)
}

// closeProjectDetail hides the project detail modal.
export function closeProjectDetail(): void {
  projectDetailOpen.set(false)
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

// createEndpointTask logs a new central endpoint task with no selection required,
// then selects it so it becomes the anchor for the next precursor.
export async function createEndpointTask(title: string, body: string, icon: string): Promise<boolean> {
  const before = new Set(get(view)?.nodes.map((node) => node.id) ?? [])
  const result = await run(async () => {
    const next = await api.createTask(title, body, icon)
    applyView(next)
    selectCreatedNode(next, before)
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
    if (mode === 'precursor') {
      const before = new Set(get(view)?.nodes.map((node) => node.id) ?? [])
      const next = await api.createPrecursor(selection, title, body, icon)
      applyView(next)
      selectCreatedNode(next, before)
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

// isEndpoint reports whether the node with the given id is a chain root: a task
// with no parent. Deleting one takes its whole chain with it, so the confirmation
// copy has to say so.
function isEndpoint(identifier: string): boolean {
  const node = get(view)?.nodes.find((candidate) => candidate.id === identifier)
  return node?.kind === 'task' && !node.parentId
}

// chainLength counts the tasks that would go with an endpoint, by walking the
// parent links back from the endpoint the same way the backend's chain walk does.
function chainLength(endpointId: string): number {
  const nodes = get(view)?.nodes ?? []
  let count = 0
  let current: string | undefined = endpointId
  while (current) {
    count += 1
    const next: string | undefined = nodes.find(
      (node) => node.kind === 'task' && node.parentId === current
    )?.id
    current = next
  }
  return count
}

// deleteMessage describes what deleting the given selection will actually do, so
// the irreversible case (an endpoint taking its chain with it) is never a surprise.
function deleteMessage(ids: string[]): string {
  if (ids.length > 1) {
    return ids.some(isEndpoint)
      ? `Delete ${ids.length} selected nodes? Any endpoint takes its entire chain with it.`
      : `Delete ${ids.length} selected nodes? Their chains will be healed.`
  }
  if (isEndpoint(ids[0])) {
    const length = chainLength(ids[0])
    return length > 1
      ? `Delete this endpoint and its entire chain (${length} tasks)?`
      : 'Delete this endpoint?'
  }
  return 'Delete this node? Its chain will be healed.'
}

// confirmAndDeleteSelected asks for confirmation, then deletes every selected node.
// Used by the Backspace shortcut for one or many nodes.
export async function confirmAndDeleteSelected(): Promise<void> {
  const ids = get(selectedNodeIds)
  if (ids.length === 0) {
    return
  }
  if (!(await requestConfirm(deleteMessage(ids)))) {
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

// deleteNodeById removes a node by id, taking its whole chain when it is an
// endpoint and healing the chain otherwise, then closes any open modal.
export async function deleteNodeById(identifier: string): Promise<void> {
  await run(async () => {
    applyView(await api.deleteNode(identifier))
    selectedNodeIds.update((ids) => ids.filter((id) => id !== identifier))
    modalNodeId.set(null)
    editNodeId.set(null)
    editModalOpen.set(false)
  })
}

// undo reverts the most recent change to the open project. applyView already drops
// any selection or open modal whose node the restore removed, so nothing else needs
// clearing here. With nothing to undo the backend returns the view unchanged.
export async function undo(): Promise<void> {
  await run(async () => {
    applyView(await api.undo())
  })
}

// redo re-applies the most recently undone change to the open project.
export async function redo(): Promise<void> {
  await run(async () => {
    applyView(await api.redo())
  })
}

// saveSettings persists settings and updates the in-memory copy.
export async function saveSettings(next: Settings): Promise<void> {
  await run(async () => {
    await api.saveSettings(next)
    settings.set(await api.getSettings())
  })
}

// exportProjectFile prompts for a path and writes a JSON backup of the open project.
export async function exportProjectFile(): Promise<void> {
  const openView = get(view)
  if (!openView) {
    return
  }
  await run(async () => {
    const savedPath = await api.exportProject(openView.project.id)
    // An empty path means the user cancelled the save dialog, which is not a success.
    if (savedPath) {
      showSuccess('Backup saved')
    }
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
