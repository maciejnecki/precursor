package graph

import (
	"testing"
	"time"

	"precursor/internal/model"
)

// stringPointer returns a pointer to the given string for optional fields.
func stringPointer(value string) *string {
	return &value
}

// task builds a task node with an optional parent.
func task(identifier string, parentID *string) model.Node {
	return model.Node{ID: identifier, Kind: model.KindTask, ParentID: parentID, CreatedAt: time.Now().UTC()}
}

// decision builds a decision node documenting a task at a sequence index.
func decision(identifier, childID string, decisionType model.DecisionType, orderIndex int) model.Node {
	return model.Node{
		ID:           identifier,
		Kind:         model.KindDecision,
		ChildID:      stringPointer(childID),
		DecisionType: decisionType,
		OrderIndex:   orderIndex,
		CreatedAt:    time.Now().UTC(),
	}
}

// idSet collapses a slice of identifiers into a set for order-independent checks.
func idSet(identifiers []string) map[string]bool {
	set := make(map[string]bool, len(identifiers))
	for _, identifier := range identifiers {
		set[identifier] = true
	}
	return set
}

// TestDeriveStatusDefaultsToScheduled verifies a task with no decisions is scheduled.
func TestDeriveStatusDefaultsToScheduled(test *testing.T) {
	nodes := []model.Node{task("alpha", nil)}
	if status := DeriveStatus(nodes, "alpha"); status != model.StatusScheduled {
		test.Fatalf("expected scheduled, got %q", status)
	}
}

// TestDeriveStatusMostRecentTypedWins verifies the latest typed decision sets status
// and that a trailing plain decision does not override it.
func TestDeriveStatusMostRecentTypedWins(test *testing.T) {
	nodes := []model.Node{
		task("alpha", nil),
		decision("d1", "alpha", model.DecisionInProgress, 0),
		decision("d2", "alpha", model.DecisionDone, 1),
		decision("d3", "alpha", model.DecisionPlain, 2),
	}
	if status := DeriveStatus(nodes, "alpha"); status != model.StatusDone {
		test.Fatalf("expected done, got %q", status)
	}
}

// TestEndpointDoneWhenPrecursorsResolved verifies an endpoint with no decisions is
// automatically done once every precursor is resolved, treating redundant
// precursors as set aside and staying scheduled while any precursor is unresolved.
func TestEndpointDoneWhenPrecursorsResolved(test *testing.T) {
	// Endpoint with one done and one redundant precursor: the endpoint is done.
	complete := []model.Node{
		task("endpoint", nil),
		task("first", stringPointer("endpoint")),
		task("second", stringPointer("first")),
		decision("firstDone", "first", model.DecisionDone, 0),
		decision("secondRedundant", "second", model.DecisionRedundant, 0),
	}
	if status := DeriveStatus(complete, "endpoint"); status != model.StatusDone {
		test.Fatalf("expected endpoint done when precursors resolved, got %q", status)
	}

	// One precursor still scheduled: the endpoint stays scheduled.
	pending := []model.Node{
		task("endpoint", nil),
		task("first", stringPointer("endpoint")),
		task("second", stringPointer("first")),
		decision("firstDone", "first", model.DecisionDone, 0),
	}
	if status := DeriveStatus(pending, "endpoint"); status != model.StatusScheduled {
		test.Fatalf("expected endpoint scheduled while a precursor is unresolved, got %q", status)
	}

	// Every precursor redundant with none done: the endpoint is not auto-completed.
	dropped := []model.Node{
		task("endpoint", nil),
		task("first", stringPointer("endpoint")),
		decision("firstRedundant", "first", model.DecisionRedundant, 0),
	}
	if status := DeriveStatus(dropped, "endpoint"); status != model.StatusScheduled {
		test.Fatalf("expected endpoint scheduled when all precursors redundant, got %q", status)
	}

	// A lone endpoint with no precursors stays scheduled.
	lone := []model.Node{task("endpoint", nil)}
	if status := DeriveStatus(lone, "endpoint"); status != model.StatusScheduled {
		test.Fatalf("expected lone endpoint scheduled, got %q", status)
	}
}

// TestChainOrdersEndpointFirst verifies a chain is returned from endpoint outward.
func TestChainOrdersEndpointFirst(test *testing.T) {
	nodes := []model.Node{
		task("endpoint", nil),
		task("middle", stringPointer("endpoint")),
		task("outer", stringPointer("middle")),
	}
	chain := Chain(nodes, "outer")
	if len(chain) != 3 || chain[0].ID != "endpoint" || chain[2].ID != "outer" {
		test.Fatalf("unexpected chain order: %+v", chain)
	}
}

// TestDeleteDecisionRemovesOnlyItself verifies deleting a decision is a simple removal.
func TestDeleteDecisionRemovesOnlyItself(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("alpha", nil),
		decision("d1", "alpha", model.DecisionDone, 0),
	}}
	changes := DeleteNode(graph, "d1")
	if len(changes.DeletedNodeIDs) != 1 || changes.DeletedNodeIDs[0] != "d1" {
		test.Fatalf("expected only d1 deleted, got %+v", changes.DeletedNodeIDs)
	}
	if len(changes.UpdatedNodes) != 0 {
		test.Fatalf("expected no node updates, got %+v", changes.UpdatedNodes)
	}
}

