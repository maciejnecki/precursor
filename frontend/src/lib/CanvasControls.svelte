<script lang="ts">
  import { onMount } from 'svelte'
  import { Panel, useSvelteFlow } from '@xyflow/svelte'
  import { registerCanvasCommands } from './canvasCommands'
  import { computeChainAreas, projectCardId } from './chains'
  import { view } from './store'

  // Navigation controls pinned to the bottom-right of the canvas: Home zooms in on
  // the project card; Zoom to fit frames the task and decision cards. Rendered
  // inside <SvelteFlow> so the flow hooks have their context.
  const { fitView, setCenter, getZoom, getNode } = useSvelteFlow()

  // How long the view animates when a control is used.
  const transitionMs = 300

  // The zoom level Home settles on, close enough to read the project card without
  // filling the whole canvas with it.
  const homeZoom = 1

  // The closest Fit is allowed to zoom, so a small project keeps its surroundings in
  // view instead of being blown up to fill the canvas.
  const fitMaxZoom = 1

  // centerOnNode pans the view to the given node's centre. The zoom level is kept
  // unless an explicit one is given. Node positions are top-left corners, so the
  // measured card size (with a fallback for cards not yet measured) shifts the
  // target to the middle.
  function centerOnNode(identifier: string, zoom?: number): void {
    const target = getNode(identifier)
    if (!target) {
      return
    }
    const centreX = target.position.x + (target.measured?.width ?? 200) / 2
    const centreY = target.position.y + (target.measured?.height ?? 60) / 2
    void setCenter(centreX, centreY, { zoom: zoom ?? getZoom(), duration: transitionMs })
  }

  // home zooms in on the project card and centres it.
  function home(): void {
    centerOnNode(projectCardId, homeZoom)
  }

  // fit frames the chain background panels, deliberately excluding the project card
  // so the graph itself fills the view. Framing the panels rather than the bare cards
  // keeps the chains' own padding, and the zoom cap stops a project with a single
  // short chain from filling the canvas.
  function fit(): void {
    const panels = computeChainAreas($view?.nodes ?? []).map((area) => ({ id: area.id }))
    if (panels.length === 0) {
      return
    }
    void fitView({ duration: transitionMs, padding: 0.2, maxZoom: fitMaxZoom, nodes: panels })
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
