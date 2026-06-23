package service

import (
	"precursor/internal/graph"
	"precursor/internal/layout"
	"precursor/internal/model"
)

// NodeView is a node enriched with the values the canvas needs but does not store:
// the derived status of task nodes and the computed canvas position.
type NodeView struct {
	ID           string             `json:"id"`
	Kind         model.NodeKind     `json:"kind"`
	Title        string             `json:"title"`
	BodyMarkdown string             `json:"bodyMarkdown"`
	Icon         string             `json:"icon"`
	ParentID     *string            `json:"parentId,omitempty"`
	ChildID      *string            `json:"childId,omitempty"`
	DecisionType model.DecisionType `json:"decisionType,omitempty"`
	OrderIndex   int                `json:"orderIndex"`
	Status       model.Status       `json:"status"`
	// DecisionCount and DecisionsCollapsed describe a task's decision link so the
	// canvas can show a collapse/expand badge with the hidden decision tally.
	DecisionCount      int     `json:"decisionCount"`
	DecisionsCollapsed bool    `json:"decisionsCollapsed"`
	X                  float64 `json:"x"`
	Y                  float64 `json:"y"`
}

// ProjectView is the full render-ready description of an open project.
type ProjectView struct {
	Project model.Project         `json:"project"`
	Nodes   []NodeView            `json:"nodes"`
	Edges   []layout.Edge         `json:"edges"`
	Bonds   []model.ProximityBond `json:"bonds"`
}

// buildView combines metadata, the graph, derived statuses, and the radial layout
// into a single render-ready view.
func buildView(project model.Project, projectGraph model.Graph) ProjectView {
	layoutResult := layout.Compute(projectGraph, layout.DefaultConfig())
	positions := make(map[string]layout.Placement, len(layoutResult.Placements))
	for _, placement := range layoutResult.Placements {
		positions[placement.NodeID] = placement
	}

	nodeViews := make([]NodeView, 0, len(projectGraph.Nodes))
	for _, node := range projectGraph.Nodes {
		// Nodes without a placement are hidden (the decisions of a collapsed task),
		// so they are left out of the render-ready view entirely.
		placement, placed := positions[node.ID]
		if !placed {
			continue
		}
		view := NodeView{
			ID:                 node.ID,
			Kind:               node.Kind,
			Title:              node.Title,
			BodyMarkdown:       node.BodyMarkdown,
			Icon:               node.Icon,
			ParentID:           node.ParentID,
			ChildID:            node.ChildID,
			DecisionType:       node.DecisionType,
			OrderIndex:         node.OrderIndex,
			DecisionsCollapsed: node.DecisionsCollapsed,
			X:                  placement.X,
			Y:                  placement.Y,
		}
		if node.Kind == model.KindTask {
			view.Status = graph.DeriveStatus(projectGraph.Nodes, node.ID)
			view.DecisionCount = len(graph.DecisionsFor(projectGraph.Nodes, node.ID))
		}
		nodeViews = append(nodeViews, view)
	}

	return ProjectView{
		Project: project,
		Nodes:   nodeViews,
		Edges:   layoutResult.Edges,
		Bonds:   projectGraph.ProximityBonds,
	}
}
