// Package graph holds the pure logic that interprets a project's nodes as
// finish-to-start chains: deriving task status from decisions, walking chains,
// and computing the changes needed to delete a node while keeping chains intact.
// Every function is side-effect free so it can be tested without a database.
package graph

import (
	"sort"

	"precursor/internal/model"
)

// statusForDecisionType maps a typed decision to the status it confers on the
// task it documents. Plain decisions confer no status, so ok is false for them.
func statusForDecisionType(decisionType model.DecisionType) (model.Status, bool) {
	switch decisionType {
	case model.DecisionScheduled:
		return model.StatusScheduled, true
	case model.DecisionInProgress:
		return model.StatusInProgress, true
	case model.DecisionDone:
		return model.StatusDone, true
	case model.DecisionRedundant:
		return model.StatusRedundant, true
	default:
		return "", false
	}
}

// nodeByID returns the node with the given identifier, if present.
func nodeByID(nodes []model.Node, identifier string) (model.Node, bool) {
	for _, node := range nodes {
		if node.ID == identifier {
			return node, true
		}
	}
	return model.Node{}, false
}

// DecisionsFor returns the decision nodes documenting the given task, ordered by
// their sequence index and then by creation time for stable tie-breaking.
func DecisionsFor(nodes []model.Node, taskID string) []model.Node {
	decisions := make([]model.Node, 0)
	for _, node := range nodes {
		if node.Kind == model.KindDecision && node.ChildID != nil && *node.ChildID == taskID {
			decisions = append(decisions, node)
		}
	}
	sort.SliceStable(decisions, func(first, second int) bool {
		if decisions[first].OrderIndex != decisions[second].OrderIndex {
			return decisions[first].OrderIndex < decisions[second].OrderIndex
		}
		return decisions[first].CreatedAt.Before(decisions[second].CreatedAt)
	})
	return decisions
}

// DeriveStatus computes a task's current status as the status conferred by its
// most recent typed decision, defaulting to scheduled when none exists. An endpoint
// takes no decisions, so its status instead reflects its chain: it is automatically
// done once every precursor is resolved (see endpointStatus).
func DeriveStatus(nodes []model.Node, taskID string) model.Status {
	decisions := DecisionsFor(nodes, taskID)
	node, found := nodeByID(nodes, taskID)
	if found && node.Kind == model.KindTask && node.ParentID == nil && len(decisions) == 0 {
		return endpointStatus(nodes, taskID)
	}
	status := model.StatusScheduled
	for _, decision := range decisions {
		conferred, ok := statusForDecisionType(decision.DecisionType)
		if ok {
			status = conferred
		}
	}
	return status
}

// endpointStatus marks a chain's endpoint done once all of its precursors are
// resolved (each one done or redundant) with at least one actually done. Redundant
// precursors are set aside and never block completion. With no precursors, or any
// precursor still scheduled or in progress, the endpoint stays scheduled.
func endpointStatus(nodes []model.Node, endpointID string) model.Status {
	precursors := Chain(nodes, endpointID)[1:]
	if len(precursors) == 0 {
		return model.StatusScheduled
	}
	anyDone := false
	for _, precursor := range precursors {
		switch DeriveStatus(nodes, precursor.ID) {
		case model.StatusDone:
			anyDone = true
		case model.StatusRedundant:
		default:
			return model.StatusScheduled
		}
	}
	if anyDone {
		return model.StatusDone
	}
	return model.StatusScheduled
}

// NextDecisionOrderIndex returns the sequence index to assign to a new decision on
// the given task, placing it after every existing decision.
func NextDecisionOrderIndex(nodes []model.Node, taskID string) int {
	next := 0
	for _, decision := range DecisionsFor(nodes, taskID) {
		if decision.OrderIndex >= next {
			next = decision.OrderIndex + 1
		}
	}
	return next
}

// PrecursorOf returns the task whose parent is the given task, that is, the single
// precursor of the given task, if one exists.
func PrecursorOf(nodes []model.Node, taskID string) (model.Node, bool) {
	for _, node := range nodes {
		if node.Kind == model.KindTask && node.ParentID != nil && *node.ParentID == taskID {
			return node, true
		}
	}
	return model.Node{}, false
}

// HasPrecursor reports whether the given task already has a precursor, which the
// editor uses to keep the one-precursor-per-parent rule.
func HasPrecursor(nodes []model.Node, taskID string) bool {
	_, exists := PrecursorOf(nodes, taskID)
	return exists
}

