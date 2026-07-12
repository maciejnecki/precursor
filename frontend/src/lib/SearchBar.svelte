<script lang="ts">
  import { Panel } from '@xyflow/svelte'
  import {
    closeSearch,
    navigateSearch,
    searchActiveIndex,
    searchFocusToken,
    searchMatchIds,
    searchOpen,
    searchQuery,
    setSearchQuery
  } from './store'

  // Search bar pinned to the top-right of the canvas (cmd+f). Typing live-filters
  // the project's nodes by title and description; non-matching cards dim on the
  // canvas. Enter pans to the next match, Shift+Enter to the previous, and Escape
  // closes the bar and clears the dimming. Rendered inside <SvelteFlow> so the
  // Panel positions itself within the canvas.
  let inputElement: HTMLInputElement | undefined = $state()

  // hasQuery controls the counter's visibility: it only shows once something has
  // actually been searched for.
  let hasQuery = $derived($searchQuery.trim().length > 0)

  // displayedIndex clamps the active index to the match list, so the counter stays
  // truthful if matches disappear (a node edit or delete) before the next Enter.
  let displayedIndex = $derived(Math.min($searchActiveIndex, Math.max($searchMatchIds.length - 1, 0)))

  // Focus and select the query text whenever the bar opens or cmd+f is pressed
  // again while it is already showing; the focus token bumps on every openSearch.
  $effect(() => {
    void $searchFocusToken
    if ($searchOpen && inputElement) {
      inputElement.focus()
      inputElement.select()
    }
  })

  // onInput pushes the live query into the store, which recomputes the match set.
  function onInput(event: Event): void {
    const target = event.target as HTMLInputElement
    setSearchQuery(target.value)
  }

  // onKeydown handles the bar's own keys: Enter cycles matches (backwards with
  // Shift) and Escape closes just the bar, stopping the window-level Escape cascade
  // from dismissing anything else.
  function onKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter') {
      event.preventDefault()
      navigateSearch(event.shiftKey ? -1 : 1)
    } else if (event.key === 'Escape') {
      event.stopPropagation()
      closeSearch()
    }
  }
</script>

{#if $searchOpen}
  <Panel position="top-right">
    <div class="search-bar">
      <input
        bind:this={inputElement}
        type="text"
        placeholder="Search nodes"
        value={$searchQuery}
        oninput={onInput}
        onkeydown={onKeydown}
      />
      {#if hasQuery}
        <span class="counter">
          {$searchMatchIds.length === 0 ? '0 / 0' : `${displayedIndex + 1} / ${$searchMatchIds.length}`}
        </span>
      {/if}
      <button type="button" class="close" onclick={closeSearch} aria-label="Close search">×</button>
    </div>
  </Panel>
{/if}

<style>
  /* Matches the light translucent panel used by the canvas controls. */
  .search-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    background-color: rgba(255, 255, 255, 0.045);
    border: 1px solid rgba(255, 255, 255, 0.07);
    border-radius: 10px;
    padding: 6px 10px;
  }

  .search-bar input {
    background: transparent;
    border: none;
    outline: none;
    color: var(--text);
    width: 180px;
    font-size: 13px;
  }

  .search-bar input::placeholder {
    color: var(--text-muted);
  }

  .counter {
    color: var(--text-muted);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .close {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 15px;
    line-height: 1;
    padding: 0 2px;
    cursor: pointer;
  }

  .close:hover {
    color: var(--text);
  }
</style>
