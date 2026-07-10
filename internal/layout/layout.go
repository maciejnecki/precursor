// Package layout turns a project's chains into concrete canvas coordinates and
// edges. Each chain is a column: its endpoint sits on the baseline and precursors
// stack upward in rows above the task they stem from, with decisions spaced down
// the segment between a precursor and its parent. Endpoints are laid out left to
// right; proximity bonds keep related chains in adjacent columns. The computation
// is deterministic and free of side effects.
package layout

import (
	"sort"

	"precursor/internal/graph"
	"precursor/internal/model"
)

// defaultRowStep is the vertical distance between successive tasks in a column.
const defaultRowStep = 160.0

// columnSpacing is the horizontal distance between neighbouring chains.
const columnSpacing = 340.0

// minSegmentGap is the smallest vertical distance allowed between two consecutive
// nodes sitting on the same segment (a task and its decisions). A segment grows
// past the row step when it carries enough decisions that they would otherwise
// overlap, so adding a decision re-opens the spacing.
const minSegmentGap = 96.0

// taskBodyExtraHeight approximates the extra height a task card gains from its
// clamped description preview. Positions anchor a node's top-left corner, so a
// segment's decisions are pushed down by this much when the task has a body,
// keeping the visual gap below the card the same as for a description-less task.
const taskBodyExtraHeight = 64.0

// Config holds the tunable geometry of the layout. RowStep is the base vertical
// distance between stacked tasks.
type Config struct {
	RowStep float64
}

// DefaultConfig returns the standard layout geometry.
func DefaultConfig() Config {
	return Config{RowStep: defaultRowStep}
}

// Placement is the computed canvas position of a single node.
type Placement struct {
	NodeID string  `json:"nodeId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// EdgeKind distinguishes a plain precursor link from a decision-annotated link.
type EdgeKind string

const (
	// EdgePrecursor is a direct parent-to-precursor link with no decisions on it.
	EdgePrecursor EdgeKind = "precursor"
	// EdgeDecision is a segment of a link that runs through decision nodes,
	// drawn in the precursor-to-parent transition direction.
	EdgeDecision EdgeKind = "decision"
)

// Edge is a directed connection between two nodes for the canvas to draw. TaskID is
// the task whose transition the edge belongs to, so the canvas can colour the whole
// link with that task's status colour.
type Edge struct {
	ID     string   `json:"id"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Kind   EdgeKind `json:"kind"`
	TaskID string   `json:"taskId"`
}

// Result is the full geometric description handed to the frontend.
type Result struct {
	Placements []Placement `json:"placements"`
	Edges      []Edge      `json:"edges"`
}

// Compute lays out every chain in the graph as a column and returns node
// placements and edges.
func Compute(projectGraph model.Graph, config Config) Result {
	if config.RowStep <= 0 {
		config.RowStep = defaultRowStep
	}
	nodes := projectGraph.Nodes
	orderedEndpoints := orderEndpoints(nodes, projectGraph.ProximityBonds)

	result := Result{Placements: make([]Placement, 0), Edges: make([]Edge, 0)}
	for columnIndex, endpoint := range orderedEndpoints {
		columnX := float64(columnIndex) * columnSpacing
		placeColumn(&result, nodes, endpoint, columnX, config.RowStep)
	}
	return result
}

// placeColumn positions a chain as a vertical column: the endpoint on the baseline
// and each precursor stacked above the task it stems from, with decisions spaced
// down the segment between a precursor and its parent. Segment lengths accumulate
// so only segments carrying decisions are widened; plain links stay close.
func placeColumn(result *Result, nodes []model.Node, endpoint model.Node, columnX, rowStep float64) {
	chain := graph.Chain(nodes, endpoint.ID)
	if len(chain) == 0 {
		return
	}

	// The endpoint sits on the baseline; its precursors climb upward (negative Y).
	appendPoint(result, endpoint.ID, columnX, 0)
	lowerY := 0.0
	for depth := 1; depth < len(chain); depth++ {
		taskNode := chain[depth]
		decisions := graph.DecisionsFor(nodes, taskNode.ID)
		// A collapsed task hides its decisions, so the segment carries none: it
		// stays at the base row step and a direct precursor edge is drawn instead.
		if taskNode.DecisionsCollapsed {
			decisions = nil
		}
		extraHeight := taskExtraHeight(taskNode, len(decisions))
		taskY := lowerY - segmentLength(len(decisions), rowStep, extraHeight)
		appendPoint(result, taskNode.ID, columnX, taskY)
		placeDecisions(result, taskNode, decisions, columnX, taskY+extraHeight)
		lowerY = taskY
	}
}

