<script lang="ts">
  // A non-interactive panel rendered behind a chain's nodes, with the chain's
  // completion percentage shown at the bottom beneath the endpoint. It is kept very
  // light so the links running over it stay legible.
  let { data }: { data: { width: number; height: number; percent: number; doneColour: string } } =
    $props()

  // A fully complete chain shows its label in the done colour to match the endpoint.
  let complete = $derived(data.percent >= 100)
</script>

<div class="chain-bg" style={`width:${data.width}px; height:${data.height}px`}>
  <span class="completion" style={complete ? `color:${data.doneColour}` : ''}>
    {data.percent}% complete
  </span>
</div>

<style>
  .chain-bg {
    box-sizing: border-box;
    display: flex;
    align-items: flex-end;
    justify-content: center;
    border-radius: 18px;
    background-color: rgba(255, 255, 255, 0.045);
    border: 1px solid rgba(255, 255, 255, 0.07);
    /* Clicks fall through to the canvas so node selection still works. */
    pointer-events: none;
  }

  .completion {
    padding-bottom: 12px;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-muted);
  }
</style>
