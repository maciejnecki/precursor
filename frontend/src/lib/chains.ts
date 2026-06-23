// Groups a project's nodes into chains and describes the light background drawn
// behind each one, along with the chain's completion percentage. A chain is an
// endpoint task together with everything that resolves to it: its precursors and
// the decisions documenting them.
import type { NodeView } from './api'

// Fixed geometry for a chain's background. Node positions come from the backend
// layout (top-left of each card) and every node in a chain shares one column x.
// The endpoint always sits on the baseline at y = 0, so the area reserves a fixed
// band beneath it for the endpoint card and the completion label.
const taskWidth = 268
const sidePadding = 20
const topPadding = 20
const bottomReserve = 150
const chainWidth = taskWidth + sidePadding * 2

// ChainArea is the render-ready background for one chain.
export type ChainArea = {
  id: string
  x: number
  y: number
  width: number
  height: number
  percent: number
}

// endpointId walks parent links from a task up to the chain root (the endpoint).
function endpointId(byId: Map<string, NodeView>, taskId: string): string {
  let current = byId.get(taskId)
  while (current && current.parentId) {
    const parent = byId.get(current.parentId)
    if (!parent) {
      break
    }
    current = parent
  }
  return current ? current.id : ''
}

// chainOf returns the endpoint a node belongs to: a task resolves through its
// parents; a decision resolves through the task it documents.
function chainOf(byId: Map<string, NodeView>, node: NodeView): string {
  if (node.kind === 'decision') {
    return node.childId ? endpointId(byId, node.childId) : ''
  }
  return endpointId(byId, node.id)
}

// completion applies the formula done / (total - redundant) over a chain's tasks,
// guarding the empty denominator (every task redundant, or none present).
function completion(tasks: NodeView[]): number {
  const total = tasks.length
  const done = tasks.filter((task) => task.status === 'done').length
  const redundant = tasks.filter((task) => task.status === 'redundant').length
  const denominator = total - redundant
  if (denominator <= 0) {
    return 0
  }
  return Math.round((done / denominator) * 100)
}

// computeChainAreas returns one background area per chain, sized to wrap the chain's
// nodes with room for the completion label beneath the endpoint.
export function computeChainAreas(nodes: NodeView[]): ChainArea[] {
  const byId = new Map(nodes.map((node) => [node.id, node]))
  const members = new Map<string, NodeView[]>()
  for (const node of nodes) {
    const endpoint = chainOf(byId, node)
    if (!endpoint) {
      continue
    }
    const list = members.get(endpoint) ?? []
    list.push(node)
    members.set(endpoint, list)
  }

  const areas: ChainArea[] = []
  for (const [endpoint, chainNodes] of members) {
    const left = Math.min(...chainNodes.map((node) => node.x))
    const top = Math.min(...chainNodes.map((node) => node.y))
    const tasks = chainNodes.filter((node) => node.kind === 'task')
    areas.push({
      id: `chain:${endpoint}`,
      x: left - sidePadding,
      y: top - topPadding,
      width: chainWidth,
      height: bottomReserve - (top - topPadding),
      percent: completion(tasks)
    })
  }
  return areas
}