// TestDeleteMiddleTaskHealsChain verifies a mid-chain delete re-links the precursor
// to the parent and takes only the decisions documenting the deleted task. The
// precursor's own decisions document its own transition, so they survive and come to
// annotate the healed link.
func TestDeleteMiddleTaskHealsChain(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("endpoint", nil),
		task("middle", stringPointer("endpoint")),
		task("outer", stringPointer("middle")),
		decision("dMiddle", "middle", model.DecisionDone, 0),
		decision("dOuter", "outer", model.DecisionInProgress, 0),
	}}
	changes := DeleteNode(graph, "middle")

	if len(changes.UpdatedNodes) != 1 || changes.UpdatedNodes[0].ID != "outer" {
		test.Fatalf("expected only outer to be re-linked, got %+v", changes.UpdatedNodes)
	}
	if changes.UpdatedNodes[0].ParentID == nil || *changes.UpdatedNodes[0].ParentID != "endpoint" {
		test.Fatalf("expected outer's parent to become endpoint, got %+v", changes.UpdatedNodes[0].ParentID)
	}

	deleted := idSet(changes.DeletedNodeIDs)
	if !deleted["middle"] || !deleted["dMiddle"] {
		test.Fatalf("expected middle and its own decision deleted, got %+v", changes.DeletedNodeIDs)
	}
	if deleted["dOuter"] {
		test.Fatalf("expected the precursor's decision to survive, got %+v", changes.DeletedNodeIDs)
	}
}

// TestDeleteEndpointRemovesChainAndBond verifies deleting an endpoint deletes its
// whole chain rather than promoting a precursor, and drops the proximity bonds that
// referenced it, since no node survives to carry them.
func TestDeleteEndpointRemovesChainAndBond(test *testing.T) {
	graph := model.Graph{
		Nodes: []model.Node{
			task("endpointA", nil),
			task("innerA", stringPointer("endpointA")),
			task("endpointB", nil),
		},
		ProximityBonds: []model.ProximityBond{
			{ID: "bond", EndpointAID: "endpointA", EndpointBID: "endpointB"},
		},
	}
	changes := DeleteNode(graph, "endpointA")

	if len(changes.UpdatedNodes) != 0 {
		test.Fatalf("expected no node promoted, got %+v", changes.UpdatedNodes)
	}
	deleted := idSet(changes.DeletedNodeIDs)
	if !deleted["endpointA"] || !deleted["innerA"] {
		test.Fatalf("expected the whole chain deleted, got %+v", changes.DeletedNodeIDs)
	}
	if deleted["endpointB"] {
		test.Fatalf("expected the other chain untouched, got %+v", changes.DeletedNodeIDs)
	}
	if len(changes.UpdatedBonds) != 0 || len(changes.DeletedBondIDs) != 1 || changes.DeletedBondIDs[0] != "bond" {
		test.Fatalf("expected the bond dropped, got updated %+v deleted %+v", changes.UpdatedBonds, changes.DeletedBondIDs)
	}
}

// TestDeleteEndpointRemovesChainDecisions verifies that deleting an endpoint also
// removes the decisions documenting every task in the chain it takes with it.
func TestDeleteEndpointRemovesChainDecisions(test *testing.T) {
	graph := model.Graph{Nodes: []model.Node{
		task("endpoint", nil),
		task("middle", stringPointer("endpoint")),
		task("outer", stringPointer("middle")),
		decision("dMiddle", "middle", model.DecisionDone, 0),
		decision("dOuter", "outer", model.DecisionInProgress, 0),
	}}
	changes := DeleteNode(graph, "endpoint")

	deleted := idSet(changes.DeletedNodeIDs)
	for _, identifier := range []string{"endpoint", "middle", "outer", "dMiddle", "dOuter"} {
		if !deleted[identifier] {
			test.Fatalf("expected %q deleted, got %+v", identifier, changes.DeletedNodeIDs)
		}
	}
}

// TestDeleteLoneEndpointDropsBond verifies deleting an endpoint with no precursor
// drops proximity bonds that referenced it.
func TestDeleteLoneEndpointDropsBond(test *testing.T) {
	graph := model.Graph{
		Nodes: []model.Node{
			task("endpointA", nil),
			task("endpointB", nil),
		},
		ProximityBonds: []model.ProximityBond{
			{ID: "bond", EndpointAID: "endpointA", EndpointBID: "endpointB"},
		},
	}
	changes := DeleteNode(graph, "endpointA")
	if len(changes.DeletedBondIDs) != 1 || changes.DeletedBondIDs[0] != "bond" {
		test.Fatalf("expected bond dropped, got %+v", changes.DeletedBondIDs)
	}
}

// TestNextDecisionOrderIndex verifies new decisions are sequenced after existing ones.
func TestNextDecisionOrderIndex(test *testing.T) {
	nodes := []model.Node{
		task("alpha", nil),
		decision("d1", "alpha", model.DecisionInProgress, 0),
		decision("d2", "alpha", model.DecisionPlain, 1),
	}
	if next := NextDecisionOrderIndex(nodes, "alpha"); next != 2 {
		test.Fatalf("expected next index 2, got %d", next)
	}
}
