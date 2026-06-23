package storage

import (
	"testing"
	"time"

	"precursor/internal/model"
)

// newTestStore creates a Store rooted in a temporary directory for isolated tests.
func newTestStore(test *testing.T) *Store {
	test.Helper()
	store, creationError := NewStore(test.TempDir())
	if creationError != nil {
		test.Fatalf("NewStore: %v", creationError)
	}
	return store
}

// makeTask builds a task node with the given identifier and optional parent.
func makeTask(identifier string, parentID *string) model.Node {
	now := time.Now().UTC()
	return model.Node{
		ID:        identifier,
		Kind:      model.KindTask,
		Title:     identifier,
		ParentID:  parentID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TestCreateAndListProjects verifies that created projects round-trip through the
// directory scan with their metadata intact.
func TestCreateAndListProjects(test *testing.T) {
	store := newTestStore(test)

	created, creationError := store.CreateProject("Alpha", "first project", "#ff0000", "🚀")
	if creationError != nil {
		test.Fatalf("CreateProject: %v", creationError)
	}

	projects, listError := store.ListProjects()
	if listError != nil {
		test.Fatalf("ListProjects: %v", listError)
	}
	if len(projects) != 1 {
		test.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].ID != created.ID || projects[0].Name != "Alpha" || projects[0].Icon != "🚀" {
		test.Fatalf("unexpected project metadata: %+v", projects[0])
	}
}

// TestNodeRoundTrip verifies inserting, reading, updating, and deleting a node.
func TestNodeRoundTrip(test *testing.T) {
	store := newTestStore(test)
	created, _ := store.CreateProject("Alpha", "", "", "")
	repository, openError := store.Open(created.ID)
	if openError != nil {
		test.Fatalf("Open: %v", openError)
	}
	defer repository.Close()

	endpoint := makeTask("endpoint", nil)
	if insertError := repository.InsertNode(endpoint); insertError != nil {
		test.Fatalf("InsertNode: %v", insertError)
	}

	read, readError := repository.Node("endpoint")
	if readError != nil {
		test.Fatalf("Node: %v", readError)
	}
	if read.ParentID != nil {
		test.Fatalf("expected endpoint to have no parent, got %v", *read.ParentID)
	}

	read.Title = "renamed"
	read.UpdatedAt = time.Now().UTC()
	if updateError := repository.UpdateNode(read); updateError != nil {
		test.Fatalf("UpdateNode: %v", updateError)
	}
	reread, _ := repository.Node("endpoint")
	if reread.Title != "renamed" {
		test.Fatalf("expected renamed title, got %q", reread.Title)
	}

	if deleteError := repository.DeleteNode("endpoint"); deleteError != nil {
		test.Fatalf("DeleteNode: %v", deleteError)
	}
	nodes, _ := repository.Nodes()
	if len(nodes) != 0 {
		test.Fatalf("expected no nodes after delete, got %d", len(nodes))
	}
}

// TestOnePrecursorConstraint verifies the unique index that forbids a parent task
// from having more than one precursor.
func TestOnePrecursorConstraint(test *testing.T) {
	store := newTestStore(test)
	created, _ := store.CreateProject("Alpha", "", "", "")
	repository, _ := store.Open(created.ID)
	defer repository.Close()

	parentID := "parent"
	if insertError := repository.InsertNode(makeTask("parent", nil)); insertError != nil {
		test.Fatalf("insert parent: %v", insertError)
	}
	if insertError := repository.InsertNode(makeTask("precursorOne", &parentID)); insertError != nil {
		test.Fatalf("insert first precursor: %v", insertError)
	}

	secondInsertError := repository.InsertNode(makeTask("precursorTwo", &parentID))
	if secondInsertError == nil {
		test.Fatalf("expected unique-constraint error when adding a second precursor, got nil")
	}
}

// TestProximityBondRoundTrip verifies bond insertion, listing, and deletion.
func TestProximityBondRoundTrip(test *testing.T) {
	store := newTestStore(test)
	created, _ := store.CreateProject("Alpha", "", "", "")
	repository, _ := store.Open(created.ID)
	defer repository.Close()

	bond := model.ProximityBond{
		ID:          "bond",
		EndpointAID: "a",
		EndpointBID: "b",
		CreatedAt:   time.Now().UTC(),
	}
	if insertError := repository.InsertProximityBond(bond); insertError != nil {
		test.Fatalf("InsertProximityBond: %v", insertError)
	}
	bonds, _ := repository.ProximityBonds()
	if len(bonds) != 1 {
		test.Fatalf("expected 1 bond, got %d", len(bonds))
	}
	if deleteError := repository.DeleteProximityBond("bond"); deleteError != nil {
		test.Fatalf("DeleteProximityBond: %v", deleteError)
	}
	bonds, _ = repository.ProximityBonds()
	if len(bonds) != 0 {
		test.Fatalf("expected 0 bonds after delete, got %d", len(bonds))
	}
}

// TestDeleteProject verifies a deleted project no longer appears in the listing.
func TestDeleteProject(test *testing.T) {
	store := newTestStore(test)
	created, _ := store.CreateProject("Alpha", "", "", "")
	if deleteError := store.DeleteProject(created.ID); deleteError != nil {
		test.Fatalf("DeleteProject: %v", deleteError)
	}
	projects, _ := store.ListProjects()
	if len(projects) != 0 {
		test.Fatalf("expected 0 projects after delete, got %d", len(projects))
	}
}
