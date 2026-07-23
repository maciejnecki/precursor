<script lang="ts">
  import { SvelteFlow, Background, MarkerType, type Node, type Edge } from '@xyflow/svelte'
  import '@xyflow/svelte/dist/style.css'
  import TaskNode from './nodes/TaskNode.svelte'
  import DecisionNode from './nodes/DecisionNode.svelte'
  import ChainBackground from './nodes/ChainBackground.svelte'
  import ProjectCard from './nodes/ProjectCard.svelte'
  import ZoomController from './ZoomController.svelte'
  import CanvasControls from './CanvasControls.svelte'
  import SearchBar from './SearchBar.svelte'
  import FloatingEdge from './edges/FloatingEdge.svelte'
  import {
    closeEditor,
    handleNodeClick,
    searchMatchIds,
    searchQuery,
    selectNode,
    selectedNodeIds,
    setEditorAnchor,
    settings,
    view
  } from './store'
  import {
    computeChainAreas,
    projectCardHeight,
    projectCardId,
    projectCardPosition,
    projectCardWidth,
    projectCompletion
  } from './chains'
  import type { NodeView, Settings } from './api'

  // nodeTypes maps our node kinds to their canvas components. The chain background
  // is a non-interactive panel drawn behind a chain's task and decision nodes.
  const nodeTypes = {
    task: TaskNode,
    decision: DecisionNode,
    chainBackground: ChainBackground,
    projectCard: ProjectCard
  }

  // edgeTypes maps every link to the radial floating edge so connections attach to
  // the node borders facing each other instead of fixed handles.
  const edgeTypes = { floating: FloatingEdge }

  // colourForNode resolves the colour a node should be drawn with from settings.
  function colourForNode(node: NodeView, currentSettings: Settings | null): string {
    const palette = currentSettings?.statusColours
    if (node.kind === 'task') {
      if (node.status === 'in_progress') return palette?.inProgress ?? '#f59e0b'
      if (node.status === 'done') return palette?.done ?? '#22c55e'
      if (node.status === 'redundant') return palette?.redundant ?? '#ef4444'
      return palette?.scheduled ?? '#64748b'
    }
    if (node.decisionType === 'in_progress') return palette?.inProgress ?? '#f59e0b'
    if (node.decisionType === 'done') return palette?.done ?? '#22c55e'
    if (node.decisionType === 'redundant') return palette?.redundant ?? '#ef4444'
    if (node.decisionType === 'scheduled') return palette?.scheduled ?? '#64748b'
    return currentSettings?.decisionColour ?? '#94a3b8'
  }

  // isPendingEndpoint reports whether a node is a chain-root task that has not yet
  // resolved, so its outline should use the standout endpoint colour.
  function isPendingEndpoint(node: NodeView): boolean {
    return node.kind === 'task' && !node.parentId && node.status !== 'done' && node.status !== 'redundant'
  }

  // outlineForNode resolves a node card's border colour: a pending endpoint stands
  // out in the endpoint colour, while every other node uses its status colour.
  function outlineForNode(node: NodeView, currentSettings: Settings | null): string {
    if (isPendingEndpoint(node)) {
      return currentSettings?.endpointColour ?? '#fc4e26'
    }
    return colourForNode(node, currentSettings)
  }

  // chainNodes are the light background panels drawn behind each chain, carrying the
  // chain's completion percentage. They sit below the task and decision nodes.
  let chainAreas = $derived(computeChainAreas($view?.nodes ?? []))

  let chainNodes = $derived<Node[]>(
    chainAreas.map((area) => ({
      id: area.id,
      type: 'chainBackground',
      position: { x: area.x, y: area.y },
      draggable: false,
      selectable: false,
      zIndex: 0,
      data: {
        width: area.width,
        height: area.height,
        percent: area.percent,
        doneColour: $settings?.statusColours?.done ?? '#22c55e'
      }
    }))
  )

  // projectCardNodes holds the single card summarising the open project, or nothing
  // when no project is open. It is derived from project metadata rather than stored
  // in the graph, so it never counts as a node the layout or the endpoint tally sees.
  let projectCardNodes = $derived<Node[]>(
    $view
      ? [
          {
            id: projectCardId,
            type: 'projectCard',
            position: projectCardPosition(chainAreas),
            draggable: false,
            selectable: false,
            zIndex: 1,
            data: {
              name: $view.project.name,
              icon: $view.project.icon,
              description: $view.project.description,
              percent: projectCompletion($view.nodes),
              width: projectCardWidth,
              height: projectCardHeight,
              doneColour: $settings?.statusColours?.done ?? '#22c55e'
            }
          }
        ]
      : []
  )

  // baseFlowNodes is the Svelte Flow node array derived from the open project's view.
  // The chain backgrounds come first so the cards stack above them. It deliberately
  // does not depend on the selection, so selecting a node reuses these data objects.
  let baseFlowNodes = $derived<Node[]>([
    ...chainNodes,
    ...projectCardNodes,
    ...($view?.nodes ?? []).map((node) => ({
      id: node.id,
      type: node.kind,
      position: { x: node.x, y: node.y },
      draggable: false,
      selectable: false,
      zIndex: 1,
      data: {
        id: node.id,
        title: node.title,
        body: node.bodyMarkdown,
        icon: node.icon,
        colour: outlineForNode(node, $settings),
        decisionCount: node.decisionCount,
        decisionsCollapsed: node.decisionsCollapsed
      }
    }))
  ])

  // searchActive reports whether a search query is being applied to the canvas.
  let searchActive = $derived($searchQuery.trim().length > 0)

  // matchIdSet indexes the search matches for constant-time membership checks while
  // stamping the dimming class onto the flow nodes.
  let matchIdSet = $derived(new Set($searchMatchIds))

  // flowNodes stamps the current selection and search dimming onto the base nodes at
  // the node level. Only the thin wrapper objects are recreated on a selection or
  // search change; each card's data keeps its identity, so the node components do
  // not re-render their content. Chain backgrounds are never dimmed.
  let flowNodes = $derived<Node[]>(
    baseFlowNodes.map((node) =>
      node.type === 'chainBackground'
        ? node
        : {
            ...node,
            selected: $selectedNodeIds.includes(node.id),
            class: searchActive && !matchIdSet.has(node.id) ? 'search-dimmed' : ''
          }
    )
  )

  // nodesById indexes the view's nodes for constant-time lookups when colouring the
  // edges, instead of scanning the node list once per edge.
  let nodesById = $derived(new Map(($view?.nodes ?? []).map((node) => [node.id, node])))

  // colourForEdge resolves an edge's colour from the status of the task whose
  // transition the edge belongs to, so a link matches its task's colour coding.
  function colourForEdge(taskId: string, currentSettings: Settings | null): string {
    const owner = nodesById.get(taskId)
    return owner ? colourForNode(owner, currentSettings) : '#64748b'
  }

  // flowEdges is the Svelte Flow edge array, coloured by the owning task's status and
  // curved more strongly for decision links.
  let flowEdges = $derived<Edge[]>(
    ($view?.edges ?? []).map((edge) => {
      const colour = colourForEdge(edge.taskId, $settings)
      const isDecision = edge.kind === 'decision'
      const dashed = isDecision ? ' stroke-dasharray: 6 4;' : ''
      return {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        type: 'floating',
        data: { curved: isDecision, curvature: 0.55 },
        markerEnd: { type: MarkerType.ArrowClosed, color: colour },
        style: `stroke: ${colour}; stroke-width: 1.6;${dashed}`
      }
    })
  )

  // hasProject reports whether a project is open, controlling the empty-state hint.
  let hasProject = $derived($view !== null)
  let isEmpty = $derived(($view?.nodes.length ?? 0) === 0)

  // recordPointer remembers where on screen the canvas was pressed so the compose
  // popup can spawn there. It runs before the click handlers that open the popup.
  function recordPointer(event: PointerEvent): void {
    setEditorAnchor(event.clientX, event.clientY)
  }
