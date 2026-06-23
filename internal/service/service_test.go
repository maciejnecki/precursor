package service

import (
	"testing"

	"precursor/internal/model"
)

// openService builds a service rooted in a temporary directory.
func openService(test *testing.T) *Service {
	test.Helper()
	created, creationError := New(test.TempDir())
	if creationError != nil {
		test.Fatalf("New: %v", creationError)
	}
	return created
}

// nodeViewByID finds a node view in a project view by identifier.
func nodeViewByID(view ProjectView, identifier string) (NodeView, bool) {
	for _, node := range view.Nodes {
		if node.ID == identifier {
			return node, true
		}
	}
	return NodeView{}, false
}

// findTaskByTitle finds a task node view by its title.
func findTaskByTitle(view ProjectView, title string) (NodeView, bool) {
	for _, node := range view.Nodes {
		if node.Kind == model.KindTask && node.Title == title {
			return node, true
		}
	}
	return NodeView{}, false
}

// TestHappyPath exercises creating a project, logging a chain, advancing status by
// decision, and deleting a node with chain healing, all through the service.
func TestHappyPath(test *testing.T) {
	service := openService(test)

	project, creationError := service.CreateProject("Release", "ship it", "#22c55e", "🚀")
	if creationError != nil {
		test.Fatalf("CreateProject: %v", creationError)
	}
	if _, openError := service.OpenProject(project.ID); openError != nil {
		test.Fatalf("OpenProject: %v", openError)
	}

	view, taskError := service.CreateTask("Ship release", "", "🚀")
	if taskError != nil {
		test.Fatalf("CreateTask: %v", taskError)
	}
	endpoint, _ := findTaskByTitle(view, "Ship release")
	if endpoint.Status != model.StatusScheduled {
		test.Fatalf("expected new task scheduled, got %q", endpoint.Status)
	}

	view, precursorError := service.CreatePrecursor(endpoint.ID, "Build binary", "", "🔧")
	if precursorError != nil {
		test.Fatalf("CreatePrecursor: %v", precursorError)
	}
	precursor, _ := findTaskByTitle(view, "Build binary")

	// A parent may only have one precursor.
	if _, secondError := service.CreatePrecursor(endpoint.ID, "Another", "", ""); secondError == nil {
		test.Fatalf("expected error adding a second precursor")
	}

	view, decisionError := service.CreateDecision(precursor.ID, string(model.DecisionDone), "Compiled", "", "")
	if decisionError != nil {
		test.Fatalf("CreateDecision: %v", decisionError)
	}
	precursor, _ = nodeViewByID(view, precursor.ID)
	if precursor.Status != model.StatusDone {
		test.Fatalf("expected precursor done after decision, got %q", precursor.Status)
	}

	// Deleting the endpoint should promote the precursor to a new endpoint.
	view, deleteError := service.DeleteNode(endpoint.ID)
	if deleteError != nil {
		test.Fatalf("DeleteNode: %v", deleteError)
	}
	if _, stillThere := nodeViewByID(view, endpoint.ID); stillThere {
		test.Fatalf("expected endpoint removed")
	}
	promoted, found := nodeViewByID(view, precursor.ID)
	if !found || promoted.ParentID != nil {
		test.Fatalf("expected precursor promoted to endpoint, got %+v", promoted)
	}
}

// TestProximityThroughService verifies bonding two chains records a bond in the view.
func TestProximityThroughService(test *testing.T) {
	service := openService(test)
	project, _ := service.CreateProject("Proximity", "", "", "")
	service.OpenProject(project.ID)

	firstView, _ := service.CreateTask("Chain A", "", "")
	chainA, _ := findTaskByTitle(firstView, "Chain A")
	secondView, _ := service.CreateTask("Chain B", "", "")
	chainB, _ := findTaskByTitle(secondView, "Chain B")

	bondedView, bondError := service.CreateProximity(chainA.ID, chainB.ID)
	if bondError != nil {
		test.Fatalf("CreateProximity: %v", bondError)
	}
	if len(bondedView.Bonds) != 1 {
		test.Fatalf("expected 1 bond, got %d", len(bondedView.Bonds))
	}

	// Bonding two nodes in the same chain is rejected.
	if _, sameError := service.CreateProximity(chainA.ID, chainA.ID); sameError == nil {
		test.Fatalf("expected error bonding a chain to itself")
	}
}

// TestImportBackupCreatesProject verifies a backup can be exported and re-imported.
func TestImportBackupCreatesProject(test *testing.T) {
	service := openService(test)
	project, _ := service.CreateProject("Original", "desc", "#fff", "📦")
	service.OpenProject(project.ID)
	service.CreateTask("Only task", "body", "")

	data, backupError := service.Backup(project.ID)
	if backupError != nil {
		test.Fatalf("Backup: %v", backupError)
	}
	imported, importError := service.ImportBackup(data)
	if importError != nil {
		test.Fatalf("ImportBackup: %v", importError)
	}
	if imported.ID == project.ID {
		test.Fatalf("expected imported project to get a fresh id")
	}

	projects, _ := service.ListProjects()
	if len(projects) != 2 {
		test.Fatalf("expected 2 projects after import, got %d", len(projects))
	}
}
