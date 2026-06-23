<script lang="ts">
  import {
    copyCompletedMarkdown,
    exportProjectFile,
    importProjectFile,
    saveSettings,
    settings,
    showSettings,
    view
  } from './store'
  import type { Settings } from './api'

  // draft is a local editable copy of the settings so changes apply only on save.
  let draft = $state<Settings | null>(null)

  // Seed the draft from the loaded settings whenever the panel opens.
  $effect(() => {
    if ($showSettings && $settings) {
      draft = JSON.parse(JSON.stringify($settings))
    }
  })

  // close hides the settings panel without saving.
  function close(): void {
    showSettings.set(false)
  }

  // apply persists the edited settings and closes the panel.
  async function apply(): Promise<void> {
    if (draft) {
      await saveSettings(draft)
    }
    close()
  }
</script>

{#if $showSettings && draft}
  <div class="overlay" onclick={close}></div>
  <div class="panel">
    <h2>Settings</h2>
    <p class="section">Status colours</p>
    <label><span>Scheduled</span><input type="color" bind:value={draft.statusColours.scheduled} /></label>
    <label><span>In Progress</span><input type="color" bind:value={draft.statusColours.inProgress} /></label>
    <label><span>Done</span><input type="color" bind:value={draft.statusColours.done} /></label>
    <label><span>Redundant</span><input type="color" bind:value={draft.statusColours.redundant} /></label>
    <p class="section">Decision colour</p>
    <label><span>Plain decision</span><input type="color" bind:value={draft.decisionColour} /></label>
    <p class="section">Data</p>
    <div class="data-actions">
      <button type="button" onclick={importProjectFile}>Import backup</button>
      {#if $view}
        <button type="button" onclick={exportProjectFile}>Backup JSON</button>
        <button type="button" onclick={copyCompletedMarkdown}>Copy completed</button>
      {/if}
    </div>
    <div class="actions">
      <button type="button" onclick={close}>Cancel</button>
      <button type="button" class="primary" onclick={apply}>Save</button>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 60;
  }

  .panel {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 70;
    width: 320px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 18px;
    background-color: var(--surface-raised);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border);
    border-radius: 10px;
  }

  h2 {
    margin: 0;
    font-size: 18px;
  }

  .section {
    margin: 8px 0 0;
    color: var(--text-muted);
    font-size: 13px;
  }

  label {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .data-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .actions {
    display: flex;
    gap: 8px;
    margin-top: 12px;
  }

  /* Save sits at the right edge; Cancel stays on the left. */
  .actions .primary {
    margin-left: auto;
  }
</style>
