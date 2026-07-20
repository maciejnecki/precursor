package service

import "testing"

// openProjectForHistory creates and opens a project, returning the ready service.
func openProjectForHistory(test *testing.T) *Service {
	test.Helper()
	service := openService(test)
	project, creationError := service.CreateProject("History", "", "", "")
	if creationError != nil {
		test.Fatalf("CreateProject: %v", creationError)
	}
	if _, openError := service.OpenProject(project.ID); openError != nil {
		test.Fatalf("OpenProject: %v", openError)
	}
	return service
}

// TestUndoRedoRoundTrip verifies that undo restores the state before a mutation and
// redo re-applies it, including the decisions a chain deletion would have destroyed.
func TestUndoRedoRoundTrip(test *testing.T) {
	service := openProjectForHistory(test)

	view, _ := service.CreateTask("Endpoint", "", "")
	endpoint, _ := findTaskByTitle(view, "Endpoint")
	view, _ = service.CreatePrecursor(endpoint.ID, "Precursor", "", "")
	precursor, _ := findTaskByTitle(view, "Precursor")
	if _, decisionError := service.CreateDecision(precursor.ID, "done", "Built", "", ""); decisionError != nil {
		test.Fatalf("CreateDecision: %v", decisionError)
	}

	deleted, deleteError := service.DeleteNode(endpoint.ID)
	if deleteError != nil {
		test.Fatalf("DeleteNode: %v", deleteError)
	}
	if len(deleted.Nodes) != 0 {
		test.Fatalf("expected the chain deleted, got %+v", deleted.Nodes)
	}

	// Undo must bring back the whole chain, decision included.
	restored, undoError := service.Undo()
	if undoError != nil {
		test.Fatalf("Undo: %v", undoError)
	}
	if len(restored.Nodes) != 3 {
		test.Fatalf("expected the chain and its decision restored, got %+v", restored.Nodes)
	}
	if _, found := nodeViewByID(restored, precursor.ID); !found {
		test.Fatalf("expected the precursor restored")
	}

	redone, redoError := service.Redo()
	if redoError != nil {
		test.Fatalf("Redo: %v", redoError)
	}
	if len(redone.Nodes) != 0 {
		test.Fatalf("expected the deletion re-applied, got %+v", redone.Nodes)
	}
}

// TestUndoOnEmptyHistoryIsNoOp verifies undo and redo with nothing recorded return
// the current view rather than an error, so the shortcut is always safe to press.
func TestUndoOnEmptyHistoryIsNoOp(test *testing.T) {
	service := openProjectForHistory(test)

	view, undoError := service.Undo()
	if undoError != nil {
		test.Fatalf("Undo on empty history: %v", undoError)
	}
	if len(view.Nodes) != 0 {
		test.Fatalf("expected an empty project, got %+v", view.Nodes)
	}
	if _, redoError := service.Redo(); redoError != nil {
		test.Fatalf("Redo on empty history: %v", redoError)
	}
}

// TestMutationClearsRedoStack verifies that acting after an undo discards the redo
// steps, which describe a future the new mutation has diverged from.
func TestMutationClearsRedoStack(test *testing.T) {
	service := openProjectForHistory(test)

	service.CreateTask("First", "", "")
	if _, undoError := service.Undo(); undoError != nil {
		test.Fatalf("Undo: %v", undoError)
	}
	if _, createError := service.CreateTask("Second", "", ""); createError != nil {
		test.Fatalf("CreateTask: %v", createError)
	}

	view, redoError := service.Redo()
	if redoError != nil {
		test.Fatalf("Redo: %v", redoError)
	}
	if _, found := findTaskByTitle(view, "First"); found {
		test.Fatalf("expected the redo stack cleared by the new mutation, got %+v", view.Nodes)
	}
	if _, found := findTaskByTitle(view, "Second"); !found {
		test.Fatalf("expected the new mutation to stand, got %+v", view.Nodes)
	}
}

// TestOpeningProjectClearsHistory verifies undo snapshots do not leak across
// projects, since they describe the content of the project that was closed.
func TestOpeningProjectClearsHistory(test *testing.T) {
	service := openProjectForHistory(test)
	service.CreateTask("First", "", "")

	other, _ := service.CreateProject("Other", "", "", "")
	if _, openError := service.OpenProject(other.ID); openError != nil {
		test.Fatalf("OpenProject: %v", openError)
	}

	view, undoError := service.Undo()
	if undoError != nil {
		test.Fatalf("Undo: %v", undoError)
	}
	if len(view.Nodes) != 0 {
		test.Fatalf("expected the newly opened project untouched, got %+v", view.Nodes)
	}
}

// TestHistoryDepthEvictsOldest verifies the stack is bounded: beyond the cap the
// oldest snapshot is dropped, so undo can no longer reach the empty project.
func TestHistoryDepthEvictsOldest(test *testing.T) {
	service := openProjectForHistory(test)

	for count := 0; count < historyDepth+5; count++ {
		if _, createError := service.CreateTask("Task", "", ""); createError != nil {
			test.Fatalf("CreateTask: %v", createError)
		}
	}
	if len(service.history.undoStack) != historyDepth {
		test.Fatalf("expected the stack capped at %d, got %d", historyDepth, len(service.history.undoStack))
	}

	var view ProjectView
	for count := 0; count < historyDepth; count++ {
		undone, undoError := service.Undo()
		if undoError != nil {
			test.Fatalf("Undo: %v", undoError)
		}
		view = undone
	}
	if len(view.Nodes) != 5 {
		test.Fatalf("expected the 5 evicted steps to be unreachable, got %d nodes", len(view.Nodes))
	}
}
