<script lang="ts">
  import { confirmRequest, resolveConfirm } from './store'
</script>

{#if $confirmRequest}
  <div class="overlay" onclick={() => resolveConfirm(false)}></div>
  <div class="panel">
    <p class="message">{$confirmRequest.message}</p>
    <div class="actions">
      <button type="button" class="danger" onclick={() => resolveConfirm(true)}>Confirm</button>
      <button type="button" onclick={() => resolveConfirm(false)}>Cancel</button>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.55);
    z-index: 100;
  }

  .panel {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 110;
    width: 360px;
    max-width: 90vw;
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 18px;
    background-color: var(--surface-raised);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border);
    border-radius: 10px;
  }

  .message {
    margin: 0;
    line-height: 1.5;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .danger {
    border-color: #ef4444;
  }

  .danger:hover {
    background-color: #7f1d1d;
  }
</style>
