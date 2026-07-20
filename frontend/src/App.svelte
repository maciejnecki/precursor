<script lang="ts">
  import { onMount } from 'svelte'
  import { get } from 'svelte/store'
  import Sidebar from './lib/Sidebar.svelte'
  import Canvas from './lib/Canvas.svelte'
  import Editor from './lib/Editor.svelte'
  import Settings from './lib/Settings.svelte'
  import NodeModal from './lib/NodeModal.svelte'
  import ProjectDetailModal from './lib/ProjectDetailModal.svelte'
  import NodeEditModal from './lib/NodeEditModal.svelte'
  import ProjectModal from './lib/ProjectModal.svelte'
  import ConfirmDialog from './lib/ConfirmDialog.svelte'
  import { canvasCommands } from './lib/canvasCommands'
  import {
    closeEditModal,
    closeEditor,
    closeNodeModal,
    closeProjectDetail,
    closeProjectModal,
    closeSearch,
    confirmAndDeleteActiveProject,
    confirmAndDeleteSelected,
    confirmRequest,
    createProximityGroup,
    dismissToast,
    editModalOpen,
    editorOpen,
    loadInitial,
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
    resolveConfirm,
    selectedNodeId,
    selectedNodeIds,
    showSettings,
    toast,
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
      get(projectDetailOpen) ||
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
  // "h" recentres on the first endpoint and "shift+h" zooms to fit; cmd+f opens the
  // canvas search bar (Enter/Shift+Enter cycle matches, Escape closes it); Backspace
  // deletes the canvas selection or, when focus is in the sidebar, the open project;
  // Escape closes the popup and the project modal.
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
      closeProjectDetail()
      closeEditModal()
      closeSearch()
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
    // cmd+f opens (or refocuses) the canvas search bar. The webview's native find is
    // always suppressed, and the isTyping guard is deliberately skipped so pressing
    // cmd+f inside the search input reselects the query text.
    if (isCommand && event.key.toLowerCase() === 'f') {
      event.preventDefault()
      if (!overlayOpen() && get(view)) {
        openSearch()
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
    // cmd+i opens the detail modal for the single selected node, or the open
    // project's own detail modal when the canvas selection is empty, since the
    // project card is not selectable.
    if (isCommand && event.key.toLowerCase() === 'i') {
      const selected = singleSelectedNode()
      event.preventDefault()
      if (selected) {
        openNodeModal(selected.id)
      } else if (get(selectedNodeIds).length === 0) {
        openProjectDetail()
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
    } else if (key === 'h') {
      // "h" mirrors the Home button and "shift+h" the Fit button, driven through the
      // canvas command registry because this handler sits outside the flow context.
      event.preventDefault()
      const commands = canvasCommands()
      if (event.shiftKey) {
        commands?.fitAll()
      } else {
        commands?.home()
      }
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
<ProjectDetailModal />
<NodeEditModal />
<ProjectModal />
<ConfirmDialog />

{#if $toast}
  <button type="button" class="toast" class:success={$toast.kind === 'success'} onclick={dismissToast}>
    {$toast.text}
  </button>
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

  /* The toast is a button so dismissing it works from the keyboard too; it keeps
     the body's font and colour rather than the browser's button defaults. */
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
    font: inherit;
    color: inherit;
    text-align: left;
  }

  /* Success confirmations reuse the toast shape in the done colour family. */
  .toast.success {
    background-color: #14532d;
    border-color: #22c55e;
  }
</style>
