<script lang="ts">
  import { get } from 'svelte/store'
  import CodeMirror from './CodeMirror.svelte'
  import EmojiButton from './EmojiButton.svelte'
  import { registerEditorCommands } from './editorCommands'
  import {
    closeEditor,
    createEndpointTask,
    editorAnchor,
    editorMode,
    editorOpen,
    saveSelected,
    selectedNodeId,
    view
  } from './store'
  import { decisionTypeOptions } from './types'
  import type { NodeView } from './api'

  // Local field state for the node being composed. Every field is optional so the
  // user is never blocked by a validation error.
  let title = $state('')
  let body = $state('')
  let icon = $state('')
  let decisionType = $state('plain')

  let titleInput: HTMLInputElement | undefined = $state()
  let codeMirror: ReturnType<typeof CodeMirror> | undefined = $state()

  // Popup geometry: position is the top-left in viewport pixels; the popup spawns at
  // the click anchor and can then be dragged by its header.
  const popupWidth = 560
  const popupHeight = 360
  let position = $state({ x: 0, y: 0 })
  let dragging = $state(false)
  let dragOffset = { x: 0, y: 0 }

  // clampPosition keeps the popup fully on screen.
  function clampPosition(x: number, y: number): { x: number; y: number } {
    const maxX = window.innerWidth - popupWidth - 8
    const maxY = window.innerHeight - popupHeight - 8
    return {
      x: Math.min(Math.max(8, x), Math.max(8, maxX)),
      y: Math.min(Math.max(36, y), Math.max(36, maxY))
    }
  }

  // Spawn the popup near the click anchor each time it opens, offset slightly so the
  // header sits under the cursor.
  $effect(() => {
    if ($editorOpen) {
      const anchor = get(editorAnchor)
      position = clampPosition(anchor.x - 40, anchor.y - 16)
    }
  })

  // startDrag begins a header drag unless the press landed on a button.
  function startDrag(event: PointerEvent): void {
    if ((event.target as HTMLElement).closest('button')) {
      return
    }
    dragging = true
    dragOffset = { x: event.clientX - position.x, y: event.clientY - position.y }
  }

  // onDrag moves the popup while a drag is in progress.
  function onDrag(event: PointerEvent): void {
    if (dragging) {
      position = clampPosition(event.clientX - dragOffset.x, event.clientY - dragOffset.y)
    }
  }

  // endDrag finishes a drag.
  function endDrag(): void {
    dragging = false
  }

  // selectedNode resolves the currently selected node from the open view.
  let selectedNode = $derived<NodeView | undefined>(
    $view?.nodes.find((node) => node.id === $selectedNodeId)
  )

  // hasSelection reports whether the popup is acting on an existing node (compose a
  // precursor or decision) rather than creating a new endpoint task.
  let hasSelection = $derived($selectedNodeId !== null && selectedNode !== undefined)

  // composeHeading is the popup's single heading. It names the target node so the
  // separate context subheading is no longer needed, e.g. "New Precursor for Ship".
  let composeHeading = $derived.by(() => {
    if (!hasSelection) {
      return 'New Task'
    }
    const target = selectedNode?.title || 'Untitled'
    if ($editorMode === 'decision') {
      return selectedNode?.kind === 'decision'
        ? `New Decision after ${target}`
        : `New Decision for ${target}`
    }
    return `New Precursor for ${target}`
  })

  // Start each popup session from empty fields and focused input: the effect reruns
  // when the popup opens or the selected node changes, but not while typing.
  $effect(() => {
    if ($editorOpen) {
      void $selectedNodeId
      clearFields()
      focusComposer()
    }
  })

  // focusComposer places keyboard focus on the title field after the DOM settles.
  function focusComposer(): void {
    queueMicrotask(() => titleInput?.focus())
  }

  // clearFields empties the composer so the next item can be logged immediately.
  function clearFields(): void {
    title = ''
    body = ''
    icon = ''
  }

  // Expose saving to the shortcut dispatcher while the popup is open, so cmd+s and
  // the menu's Save item commit the composer from any of its fields.
  $effect(() => {
    if ($editorOpen) {
      return registerEditorCommands({ save: () => void save() })
    }
  })

  // handleTitleKeydown maps Enter to entering the body, so logging stays on the
  // keyboard; saving is handled centrally by the shortcut dispatcher.
  function handleTitleKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter') {
      event.preventDefault()
      codeMirror?.focus()
    }
  }

  // save commits the composer according to the current mode and selection, then
  // hides the popup once the node, precursor, or decision has been added.
  async function save(): Promise<void> {
    if (!hasSelection) {
      const created = await createEndpointTask(title, body, icon)
      if (created) {
        closeEditor()
      }
      return
    }
    const added = await saveSelected($editorMode, title, body, icon, decisionType)
    if (added) {
      closeEditor()
    }
  }
</script>

<svelte:window onpointermove={onDrag} onpointerup={endDrag} />

{#if $editorOpen}
  <div class="editor" style={`left:${position.x}px; top:${position.y}px`}>
    <!-- The header is the popup's mouse drag handle; the popup needs no keyboard
         positioning, so the handle is presentational to assistive technology. -->
    <div class="header" role="presentation" onpointerdown={startDrag}>
      <h2>{composeHeading}</h2>
    </div>

    <div class="compose">
      <div class="row">
          <EmojiButton bind:value={icon} />
          <input
            type="text"
            class="title-input"
            placeholder="Title"
            bind:value={title}
            bind:this={titleInput}
            onkeydown={handleTitleKeydown}
          />
        </div>
        {#if hasSelection && $editorMode === 'decision'}
          <div class="decision-types">
            {#each decisionTypeOptions as option}
              <button
                type="button"
                class="decision-type-btn"
                class:active={decisionType === option.value}
                title={option.label}
                onclick={() => (decisionType = option.value)}
              >
                <span class="decision-type-icon">{option.icon}</span>
                <span class="decision-type-label">{option.label}</span>
              </button>
            {/each}
          </div>
        {/if}
      <div class="body">
        <CodeMirror bind:value={body} placeholder="Markdown description" onSave={save} bind:this={codeMirror} />
      </div>
      <button type="button" class="primary submit" onclick={save}>
        {#if !hasSelection}Create{:else}Add{/if}
      </button>
    </div>
  </div>
{/if}

<style>
  .editor {
    position: fixed;
    z-index: 75;
    width: 560px;
    max-width: 90vw;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 18px;
    /* The composer wears the same material as every modal so the two read as one
       surface. It keeps its shadow because, unlike a modal, it has no backdrop
       behind it and would otherwise look painted onto the canvas. */
    background-color: var(--surface-panel);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border-panel);
    border-radius: 10px;
    box-shadow: 0 18px 44px rgba(0, 0, 0, 0.55);
  }

  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    cursor: move;
  }

  h2 {
    margin: 0;
    font-size: 18px;
  }

  .compose {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .title-input {
    flex: 1;
  }

  .decision-types {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .decision-type-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    font-size: 13px;
    opacity: 0.55;
  }

  .decision-type-icon {
    font-size: 16px;
  }

  .decision-type-btn.active {
    opacity: 1;
    border-color: var(--accent);
    background-color: color-mix(in srgb, var(--accent) 22%, var(--surface-raised));
  }

  .body {
    height: 220px;
  }

  /* The submit button sits at the very bottom, spanning the popup as a prominent,
     taller call to action now that the inline button is gone. */
  .submit {
    width: 100%;
    padding: 12px;
    font-weight: 600;
  }
</style>
