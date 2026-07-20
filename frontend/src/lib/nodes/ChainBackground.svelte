<script lang="ts">
  // A non-interactive panel rendered behind a chain's nodes, with the chain's
  // completion bar shown at the bottom beneath the endpoint. It is kept very light so
  // the links running over it stay legible.
  import ProgressBar from '../ProgressBar.svelte'

  let { data }: { data: { width: number; height: number; percent: number; doneColour: string } } =
    $props()
</script>

<div class="chain-bg" style={`width:${data.width}px; height:${data.height}px`}>
  <div class="completion">
    <ProgressBar percent={data.percent} colour={data.doneColour} />
  </div>
</div>

<style>
  .chain-bg {
    box-sizing: border-box;
    display: flex;
    align-items: flex-end;
    justify-content: center;
    border-radius: 18px;
    background-color: var(--surface-panel);
    border: 1px solid var(--border-panel);
    /* Clicks fall through to the canvas so node selection still works. */
    pointer-events: none;
  }

  /* The bar sits in the band reserved beneath the endpoint, inset from the panel
     edges so it lines up with the endpoint card above it. */
  .completion {
    width: 100%;
    padding: 0 20px 12px;
    box-sizing: border-box;
  }
</style>
