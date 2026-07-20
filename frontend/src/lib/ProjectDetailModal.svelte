<script lang="ts">
  import { closeProjectDetail, projectDetailOpen, view } from './store'
  import { renderMarkdown } from './markdown'

  // project is the open project, whose full description this modal presents.
  let project = $derived($view?.project)

  // description is the project's description rendered to HTML for display.
  let description = $derived(project ? renderMarkdown(project.description) : '')
</script>

{#if $projectDetailOpen && project}
  <!-- The backdrop dismisses on click; keyboard users close the modal with Escape,
       so it is presentational to assistive technology. -->
  <div class="overlay" role="presentation" onclick={closeProjectDetail}></div>
  <div class="panel">
    <header>
      <h2>
        {#if project.icon}<span class="icon">{project.icon}</span>{/if}
        {project.name || 'Untitled'}
      </h2>
    </header>
    <div class="content">
      {#if description}
        <div class="markdown">{@html description}</div>
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
    background-color: var(--surface-panel);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border-panel);
    border-radius: 10px;
    overflow: hidden;
  }

  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 8px;
    padding: 16px 18px;
    border-bottom: 1px solid var(--border-panel);
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
