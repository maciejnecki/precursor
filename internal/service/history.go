package service

import (
	"precursor/internal/model"
	"precursor/internal/storage"
)

// historyDepth caps how many undo steps are retained. Snapshots hold a whole
// project graph, so the cap bounds memory on a long editing session; reaching back
// further than this is not something a user does with a keyboard shortcut.
const historyDepth = 50

// history is the undo/redo stack for the active project. It stores whole-graph
// snapshots rather than reversible operations, because every mutation already
// funnels through Service.mutate and a graph is small enough to copy wholesale.
// The stacks belong to one open project and are discarded when another is opened.
type history struct {
	undoStack []model.Graph
	redoStack []model.Graph
}

// clear discards all recorded history, used when the active project changes.
func (recorded *history) clear() {
	recorded.undoStack = nil
	recorded.redoStack = nil
}

// push appends a snapshot to a stack, evicting the oldest entry once the stack is
// at its cap so the stack never grows without bound.
func push(stack []model.Graph, snapshot model.Graph) []model.Graph {
	stack = append(stack, snapshot)
	if len(stack) > historyDepth {
		stack = stack[len(stack)-historyDepth:]
	}
	return stack
}

// pop removes and returns the most recent snapshot, reporting whether one existed.
func pop(stack []model.Graph) ([]model.Graph, model.Graph, bool) {
	if len(stack) == 0 {
		return stack, model.Graph{}, false
	}
	last := len(stack) - 1
	return stack[:last], stack[last], true
}

// recordMutation stores the pre-mutation snapshot as the next undo step. Applying a
// fresh mutation invalidates any redo steps, since they describe a future that the
// new mutation has just diverged from.
func (service *Service) recordMutation(before model.Graph) {
	service.history.undoStack = push(service.history.undoStack, before)
	service.history.redoStack = nil
}

// Undo restores the project to the state before the most recent mutation, moving
// that mutation onto the redo stack. With nothing to undo it returns the current
// view unchanged, so the menu item and shortcut are harmless no-ops rather than
// errors, matching how every other inapplicable action behaves.
func (service *Service) Undo() (ProjectView, error) {
	return service.stepHistory(&service.history.undoStack, &service.history.redoStack)
}

// Redo re-applies the most recently undone mutation, moving it back onto the undo
// stack. Like Undo, it is a no-op when there is nothing to redo.
func (service *Service) Redo() (ProjectView, error) {
	return service.stepHistory(&service.history.redoStack, &service.history.undoStack)
}

// stepHistory moves one snapshot from the source stack onto the destination stack,
// restoring the project to it. Undo and redo are the same operation with the stacks
// swapped: the state being left behind always becomes the way back. It deliberately
// does not go through mutate, which would record the restore as a new mutation.
func (service *Service) stepHistory(source, destination *[]model.Graph) (ProjectView, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.active == nil {
		return ProjectView{}, errNoActiveProject
	}
	remaining, snapshot, found := pop(*source)
	if !found {
		return service.activeView()
	}
	current, graphError := service.active.Graph()
	if graphError != nil {
		return ProjectView{}, graphError
	}
	restoreError := service.active.WithinTransaction(func(repository *storage.Repository) error {
		return repository.ReplaceGraph(snapshot)
	})
	if restoreError != nil {
		return ProjectView{}, restoreError
	}
	*source = remaining
	*destination = push(*destination, current)
	return service.activeView()
}
