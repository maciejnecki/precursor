package service

import (
	"strings"
	"testing"

	"precursor/internal/config"
)

// knownProjects builds the membership set normaliseSidebar checks identifiers against.
func knownProjects(identifiers ...string) map[string]bool {
	known := make(map[string]bool, len(identifiers))
	for _, identifier := range identifiers {
		known[identifier] = true
	}
	return known
}

// A group's members are pulled together at the position of its first member, so the
// sidebar can draw the group as one unbroken band.
func TestNormaliseSidebarMakesGroupsContiguous(test *testing.T) {
	order := []string{"a", "b", "c", "d"}
	groups := []config.ProjectGroup{{ID: "g1", Name: "Group", Members: []string{"a", "c"}}}

	normalisedOrder, normalisedGroups := normaliseSidebar(order, groups, knownProjects("a", "b", "c", "d"))

	if strings.Join(normalisedOrder, ",") != "a,c,b,d" {
		test.Fatalf("order = %v, want a,c,b,d", normalisedOrder)
	}
	if len(normalisedGroups) != 1 || len(normalisedGroups[0].Members) != 2 {
		test.Fatalf("groups = %v, want one group of two", normalisedGroups)
	}
}

// Identifiers of projects that no longer exist are dropped from both the order and
// group membership, and a group left with no members disappears.
func TestNormaliseSidebarDropsUnknownAndEmpty(test *testing.T) {
	order := []string{"a", "gone", "b"}
	groups := []config.ProjectGroup{
		{ID: "g1", Members: []string{"gone"}},
		{ID: "g2", Members: []string{"b"}},
	}

	normalisedOrder, normalisedGroups := normaliseSidebar(order, groups, knownProjects("a", "b"))

	if strings.Join(normalisedOrder, ",") != "a,b" {
		test.Fatalf("order = %v, want a,b", normalisedOrder)
	}
	if len(normalisedGroups) != 1 || normalisedGroups[0].ID != "g2" {
		test.Fatalf("groups = %v, want only g2", normalisedGroups)
	}
}

// A project listed by two groups belongs to the first one only, so membership stays
// unambiguous however the frontend sends it.
func TestNormaliseSidebarClaimsMemberOnce(test *testing.T) {
	order := []string{"a", "b"}
	groups := []config.ProjectGroup{
		{ID: "g1", Members: []string{"a", "b"}},
		{ID: "g2", Members: []string{"b"}},
	}

	_, normalisedGroups := normaliseSidebar(order, groups, knownProjects("a", "b"))

	if len(normalisedGroups) != 1 || normalisedGroups[0].ID != "g1" {
		test.Fatalf("groups = %v, want only g1 holding both", normalisedGroups)
	}
}

// Saving a sidebar layout round-trips through the settings file.
func TestSaveSidebarPersists(test *testing.T) {
	service := openService(test)
	first, creationError := service.CreateProject("First", "", "", "")
	if creationError != nil {
		test.Fatalf("CreateProject: %v", creationError)
	}
	second, creationError := service.CreateProject("Second", "", "", "")
	if creationError != nil {
		test.Fatalf("CreateProject: %v", creationError)
	}

	saved, saveError := service.SaveSidebar(
		[]string{second.ID, first.ID},
		[]config.ProjectGroup{{ID: "g1", Name: "Band", Members: []string{first.ID}}},
	)
	if saveError != nil {
		test.Fatalf("SaveSidebar: %v", saveError)
	}
	if saved.Projects[0].ID != second.ID {
		test.Fatalf("first project = %q, want %q", saved.Projects[0].ID, second.ID)
	}

	reloaded, readError := service.Sidebar()
	if readError != nil {
		test.Fatalf("Sidebar: %v", readError)
	}
	if len(reloaded.Groups) != 1 || reloaded.Groups[0].Name != "Band" {
		test.Fatalf("groups = %v, want the stored band", reloaded.Groups)
	}
}
