<script lang="ts">
  import { onMount } from 'svelte'
  import { get } from 'svelte/store'
  import Sidebar from './lib/Sidebar.svelte'
  import Canvas from './lib/Canvas.svelte'
  import Editor from './lib/Editor.svelte'
  import Settings from './lib/Settings.svelte'
  import NodeModal from './lib/NodeModal.svelte'
  import NodeEditModal from './lib/NodeEditModal.svelte'
  import ProjectModal from './lib/ProjectModal.svelte'
  import ConfirmDialog from './lib/ConfirmDialog.svelte'
  import {
    closeEditModal,
    closeEditor,
    closeNodeModal,
    closeProjectModal,
    confirmAndDeleteActiveProject,
    confirmAndDeleteSelected,
    confirmRequest,
    createProximityGroup,
    editModalOpen,
    editorOpen,
    errorMessage,
    loadInitial,
    modalNodeId,
    openComposeForSelection,
    openEditModal,
    openNewTaskAtCenter,
    openNodeModal,
    openProjectEditModal,
    openProjectModal,
    projectModalOpen,
    resolveConfirm,
    selectedNodeId,
    selectedNodeIds,
    showSettings,
    view
  } from './lib/store'
  import type { NodeView } from './lib/api'

  // Load projects and settings once the component is mounted.
  onMount(loadInitial)

  // singleSelectedNode returns the node when exactly one is selected, else undefined.
  function singleSelectedNode(): NodeView | undefined {
    const id = get(selectedNodeId)
    if (!id) {
      return undefined
    }
    return get(view)?.nodes.find((node) => node.id === id)
  }

  // isTyping reports whether keystrokes are going into a text field, so the canvas
  // shortcuts do not fire while the user is writing.
  function isTyping(target: EventTarget | null): boolean {
    const element = target as HTMLElement | null
    if (!element) {
      return false
    }
    return (
      element.tagName === 'INPUT' ||
      element.tagName === 'TEXTAREA' ||
      element.isContentEditable ||
      element.closest('.cm-editor') !== null
    )
  }

  // overlayOpen reports whether any popup or modal is showing, so shortcuts do not
  // act behind them.
  function overlayOpen(): boolean {
    return (
      get(editorOpen) ||
      get(modalNodeId) !== null ||
      get(editModalOpen) ||
      get(projectModalOpen) ||
      get(showSettings) ||
      get(confirmRequest) !== null
    )
  }

  // inSidebar reports whether the keystroke originated from the sidebar, used to
  // route Backspace to project deletion rather than canvas node deletion.
  function inSidebar(target: EventTarget | null): boolean {
    const element = target as HTMLElement | null
    return element !== null && element.closest('.sidebar') !== null
  }

  // onKeydown wires the canvas shortcuts. "shift+t" always creates a new endpoint
  // task; with one task selected "t" adds a precursor (inserted ahead of any existing
  // one), "e" edits it, "d" adds a decision, and cmd+i opens its detail modal; with
  // several tasks selected "p" clusters them; cmd+n opens the new-project modal;
  // Backspace deletes the canvas selection or, when focus is in the sidebar, the open
  // project; Escape closes the popup and the project modal.
  function onKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      // A pending confirmation sits on top of everything, so Escape cancels just it;
      // otherwise Escape dismisses whichever popup or modal is open.
      if (get(confirmRequest)) {
        resolveConfirm(false)
        return
      }
      closeEditor()
      closeProjectModal()
      closeNodeModal()
      closeEditModal()
      showSettings.set(false)
      return
    }
    const isCommand = event.metaKey || event.ctrlKey
    // cmd+n opens the new-project modal from anywhere except while typing or behind
    // another overlay.
    if (isCommand && event.key.toLowerCase() === 'n') {
      if (!isTyping(event.target) && !overlayOpen()) {
        event.preventDefault()
        openProjectModal()
      }
      return
    }
    if (isTyping(event.target) || overlayOpen() || !get(view)) {
      return
    }
    if (event.key === 'Backspace' || event.key === 'Delete') {
      if (inSidebar(event.target)) {
        event.preventDefault()
        void confirmAndDeleteActiveProject()
      } else if (get(selectedNodeIds).length > 0) {
        event.preventDefault()
        void confirmAndDeleteSelected()
      }
      return
    }
    // cmd+i opens the detail modal for the single selected node.
    if (isCommand && event.key.toLowerCase() === 'i') {
      const selected = singleSelectedNode()
      if (selected) {
        event.preventDefault()
        openNodeModal(selected.id)
      }
      return
    }
    // cmd+e opens the edit modal for the open project when focus is in the sidebar,
    // mirroring how Backspace there targets the project rather than the canvas.
    if (isCommand && event.key.toLowerCase() === 'e') {
      const openView = get(view)
      if (inSidebar(event.target) && openView) {
        event.preventDefault()
        openProjectEditModal(openView.project.id)
      }
      return
    }
    const key = event.key.toLowerCase()
    const node = singleSelectedNode()
    if (key === 't') {
      event.preventDefault()
      if (event.shiftKey) {
        openNewTaskAtCenter()
      } else if (node && node.kind === 'task') {
        openComposeForSelection('precursor')
      }
    } else if (key === 'e' && node) {
      event.preventDefault()
      openEditModal(node.id)
    } else if (key === 'd' && node && ((node.kind === 'task' && node.parentId) || node.kind === 'decision')) {
      event.preventDefault()
      openComposeForSelection('decision')
    } else if (key === 'p' && get(selectedNodeIds).length >= 2) {
      event.preventDefault()
      void createProximityGroup()
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<!-- A solid strip across the top acts as the window drag handle and keeps the canvas
     from reaching the very top; the native traffic lights render above it. The app
     name sits centred in it. -->
<div class="titlebar" style="--wails-draggable:drag">
  <span class="app-name">{$view ? `Precursor - ${$view.project.name || 'Untitled'}` : 'Precursor'}</span>
</div>

<div class="layout">
  <Sidebar />
  <div class="main">
    <div class="canvas-wrap">
      <Canvas />
    </div>
  </div>
</div>

<Editor />
<Settings />
<NodeModal />
<NodeEditModal />
<ProjectModal />
<ConfirmDialog />

{#if $errorMessage}
  <div class="toast" onclick={() => errorMessage.set('')}>{$errorMessage}</div>
{/if}

<style>
  .titlebar {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 28px;
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: var(--slate-veil);
    backdrop-filter: blur(24px) saturate(1.3);
  }

  .app-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
  }

  .layout {
    display: flex;
    height: calc(100vh - 28px);
    margin-top: 28px;
  }

  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .canvas-wrap {
    flex: 1;
    min-height: 0;
    /* Matches the sidebar's tint so the canvas's rounded corners reveal the same
       colour behind them instead of bare window vibrancy. */
    background-color: var(--slate-veil);
    backdrop-filter: blur(24px) saturate(1.3);
  }

  .toast {
    position: fixed;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 80;
    max-width: 70%;
    padding: 10px 14px;
    background-color: #7f1d1d;
    border: 1px solid #ef4444;
    border-radius: 8px;
    cursor: pointer;
  }
</style>
