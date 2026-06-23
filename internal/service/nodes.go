package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"precursor/internal/graph"
	"precursor/internal/model"
	"precursor/internal/storage"
)

// now returns the current time in UTC, the single time source for new records.
func now() time.Time {
	return time.Now().UTC()
}

// newTaskNode builds a task node with a fresh identifier and timestamps.
func newTaskNode(title, body, icon string, parentID *string) model.Node {
	moment := now()
	return model.Node{
		ID:           uuid.NewString(),
		Kind:         model.KindTask,
		Title:        title,
		BodyMarkdown: body,
		Icon:         icon,
		ParentID:     parentID,
		CreatedAt:    moment,
		UpdatedAt:    moment,
	}
}

// newDecisionNode builds a decision node documenting a task at a sequence index.
func newDecisionNode(childID string, decisionType model.DecisionType, title, body, icon string, orderIndex int) model.Node {
	moment := now()
	if decisionType == "" {
		decisionType = model.DecisionPlain
	}
	owned := childID
	return model.Node{
		ID:           uuid.NewString(),
		Kind:         model.KindDecision,
		Title:        title,
		BodyMarkdown: body,
		Icon:         icon,
		ChildID:      &owned,
		DecisionType: decisionType,
		OrderIndex:   orderIndex,
		CreatedAt:    moment,
		UpdatedAt:    moment,
	}
}

// mutate runs an operation against the active project's repository and returns the
// refreshed view. It centralises the locking and active-project checks.
func (service *Service) mutate(operation func(repository *storage.Repository) error) (ProjectView, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.active == nil {
		return ProjectView{}, errNoActiveProject
	}
	if operationError := operation(service.active); operationError != nil {
		return ProjectView{}, operationError
	}
	return service.activeView()
}

// CreateTask adds a new endpoint task to the active project.
func (service *Service) CreateTask(title, body, icon string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		return repository.InsertNode(newTaskNode(title, body, icon, nil))
	})
}

// CreatePrecursor adds a precursor task to the given parent, enforcing the rule
// that a parent may have at most one precursor.
func (service *Service) CreatePrecursor(parentID, title, body, icon string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		nodes, nodesError := repository.Nodes()
		if nodesError != nil {
			return nodesError
		}
		if graph.HasPrecursor(nodes, parentID) {
			return errors.New("the selected task already has a precursor")
		}
		return repository.InsertNode(newTaskNode(title, body, icon, &parentID))
	})
}

// CreateDecision adds a decision documenting the given task, sequenced after any
// existing decisions on that task.
func (service *Service) CreateDecision(childID, decisionType, title, body, icon string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		childNode, readError := repository.Node(childID)
		if readError != nil {
			return readError
		}
		// The endpoint (a task with no parent) is the final parent of its chain and
		// has no transition to document, so it cannot take decisions.
		if childNode.Kind == model.KindTask && childNode.ParentID == nil {
			return errors.New("the final task in a chain cannot take decisions")
		}
		nodes, nodesError := repository.Nodes()
		if nodesError != nil {
			return nodesError
		}
		if availableError := decisionTypeAvailable(nodes, childID, decisionType); availableError != nil {
			return availableError
		}
		orderIndex := graph.NextDecisionOrderIndex(nodes, childID)
		decision := newDecisionNode(childID, model.DecisionType(decisionType), title, body, icon, orderIndex)
		return repository.InsertNode(decision)
	})
}

// CreateDecisionAfter inserts a decision immediately downstream of an existing
// decision (toward the parent), documenting the same task and shifting later
// decisions to keep the sequence contiguous.
func (service *Service) CreateDecisionAfter(decisionID, decisionType, title, body, icon string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		decisionNode, readError := repository.Node(decisionID)
		if readError != nil {
			return readError
		}
		if decisionNode.Kind != model.KindDecision || decisionNode.ChildID == nil {
			return errors.New("select a decision to insert after")
		}
		childID := *decisionNode.ChildID
		nodes, nodesError := repository.Nodes()
		if nodesError != nil {
			return nodesError
		}
		if availableError := decisionTypeAvailable(nodes, childID, decisionType); availableError != nil {
			return availableError
		}

		// Shift decisions sitting after the selected one to open a slot for the new one.
		for _, node := range nodes {
			if node.Kind == model.KindDecision && node.ChildID != nil && *node.ChildID == childID && node.OrderIndex > decisionNode.OrderIndex {
				node.OrderIndex++
				node.UpdatedAt = now()
				if updateError := repository.UpdateNode(node); updateError != nil {
					return updateError
				}
			}
		}
		decision := newDecisionNode(childID, model.DecisionType(decisionType), title, body, icon, decisionNode.OrderIndex+1)
		return repository.InsertNode(decision)
	})
}

// decisionTypeAvailable rejects a second Done or Redundant decision on a task, as
// each terminal outcome may be recorded only once.
func decisionTypeAvailable(nodes []model.Node, childID, decisionType string) error {
	if decisionType != string(model.DecisionDone) && decisionType != string(model.DecisionRedundant) {
		return nil
	}
	for _, decision := range graph.DecisionsFor(nodes, childID) {
		if string(decision.DecisionType) == decisionType {
			return errors.New("only one " + decisionType + " decision is allowed per task")
		}
	}
	return nil
}

// UpdateNode changes the editable content of an existing node, leaving its kind,
// relationships, and derived status untouched.
func (service *Service) UpdateNode(identifier, title, body, icon string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		node, readError := repository.Node(identifier)
		if readError != nil {
			return readError
		}
		node.Title = title
		node.BodyMarkdown = body
		node.Icon = icon
		node.UpdatedAt = now()
		return repository.UpdateNode(node)
	})
}

