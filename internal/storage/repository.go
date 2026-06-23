package storage

import (
	"database/sql"
	"fmt"
	"time"

	"precursor/internal/model"
)

// Repository exposes CRUD operations over a single open project database.
type Repository struct {
	database   *sql.DB
	identifier string
}

// Close releases the underlying database connection.
func (repository *Repository) Close() error {
	return repository.database.Close()
}

// ID returns the identifier of the project this repository is bound to.
func (repository *Repository) ID() string {
	return repository.identifier
}

// formatTimestamp renders a time value in the stored textual layout.
func formatTimestamp(value time.Time) string {
	return value.UTC().Format(timestampLayout)
}

// parseTimestamp reads a stored textual timestamp back into a time value.
func parseTimestamp(value string) time.Time {
	parsed, parseError := time.Parse(timestampLayout, value)
	if parseError != nil {
		return time.Time{}
	}
	return parsed
}

// optionalString converts a database string column into an optional pointer,
// treating the empty string as absent.
func optionalString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	owned := value.String
	return &owned
}

// nullableString converts an optional pointer into a database string column.
func nullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

// boolToInt maps a boolean onto the integer SQLite stores it as.
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// writeMeta inserts or replaces the single metadata row of a project database.
func writeMeta(database *sql.DB, project model.Project) error {
	_, executionError := database.Exec(
		`INSERT INTO meta (id, name, description, colour, icon, created_at, updated_at)
		 VALUES (1, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name = excluded.name,
		   description = excluded.description,
		   colour = excluded.colour,
		   icon = excluded.icon,
		   updated_at = excluded.updated_at`,
		project.Name,
		project.Description,
		project.Colour,
		project.Icon,
		formatTimestamp(project.CreatedAt),
		formatTimestamp(project.UpdatedAt),
	)
	if executionError != nil {
		return fmt.Errorf("write project metadata: %w", executionError)
	}
	return nil
}

// Meta reads the project's metadata row.
func (repository *Repository) Meta() (model.Project, error) {
	row := repository.database.QueryRow(
		`SELECT name, description, colour, icon, created_at, updated_at FROM meta WHERE id = 1`,
	)
	project := model.Project{ID: repository.identifier}
	var createdAt, updatedAt string
	scanError := row.Scan(
		&project.Name,
		&project.Description,
		&project.Colour,
		&project.Icon,
		&createdAt,
		&updatedAt,
	)
	if scanError != nil {
		return model.Project{}, fmt.Errorf("read project metadata: %w", scanError)
	}
	project.CreatedAt = parseTimestamp(createdAt)
	project.UpdatedAt = parseTimestamp(updatedAt)
	return project, nil
}

// UpdateMeta updates the editable metadata fields of the project.
func (repository *Repository) UpdateMeta(name, description, colour, icon string) error {
	_, executionError := repository.database.Exec(
		`UPDATE meta SET name = ?, description = ?, colour = ?, icon = ?, updated_at = ? WHERE id = 1`,
		name,
		description,
		colour,
		icon,
		formatTimestamp(time.Now().UTC()),
	)
	if executionError != nil {
		return fmt.Errorf("update project metadata: %w", executionError)
	}
	return nil
}

// scanNode reads a single node from a query row source.
func scanNode(scanner interface{ Scan(...any) error }) (model.Node, error) {
	var node model.Node
	var parentID, childID sql.NullString
	var decisionType string
	var decisionsCollapsed int
	var createdAt, updatedAt string
	scanError := scanner.Scan(
		&node.ID,
		&node.Kind,
		&node.Title,
		&node.BodyMarkdown,
		&node.Icon,
		&parentID,
		&childID,
		&decisionType,
		&node.OrderIndex,
		&decisionsCollapsed,
		&createdAt,
		&updatedAt,
	)
	if scanError != nil {
		return model.Node{}, scanError
	}
	node.ParentID = optionalString(parentID)
	node.ChildID = optionalString(childID)
	node.DecisionType = model.DecisionType(decisionType)
	node.DecisionsCollapsed = decisionsCollapsed != 0
	node.CreatedAt = parseTimestamp(createdAt)
	node.UpdatedAt = parseTimestamp(updatedAt)
	return node, nil
}

// nodeColumns lists the node columns in the order scanNode expects them.
const nodeColumns = `id, kind, title, body_markdown, icon, parent_id, child_id, decision_type, order_index, decisions_collapsed, created_at, updated_at`

