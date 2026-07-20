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
  import { overlayOpen, runShortcut, shortcutForEvent, type ShortcutId } from './lib/shortcuts'
  import { EventsOn } from '../wailsjs/runtime/runtime'
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
    dismissToast,
    loadInitial,
    resolveConfirm,
    selectedNodeIds,
    showSettings,
    toast,
    view
  } from './lib/store'

  // Load projects and settings once the component is mounted, and subscribe to the
  // native menu bar so its items run the same actions as the keyboard shortcuts.
  onMount(() => {
    void loadInitial()
    return EventsOn('menu:action', (identifier: ShortcutId) => runShortcut(identifier))
  })

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

  // inSidebar reports whether the keystroke originated from the sidebar, used to
  // route Backspace to project deletion rather than canvas node deletion.
  function inSidebar(target: EventTarget | null): boolean {
    const element = target as HTMLElement | null
    return element !== null && element.closest('.sidebar') !== null
  }

  // onKeydown handles the two keystrokes that depend on where focus sits, then hands
  // everything else to the shared shortcut dispatcher, which the native menu bar also
  // drives. Escape cascades through the open overlays, and bare Backspace deletes the
  // canvas selection or, when focus is in the sidebar, the open project; both stay
  // here because they must never be registered as menu accelerators.
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
    // Bare Backspace still deletes, as an unlisted convenience alongside the menu's
    // cmd+Backspace. It stays out of the dispatcher because it needs the focus target
    // to choose between deleting the open project and the canvas selection.
    if ((event.key === 'Backspace' || event.key === 'Delete') && !(event.metaKey || event.ctrlKey)) {
      if (isTyping(event.target) || overlayOpen() || get(view) === null) {
        return
      }
      if (inSidebar(event.target)) {
        event.preventDefault()
        void confirmAndDeleteActiveProject()
      } else if (get(selectedNodeIds).length > 0) {
        event.preventDefault()
        void confirmAndDeleteSelected()
      }
      return
    }
    const identifier = shortcutForEvent(event)
    if (!identifier) {
      return
    }
    // Save and find deliberately skip the isTyping guard: both are aimed at the field
    // the user is already typing in. Find always suppresses the webview's native find,
    // and pressing it inside the search input reselects the query text.
    const worksWhileTyping = identifier === 'editor.save' || identifier === 'view.find'
    if (isTyping(event.target) && !worksWhileTyping) {
      return
    }
    event.preventDefault()
    runShortcut(identifier)
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
