<script lang="ts">
  import {
    confirmAndDeleteProject,
    openProject,
    openProjectModal,
    projects,
    showSettings,
    sidebarCollapsed,
    view
  } from './store'

  // toggleCollapse switches the sidebar between full and narrow icon-rail widths.
  function toggleCollapse(): void {
    sidebarCollapsed.update((collapsed) => !collapsed)
  }

  // selectProject opens a project and forces keyboard focus onto its button. macOS
  // WebKit does not focus a button on click, so this is what lets the Backspace
  // shortcut know the sidebar (not the canvas) holds the selection.
  function selectProject(identifier: string, event: MouseEvent): void {
    ;(event.currentTarget as HTMLButtonElement).focus()
    void openProject(identifier)
  }

  // activeId is the id of the currently open project, for highlighting.
  let activeId = $derived($view?.project.id ?? null)

  // contextMenu holds the right-click menu's screen position and the project it acts
  // on, or null when no menu is open.
  let contextMenu = $state<{ x: number; y: number; id: string; name: string } | null>(null)

  // The context menu's footprint, used to keep it on screen near the cursor.
  const contextMenuWidth = 170
  const contextMenuHeight = 44

  // openContextMenu shows the project context menu at the cursor, clamped on screen.
  function openContextMenu(event: MouseEvent, id: string, name: string): void {
    event.preventDefault()
    const x = Math.min(event.clientX, window.innerWidth - contextMenuWidth - 8)
    const y = Math.min(event.clientY, window.innerHeight - contextMenuHeight - 8)
    contextMenu = { x, y, id, name }
  }

  // closeContextMenu hides the project context menu.
  function closeContextMenu(): void {
    contextMenu = null
  }

  // deleteFromContextMenu confirms and deletes the project the menu was opened on.
  async function deleteFromContextMenu(): Promise<void> {
    if (contextMenu) {
      const target = contextMenu
      closeContextMenu()
      await confirmAndDeleteProject(target.id, target.name)
    }
  }
</script>

<aside class="sidebar" class:collapsed={$sidebarCollapsed}>
  <header>
    <button type="button" class="collapse" onclick={toggleCollapse} title={$sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
      {$sidebarCollapsed ? '▶' : '◀'}
    </button>
    {#if !$sidebarCollapsed}
      <button type="button" class="add" onclick={openProjectModal} title="New project">+</button>
    {/if}
  </header>

  <div class="projects">
    {#each $projects as project}
      <div class="project" class:active={project.id === activeId}>
        <button
          type="button"
          class="open"
          onclick={(event) => selectProject(project.id, event)}
          oncontextmenu={(event) => openContextMenu(event, project.id, project.name || 'Untitled')}
          title={project.name || 'Untitled'}
        >
          <span class="icon">{project.icon || '📁'}</span>
          {#if !$sidebarCollapsed}
            <span class="name">{project.name || 'Untitled'}</span>
          {/if}
        </button>
      </div>
    {/each}
    {#if $projects.length === 0 && !$sidebarCollapsed}
      <p class="empty">No projects yet.</p>
    {/if}
  </div>

  {#if !$sidebarCollapsed}
    <footer>
      <button type="button" class="settings" onclick={() => showSettings.set(true)} title="Settings">
        Settings
      </button>
    </footer>
  {/if}
</aside>

<!-- Right-click menu for a project. A full-screen backdrop dismisses it on any click
     or further right-click outside the menu. -->
{#if contextMenu}
  <div
    class="context-backdrop"
    onclick={closeContextMenu}
    oncontextmenu={(event) => {
      event.preventDefault()
      closeContextMenu()
    }}
  ></div>
  <div class="context-menu" style={`left:${contextMenu.x}px; top:${contextMenu.y}px`}>
    <button type="button" class="context-item danger" onclick={deleteFromContextMenu}>Delete project</button>
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
    width: 60px;
    padding: 12px 8px;
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

  .add {
    /* Pushed to the right edge of the header now that the title has moved out. */
    margin-left: auto;
    font-size: 18px;
    line-height: 1;
    padding: 2px 10px;
  }

  .projects {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .project {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .project.active .open {
    border-color: var(--accent);
  }

  .open {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    text-align: left;
    overflow: hidden;
  }

  .sidebar.collapsed .open {
    justify-content: center;
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty {
    color: var(--text-muted);
    font-size: 13px;
  }

  footer {
    border-top: 1px solid var(--border);
    padding-top: 8px;
  }

  .settings {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }

  .context-backdrop {
    position: fixed;
    inset: 0;
    z-index: 120;
  }

  .context-menu {
    position: fixed;
    z-index: 121;
    min-width: 150px;
    display: flex;
    flex-direction: column;
    padding: 4px;
    background-color: var(--surface-raised);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
  }

  .context-item {
    width: 100%;
    text-align: left;
    background-color: transparent;
    border-color: transparent;
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
