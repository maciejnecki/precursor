<script lang="ts">
  import {
    confirmAndDeleteProject,
    openProject,
    openProjectEditModal,
    openProjectModal,
    projectGroups,
    projects,
    saveSidebar,
    showSettings,
    sidebarCollapsed,
    view
  } from './store'
  import {
    bandsFor,
    groupIdOf,
    placeAtEnd,
    placeAtGroupHead,
    placeAtProject,
    withGroupChanged,
    withGroupCreated,
    withGroupDissolved,
    withGroupMoved,
    type Layout
  } from './sidebarLayout'

  // toggleCollapse switches the sidebar between full and narrow icon-rail widths.
  function toggleCollapse(): void {
    sidebarCollapsed.update((collapsed) => !collapsed)
  }

  // activeId is the id of the currently open project, for highlighting.
  let activeId = $derived($view?.project.id ?? null)

  // selectedIds holds the projects the grouping action acts on. Opening a project
  // makes it the selection, and shift-clicking extends it from there.
  let selectedIds = $state<string[]>([])

  // selectProject opens a project and forces keyboard focus onto its button. macOS
  // WebKit does not focus a button on click, so this is what lets the Backspace
  // shortcut know the sidebar (not the canvas) holds the selection. A plain click
  // makes that project the whole selection, and shift-clicking extends it without
  // opening anything, so the project you clicked first is grouped along with the
  // ones you added.
  function selectProject(identifier: string, event: MouseEvent): void {
    ;(event.currentTarget as HTMLButtonElement).focus()
    if (event.shiftKey) {
      selectedIds = selectedIds.includes(identifier)
        ? selectedIds.filter((selected) => selected !== identifier)
        : [...selectedIds, identifier]
      return
    }
    selectedIds = [identifier]
    void openProject(identifier)
  }

  // storedLayout is the persisted order and grouping; previewLayout is the version
  // shown while a drag is in flight, so the list rearranges under the pointer.
  let storedLayout = $derived<Layout>({
    order: $projects.map((project) => project.id),
    groups: $projectGroups
  })
  let previewLayout = $state<Layout | null>(null)
  let layout = $derived(previewLayout ?? storedLayout)

  // bands are the runs the list renders: each group with its members, and each
  // ungrouped project on its own.
  let bands = $derived.by(() => {
    const byId = new Map($projects.map((project) => [project.id, project]))
    const ordered = layout.order
      .map((identifier) => byId.get(identifier))
      .filter((project) => project !== undefined)
    return bandsFor(ordered, layout.groups)
  })

  // apply stores a rearranged layout, replacing any drag preview.
  function apply(next: Layout): void {
    previewLayout = null
    void saveSidebar(next.order, next.groups)
  }

  // groupSelected bands the shift-clicked projects together and starts renaming the
  // new group in place.
  function groupSelected(): void {
    const next = withGroupCreated(layout, selectedIds, 'New group')
    const created = next.groups.find((group) => !layout.groups.some((existing) => existing.id === group.id))
    selectedIds = []
    apply(next)
    if (created) {
      editingGroupId = created.id
    }
  }

  // toggleGroupCollapsed folds a group down to its header row, or opens it again.
  function toggleGroupCollapsed(groupId: string, collapsed: boolean): void {
    apply(withGroupChanged(layout, groupId, { collapsed }))
  }

  // editingGroupId is the group whose name is being edited in place, or null.
  let editingGroupId = $state<string | null>(null)

  // commitGroupName stores an edited group name, ignoring an empty one.
  function commitGroupName(groupId: string, name: string): void {
    // Escape clears the editing state first, so a blur that follows it stores nothing.
    if (editingGroupId !== groupId) {
      return
    }
    editingGroupId = null
    const trimmed = name.trim()
    if (trimmed.length > 0) {
      apply(withGroupChanged(layout, groupId, { name: trimmed }))
    }
  }

  // dragged is the row being dragged: a project or a whole group.
  let dragged = $state<{ kind: 'project' | 'group'; id: string } | null>(null)

  // startDrag begins dragging a row and seeds the preview with the stored layout.
  function startDrag(kind: 'project' | 'group', identifier: string, event: DragEvent): void {
    dragged = { kind, id: identifier }
    previewLayout = storedLayout
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move'
      event.dataTransfer.setData('text/plain', identifier)
    }
  }

  // dragOverProject previews the dragged row landing on a project's row: a dragged
  // project takes that row's place and its group, and a dragged group moves its whole
  // block there. A group cannot land on a row inside another group, since a group
  // never nests inside one.
  function dragOverProject(targetId: string, event: DragEvent): void {
    event.preventDefault()
    if (!dragged) {
      return
    }
    if (dragged.kind === 'group') {
      if (groupIdOf(layout.groups, targetId) === null) {
        previewLayout = withGroupMoved(layout, dragged.id, targetId)
      }
      return
    }
    if (dragged.id !== targetId) {
      previewLayout = placeAtProject(layout, dragged.id, targetId)
    }
  }

  // dragOverGroupHeader previews a dragged project joining that group at its top, or
  // a dragged group taking the hovered group's place.
  function dragOverGroupHeader(groupId: string, event: DragEvent): void {
    event.preventDefault()
    if (!dragged) {
      return
    }
    if (dragged.kind === 'group') {
      const target = layout.groups.find((group) => group.id === groupId)
      if (target && target.id !== dragged.id) {
        previewLayout = withGroupMoved(layout, dragged.id, target.members[0])
      }
      return
    }
    previewLayout = placeAtGroupHead(layout, dragged.id, groupId)
  }

  // dragOverEnd previews the dragged row landing below every other one, which is how
  // a project leaves a group without joining another.
  function dragOverEnd(event: DragEvent): void {
    event.preventDefault()
    if (!dragged) {
      return
    }
    previewLayout =
      dragged.kind === 'group' ? withGroupMoved(layout, dragged.id, null) : placeAtEnd(layout, dragged.id)
  }

  // endDrag persists the previewed layout when it differs from the stored one, and
  // clears the drag state either way.
  function endDrag(): void {
    const previewed = previewLayout
    dragged = null
    previewLayout = null
    if (previewed && !sameLayout(previewed, storedLayout)) {
      void saveSidebar(previewed.order, previewed.groups)
    }
  }

  // sameLayout reports whether two layouts would store identically, so a drag that
  // ends where it started writes nothing.
  function sameLayout(first: Layout, second: Layout): boolean {
    const signature = (candidate: Layout) =>
      `${candidate.order.join()}|${candidate.groups.map((group) => `${group.id}:${group.members.join()}`).join(';')}`
    return signature(first) === signature(second)
  }

  // contextMenu holds the right-click menu's screen position and the row it acts on,
  // or null when no menu is open.
  let contextMenu = $state<{ x: number; y: number; kind: 'project' | 'group'; id: string; name: string } | null>(
    null
  )

  // The context menu's footprint, used to keep it on screen near the cursor.
  const contextMenuWidth = 170
  const contextMenuHeight = 84

  // openContextMenu shows the row's context menu at the cursor, clamped on screen.
  function openContextMenu(event: MouseEvent, kind: 'project' | 'group', id: string, name: string): void {
    event.preventDefault()
    const x = Math.min(event.clientX, window.innerWidth - contextMenuWidth - 8)
    const y = Math.min(event.clientY, window.innerHeight - contextMenuHeight - 8)
    contextMenu = { x, y, kind, id, name }
  }

  // closeContextMenu hides the context menu.
  function closeContextMenu(): void {
    contextMenu = null
  }

  // editFromContextMenu opens the project modal in edit mode for the project the menu
  // was opened on. The project is edited in place, so whichever project is currently
  // open on the canvas stays open.
  function editFromContextMenu(): void {
    if (contextMenu) {
      const target = contextMenu
      closeContextMenu()
      openProjectEditModal(target.id)
    }
  }

  // deleteFromContextMenu confirms and deletes the project the menu was opened on.
  async function deleteFromContextMenu(): Promise<void> {
    if (contextMenu) {
      const target = contextMenu
      closeContextMenu()
      await confirmAndDeleteProject(target.id, target.name)
    }
  }

  // renameFromContextMenu starts editing the group's name in place.
  function renameFromContextMenu(): void {
    if (contextMenu) {
      const target = contextMenu
      closeContextMenu()
      editingGroupId = target.id
    }
  }

  // ungroupFromContextMenu dissolves the group, leaving its projects where they are.
  function ungroupFromContextMenu(): void {
    if (contextMenu) {
      const target = contextMenu
      closeContextMenu()
      apply(withGroupDissolved(layout, target.id))
    }
  }

  // onEscapeCapture closes an open context menu on Escape during the capture phase
  // and stops the event, so the global Escape handler does not also act.
  function onEscapeCapture(event: KeyboardEvent): void {
    if (contextMenu && event.key === 'Escape') {
      event.stopPropagation()
      closeContextMenu()
    }
  }
