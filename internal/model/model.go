// Package model defines the core domain types shared across the storage, graph,
// layout, and export layers. The types deliberately contain no behaviour so that
// they can be reused freely without creating dependencies between packages.
package model

import "time"

// NodeKind distinguishes the two kinds of nodes that live on a project canvas.
type NodeKind string

const (
	// KindTask marks a task node: a unit of work in a finish-to-start chain.
	KindTask NodeKind = "task"
	// KindDecision marks a decision node: a record of a transition that sits
	// between a precursor task and its parent.
	KindDecision NodeKind = "decision"
)

// Status is the lifecycle state of a task node. It is never stored directly; it
// is derived from the task's typed decisions by the graph layer.
type Status string

const (
	// StatusScheduled is the default state of a task that has no typed decision yet.
	StatusScheduled Status = "scheduled"
	// StatusInProgress marks a task that work has started on.
	StatusInProgress Status = "in_progress"
	// StatusDone marks a completed task.
	StatusDone Status = "done"
	// StatusRedundant marks a task that is no longer needed.
	StatusRedundant Status = "redundant"
)

// DecisionType classifies a decision node. A typed decision (anything other than
// DecisionPlain) drives the status of the precursor task it documents; a plain
// decision records design intent only and changes no status.
type DecisionType string

const (
	// DecisionScheduled sets the documented task back to the scheduled state.
	DecisionScheduled DecisionType = "scheduled"
	// DecisionInProgress moves the documented task into progress.
	DecisionInProgress DecisionType = "in_progress"
	// DecisionDone marks the documented task as done.
	DecisionDone DecisionType = "done"
	// DecisionRedundant marks the documented task as redundant.
	DecisionRedundant DecisionType = "redundant"
	// DecisionPlain records design intent without changing any status.
	DecisionPlain DecisionType = "plain"
)

// Project holds the metadata of a single project, mirroring the meta row of its
// database. The ID is the database file stem and acts as a stable handle.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Colour      string    `json:"colour"`
	Icon        string    `json:"icon"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Node is the unified record for both task and decision nodes. Which fields are
// meaningful depends on Kind: task nodes use ParentID, decision nodes use
// ChildID, DecisionType, and OrderIndex. DecisionsCollapsed applies to task nodes
// and hides the decisions on the link leading to the task's parent.
type Node struct {
	ID                 string       `json:"id"`
	Kind               NodeKind     `json:"kind"`
	Title              string       `json:"title"`
	BodyMarkdown       string       `json:"bodyMarkdown"`
	Icon               string       `json:"icon"`
	ParentID           *string      `json:"parentId,omitempty"`
	ChildID            *string      `json:"childId,omitempty"`
	DecisionType       DecisionType `json:"decisionType,omitempty"`
	OrderIndex         int          `json:"orderIndex"`
	DecisionsCollapsed bool         `json:"decisionsCollapsed"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
}

// ProximityBond requests that the two chains owning the referenced endpoint tasks
// be placed next to each other in the radial layout.
type ProximityBond struct {
	ID          string    `json:"id"`
	EndpointAID string    `json:"endpointAId"`
	EndpointBID string    `json:"endpointBId"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Graph is the full content of one project: every node plus every proximity bond.
// It is the unit passed to the graph and layout layers and serialized for backup.
type Graph struct {
	Nodes           []Node          `json:"nodes"`
	ProximityBonds  []ProximityBond `json:"proximityBonds"`
}