// Endpoints returns every endpoint task: a task with no parent, which sits at the
// centre of the radial layout.
func Endpoints(nodes []model.Node) []model.Node {
	endpoints := make([]model.Node, 0)
	for _, node := range nodes {
		if node.Kind == model.KindTask && node.ParentID == nil {
			endpoints = append(endpoints, node)
		}
	}
	return endpoints
}

// EndpointID walks parent links from the given task up to the chain root and
// returns the endpoint's identifier.
func EndpointID(nodes []model.Node, taskID string) string {
	current, found := nodeByID(nodes, taskID)
	if !found {
		return ""
	}
	for current.ParentID != nil {
		parent, parentFound := nodeByID(nodes, *current.ParentID)
		if !parentFound {
			break
		}
		current = parent
	}
	return current.ID
}

// Chain returns the tasks of the chain containing the given task, ordered from the
// endpoint outward to the most distant precursor.
func Chain(nodes []model.Node, taskID string) []model.Node {
	endpointID := EndpointID(nodes, taskID)
	chain := make([]model.Node, 0)
	current, found := nodeByID(nodes, endpointID)
	for found {
		chain = append(chain, current)
		precursor, hasPrecursor := PrecursorOf(nodes, current.ID)
		if !hasPrecursor {
			break
		}
		current = precursor
		found = true
	}
	return chain
}

// ChangeSet describes the persistence operations that realise a graph mutation.
// The application layer applies it against the repository.
type ChangeSet struct {
	UpdatedNodes   []model.Node
	DeletedNodeIDs []string
	UpdatedBonds   []model.ProximityBond
	DeletedBondIDs []string
}

// DeleteNode computes the changes required to delete the target node. Deleting a
// decision simply removes it. Deleting a task heals its chain: the task's
// precursor re-links to the task's parent, decisions on the removed links are
// discarded, and proximity bonds that referenced a removed endpoint are repointed
// to the new endpoint or dropped.
func DeleteNode(projectGraph model.Graph, targetID string) ChangeSet {
	nodes := projectGraph.Nodes
	target, found := nodeByID(nodes, targetID)
	if !found {
		return ChangeSet{}
	}

	if target.Kind == model.KindDecision {
		return ChangeSet{DeletedNodeIDs: []string{targetID}}
	}

	changes := ChangeSet{DeletedNodeIDs: []string{targetID}}

	// Discard every decision that documented the task being deleted, because the
	// link those decisions annotated no longer exists.
	for _, decision := range DecisionsFor(nodes, targetID) {
		changes.DeletedNodeIDs = append(changes.DeletedNodeIDs, decision.ID)
	}

	precursor, hasPrecursor := PrecursorOf(nodes, targetID)
	if hasPrecursor {
		// Heal the chain by re-linking the precursor to the deleted task's parent,
		// and discard the decisions that annotated the now-removed precursor link.
		precursor.ParentID = target.ParentID
		changes.UpdatedNodes = append(changes.UpdatedNodes, precursor)
		for _, decision := range DecisionsFor(nodes, precursor.ID) {
			changes.DeletedNodeIDs = append(changes.DeletedNodeIDs, decision.ID)
		}
	}

	changes.applyBondHealing(projectGraph.ProximityBonds, target, precursor, hasPrecursor)
	return changes
}

// applyBondHealing repoints or drops proximity bonds that referenced the deleted
// task. When the deleted task was an endpoint, its precursor becomes the new
// endpoint and bonds are repointed to it; otherwise such bonds are dropped.
func (changes *ChangeSet) applyBondHealing(
	bonds []model.ProximityBond,
	target model.Node,
	precursor model.Node,
	hasPrecursor bool,
) {
	targetWasEndpoint := target.ParentID == nil
	for _, bond := range bonds {
		referencesTarget := bond.EndpointAID == target.ID || bond.EndpointBID == target.ID
		if !referencesTarget {
			continue
		}
		if targetWasEndpoint && hasPrecursor {
			repointed := bond
			if repointed.EndpointAID == target.ID {
				repointed.EndpointAID = precursor.ID
			}
			if repointed.EndpointBID == target.ID {
				repointed.EndpointBID = precursor.ID
			}
			changes.UpdatedBonds = append(changes.UpdatedBonds, repointed)
			continue
		}
		changes.DeletedBondIDs = append(changes.DeletedBondIDs, bond.ID)
	}
}
