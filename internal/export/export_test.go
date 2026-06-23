package export

import (
	"strings"
	"testing"
	"time"

	"precursor/internal/model"
)

// stringPointer returns a pointer to the given string for optional fields.
func stringPointer(value string) *string {
	return &value
}

// TestCompletedTableListsOnlyDoneTasks verifies the table includes done tasks with
// their decisions and excludes tasks that are not done.
func TestCompletedTableListsOnlyDoneTasks(test *testing.T) {
	nodes := []model.Node{
		{ID: "endpoint", Kind: model.KindTask, Title: "Ship release"},
		{ID: "build", Kind: model.KindTask, Title: "Build binary", ParentID: stringPointer("endpoint")},
		{ID: "draft", Kind: model.KindTask, Title: "Draft notes", ParentID: stringPointer("endpoint")},
		{ID: "d1", Kind: model.KindDecision, ChildID: stringPointer("build"), DecisionType: model.DecisionDone, Title: "Compiled cleanly", OrderIndex: 0},
	}

	table := CompletedTable(nodes)
	if !strings.Contains(table, "Build binary") {
		test.Fatalf("expected completed task in table:\n%s", table)
	}
	if strings.Contains(table, "Draft notes") {
		test.Fatalf("did not expect non-done task in table:\n%s", table)
	}
	if !strings.Contains(table, "done: Compiled cleanly") {
		test.Fatalf("expected decision description in table:\n%s", table)
	}
	if !strings.Contains(table, "| Ship release |") {
		test.Fatalf("expected endpoint grouping in table:\n%s", table)
	}
}

// TestBackupRoundTrip verifies a project and graph survive marshal then unmarshal.
func TestBackupRoundTrip(test *testing.T) {
	project := model.Project{ID: "p1", Name: "Alpha", Icon: "🚀", CreatedAt: time.Now().UTC().Truncate(time.Second)}
	graph := model.Graph{
		Nodes: []model.Node{
			{ID: "endpoint", Kind: model.KindTask, Title: "Ship"},
			{ID: "d1", Kind: model.KindDecision, ChildID: stringPointer("endpoint"), DecisionType: model.DecisionDone},
		},
		ProximityBonds: []model.ProximityBond{{ID: "b1", EndpointAID: "endpoint", EndpointBID: "endpoint"}},
	}

	data, marshalError := MarshalBackup(project, graph)
	if marshalError != nil {
		test.Fatalf("MarshalBackup: %v", marshalError)
	}
	restored, unmarshalError := UnmarshalBackup(data)
	if unmarshalError != nil {
		test.Fatalf("UnmarshalBackup: %v", unmarshalError)
	}
	if restored.Project.Name != "Alpha" || len(restored.Graph.Nodes) != 2 || len(restored.Graph.ProximityBonds) != 1 {
		test.Fatalf("backup did not round-trip: %+v", restored)
	}
}
