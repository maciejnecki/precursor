package layout

import (
	"math"
	"reflect"
	"testing"
	"time"

	"precursor/internal/model"
)

// stringPointer returns a pointer to the given string for optional fields.
func stringPointer(value string) *string {
	return &value
}

// task builds a task node created at a fixed offset so ordering is deterministic.
func task(identifier string, parentID *string, secondsOffset int) model.Node {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return model.Node{
		ID:        identifier,
		Kind:      model.KindTask,
		ParentID:  parentID,
		CreatedAt: base.Add(time.Duration(secondsOffset) * time.Second),
	}
}

// decision builds a decision node documenting a task at a sequence index.
func decision(identifier, childID string, orderIndex int) model.Node {
	return model.Node{
		ID:           identifier,
		Kind:         model.KindDecision,
		ChildID:      stringPointer(childID),
		DecisionType: model.DecisionDone,
		OrderIndex:   orderIndex,
	}
}

// placementFor finds the placement of a node by identifier.
func placementFor(result Result, nodeID string) (Placement, bool) {
	for _, placement := range result.Placements {
		if placement.NodeID == nodeID {
			return placement, true
		}
	}
	return Placement{}, false
}

// TestColumnStacksUpward verifies the endpoint sits on the baseline and its
// precursor is placed above it in the same column.
func TestColumnStacksUpward(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("endpoint", nil, 0),
		task("precursor", stringPointer("endpoint"), 1),
	}}
	result := Compute(graph, DefaultConfig())

	endpoint, _ := placementFor(result, "endpoint")
	precursor, _ := placementFor(result, "precursor")
	if math.Abs(endpoint.Y) > 0.001 {
		test.Fatalf("expected endpoint on the baseline, got Y %.3f", endpoint.Y)
	}
	if precursor.Y >= endpoint.Y {
		test.Fatalf("expected precursor above endpoint, got precursor Y %.3f endpoint Y %.3f", precursor.Y, endpoint.Y)
	}
	if math.Abs(precursor.X-endpoint.X) > 0.001 {
		test.Fatalf("expected precursor in the same column, got X %.3f vs %.3f", precursor.X, endpoint.X)
	}
}

// TestEndpointsInColumns verifies separate chains occupy distinct, evenly spaced columns.
func TestEndpointsInColumns(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("a", nil, 0),
		task("b", nil, 1),
	}}
	result := Compute(graph, DefaultConfig())
	first, _ := placementFor(result, "a")
	second, _ := placementFor(result, "b")
	if math.Abs(first.X-second.X) < 0.001 {
		test.Fatalf("expected endpoints in different columns, both at X %.3f", first.X)
	}
}

// TestPlainPrecursorEdge verifies a link with no decisions produces one precursor edge.
func TestPlainPrecursorEdge(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("endpoint", nil, 0),
		task("precursor", stringPointer("endpoint"), 1),
	}}
	result := Compute(graph, DefaultConfig())
	if len(result.Edges) != 1 {
		test.Fatalf("expected 1 edge, got %d: %+v", len(result.Edges), result.Edges)
	}
	edge := result.Edges[0]
	if edge.Kind != EdgePrecursor || edge.Source != "precursor" || edge.Target != "endpoint" {
		test.Fatalf("unexpected edge: %+v", edge)
	}
}