</script>

<svelte:window onkeydowncapture={onEscapeCapture} />

<aside class="sidebar" class:collapsed={$sidebarCollapsed}>
  <!-- An empty strip that both drags the window and keeps the controls below clear of
       the native traffic lights, which render over this corner. -->
  <div class="drag-strip" style="--wails-draggable:drag"></div>

  <header>
    <!-- Grouping needs at least two projects, so the button only appears once a
         shift-click has extended the selection past the open project. -->
    {#if !$sidebarCollapsed && selectedIds.length > 1}
      <button type="button" class="group-action" onclick={groupSelected} title="Group selected projects">
        Group {selectedIds.length}
      </button>
    {/if}
    <button type="button" class="add" onclick={openProjectModal} title="New project">+</button>
  </header>

  <div class="projects">
    {#each bands as band (band.group ? band.group.id : band.projects[0].id)}
      {#if band.group}
        <!-- A group draws its members on a delicate panel, the same treatment chain
             groups get on the canvas. -->
        <div class="group">
          <div
            class="group-header"
            role="presentation"
            draggable={editingGroupId !== band.group.id}
            ondragstart={(event) => band.group && startDrag('group', band.group.id, event)}
            ondragover={(event) => band.group && dragOverGroupHeader(band.group.id, event)}
            ondrop={(event) => event.preventDefault()}
            ondragend={endDrag}
            oncontextmenu={(event) => band.group && openContextMenu(event, 'group', band.group.id, band.group.name)}
          >
            <button
              type="button"
              class="caret"
              onclick={() => band.group && toggleGroupCollapsed(band.group.id, !band.group.collapsed)}
              title={band.group.collapsed ? 'Expand group' : 'Collapse group'}
            >
              {band.group.collapsed ? '▶' : '▼'}
            </button>
            {#if !$sidebarCollapsed}
              {#if editingGroupId === band.group.id}
                <!-- svelte-ignore a11y_autofocus -->
                <input
                  class="group-name-input"
                  autofocus
                  value={band.group.name}
                  onblur={(event) => band.group && commitGroupName(band.group.id, event.currentTarget.value)}
                  onkeydown={(event) => {
                    if (event.key === 'Enter') {
                      event.currentTarget.blur()
                    }
                    if (event.key === 'Escape') {
                      event.stopPropagation()
                      editingGroupId = null
                    }
                  }}
                />
              {:else}
                <span class="group-name">{band.group.name}</span>
                <span class="group-count">{band.projects.length}</span>
              {/if}
            {/if}
          </div>

          {#if !band.group.collapsed}
            <div class="group-members">
              {#each band.projects as project (project.id)}
                {@render projectRow(project)}
              {/each}
            </div>
          {/if}
        </div>
      {:else}
        {@render projectRow(band.projects[0])}
      {/if}
    {/each}

    <!-- Dropping below the list moves a row to the end and out of any group; it is
         also how the last project leaves a group. -->
    <div
      class="drop-end"
      class:active={dragged !== null}
      role="presentation"
      ondragover={dragOverEnd}
      ondrop={(event) => event.preventDefault()}
    ></div>

    {#if $projects.length === 0 && !$sidebarCollapsed}
      <p class="empty">No projects yet.</p>
    {/if}
  </div>

  <!-- The footer carries the two chrome controls, sat where the account row lives in
       a native sidebar: Settings takes the width, Collapse tucks beside it. -->
  <footer>
    <button type="button" class="settings" onclick={() => showSettings.set(true)} title="Settings">
      <span class="foot-icon">⚙</span>
      {#if !$sidebarCollapsed}
        <span>Settings</span>
      {/if}
    </button>
    <button type="button" class="collapse" onclick={toggleCollapse} title={$sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
      {$sidebarCollapsed ? '▶' : '◀'}
    </button>
  </footer>
</aside>

<!-- One project row, used both at the top level and inside a group. -->
{#snippet projectRow(project: import('./api').Project)}
  <div
    class="project"
    class:active={project.id === activeId}
    class:picked={selectedIds.length > 1 && selectedIds.includes(project.id)}
    class:dragging={dragged?.kind === 'project' && dragged.id === project.id}
    draggable="true"
    role="presentation"
    ondragstart={(event) => startDrag('project', project.id, event)}
    ondragover={(event) => dragOverProject(project.id, event)}
    ondrop={(event) => event.preventDefault()}
    ondragend={endDrag}
  >
    <button
      type="button"
      class="open"
      onclick={(event) => selectProject(project.id, event)}
      oncontextmenu={(event) => openContextMenu(event, 'project', project.id, project.name || 'Untitled')}
      title={project.name || 'Untitled'}
    >
      <span class="icon">{project.icon || '📁'}</span>
      {#if !$sidebarCollapsed}
        <span class="name">{project.name || 'Untitled'}</span>
      {/if}
    </button>
  </div>
{/snippet}

<!-- Right-click menu for a project or a group. A full-screen backdrop dismisses it on
     any click or further right-click outside the menu. -->
{#if contextMenu}
  <div
    class="context-backdrop"
    role="presentation"
    onclick={closeContextMenu}
    oncontextmenu={(event) => {
      event.preventDefault()
      closeContextMenu()
    }}
  ></div>
  <div class="context-menu" style={`left:${contextMenu.x}px; top:${contextMenu.y}px`}>
    {#if contextMenu.kind === 'group'}
      <button type="button" class="context-item" onclick={renameFromContextMenu}>Rename group</button>
      <div class="context-separator"></div>
      <button type="button" class="context-item" onclick={ungroupFromContextMenu}>Ungroup</button>
    {:else}
      <button type="button" class="context-item" onclick={editFromContextMenu}>Edit project</button>
      <div class="context-separator"></div>
      <button type="button" class="context-item danger" onclick={deleteFromContextMenu}>Delete project</button>
    {/if}
  </div>
{/if}

<style>
  .sidebar {
    width: 260px;
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
    /* Translucent tint with a blur, the way native macOS sidebars look. */
    background-color: var(--slate-veil);
    backdrop-filter: blur(24px) saturate(1.3);
    transition: width 0.12s ease;
  }

  .sidebar.collapsed {
    /* Kept at least as wide as the native traffic lights so the rightmost button never
       spills onto the canvas beneath them. */
    width: 78px;
    padding: 12px 8px;
  }

  /* Clears the traffic lights and gives a comfortable grab area for moving the window. */
  .drag-strip {
    height: 22px;
    flex: none;
  }

  header {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .sidebar.collapsed header {
    flex-direction: column;
  }

  .collapse {
    padding: 4px 8px;
  }

  .group-action {
    /* Sits beside the add button, pushed to the right with it. */
    margin-left: auto;
    font-size: 12px;
    padding: 4px 8px;
  }

  .add {
    /* Pushed to the right edge of the header now that the title has moved out. */
    margin-left: auto;
    font-size: 18px;
    line-height: 1;
    padding: 2px 10px;
  }

  .sidebar.collapsed .add {
    /* Centred in the rail so it lines up with the stacked footer controls. */
    margin: 0 auto;
  }

  /* With a group button present, only the first of the two needs the auto margin. */
  .group-action ~ .add {
    margin-left: 0;
  }

  .projects {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  /* The same delicate panel the canvas draws behind a chain's nodes. */
  .group {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 6px;
    border-radius: 12px;
    background-color: var(--surface-panel);
    border: 1px solid var(--border-panel);
  }

  .group-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 2px;
    cursor: grab;
  }

  .caret {
    background-color: transparent;
    border-color: transparent;
    color: var(--text-muted);
    font-size: 10px;
    line-height: 1;
    padding: 2px 4px;
  }

  .group-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.02em;
    color: var(--text-muted);
    text-transform: uppercase;
  }

  .group-count {
    color: var(--text-muted);
    font-size: 11px;
  }

  .group-name-input {
    flex: 1;
    min-width: 0;
    font-size: 12px;
    font-weight: 600;
    padding: 2px 6px;
  }

  .group-members {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .project {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  /* The row being dragged fades back so the gap it will land in reads clearly. */
  .project.dragging {
    opacity: 0.4;
  }

  /* The open project reads as a filled accent pill, the one loud row in the list. It
     uses the deepened accent so its white label stays legible. */
  .project.active .open,
  .project.active .open:hover {
    background-color: var(--accent-strong);
    border-color: var(--accent-strong);
    color: #ffffff;
    font-weight: 600;
  }

  /* A shift-clicked project waiting to be grouped. */
  .project.picked .open {
    background-color: rgba(255, 255, 255, 0.09);
    border-color: var(--text-muted);
  }

  .open {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    text-align: left;
    overflow: hidden;
    /* Flat rows carry no button chrome; only a faint wash on hover, the way a native
       sidebar list reads. */
    background-color: transparent;
    border-color: transparent;
  }

  .open:hover {
    background-color: rgba(255, 255, 255, 0.05);
    border-color: transparent;
  }

  /* A project sitting on its own carries the same faint wash a group panel lends its
     members, so a row reads identically whether or not it is in a group. Grouped rows
     stay transparent and take the wash from the panel behind them. */
  .projects > .project:not(.active):not(.picked) > .open {
    background-color: var(--surface-panel);
  }

  .projects > .project:not(.active):not(.picked) > .open:hover {
    background-color: rgba(255, 255, 255, 0.09);
  }

  .sidebar.collapsed .open {
    justify-content: center;
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Only takes up room during a drag, where it is the target for leaving a group. */
  .drop-end {
    height: 8px;
    border-radius: 6px;
  }

  .drop-end.active {
    flex: 1;
    min-height: 32px;
  }

  .empty {
    color: var(--text-muted);
    font-size: 13px;
  }

  footer {
    display: flex;
    gap: 6px;
    border-top: 1px solid var(--border);
    padding-top: 10px;
  }

  /* The narrow rail stacks the two controls so each stays a full-width icon. */
  .sidebar.collapsed footer {
    flex-direction: column;
  }

  .settings {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }

  .sidebar.collapsed .settings {
    flex: none;
  }

  .foot-icon {
    font-size: 18px;
    line-height: 1;
  }

  .context-backdrop {
    position: fixed;
    inset: 0;
    z-index: 120;
  }

  /* Menu chrome follows the macOS context-menu proportions: a tight 5px frame, a
     small corner radius, and a soft shadow that reads as floating rather than heavy. */
  .context-menu {
    position: fixed;
    z-index: 121;
    min-width: 170px;
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 5px;
    background-color: var(--surface-raised);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border);
    border-radius: 6px;
    box-shadow: 0 10px 28px rgba(0, 0, 0, 0.45);
  }

  .context-item {
    width: 100%;
    padding: 4px 8px;
    text-align: left;
    font-size: 13px;
    line-height: 18px;
    background-color: transparent;
    border-color: transparent;
    border-radius: 4px;
  }

  /* A hairline rule keeps the destructive item from reading as part of the group
     above it. It insets slightly so it stops short of the menu's rounded corners. */
  .context-separator {
    height: 1px;
    margin: 4px 6px;
    background-color: var(--border);
  }

  .context-item.danger {
    color: #fca5a5;
  }

  .context-item.danger:hover {
    background-color: #7f1d1d;
    border-color: #ef4444;
    color: var(--text);
  }
</style>
