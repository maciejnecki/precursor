<script lang="ts">
  import { closeNodeModal, modalNodeId, view } from './store'
  import { renderMarkdown } from './markdown'
  import type { NodeView } from './api'

  // node resolves the node currently shown in the detail modal from the open view.
  let node = $derived<NodeView | undefined>(
    $view?.nodes.find((candidate) => candidate.id === $modalNodeId)
  )

  // body is the node's description rendered to HTML for display.
  let body = $derived(node ? renderMarkdown(node.bodyMarkdown) : '')
</script>

{#if node}
  <div class="overlay" onclick={closeNodeModal}></div>
  <div class="panel">
    <header>
      <h2>
        {#if node.icon}<span class="icon">{node.icon}</span>{/if}
        {node.title || 'Untitled'}
      </h2>
    </header>
    <div class="content">
      {#if body}
        <div class="markdown">{@html body}</div>
      {:else}
        <p class="empty">No description.</p>
      {/if}
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.55);
    z-index: 60;
  }

  .panel {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 70;
    width: 520px;
    max-width: 90vw;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    background-color: var(--surface-raised);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
  }

  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 8px;
    padding: 16px 18px;
    border-bottom: 1px solid var(--border);
  }

  h2 {
    margin: 0;
    font-size: 18px;
  }

  .icon {
    margin-right: 6px;
  }

  .content {
    padding: 16px 18px;
    overflow-y: auto;
  }

  .empty {
    color: var(--text-muted);
    margin: 0;
  }
</style>
