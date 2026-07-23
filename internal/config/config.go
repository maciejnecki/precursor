// Package config persists user settings as a human-editable TOML file. Settings
// currently hold the colours used to render task statuses and decision nodes.
// Compile-time fallback values guarantee the app always has a usable palette even
// when the file is missing or partially filled in.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Fallback colours form the shipped Classic palette and back-fill any missing field.
const (
	fallbackScheduled  = "#64748b"
	fallbackInProgress = "#f59e0b"
	fallbackDone       = "#22c55e"
	fallbackRedundant  = "#ef4444"
	fallbackDecision   = "#94a3b8"
	fallbackEndpoint   = "#fc4e26"
)

// StatusColours holds one colour per task status.
type StatusColours struct {
	Scheduled  string `toml:"scheduled" json:"scheduled"`
	InProgress string `toml:"inProgress" json:"inProgress"`
	Done       string `toml:"done" json:"done"`
	Redundant  string `toml:"redundant" json:"redundant"`
}

// Settings is the full set of user-configurable values.
type Settings struct {
	StatusColours  StatusColours `toml:"statusColours" json:"statusColours"`
	DecisionColour string        `toml:"decisionColour" json:"decisionColour"`
	// EndpointColour outlines a pending chain-root task so an incomplete chain
	// stands out until every node in it resolves.
	EndpointColour string `toml:"endpointColour" json:"endpointColour"`
	// ProjectOrder lists project identifiers in the order the sidebar shows them.
	// It is a stored preference rather than part of the settings the UI edits, so
	// it is kept out of the JSON the frontend exchanges.
	ProjectOrder []string `toml:"projectOrder" json:"-"`
	// ProjectGroups are the sidebar's named bands of projects, also a stored
	// preference rather than a value the settings UI edits.
	ProjectGroups []ProjectGroup `toml:"projectGroups" json:"-"`
	// LastProject is the identifier of the project that was open most recently, so
	// the app can reopen it on launch. Another stored preference the settings UI
	// does not edit.
	LastProject string `toml:"lastProject" json:"-"`
}

// ProjectGroup is a named, collapsible band of projects in the sidebar. A project
// belongs to at most one group, and groups do not nest.
type ProjectGroup struct {
	ID        string   `toml:"id" json:"id"`
	Name      string   `toml:"name" json:"name"`
	Collapsed bool     `toml:"collapsed" json:"collapsed"`
	Members   []string `toml:"members" json:"members"`
}

// DefaultSettings returns the shipped settings using the Classic palette.
func DefaultSettings() Settings {
	return Settings{
		StatusColours: StatusColours{
			Scheduled:  fallbackScheduled,
			InProgress: fallbackInProgress,
			Done:       fallbackDone,
			Redundant:  fallbackRedundant,
		},
		DecisionColour: fallbackDecision,
		EndpointColour: fallbackEndpoint,
	}
}

// withFallbacks fills any empty colour in the settings with its shipped default so
// a partially edited file never produces a blank colour.
func withFallbacks(settings Settings) Settings {
	defaults := DefaultSettings()
	if settings.StatusColours.Scheduled == "" {
		settings.StatusColours.Scheduled = defaults.StatusColours.Scheduled
	}
	if settings.StatusColours.InProgress == "" {
		settings.StatusColours.InProgress = defaults.StatusColours.InProgress
	}
	if settings.StatusColours.Done == "" {
		settings.StatusColours.Done = defaults.StatusColours.Done
	}
	if settings.StatusColours.Redundant == "" {
		settings.StatusColours.Redundant = defaults.StatusColours.Redundant
	}
	if settings.DecisionColour == "" {
		settings.DecisionColour = defaults.DecisionColour
	}
	if settings.EndpointColour == "" {
		settings.EndpointColour = defaults.EndpointColour
	}
	return settings
}

// Manager reads and writes the settings file at a fixed path.
type Manager struct {
	path string
}

