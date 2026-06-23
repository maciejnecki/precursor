<script lang="ts">
  import { onMount } from 'svelte'
  import { useSvelteFlow, type Viewport } from '@xyflow/svelte'

  // Replaces the default wheel zoom with an instant zoom-to-cursor gesture: the point
  // under the cursor stays pinned under the cursor as you zoom in or out, so you can
  // keep zooming into the same spot. (A webview cannot move the OS cursor, so the
  // point is held under the cursor rather than warped to the view centre.) Rendered
  // inside <SvelteFlow> so the flow hooks have context.
  const { getViewport, setViewport } = useSvelteFlow()

  // Zoom bounds and how sharply a wheel notch changes the zoom level.
  const minZoom = 0.2
  const maxZoom = 2.5
  const wheelSensitivity = 0.0015

  // The gap after which a wheel event is treated as a fresh gesture, so any panning
  // done between gestures is picked up from the live viewport.
  const gestureGapMs = 120

  // pane is the flow surface the wheel gesture acts on.
  let pane: HTMLElement | null = null

  // tracked is the viewport applied so far this gesture, kept in sync locally so a
  // fast burst of wheel events never reads a stale, not-yet-applied viewport.
  let tracked: Viewport | null = null
  let lastWheelTime = 0

  // onWheel zooms toward the cursor and recentres the view on that point instantly.
  function onWheel(event: WheelEvent): void {
    event.preventDefault()
    if (!pane) {
      return
    }
    const now = performance.now()
    if (!tracked || now - lastWheelTime > gestureGapMs) {
      tracked = getViewport()
    }
    lastWheelTime = now

    const rect = pane.getBoundingClientRect()
    const pointerX = event.clientX - rect.left
    const pointerY = event.clientY - rect.top
    // World point currently under the cursor, from the locally tracked viewport.
    const worldX = (pointerX - tracked.x) / tracked.zoom
    const worldY = (pointerY - tracked.y) / tracked.zoom

    const zoom = Math.min(maxZoom, Math.max(minZoom, tracked.zoom * Math.exp(-event.deltaY * wheelSensitivity)))
    // Keep that world point under the cursor at the new zoom, so repeated zooming
    // stays anchored to where the user is pointing.
    tracked = {
      x: pointerX - worldX * zoom,
      y: pointerY - worldY * zoom,
      zoom
    }
    void setViewport(tracked, { duration: 0 })
  }

  onMount(() => {
    pane = document.querySelector('.svelte-flow')
    if (!pane) {
      return
    }
    pane.addEventListener('wheel', onWheel, { passive: false })
    return () => pane?.removeEventListener('wheel', onWheel)
  })
</script>
