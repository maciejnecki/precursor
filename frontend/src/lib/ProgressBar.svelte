<script lang="ts">
  // A completion bar with its percentage beside it, shared by the project card and
  // the chain backgrounds so both report progress the same way. The fill always uses
  // the done status colour, tying the bar to the colour completed tasks already wear.
  let { percent, colour }: { percent: number; colour: string } = $props()
</script>

<div class="progress">
  <div class="track">
    <div class="fill" style={`width:${percent}%; background-color:${colour}`}></div>
  </div>
  <span class="percent" style={percent >= 100 ? `color:${colour}` : ''}>{percent}%</span>
</div>

<style>
  .progress {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
  }

  /* One size serves both the chain backgrounds and the project card, so progress
     reads identically wherever it appears. */
  .track {
    flex: 1;
    height: 6px;
    border-radius: 3px;
    background-color: rgba(255, 255, 255, 0.09);
    overflow: hidden;
  }

  .fill {
    height: 100%;
    border-radius: 3px;
  }

  .percent {
    flex-shrink: 0;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-muted);
  }
</style>
