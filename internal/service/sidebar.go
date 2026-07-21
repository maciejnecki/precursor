package service

import (
	"precursor/internal/config"
	"precursor/internal/model"
)

// SidebarState is everything the sidebar renders: the projects in their stored
// order, and the groups banding some of them together.
type SidebarState struct {
	Projects []model.Project       `json:"projects"`
	Groups   []config.ProjectGroup `json:"groups"`
}

// normaliseSidebar cleans an incoming order and set of groups against the projects
// that actually exist. It drops unknown identifiers, keeps a project in at most one
// group, drops groups left without members, and rewrites the order so each group's
// members sit together at the position of its first member. Keeping members
// contiguous is what lets the sidebar render a group as one unbroken band.
func normaliseSidebar(order []string, groups []config.ProjectGroup, known map[string]bool) ([]string, []config.ProjectGroup) {
	ordered := make([]string, 0, len(order))
	seen := make(map[string]bool, len(order))
	for _, identifier := range order {
		if known[identifier] && !seen[identifier] {
			seen[identifier] = true
			ordered = append(ordered, identifier)
		}
	}

	// Build the surviving groups, claiming each member for the first group listing it.
	claimed := make(map[string]string, len(ordered))
	cleanGroups := make([]config.ProjectGroup, 0, len(groups))
	for _, group := range groups {
		members := make([]string, 0, len(group.Members))
		for _, identifier := range group.Members {
			if seen[identifier] && claimed[identifier] == "" {
				claimed[identifier] = group.ID
				members = append(members, identifier)
			}
		}
		if len(members) == 0 {
			continue
		}
		group.Members = members
		cleanGroups = append(cleanGroups, group)
	}

	membersOf := make(map[string][]string, len(cleanGroups))
	for _, group := range cleanGroups {
		membersOf[group.ID] = group.Members
	}

	// Walk the order once, expanding each group in full the first time one of its
	// members is met so the block lands where the group already sat.
	contiguous := make([]string, 0, len(ordered))
	emitted := make(map[string]bool, len(ordered))
	for _, identifier := range ordered {
		if emitted[identifier] {
			continue
		}
		groupID := claimed[identifier]
		if groupID == "" {
			emitted[identifier] = true
			contiguous = append(contiguous, identifier)
			continue
		}
		for _, member := range membersOf[groupID] {
			emitted[member] = true
			contiguous = append(contiguous, member)
		}
	}
	return contiguous, cleanGroups
}
