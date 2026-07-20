<script lang="ts">
  import { openProjectDetail } from '../store'
  import { renderMarkdown } from '../markdown'
  import ProgressBar from '../ProgressBar.svelte'

  // The project card is a non-interactive summary of the open project, drawn to the
  // left of the first chain: its icon and name, its rendered description, and the
  // share of endpoints completed. Its height is fixed, so a long description scrolls
  // within the card; double-clicking opens the detail modal with the full text, the
  // way a task card opens its own.
  let { data }: {
    data: {
      name: string
      icon: string
      description: string
      percent: number
      width: number
      height: number
      doneColour: string
    }
  } = $props()

  // descriptionFor caches the last rendered description so an unchanged one is not
  // re-parsed when a view refresh replaces the surrounding data object.
  let cachedDescription: string | null = null
  let cachedRender = ''
  function descriptionFor(description: string): string {
    if (description !== cachedDescription) {
      cachedDescription = description
      cachedRender = renderMarkdown(description)
    }
    return cachedRender
  }

  // body is the project description rendered to HTML.
  let body = $derived(descriptionFor(data.description))
</script>

<div
  class="project-card"
  style={`width:${data.width}px; height:${data.height}px`}
  ondblclick={openProjectDetail}
  role="button"
  tabindex="-1"
>
  <div class="header">
    {#if data.icon}<span class="icon">{data.icon}</span>{/if}
    <span class="name">{data.name || 'Untitled'}</span>
  </div>
  <div class="body">
    {#if body}
      <div class="markdown">{@html body}</div>
    {:else}
      <p class="empty">No description.</p>
    {/if}
  </div>
  <div class="footer">
    <ProgressBar percent={data.percent} colour={data.doneColour} />
  </div>
</div>

<style>
  /* The fill matches the chain background panels rather than a task card, so the
     project reads as the canvas's backdrop element; only the outline keeps it as a
     distinct card. */
  .project-card {
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    border-radius: 18px;
    background-color: var(--surface-panel);
    border: 1.5px solid var(--border-panel);
    color: var(--text);
    text-align: left;
    font-size: 17px;
    overflow: hidden;
  }

  .header {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
    padding: 20px 24px;
    border-bottom: 1px solid var(--border-panel);
  }

  .icon {
    font-size: 30px;
  }

  .name {
    flex: 1;
    font-size: 26px;
    font-weight: 600;
    color: #f5f7fa;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* The description takes the height left between the fixed header and footer and
     scrolls within it, so the card's footprint never depends on how much was typed. */
  .body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 18px 24px;
    font-size: 16px;
    line-height: 1.55;
    color: var(--text-muted);
  }

  .empty {
    margin: 0;
    color: var(--text-muted);
  }

  .footer {
    display: flex;
    align-items: center;
    flex-shrink: 0;
    padding: 18px 24px;
    border-top: 1px solid var(--border-panel);
  }
</style>
