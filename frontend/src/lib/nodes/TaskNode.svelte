<script lang="ts">
  import { Handle, Position } from '@xyflow/svelte'
  import { openNodeModal, toggleDecisions } from '../store'
  import { renderMarkdown } from '../markdown'

  // A task node renders as a dark card with a titled header, a divider, and a few
  // lines of its rendered description. Selection arrives as Svelte Flow's node-level
  // flag so the data object stays stable. Double-clicking opens the detail modal.
  let { data, selected = false }: {
    data: {
      id: string
      title: string
      body: string
      icon: string
      colour: string
      decisionCount: number
      decisionsCollapsed: boolean
    }
    selected?: boolean
  } = $props()

  // previewFor caches the last rendered body so an unchanged body is not re-parsed
  // when the surrounding data object is replaced by a view refresh.
  let cachedBody: string | null = null
  let cachedPreview = ''
  function previewFor(body: string): string {
    if (body !== cachedBody) {
      cachedBody = body
      cachedPreview = renderMarkdown(body)
    }
    return cachedPreview
  }

  // preview is the node body rendered to HTML for the in-node snippet.
  let preview = $derived(previewFor(data.body))

  // toggle flips the collapsed state of this task's decision link without letting
  // the click bubble up to node selection or the detail modal.
  function toggle(event: MouseEvent): void {
    event.stopPropagation()
    toggleDecisions(data.id, !data.decisionsCollapsed)
  }
</script>

<div
  class="task-node"
  class:selected
  style={`--node-colour:${data.colour}`}
  ondblclick={() => openNodeModal(data.id)}
  role="button"
  tabindex="-1"
>
  <Handle type="target" position={Position.Top} isConnectable={false} />
  <div class="header">
    {#if data.icon}<span class="icon">{data.icon}</span>{/if}
    <span class="title">{data.title || 'Untitled'}</span>
    {#if data.decisionCount > 0}
      <button
        type="button"
        class="badge"
        onclick={toggle}
        title={data.decisionsCollapsed ? 'Expand decisions' : 'Collapse decisions'}
      >
        {data.decisionCount}{data.decisionsCollapsed ? ' ▸' : ' ▾'}
      </button>
    {/if}
  </div>
  {#if preview}
    <div class="body markdown">{@html preview}</div>
  {/if}
  <Handle type="source" position={Position.Bottom} isConnectable={false} />
</div>

<style>
  .task-node {
    min-width: 168px;
    max-width: 268px;
    border-radius: 14px;
    background-color: rgba(12, 13, 17, 0.88);
    /* Status colour (derived from decisions) is shown on the border at all times. */
    border: 1.5px solid var(--node-colour);
    box-shadow: 0 10px 28px rgba(0, 0, 0, 0.55);
    color: var(--text);
    text-align: left;
    font-size: 13px;
    overflow: hidden;
  }

  /* Selection overrides the status border with the accent blue to stand out. */
  .task-node.selected {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--accent), 0 10px 28px rgba(0, 0, 0, 0.55);
  }

  .header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  }

  .icon {
    font-size: 15px;
  }

  .title {
    flex: 1;
    font-weight: 600;
    color: #f5f7fa;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .badge {
    flex-shrink: 0;
    padding: 1px 6px;
    font-size: 11px;
    line-height: 1.4;
    border-radius: 10px;
    background-color: rgba(255, 255, 255, 0.06);
    border-color: rgba(255, 255, 255, 0.12);
  }

  .body {
    padding: 9px 12px;
    font-size: 11px;
    color: var(--text-muted);
    max-height: 4.5em;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
  }
</style>
