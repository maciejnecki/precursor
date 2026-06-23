<script lang="ts">
  import { Handle, Position } from '@xyflow/svelte'
  import { openNodeModal } from '../store'

  // A decision node renders as a small dark card matching the task nodes, with a
  // status dot tinted by the decision colour. Double-clicking opens the detail modal.
  let { data }: { data: { id: string; title: string; icon: string; colour: string; selected: boolean } } = $props()
</script>

<div
  class="decision-node"
  class:selected={data.selected}
  style={`--node-colour:${data.colour}`}
  ondblclick={() => openNodeModal(data.id)}
  role="button"
  tabindex="-1"
>
  <Handle type="target" position={Position.Top} isConnectable={false} />
  {#if data.icon}<span class="icon">{data.icon}</span>{/if}
  <span class="label">{data.title || 'Decision'}</span>
  <Handle type="source" position={Position.Bottom} isConnectable={false} />
</div>

<style>
  .decision-node {
    display: flex;
    align-items: center;
    gap: 7px;
    width: 132px;
    padding: 8px 11px;
    border-radius: 10px;
    background-color: rgba(12, 13, 17, 0.85);
    /* Border carries the decision-type colour at all times. */
    border: 1.5px solid var(--node-colour);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.5);
  }

  .decision-node.selected {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--accent), 0 6px 18px rgba(0, 0, 0, 0.5);
  }

  .icon {
    font-size: 13px;
  }

  .label {
    flex: 1;
    font-size: 11px;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
