// Package export produces the two open output formats the app offers: a Markdown
// table summarising completed work, and a JSON backup that round-trips a whole
// project. Both operate on plain model values so they stay independent of storage.
package export

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"precursor/internal/graph"
	"precursor/internal/model"
)

// Backup is the serializable form of a complete project used for JSON export and
// import.
type Backup struct {
	Project model.Project `json:"project"`
	Graph   model.Graph   `json:"graph"`
}

// MarshalBackup serializes a project and its graph to indented JSON.
func MarshalBackup(project model.Project, projectGraph model.Graph) ([]byte, error) {
	data, marshalError := json.MarshalIndent(Backup{Project: project, Graph: projectGraph}, "", "  ")
	if marshalError != nil {
		return nil, fmt.Errorf("marshal backup: %w", marshalError)
	}
	return data, nil
}

// UnmarshalBackup parses a JSON backup produced by MarshalBackup.
func UnmarshalBackup(data []byte) (Backup, error) {
	var backup Backup
	unmarshalError := json.Unmarshal(data, &backup)
	if unmarshalError != nil {
		return Backup{}, fmt.Errorf("unmarshal backup: %w", unmarshalError)
	}
	return backup, nil
}

// escapeCell makes a string safe to place inside a Markdown table cell by
// neutralising pipe and newline characters.
func escapeCell(value string) string {
	replacer := strings.NewReplacer("|", "\\|", "\n", " ", "\r", " ")
	return replacer.Replace(value)
}

// describeDecision renders a decision as its type and title for the summary cell.
func describeDecision(decision model.Node) string {
	title := decision.Title
	if title == "" {
		title = "(untitled)"
	}
	return fmt.Sprintf("%s: %s", decision.DecisionType, title)
}

// CompletedTable builds a Markdown table of every completed task together with the
// decisions that led to its completion, grouped by the endpoint of its chain.
func CompletedTable(nodes []model.Node) string {
	type row struct {
		endpoint  string
		task      string
		decisions string
	}

	rows := make([]row, 0)
	for _, node := range nodes {
		if node.Kind != model.KindTask || graph.DeriveStatus(nodes, node.ID) != model.StatusDone {
			continue
		}
		descriptions := make([]string, 0)
		for _, decision := range graph.DecisionsFor(nodes, node.ID) {
			descriptions = append(descriptions, describeDecision(decision))
		}
		endpointID := graph.EndpointID(nodes, node.ID)
		endpointTitle := titleFor(nodes, endpointID)
		rows = append(rows, row{
			endpoint:  endpointTitle,
			task:      titleOrUntitled(node.Title),
			decisions: strings.Join(descriptions, " → "),
		})
	}

	sort.SliceStable(rows, func(first, second int) bool {
		if rows[first].endpoint != rows[second].endpoint {
			return rows[first].endpoint < rows[second].endpoint
		}
		return rows[first].task < rows[second].task
	})

	var builder strings.Builder
	builder.WriteString("| Endpoint | Completed Task | Decisions |\n")
	builder.WriteString("| --- | --- | --- |\n")
	for _, current := range rows {
		builder.WriteString(fmt.Sprintf(
			"| %s | %s | %s |\n",
			escapeCell(current.endpoint),
			escapeCell(current.task),
			escapeCell(current.decisions),
		))
	}
	return builder.String()
}

// titleFor returns the title of the node with the given identifier, or an empty
// string when it is not found.
func titleFor(nodes []model.Node, identifier string) string {
	for _, node := range nodes {
		if node.ID == identifier {
			return titleOrUntitled(node.Title)
		}
	}
	return ""
}

// titleOrUntitled substitutes a placeholder for an empty title.
func titleOrUntitled(title string) string {
	if title == "" {
		return "(untitled)"
	}
	return title
}
