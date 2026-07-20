<script lang="ts">
  import CodeMirror from './CodeMirror.svelte'
  import EmojiButton from './EmojiButton.svelte'
  import {
    closeEditModal,
    deleteNodeById,
    editModalOpen,
    editNodeId,
    requestConfirm,
    saveNodeEdit,
    view
  } from './store'
  import type { NodeView } from './api'

  // node resolves the node being edited from the open view.
  let node = $derived<NodeView | undefined>(
    $view?.nodes.find((candidate) => candidate.id === $editNodeId)
  )

  // Editable field copies, seeded from the node whenever the editor modal opens.
  let title = $state('')
  let body = $state('')
  let icon = $state('')

  // Seed the fields from the node each time the modal becomes visible.
  $effect(() => {
    if ($editModalOpen && node) {
      title = node.title
      body = node.bodyMarkdown
      icon = node.icon
    }
  })

  // save persists the edited fields and closes the editor modal.
  async function save(): Promise<void> {
    if (node) {
      await saveNodeEdit(node.id, title, body, icon)
    }
  }

  // handleKeydown lets cmd+s (or ctrl+s) save from anywhere in the modal, not just
  // while the markdown editor holds focus, so editing a node stays keyboard-driven.
  function handleKeydown(event: KeyboardEvent): void {
    if ($editModalOpen && (event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
      event.preventDefault()
      void save()
    }
  }

  // remove deletes the node after an in-app confirmation prompt.
  async function remove(): Promise<void> {
    if (node && (await requestConfirm('Delete this node? Its chain will be healed.'))) {
      await deleteNodeById(node.id)
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if $editModalOpen && node}
  <!-- The backdrop dismisses on click; keyboard users close the modal with Escape,
       so it is presentational to assistive technology. -->
  <div class="overlay" role="presentation" onclick={closeEditModal}></div>
  <div class="panel">
    <h2>Edit node</h2>
    <div class="row">
      <EmojiButton bind:value={icon} />
      <input type="text" class="title-input" placeholder="Title (optional)" bind:value={title} />
    </div>
    <div class="body">
      <CodeMirror bind:value={body} placeholder="Markdown description (optional)" onSave={save} />
    </div>
    <div class="actions">
      <button type="button" class="danger" onclick={remove}>Delete</button>
      <button type="button" class="primary" onclick={save}>Save</button>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.55);
    z-index: 80;
  }

  .panel {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 90;
    width: 560px;
    max-width: 92vw;
    height: 60vh;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 18px;
    background-color: var(--surface-panel);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border-panel);
    border-radius: 10px;
  }

  h2 {
    margin: 0;
    font-size: 18px;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .title-input {
    flex: 1;
  }

  .body {
    flex: 1;
    min-height: 0;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .primary {
    margin-left: auto;
  }

  .danger:hover {
    border-color: #ef4444;
  }
</style>
