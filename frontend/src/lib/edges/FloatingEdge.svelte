<script lang="ts">
  import {
    BaseEdge,
    getBezierPath,
    getStraightPath,
    Position,
    useInternalNode,
    type EdgeProps
  } from '@xyflow/svelte'

  // A floating edge attaches to the borders of its two nodes facing each other. In
  // the radial layout this runs from the innermost edge of the outer node to the
  // outermost edge of the inner node, along the chain's ray. Decision links curve;
  // plain precursor links stay straight.
  let { id, source, target, markerEnd, style, data }: EdgeProps = $props()

  // useInternalNode exposes each node's measured size and absolute position as a
  // reactive object whose latest value is read from its `current` property. Reading
  // the initial prop values here is intentional: an edge's source and target never
  // change for its lifetime, because Canvas recreates edges under new ids instead.
  // svelte-ignore state_referenced_locally
  const sourceNode = useInternalNode(source)
  // svelte-ignore state_referenced_locally
  const targetNode = useInternalNode(target)

  // centre returns the absolute centre point of a node.
  function centre(node: NonNullable<typeof sourceNode.current>): { x: number; y: number } {
    return {
      x: node.internals.positionAbsolute.x + (node.measured.width ?? 0) / 2,
      y: node.internals.positionAbsolute.y + (node.measured.height ?? 0) / 2
    }
  }

  // borderPoint returns where the straight line between two node centres crosses
  // the border of the first node, so the edge meets the node edge cleanly.
  function borderPoint(
    node: NonNullable<typeof sourceNode.current>,
    other: NonNullable<typeof targetNode.current>
  ): { x: number; y: number } {
    const halfWidth = (node.measured.width ?? 0) / 2
    const halfHeight = (node.measured.height ?? 0) / 2
    const here = centre(node)
    const there = centre(other)

    const normalisedX = (there.x - here.x) / (2 * halfWidth) - (there.y - here.y) / (2 * halfHeight)
    const normalisedY = (there.x - here.x) / (2 * halfWidth) + (there.y - here.y) / (2 * halfHeight)
    const scale = 1 / (Math.abs(normalisedX) + Math.abs(normalisedY))
    const unitX = scale * normalisedX
    const unitY = scale * normalisedY
    return {
      x: halfWidth * (unitX + unitY) + here.x,
      y: halfHeight * (-unitX + unitY) + here.y
    }
  }

  // sideFacing reports which border a point sits on relative to a centre, which sets
  // the direction the curve leaves or enters the node.
  function sideFacing(point: { x: number; y: number }, c: { x: number; y: number }): Position {
    if (Math.abs(point.x - c.x) > Math.abs(point.y - c.y)) {
      return point.x > c.x ? Position.Right : Position.Left
    }
    return point.y > c.y ? Position.Bottom : Position.Top
  }

  // path is the SVG path between the two computed border points: a straight line for
  // precursor links, or a bezier curve for decision links.
  let path = $derived.by(() => {
    const start = sourceNode.current
    const end = targetNode.current
    if (!start || !end || !start.measured.width || !end.measured.width) {
      return ''
    }
    const from = borderPoint(start, end)
    const to = borderPoint(end, start)
    const options = data as { curved?: boolean; curvature?: number } | undefined
    if (!options?.curved) {
      const [straightPath] = getStraightPath({
        sourceX: from.x,
        sourceY: from.y,
        targetX: to.x,
        targetY: to.y
      })
      return straightPath
    }
    const [bezierPath] = getBezierPath({
      sourceX: from.x,
      sourceY: from.y,
      sourcePosition: sideFacing(from, centre(start)),
      targetX: to.x,
      targetY: to.y,
      targetPosition: sideFacing(to, centre(end)),
      curvature: options.curvature ?? 0.5
    })
    return bezierPath
  })
</script>

{#if path}
  <BaseEdge {id} {path} {markerEnd} {style} />
{/if}