// Nodes returns every node in the project.
func (repository *Repository) Nodes() ([]model.Node, error) {
	rows, queryError := repository.database.Query(`SELECT ` + nodeColumns + ` FROM nodes`)
	if queryError != nil {
		return nil, fmt.Errorf("query nodes: %w", queryError)
	}
	defer rows.Close()

	nodes := make([]model.Node, 0)
	for rows.Next() {
		node, scanError := scanNode(rows)
		if scanError != nil {
			return nil, fmt.Errorf("scan node: %w", scanError)
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// Node returns a single node by identifier.
func (repository *Repository) Node(identifier string) (model.Node, error) {
	row := repository.database.QueryRow(`SELECT `+nodeColumns+` FROM nodes WHERE id = ?`, identifier)
	node, scanError := scanNode(row)
	if scanError != nil {
		return model.Node{}, fmt.Errorf("read node %q: %w", identifier, scanError)
	}
	return node, nil
}

// InsertNode stores a new node.
func (repository *Repository) InsertNode(node model.Node) error {
	_, executionError := repository.database.Exec(
		`INSERT INTO nodes (`+nodeColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID,
		node.Kind,
		node.Title,
		node.BodyMarkdown,
		node.Icon,
		nullableString(node.ParentID),
		nullableString(node.ChildID),
		string(node.DecisionType),
		node.OrderIndex,
		boolToInt(node.DecisionsCollapsed),
		formatTimestamp(node.CreatedAt),
		formatTimestamp(node.UpdatedAt),
	)
	if executionError != nil {
		return fmt.Errorf("insert node: %w", executionError)
	}
	return nil
}

// UpdateNode overwrites an existing node identified by its ID.
func (repository *Repository) UpdateNode(node model.Node) error {
	_, executionError := repository.database.Exec(
		`UPDATE nodes SET kind = ?, title = ?, body_markdown = ?, icon = ?, parent_id = ?,
		 child_id = ?, decision_type = ?, order_index = ?, decisions_collapsed = ?, updated_at = ? WHERE id = ?`,
		node.Kind,
		node.Title,
		node.BodyMarkdown,
		node.Icon,
		nullableString(node.ParentID),
		nullableString(node.ChildID),
		string(node.DecisionType),
		node.OrderIndex,
		boolToInt(node.DecisionsCollapsed),
		formatTimestamp(node.UpdatedAt),
		node.ID,
	)
	if executionError != nil {
		return fmt.Errorf("update node %q: %w", node.ID, executionError)
	}
	return nil
}

// DeleteNode removes a single node by identifier.
func (repository *Repository) DeleteNode(identifier string) error {
	_, executionError := repository.database.Exec(`DELETE FROM nodes WHERE id = ?`, identifier)
	if executionError != nil {
		return fmt.Errorf("delete node %q: %w", identifier, executionError)
	}
	return nil
}

// ProximityBonds returns every proximity bond in the project.
func (repository *Repository) ProximityBonds() ([]model.ProximityBond, error) {
	rows, queryError := repository.database.Query(
		`SELECT id, endpoint_a_id, endpoint_b_id, created_at FROM proximity_bonds`,
	)
	if queryError != nil {
		return nil, fmt.Errorf("query proximity bonds: %w", queryError)
	}
	defer rows.Close()

	bonds := make([]model.ProximityBond, 0)
	for rows.Next() {
		var bond model.ProximityBond
		var createdAt string
		scanError := rows.Scan(&bond.ID, &bond.EndpointAID, &bond.EndpointBID, &createdAt)
		if scanError != nil {
			return nil, fmt.Errorf("scan proximity bond: %w", scanError)
		}
		bond.CreatedAt = parseTimestamp(createdAt)
		bonds = append(bonds, bond)
	}
	return bonds, rows.Err()
}

// InsertProximityBond stores a new proximity bond.
func (repository *Repository) InsertProximityBond(bond model.ProximityBond) error {
	_, executionError := repository.database.Exec(
		`INSERT INTO proximity_bonds (id, endpoint_a_id, endpoint_b_id, created_at) VALUES (?, ?, ?, ?)`,
		bond.ID,
		bond.EndpointAID,
		bond.EndpointBID,
		formatTimestamp(bond.CreatedAt),
	)
	if executionError != nil {
		return fmt.Errorf("insert proximity bond: %w", executionError)
	}
	return nil
}

// UpdateProximityBond overwrites the endpoints of an existing bond.
func (repository *Repository) UpdateProximityBond(bond model.ProximityBond) error {
	_, executionError := repository.database.Exec(
		`UPDATE proximity_bonds SET endpoint_a_id = ?, endpoint_b_id = ? WHERE id = ?`,
		bond.EndpointAID,
		bond.EndpointBID,
		bond.ID,
	)
	if executionError != nil {
		return fmt.Errorf("update proximity bond %q: %w", bond.ID, executionError)
	}
	return nil
}

// DeleteProximityBond removes a proximity bond by identifier.
func (repository *Repository) DeleteProximityBond(identifier string) error {
	_, executionError := repository.database.Exec(`DELETE FROM proximity_bonds WHERE id = ?`, identifier)
	if executionError != nil {
		return fmt.Errorf("delete proximity bond %q: %w", identifier, executionError)
	}
	return nil
}

// Graph returns the full content of the project: all nodes and all bonds.
func (repository *Repository) Graph() (model.Graph, error) {
	nodes, nodesError := repository.Nodes()
	if nodesError != nil {
		return model.Graph{}, nodesError
	}
	bonds, bondsError := repository.ProximityBonds()
	if bondsError != nil {
		return model.Graph{}, bondsError
	}
	return model.Graph{Nodes: nodes, ProximityBonds: bonds}, nil
}