// SetDecisionsCollapsed toggles whether a task hides the decisions on the link to
// its parent, persisting the choice so it survives reopening the project.
func (service *Service) SetDecisionsCollapsed(identifier string, collapsed bool) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		node, readError := repository.Node(identifier)
		if readError != nil {
			return readError
		}
		node.DecisionsCollapsed = collapsed
		node.UpdatedAt = now()
		return repository.UpdateNode(node)
	})
}

// DeleteNode removes a node and applies the chain-healing change set in an order
// that never momentarily violates the one-precursor constraint.
func (service *Service) DeleteNode(identifier string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		projectGraph, graphError := repository.Graph()
		if graphError != nil {
			return graphError
		}
		changes := graph.DeleteNode(projectGraph, identifier)
		return applyChanges(repository, changes)
	})
}

// applyChanges persists a change set: deletions first so re-linked precursors do
// not collide with the node being removed, then node and bond updates.
func applyChanges(repository *storage.Repository, changes graph.ChangeSet) error {
	for _, deletedID := range changes.DeletedNodeIDs {
		if deletionError := repository.DeleteNode(deletedID); deletionError != nil {
			return deletionError
		}
	}
	for _, updatedNode := range changes.UpdatedNodes {
		updatedNode.UpdatedAt = now()
		if updateError := repository.UpdateNode(updatedNode); updateError != nil {
			return updateError
		}
	}
	for _, deletedBondID := range changes.DeletedBondIDs {
		if deletionError := repository.DeleteProximityBond(deletedBondID); deletionError != nil {
			return deletionError
		}
	}
	for _, updatedBond := range changes.UpdatedBonds {
		if updateError := repository.UpdateProximityBond(updatedBond); updateError != nil {
			return updateError
		}
	}
	return nil
}

// CreateProximity bonds the two chains that own the given nodes so they are laid
// out next to each other. Selecting two nodes in the same chain is rejected.
func (service *Service) CreateProximity(nodeAID, nodeBID string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		nodes, nodesError := repository.Nodes()
		if nodesError != nil {
			return nodesError
		}
		endpointA := graph.EndpointID(nodes, nodeAID)
		endpointB := graph.EndpointID(nodes, nodeBID)
		if endpointA == "" || endpointB == "" || endpointA == endpointB {
			return errors.New("select nodes in two different chains")
		}

		bonds, bondsError := repository.ProximityBonds()
		if bondsError != nil {
			return bondsError
		}
		for _, bond := range bonds {
			if bondsSamePair(bond, endpointA, endpointB) {
				return nil
			}
		}
		bond := model.ProximityBond{
			ID:          uuid.NewString(),
			EndpointAID: endpointA,
			EndpointBID: endpointB,
			CreatedAt:   now(),
		}
		return repository.InsertProximityBond(bond)
	})
}

// CreateProximityGroup bonds the chains of every given node so they cluster
// together in the layout. The distinct endpoints are bonded in sequence, which
// makes the union-find ordering place their pie-slices next to one another.
func (service *Service) CreateProximityGroup(nodeIDs []string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		nodes, nodesError := repository.Nodes()
		if nodesError != nil {
			return nodesError
		}

		// Resolve each selected node to its chain endpoint, keeping distinct ones in
		// the order they were selected.
		seen := make(map[string]bool)
		endpoints := make([]string, 0, len(nodeIDs))
		for _, nodeID := range nodeIDs {
			endpoint := graph.EndpointID(nodes, nodeID)
			if endpoint != "" && !seen[endpoint] {
				seen[endpoint] = true
				endpoints = append(endpoints, endpoint)
			}
		}
		if len(endpoints) < 2 {
			return errors.New("select tasks in at least two different chains")
		}

		bonds, bondsError := repository.ProximityBonds()
		if bondsError != nil {
			return bondsError
		}
		for index := 0; index < len(endpoints)-1; index++ {
			endpointA := endpoints[index]
			endpointB := endpoints[index+1]
			if hasBondPair(bonds, endpointA, endpointB) {
				continue
			}
			bond := model.ProximityBond{
				ID:          uuid.NewString(),
				EndpointAID: endpointA,
				EndpointBID: endpointB,
				CreatedAt:   now(),
			}
			if insertError := repository.InsertProximityBond(bond); insertError != nil {
				return insertError
			}
			bonds = append(bonds, bond)
		}
		return nil
	})
}

// hasBondPair reports whether the bonds already contain the given endpoint pair.
func hasBondPair(bonds []model.ProximityBond, endpointA, endpointB string) bool {
	for _, bond := range bonds {
		if bondsSamePair(bond, endpointA, endpointB) {
			return true
		}
	}
	return false
}

// DeleteProximity removes a proximity bond from the active project.
func (service *Service) DeleteProximity(identifier string) (ProjectView, error) {
	return service.mutate(func(repository *storage.Repository) error {
		return repository.DeleteProximityBond(identifier)
	})
}

// bondsSamePair reports whether a bond connects the given pair of endpoints in
// either order.
func bondsSamePair(bond model.ProximityBond, endpointA, endpointB string) bool {
	forward := bond.EndpointAID == endpointA && bond.EndpointBID == endpointB
	reverse := bond.EndpointAID == endpointB && bond.EndpointBID == endpointA
	return forward || reverse
}
