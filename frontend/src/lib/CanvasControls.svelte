<script lang="ts">
  import { onMount } from 'svelte'
  import { Panel, useSvelteFlow } from '@xyflow/svelte'
  import { registerCanvasCommands } from './canvasCommands'
  import { computeChainAreas, projectCardHeight, projectCardId, projectCardWidth } from './chains'
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
  // unless an explicit one is given. Node positions are top-left corners, so the card
  // size shifts the target to the middle: the given size when the caller knows it,
  // otherwise the measured one, falling back to a generic card for nodes the flow has
  // not measured yet.
  function centerOnNode(
    identifier: string,
    zoom?: number,
    size?: { width: number; height: number },
    durationMs: number = transitionMs
  ): void {
    const target = getNode(identifier)
    if (!target) {
      return
    }
    const centreX = target.position.x + (size?.width ?? target.measured?.width ?? 200) / 2
    const centreY = target.position.y + (size?.height ?? target.measured?.height ?? 60) / 2
    void setCenter(centreX, centreY, { zoom: zoom ?? getZoom(), duration: durationMs })
  }

  // home zooms in on the project card and centres it. The card's size is laid out
  // rather than measured, so passing it keeps the snap correct even on the frame the
  // card first appears.
  function home(durationMs: number = transitionMs): void {
    centerOnNode(projectCardId, homeZoom, { width: projectCardWidth, height: projectCardHeight }, durationMs)
  }

  // chainPanels lists the chain background panels of the open project, which is what
  // Fit frames.
  function chainPanels(): { id: string }[] {
    return computeChainAreas($view?.nodes ?? []).map((area) => ({ id: area.id }))
  }

  // fit frames the chain background panels, deliberately excluding the project card
  // so the graph itself fills the view. Framing the panels rather than the bare cards
  // keeps the chains' own padding, and the zoom cap stops a project with a single
  // short chain from filling the canvas.
  function fit(durationMs: number = transitionMs): void {
    const panels = chainPanels()
    if (panels.length === 0) {
      return
    }
    void fitView({ duration: durationMs, padding: 0.2, maxZoom: fitMaxZoom, nodes: panels })
  }

  // How many frames the automatic framing waits for the flow to register the nodes of
  // a newly opened project. They normally appear on the first or second.
  const framingFrameBudget = 10

  // snapToDefaultView frames a newly opened project the way Fit does, without
  // animating, since there is no previous viewport worth showing the move from. It
  // retries for a few frames because a project that has only just opened has not been
  // turned into flow nodes yet, and falls back to the project card for a project that
  // has no chains to frame.
  function snapToDefaultView(framesLeft: number): void {
    const panels = chainPanels()
    const awaited = panels.length > 0 ? panels[0].id : projectCardId
    if (getNode(awaited)) {
      if (panels.length > 0) {
        fit(0)
      } else {
        home(0)
      }
      return
    }
    if (framesLeft > 0) {
      requestAnimationFrame(() => snapToDefaultView(framesLeft - 1))
    }
  }

  // Every project opens framed on its chains, so the canvas starts somewhere known
  // instead of keeping the viewport the previously open project left behind.
  // Plain state rather than $state: it only guards the effect from re-framing the
  // same project, and nothing renders from it.
  let framedProjectId: string | null = null

  $effect(() => {
    const openProjectId = $view?.project.id ?? null
    if (!openProjectId || openProjectId === framedProjectId) {
      return
    }
    framedProjectId = openProjectId
    snapToDefaultView(framingFrameBudget)
  })

  // Register the viewport commands so the global shortcuts (h, shift+h) and the
  // search bar's match cycling can drive the canvas from outside the flow context.
  // The returned unregister runs on unmount, making the shortcuts safe no-ops when
  // no project is open.
  onMount(() => registerCanvasCommands({ home, fitAll: fit, centerOnNode }))
</script>

<Panel position="bottom-right">
  <div class="controls">
    <!-- Wrapped rather than passed directly, so the click event is not taken for the
         optional animation duration. -->
    <button type="button" onclick={() => home()}>Home</button>
    <button type="button" onclick={() => fit()}>Fit</button>
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
