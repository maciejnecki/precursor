<script lang="ts">
  import { onMount } from 'svelte'
  import { Panel, useSvelteFlow } from '@xyflow/svelte'
  import { registerCanvasCommands } from './canvasCommands'
  import { view } from './store'

  // Navigation controls pinned to the bottom-right of the canvas: Home recentres on
  // the first chain's endpoint; Zoom to fit frames every chain. Rendered inside
  // <SvelteFlow> so the flow hooks have their context.
  const { fitView, setCenter, getZoom, getNode } = useSvelteFlow()

  // How long the view animates when a control is used.
  const transitionMs = 300

  // centerOnNode pans the view to the given node's centre, keeping the current zoom
  // level. Node positions are top-left corners, so the measured card size (with a
  // fallback for cards not yet measured) shifts the target to the middle.
  function centerOnNode(identifier: string): void {
    const target = ($view?.nodes ?? []).find((node) => node.id === identifier)
    if (!target) {
      return
    }
    const measured = getNode(target.id)?.measured
    const centreX = target.x + (measured?.width ?? 200) / 2
    const centreY = target.y + (measured?.height ?? 60) / 2
    void setCenter(centreX, centreY, { zoom: getZoom(), duration: transitionMs })
  }

  // home recentres the view on the first endpoint (the leftmost chain), keeping the
  // current zoom level.
  function home(): void {
    const endpoints = ($view?.nodes ?? []).filter((node) => node.kind === 'task' && !node.parentId)
    if (endpoints.length === 0) {
      return
    }
    const first = endpoints.reduce((leftmost, candidate) => (candidate.x < leftmost.x ? candidate : leftmost))
    centerOnNode(first.id)
  }

  // fit frames all chains within the view.
  function fit(): void {
    void fitView({ duration: transitionMs, padding: 0.2 })
  }

  // Register the viewport commands so the global shortcuts (h, shift+h) and the
  // search bar's match cycling can drive the canvas from outside the flow context.
  // The returned unregister runs on unmount, making the shortcuts safe no-ops when
  // no project is open.
  onMount(() => registerCanvasCommands({ home, fitAll: fit, centerOnNode }))
</script>

<Panel position="bottom-right">
  <div class="controls">
    <button type="button" onclick={home}>Home</button>
    <button type="button" onclick={fit}>Fit</button>
  </div>
</Panel>

<style>
  .controls {
    display: flex;
    gap: 6px;
  }

  /* Matches the light translucent panel drawn behind chains. */
  .controls button {
    background-color: rgba(255, 255, 255, 0.045);
    border: 1px solid rgba(255, 255, 255, 0.07);
    border-radius: 10px;
    color: var(--text-muted);
    font-weight: 600;
    padding: 6px 12px;
  }

  .controls button:hover {
    background-color: rgba(255, 255, 255, 0.09);
    border-color: rgba(255, 255, 255, 0.14);
    color: var(--text);
  }
</style>