// taskExtraHeight estimates how much taller a task card renders than a bare header
// card. It only matters for spacing the decisions hanging off the task, so a
// decision-less segment reports zero and keeps the plain-link geometry.
func taskExtraHeight(taskNode model.Node, decisionCount int) float64 {
	if decisionCount == 0 || taskNode.BodyMarkdown == "" {
		return 0
	}
	return taskBodyExtraHeight
}

// segmentLength returns the vertical length of the segment leading to a task. A
// segment keeps the base row step unless its decisions, pushed down by the task
// card's extra height, need more room to sit a minimum gap apart, in which case
// it grows just enough to fit them.
func segmentLength(decisionCount int, baseStep, extraHeight float64) float64 {
	required := float64(decisionCount+1)*minSegmentGap + extraHeight
	if required > baseStep {
		return required
	}
	return baseStep
}

// placeDecisions positions the decisions documenting a task down the segment from
// the task toward its parent and wires the edges in flow direction. upperY is the
// task's position shifted by its extra card height, so every visual gap is the
// same minimum step: the task-to-first-decision spacing matches the spacing
// between decisions and between the last decision and the parent.
func placeDecisions(result *Result, taskNode model.Node, decisions []model.Node, columnX, upperY float64) {
	if len(decisions) == 0 {
		if taskNode.ParentID != nil {
			appendEdge(result, EdgePrecursor, taskNode.ID, *taskNode.ParentID, taskNode.ID)
		}
		return
	}

	previousNodeID := taskNode.ID
	for sequencePosition, decision := range decisions {
		decisionY := upperY + float64(sequencePosition+1)*minSegmentGap
		appendPoint(result, decision.ID, columnX, decisionY)
		appendEdge(result, EdgeDecision, previousNodeID, decision.ID, taskNode.ID)
		previousNodeID = decision.ID
	}
	if taskNode.ParentID != nil {
		appendEdge(result, EdgeDecision, previousNodeID, *taskNode.ParentID, taskNode.ID)
	}
}

// appendPoint records a node's canvas position.
func appendPoint(result *Result, nodeID string, x, y float64) {
	result.Placements = append(result.Placements, Placement{NodeID: nodeID, X: x, Y: y})
}

// appendEdge records a directed edge with a deterministic identifier, tagged with
// the task whose transition it belongs to.
func appendEdge(result *Result, kind EdgeKind, source, target, taskID string) {
	result.Edges = append(result.Edges, Edge{
		ID:     string(kind) + ":" + source + "->" + target,
		Source: source,
		Target: target,
		Kind:   kind,
		TaskID: taskID,
	})
}

// orderEndpoints returns the endpoints in the angular order they should occupy,
// keeping proximity-bonded chains contiguous. Endpoints are grouped by the
// connected component their bonds form; groups and members keep a stable order.
func orderEndpoints(nodes []model.Node, bonds []model.ProximityBond) []model.Node {
	endpoints := graph.Endpoints(nodes)
	sort.SliceStable(endpoints, func(first, second int) bool {
		if !endpoints[first].CreatedAt.Equal(endpoints[second].CreatedAt) {
			return endpoints[first].CreatedAt.Before(endpoints[second].CreatedAt)
		}
		return endpoints[first].ID < endpoints[second].ID
	})

	isEndpoint := make(map[string]bool, len(endpoints))
	for _, endpoint := range endpoints {
		isEndpoint[endpoint.ID] = true
	}

	roots := newUnionFind()
	for _, endpoint := range endpoints {
		roots.add(endpoint.ID)
	}
	for _, bond := range bonds {
		endpointA := graph.EndpointID(nodes, bond.EndpointAID)
		endpointB := graph.EndpointID(nodes, bond.EndpointBID)
		if isEndpoint[endpointA] && isEndpoint[endpointB] {
			roots.union(endpointA, endpointB)
		}
	}

	ordered := make([]model.Node, 0, len(endpoints))
	emitted := make(map[string]bool, len(endpoints))
	for _, endpoint := range endpoints {
		if emitted[endpoint.ID] {
			continue
		}
		group := roots.find(endpoint.ID)
		for _, candidate := range endpoints {
			if !emitted[candidate.ID] && roots.find(candidate.ID) == group {
				ordered = append(ordered, candidate)
				emitted[candidate.ID] = true
			}
		}
	}
	return ordered
}