// TestDecisionsOnLink verifies decisions sit between task and parent and chain edges.
func TestDecisionsOnLink(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("endpoint", nil, 0),
		task("precursor", stringPointer("endpoint"), 1),
		decision("decisionOne", "precursor", 0),
		decision("decisionTwo", "precursor", 1),
	}}
	result := Compute(graph, DefaultConfig())

	precursor, _ := placementFor(result, "precursor")
	endpoint, _ := placementFor(result, "endpoint")
	decisionOne, foundOne := placementFor(result, "decisionOne")
	decisionTwo, foundTwo := placementFor(result, "decisionTwo")
	if !foundOne || !foundTwo {
		test.Fatalf("expected both decisions placed")
	}
	// Decisions descend from the precursor (upper) toward the endpoint (lower), so
	// the first sits just below the precursor and the second nearer the endpoint.
	if !(precursor.Y < decisionOne.Y && decisionOne.Y < decisionTwo.Y && decisionTwo.Y < endpoint.Y) {
		test.Fatalf("expected precursor < decisionOne < decisionTwo < endpoint in Y, got %.1f %.1f %.1f %.1f",
			precursor.Y, decisionOne.Y, decisionTwo.Y, endpoint.Y)
	}

	decisionEdges := 0
	for _, edge := range result.Edges {
		if edge.Kind == EdgeDecision {
			decisionEdges++
		}
	}
	if decisionEdges != 3 {
		test.Fatalf("expected 3 decision edges (precursor->d1->d2->endpoint), got %d", decisionEdges)
	}
}

// TestBodyPushesDecisionsDown verifies a task with a description places its first
// decision farther below than a description-less task does, compensating for the
// taller card so the visual gap stays the same.
func TestBodyPushesDecisionsDown(test *testing.T) {
	plainGraph := model.Graph{Nodes: []model.Node{
		task("endpoint", nil, 0),
		task("precursor", stringPointer("endpoint"), 1),
		decision("decisionOne", "precursor", 0),
	}}
	describedNodes := []model.Node{
		task("endpoint", nil, 0),
		task("precursor", stringPointer("endpoint"), 1),
		decision("decisionOne", "precursor", 0),
	}
	describedNodes[1].BodyMarkdown = "a long description that the card truncates"
	describedGraph := model.Graph{Nodes: describedNodes}

	plainResult := Compute(plainGraph, DefaultConfig())
	describedResult := Compute(describedGraph, DefaultConfig())

	plainTask, _ := placementFor(plainResult, "precursor")
	plainDecision, _ := placementFor(plainResult, "decisionOne")
	describedTask, _ := placementFor(describedResult, "precursor")
	describedDecision, _ := placementFor(describedResult, "decisionOne")

	plainGap := plainDecision.Y - plainTask.Y
	describedGap := describedDecision.Y - describedTask.Y
	if describedGap <= plainGap {
		test.Fatalf("expected the described task's decision pushed farther down, got gap %.1f vs plain %.1f", describedGap, plainGap)
	}
	if math.Abs((describedGap-plainGap)-taskBodyExtraHeight) > 0.001 {
		test.Fatalf("expected the gap to grow by the body height %.1f, grew by %.1f", taskBodyExtraHeight, describedGap-plainGap)
	}
}

// TestDeterministic verifies identical input yields identical output.
func TestDeterministic(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("a", nil, 0),
		task("b", nil, 1),
		task("aInner", stringPointer("a"), 2),
	}}
	first := Compute(graph, DefaultConfig())
	second := Compute(graph, DefaultConfig())
	if !reflect.DeepEqual(first, second) {
		test.Fatalf("layout is not deterministic")
	}
}

// TestProximityKeepsBondedChainsAdjacent verifies a proximity bond makes two chains
// occupy neighbouring angular slots among several endpoints.
func TestProximityKeepsBondedChainsAdjacent(test *testing.T) {
	graph := model.Graph{
		Nodes: []model.Node{
			task("a", nil, 0),
			task("b", nil, 1),
			task("c", nil, 2),
			task("d", nil, 3),
		},
		ProximityBonds: []model.ProximityBond{
			{ID: "bond", EndpointAID: "a", EndpointBID: "d"},
		},
	}
	ordered := orderEndpoints(graph.Nodes, graph.ProximityBonds)
	positions := make(map[string]int, len(ordered))
	for index, endpoint := range ordered {
		positions[endpoint.ID] = index
	}
	separation := positions["a"] - positions["d"]
	if separation < 0 {
		separation = -separation
	}
	if separation != 1 {
		test.Fatalf("expected bonded endpoints a and d to be adjacent, got positions %+v", positions)
	}
}
