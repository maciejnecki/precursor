package storage

import (
	"database/sql"
	"fmt"
	"time"

	"precursor/internal/model"
)

// executor is the query surface shared by *sql.DB and *sql.Tx, letting every
// repository method run identically inside and outside a transaction.
type executor interface {
	Exec(query string, arguments ...any) (sql.Result, error)
	Query(query string, arguments ...any) (*sql.Rows, error)
	QueryRow(query string, arguments ...any) *sql.Row
}

// Repository exposes CRUD operations over a single open project database. Its
// methods run against the executor, which is the database itself or, inside
// WithinTransaction, a single transaction.
type Repository struct {
	database   *sql.DB
	executor   executor
	identifier string
}

// newRepository binds a Repository to an open database, keeping the invariant
// that the executor starts as the database itself in one place.
func newRepository(database *sql.DB, identifier string) *Repository {
	return &Repository{database: database, executor: database, identifier: identifier}
}

// WithinTransaction runs the operation against a repository bound to a single
// transaction, committing on success and rolling back on error, so multi-statement
// mutations are atomic.
func (repository *Repository) WithinTransaction(operation func(transactional *Repository) error) error {
	transaction, beginError := repository.database.Begin()
	if beginError != nil {
		return fmt.Errorf("begin transaction: %w", beginError)
	}
	transactional := &Repository{database: repository.database, executor: transaction, identifier: repository.identifier}
	if operationError := operation(transactional); operationError != nil {
		transaction.Rollback()
		return operationError
	}
	if commitError := transaction.Commit(); commitError != nil {
		return fmt.Errorf("commit transaction: %w", commitError)
	}
	return nil
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
func writeMeta(database executor, project model.Project) error {
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
	return readMeta(repository.executor, repository.identifier)
}

// readMeta reads the single metadata row of a project database from any query
// surface, so both an open Repository and the lightweight project listing share
// one implementation.
func readMeta(database executor, identifier string) (model.Project, error) {
	row := database.QueryRow(
		`SELECT name, description, colour, icon, created_at, updated_at FROM meta WHERE id = 1`,
	)
	project := model.Project{ID: identifier}
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
	_, executionError := repository.executor.Exec(
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
	rows, queryError := repository.executor.Query(`SELECT ` + nodeColumns + ` FROM nodes`)
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
	row := repository.executor.QueryRow(`SELECT `+nodeColumns+` FROM nodes WHERE id = ?`, identifier)
	node, scanError := scanNode(row)
	if scanError != nil {
		return model.Node{}, fmt.Errorf("read node %q: %w", identifier, scanError)
	}
	return node, nil
}

// InsertNode stores a new node.
func (repository *Repository) InsertNode(node model.Node) error {
	_, executionError := repository.executor.Exec(
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
	_, executionError := repository.executor.Exec(
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
	_, executionError := repository.executor.Exec(`DELETE FROM nodes WHERE id = ?`, identifier)
	if executionError != nil {
		return fmt.Errorf("delete node %q: %w", identifier, executionError)
	}
	return nil
}

// ProximityBonds returns every proximity bond in the project.
func (repository *Repository) ProximityBonds() ([]model.ProximityBond, error) {
	rows, queryError := repository.executor.Query(
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
	_, executionError := repository.executor.Exec(
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
	_, executionError := repository.executor.Exec(
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
	_, executionError := repository.executor.Exec(`DELETE FROM proximity_bonds WHERE id = ?`, identifier)
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

// ReplaceGraph overwrites the project's content with the given snapshot. Every
// existing node and bond is removed before the snapshot is written, so a restored
// parent link can never transiently collide with the unique-parent index against a
// row that the snapshot is about to replace. Callers must run this inside a
// transaction so a failed restore leaves the project untouched.
func (repository *Repository) ReplaceGraph(replacement model.Graph) error {
	if _, executionError := repository.executor.Exec(`DELETE FROM proximity_bonds`); executionError != nil {
		return fmt.Errorf("clear proximity bonds: %w", executionError)
	}
	if _, executionError := repository.executor.Exec(`DELETE FROM nodes`); executionError != nil {
		return fmt.Errorf("clear nodes: %w", executionError)
	}
	for _, node := range replacement.Nodes {
		if insertError := repository.InsertNode(node); insertError != nil {
			return insertError
		}
	}
	for _, bond := range replacement.ProximityBonds {
		if insertError := repository.InsertProximityBond(bond); insertError != nil {
			return insertError
		}
	}
	return nil
}