</script>

<!-- The wrapper only records pointer positions for the compose popup; it exposes no
     interaction of its own, so it is presentational to assistive technology. -->
<div class="canvas" role="presentation" onpointerdown={recordPointer}>
  {#if hasProject}
    <SvelteFlow
      nodes={flowNodes}
      edges={flowEdges}
      {nodeTypes}
      {edgeTypes}
      nodesDraggable={false}
      deleteKey={null}
      zoomOnDoubleClick={false}
      zoomOnScroll={false}
      zoomOnPinch={false}
      minZoom={0.2}
      maxZoom={2.5}
      proOptions={{ hideAttribution: true }}
      fitView
      onnodeclick={({ node, event }) => {
        if (node.type === 'chainBackground') {
          return
        }
        handleNodeClick(node.id, event.shiftKey)
      }}
      onpaneclick={() => {
        selectNode(null)
        closeEditor()
      }}
    >
      <Background gap={36} size={1} />
      <ZoomController />
      <CanvasControls />
      <SearchBar />
    </SvelteFlow>

    {#if isEmpty}
      <div class="hint">Press <span class="keys">⇧⌘T</span> to create your first task.</div>
    {/if}
  {:else}
    <div class="hint">Select or create a project from the sidebar to begin.</div>
  {/if}
</div>

<style>
  .canvas {
    position: relative;
    height: 100%;
    /* Translucent black so the window vibrancy reads through as a dark canvas. */
    background-color: rgba(0, 0, 0, 0.55);
    /* Square on every corner so the canvas runs flush to the window edges now that it
       reaches the top. */
    overflow: hidden;
  }

  /* Let the canvas tint show through Svelte Flow's own background layer. */
  :global(.svelte-flow),
  :global(.svelte-flow__background) {
    background: transparent;
  }

  .hint {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    color: var(--text-muted);
    pointer-events: none;
    text-align: center;
    max-width: 70%;
  }

  /* The key combination reads as a key cap inside the hint sentence. */
  .hint .keys {
    background-color: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text);
    font-weight: 600;
    padding: 2px 6px;
    white-space: nowrap;
  }

  :global(.svelte-flow__handle) {
    opacity: 0;
    pointer-events: none;
  }

  /* The chain background is purely decorative: let clicks fall through to the canvas
     so selecting and deselecting nodes still works over it. */
  :global(.svelte-flow__node-chainBackground) {
    pointer-events: none;
    cursor: default;
  }

  /* Search dimming fades cards in and out smoothly as the query changes. */
  :global(.svelte-flow__node-task),
  :global(.svelte-flow__node-decision) {
    transition: opacity 160ms ease;
  }

  /* Nodes that do not match the active search query fade back so matches stand out. */
  :global(.svelte-flow__node.search-dimmed) {
    opacity: 0.15;
  }
</style>
