// Pure layout arithmetic for the sidebar's project list. The component owns the
// pointer handling and the rendering; every rearrangement of the order and the
// groups happens here, so the rules stay in one readable place.
//
// Two invariants hold throughout: a project belongs to at most one group, and a
// group's members sit together in the order. Every function below preserves them,
// which is what lets a group be drawn as one unbroken band. The backend enforces
// the same invariants when it stores the result.

import type { Project, ProjectGroup } from './api'

// Layout is an order of project identifiers alongside the groups banding them.
export type Layout = { order: string[]; groups: ProjectGroup[] }

// Band is one rendered run of the list: a group with its members, or a single
// ungrouped project (carried as a one-project band with no group).
export type Band = { group: ProjectGroup | null; projects: Project[] }

// groupIdOf returns the identifier of the group holding the project, or null.
export function groupIdOf(groups: ProjectGroup[], projectId: string): string | null {
  const owner = groups.find((group) => group.members.includes(projectId))
  return owner ? owner.id : null
}

// bandsFor splits the ordered projects into the runs the sidebar renders. Group
// members arrive contiguously, so a single pass over the projects is enough.
export function bandsFor(projects: Project[], groups: ProjectGroup[]): Band[] {
  const bands: Band[] = []
  for (const project of projects) {
    const owner = groups.find((group) => group.members.includes(project.id)) ?? null
    const previous = bands[bands.length - 1]
    if (owner && previous && previous.group?.id === owner.id) {
      previous.projects.push(project)
      continue
    }
    bands.push({ group: owner, projects: [project] })
  }
  return bands
}

// rebuild recomputes each group's members from the order and a membership map, and
// drops groups left empty — a group whose last project was dragged out disappears.
function rebuild(order: string[], groups: ProjectGroup[], membership: Map<string, string>): Layout {
  const rebuilt = groups
    .map((group) => ({ ...group, members: order.filter((identifier) => membership.get(identifier) === group.id) }))
    .filter((group) => group.members.length > 0)
  return { order, groups: rebuilt }
}

// membershipOf indexes which group each project belongs to, optionally ignoring one
// project that is about to be placed somewhere else.
function membershipOf(groups: ProjectGroup[], ignoreId?: string): Map<string, string> {
  const membership = new Map<string, string>()
  for (const group of groups) {
    for (const member of group.members) {
      if (member !== ignoreId) {
        membership.set(member, group.id)
      }
    }
  }
  return membership
}

// withProjectPlaced moves a project to the given position in the order and into the
// given group, or out of every group when the group is null.
function withProjectPlaced(layout: Layout, projectId: string, index: number, groupId: string | null): Layout {
  const order = layout.order.filter((identifier) => identifier !== projectId)
  order.splice(Math.max(0, Math.min(index, order.length)), 0, projectId)
  const membership = membershipOf(layout.groups, projectId)
  if (groupId) {
    membership.set(projectId, groupId)
  }
  return rebuild(order, layout.groups, membership)
}

// placeAtProject drops a dragged project onto another project's row: it takes that
// row's position and joins whatever group the row belongs to.
export function placeAtProject(layout: Layout, projectId: string, targetId: string): Layout {
  const without = layout.order.filter((identifier) => identifier !== projectId)
  const index = without.indexOf(targetId)
  return withProjectPlaced(layout, projectId, index === -1 ? without.length : index, groupIdOf(layout.groups, targetId))
}

// placeAtGroupHead drops a dragged project onto a group's header, making it the
// group's first member.
export function placeAtGroupHead(layout: Layout, projectId: string, groupId: string): Layout {
  const group = layout.groups.find((candidate) => candidate.id === groupId)
  if (!group) {
    return layout
  }
  const without = layout.order.filter((identifier) => identifier !== projectId)
  const first = group.members.find((member) => member !== projectId)
  const index = first ? without.indexOf(first) : without.length
  return withProjectPlaced(layout, projectId, index === -1 ? without.length : index, groupId)
}

// placeAtEnd drops a dragged project below the list, leaving any group it was in.
export function placeAtEnd(layout: Layout, projectId: string): Layout {
  return withProjectPlaced(layout, projectId, layout.order.length, null)
}

// withGroupMoved moves a whole group's block of members to sit at the given
// project's position, so a group can be dragged past the rows around it.
export function withGroupMoved(layout: Layout, groupId: string, targetId: string | null): Layout {
  const group = layout.groups.find((candidate) => candidate.id === groupId)
  if (!group) {
    return layout
  }
  const members = new Set(group.members)
  const order = layout.order.filter((identifier) => !members.has(identifier))
  const index = targetId === null ? order.length : order.indexOf(targetId)
  order.splice(index === -1 ? order.length : index, 0, ...group.members)
  return rebuild(order, layout.groups, membershipOf(layout.groups))
}

// withGroupCreated bands the given projects together in a new group placed where the
// first of them already sat, pulling in the rest from wherever they were.
export function withGroupCreated(layout: Layout, projectIds: string[], name: string): Layout {
  const members = layout.order.filter((identifier) => projectIds.includes(identifier))
  if (members.length === 0) {
    return layout
  }
  const group: ProjectGroup = { id: `group-${Date.now()}`, name, collapsed: false, members }
  const membership = membershipOf(layout.groups)
  for (const member of members) {
    membership.set(member, group.id)
  }
  // The block lands where the first of the chosen projects already sat. Everything
  // before that position is a project staying put, so the index carries over.
  const anchor = layout.order.findIndex((identifier) => members.includes(identifier))
  const order = layout.order.filter((identifier) => !members.includes(identifier))
  order.splice(anchor, 0, ...members)
  return rebuild(order, [...layout.groups, group], membership)
}

// withGroupDissolved removes a group, returning its projects to the top level in the
// positions they already occupy.
export function withGroupDissolved(layout: Layout, groupId: string): Layout {
  return { order: layout.order, groups: layout.groups.filter((group) => group.id !== groupId) }
}

// withGroupChanged applies a change to one group, used for renaming and collapsing.
export function withGroupChanged(layout: Layout, groupId: string, change: Partial<ProjectGroup>): Layout {
  return {
    order: layout.order,
    groups: layout.groups.map((group) => (group.id === groupId ? { ...group, ...change } : group))
  }
}