// NewManager returns a settings manager bound to the given file path.
func NewManager(path string) *Manager {
	return &Manager{path: path}
}

// Load reads the settings file, writing and returning defaults when it is absent.
func (manager *Manager) Load() (Settings, error) {
	data, readError := os.ReadFile(manager.path)
	if errors.Is(readError, fs.ErrNotExist) {
		defaults := DefaultSettings()
		writeError := manager.Save(defaults)
		if writeError != nil {
			return defaults, writeError
		}
		return defaults, nil
	}
	if readError != nil {
		return Settings{}, fmt.Errorf("read settings: %w", readError)
	}

	var settings Settings
	decodeError := toml.Unmarshal(data, &settings)
	if decodeError != nil {
		return Settings{}, fmt.Errorf("decode settings: %w", decodeError)
	}
	return withFallbacks(settings), nil
}

// storedPreferences reads the settings straight from the file, without the
// default-writing Load performs, so Save can consult the stored preferences the UI
// does not send back while writing. A missing or unreadable file yields zero values.
func (manager *Manager) storedPreferences() Settings {
	data, readError := os.ReadFile(manager.path)
	if readError != nil {
		return Settings{}
	}
	var stored Settings
	decodeError := toml.Unmarshal(data, &stored)
	if decodeError != nil {
		return Settings{}
	}
	return stored
}

// Sidebar returns the stored order of project identifiers and the stored groups.
func (manager *Manager) Sidebar() ([]string, []ProjectGroup, error) {
	settings, loadError := manager.Load()
	if loadError != nil {
		return nil, nil, loadError
	}
	return settings.ProjectOrder, settings.ProjectGroups, nil
}

// SetSidebar stores the sidebar order and groups, leaving every other setting as it
// is. Both are written as given, so an empty list does clear them.
func (manager *Manager) SetSidebar(identifiers []string, groups []ProjectGroup) error {
	settings, loadError := manager.Load()
	if loadError != nil {
		return loadError
	}
	settings.ProjectOrder = identifiers
	settings.ProjectGroups = groups
	return manager.writeSettings(settings)
}

// LastProject returns the identifier of the project that was open most recently, or
// an empty string when none was recorded.
func (manager *Manager) LastProject() (string, error) {
	settings, loadError := manager.Load()
	if loadError != nil {
		return "", loadError
	}
	return settings.LastProject, nil
}

// SetLastProject records the project that is now open, leaving every other setting
// as it is. An empty identifier clears the record.
func (manager *Manager) SetLastProject(identifier string) error {
	settings, loadError := manager.Load()
	if loadError != nil {
		return loadError
	}
	settings.LastProject = identifier
	return manager.writeSettings(settings)
}

// Save writes the settings to the file. The stored preferences the settings UI does
// not edit are carried over when the incoming settings omit them; SetSidebar and
// SetLastProject write them directly and so can clear them.
func (manager *Manager) Save(settings Settings) error {
	stored := manager.storedPreferences()
	if len(settings.ProjectOrder) == 0 && len(settings.ProjectGroups) == 0 {
		settings.ProjectOrder, settings.ProjectGroups = stored.ProjectOrder, stored.ProjectGroups
	}
	if settings.LastProject == "" {
		settings.LastProject = stored.LastProject
	}
	return manager.writeSettings(settings)
}

// writeSettings encodes the settings to the file, creating the parent directory if
// needed.
func (manager *Manager) writeSettings(settings Settings) error {
	directoryError := os.MkdirAll(filepath.Dir(manager.path), 0o755)
	if directoryError != nil {
		return fmt.Errorf("create config directory: %w", directoryError)
	}
	data, encodeError := toml.Marshal(withFallbacks(settings))
	if encodeError != nil {
		return fmt.Errorf("encode settings: %w", encodeError)
	}
	writeError := os.WriteFile(manager.path, data, 0o644)
	if writeError != nil {
		return fmt.Errorf("write settings: %w", writeError)
	}
	return nil
}
