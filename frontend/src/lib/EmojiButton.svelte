<script lang="ts">
  import { iconGlyphs } from './types'

  // A compact button that shows the chosen glyph and opens a small popover grid of
  // curated icons. This replaces the heavy full emoji-picker web component with a
  // lightweight, dependency-free set suited to work tracking.
  let { value = $bindable('') }: { value?: string } = $props()

  let open = $state(false)

  // toggle opens or closes the glyph popover.
  function toggle(): void {
    open = !open
  }

  // close hides the glyph popover.
  function close(): void {
    open = false
  }

  // choose records the selected glyph and closes the popover.
  function choose(glyph: string): void {
    value = glyph
    open = false
  }

  // clear removes the current glyph, keeping the field optional.
  function clear(): void {
    value = ''
    open = false
  }
</script>

<div class="emoji-button">
  <button type="button" onclick={toggle} title="Choose an icon">
    {value || '🙂'}
  </button>

  {#if open}
    <div class="backdrop" onclick={close}></div>
    <div class="popover">
      <div class="grid">
        {#each iconGlyphs as glyph}
          <button type="button" class="glyph" class:active={glyph === value} onclick={() => choose(glyph)}>
            {glyph}
          </button>
        {/each}
      </div>
      <button type="button" class="clear" onclick={clear}>Clear icon</button>
    </div>
  {/if}
</div>

<style>
  .emoji-button {
    position: relative;
    display: inline-block;
  }

  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 40;
  }

  .popover {
    position: absolute;
    bottom: calc(100% + 6px);
    left: 0;
    z-index: 50;
    display: flex;
    flex-direction: column;
    gap: 6px;
    background-color: var(--surface-raised);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 8px;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 4px;
  }

  .glyph {
    width: 36px;
    height: 36px;
    font-size: 18px;
    padding: 0;
  }

  .glyph.active {
    border-color: var(--accent);
  }

  .clear {
    font-size: 12px;
  }
</style>
