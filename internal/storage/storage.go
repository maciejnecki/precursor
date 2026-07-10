// Package storage persists projects as individual SQLite databases. Each project
// is one file under a projects directory; the file holds the project metadata,
// every node, and every proximity bond. A Store manages the directory as a whole,
// while a Repository exposes CRUD over a single open project database.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"precursor/internal/model"
)

// timestampLayout is the textual format used for every stored timestamp so that
// databases stay human-readable and portable across platforms.
const timestampLayout = time.RFC3339Nano

// schema creates every table and index a project database needs. It is safe to
// run on every open because each statement is guarded with IF NOT EXISTS.
const schema = `
CREATE TABLE IF NOT EXISTS meta (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	name TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	colour TEXT NOT NULL DEFAULT '',
	icon TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS nodes (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	body_markdown TEXT NOT NULL DEFAULT '',
	icon TEXT NOT NULL DEFAULT '',
	parent_id TEXT,
	child_id TEXT,
	decision_type TEXT NOT NULL DEFAULT '',
	order_index INTEGER NOT NULL DEFAULT 0,
	decisions_collapsed INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_unique_parent
	ON nodes(parent_id) WHERE kind = 'task' AND parent_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS proximity_bonds (
	id TEXT PRIMARY KEY,
	endpoint_a_id TEXT NOT NULL,
	endpoint_b_id TEXT NOT NULL,
	created_at TEXT NOT NULL
);
`

// Store owns the directory that holds every project database file.
type Store struct {
	directory string
}

// NewStore returns a Store rooted at the given directory, creating the directory
// if it does not already exist.
func NewStore(directory string) (*Store, error) {
	creationError := os.MkdirAll(directory, 0o755)
	if creationError != nil {
		return nil, fmt.Errorf("create projects directory: %w", creationError)
	}
	return &Store{directory: directory}, nil
}

// pathForID maps a project identifier to its database file path.
func (store *Store) pathForID(identifier string) string {
	return filepath.Join(store.directory, identifier+".db")
}

// migrations are idempotent statements that bring databases created by older
// versions up to the current schema. SQLite cannot add a column conditionally, so
// a duplicate-column error from a re-run is expected and ignored.
var migrations = []string{
	`ALTER TABLE nodes ADD COLUMN decisions_collapsed INTEGER NOT NULL DEFAULT 0`,
}

// openDatabase opens (and migrates) a SQLite database at the given path, tuned
// for a single desktop process: one pooled connection, write-ahead logging, and a
// busy timeout so a briefly locked file retries instead of failing.
func openDatabase(path string) (*sql.DB, error) {
	database, openError := openRawDatabase(path)
	if openError != nil {
		return nil, openError
	}
	_, migrationError := database.Exec(schema)
	if migrationError != nil {
		database.Close()
		return nil, fmt.Errorf("apply schema: %w", migrationError)
	}
	if upgradeError := applyMigrations(database); upgradeError != nil {
		database.Close()
		return nil, upgradeError
	}
	return database, nil
}

// openRawDatabase opens a SQLite database with the shared connection tuning but
// without touching the schema, for callers that only read from it.
func openRawDatabase(path string) (*sql.DB, error) {
	database, openError := sql.Open("sqlite", path)
	if openError != nil {
		return nil, fmt.Errorf("open database: %w", openError)
	}
	database.SetMaxOpenConns(1)
	_, pragmaError := database.Exec(`PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`)
	if pragmaError != nil {
		database.Close()
		return nil, fmt.Errorf("apply connection pragmas: %w", pragmaError)
	}
	return database, nil
}

// applyMigrations runs each migration, ignoring duplicate-column errors so the
// statements stay safe to apply to databases that already have the change.
func applyMigrations(database *sql.DB) error {
	for _, statement := range migrations {
		_, migrationError := database.Exec(statement)
		if migrationError != nil && !strings.Contains(migrationError.Error(), "duplicate column name") {
			return fmt.Errorf("apply migration: %w", migrationError)
		}
	}
	return nil
}

// Open returns a Repository over an existing project database. The caller is
// responsible for closing the returned Repository.
func (store *Store) Open(identifier string) (*Repository, error) {
	path := store.pathForID(identifier)
	_, statError := os.Stat(path)
	if statError != nil {
		return nil, fmt.Errorf("project %q not found: %w", identifier, statError)
	}
	database, openError := openDatabase(path)
	if openError != nil {
		return nil, openError
	}
	return newRepository(database, identifier), nil
}

// CreateProject creates a new project database, writes its metadata, and returns
// the stored project. The identifier is a freshly generated UUID.
func (store *Store) CreateProject(name, description, colour, icon string) (model.Project, error) {
	identifier := uuid.NewString()
	database, openError := openDatabase(store.pathForID(identifier))
	if openError != nil {
		return model.Project{}, openError
	}
	defer database.Close()

	now := time.Now().UTC()
	project := model.Project{
		ID:          identifier,
		Name:        name,
		Description: description,
		Colour:      colour,
		Icon:        icon,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	insertError := writeMeta(database, project)
	if insertError != nil {
		return model.Project{}, insertError
	}
	return project, nil
}

// DeleteProject removes a project database file permanently, along with the
// write-ahead-log sidecar files SQLite keeps beside it.
func (store *Store) DeleteProject(identifier string) error {
	path := store.pathForID(identifier)
	removalError := os.Remove(path)
	if removalError != nil {
		return fmt.Errorf("delete project %q: %w", identifier, removalError)
	}
	os.Remove(path + "-wal")
	os.Remove(path + "-shm")
	return nil
}

// ListProjects scans the directory and returns the metadata of every project,
// sorted by name so the sidebar order is stable.
func (store *Store) ListProjects() ([]model.Project, error) {
	entries, readError := os.ReadDir(store.directory)
	if readError != nil {
		return nil, fmt.Errorf("read projects directory: %w", readError)
	}

	projects := make([]model.Project, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		identifier := strings.TrimSuffix(entry.Name(), ".db")
		project, openError := store.readProjectMeta(identifier)
		if openError != nil {
			return nil, openError
		}
		projects = append(projects, project)
	}

	sort.Slice(projects, func(first, second int) bool {
		return projects[first].Name < projects[second].Name
	})
	return projects, nil
}

// readProjectMeta opens a project database briefly to read its metadata row. It
// skips the schema and migration work a full Open performs, since listing only
// needs the single meta row.
func (store *Store) readProjectMeta(identifier string) (model.Project, error) {
	path := store.pathForID(identifier)
	_, statError := os.Stat(path)
	if statError != nil {
		return model.Project{}, fmt.Errorf("project %q not found: %w", identifier, statError)
	}
	database, openError := openRawDatabase(path)
	if openError != nil {
		return model.Project{}, openError
	}
	defer database.Close()
	return readMeta(database, identifier)
}
