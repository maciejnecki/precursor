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

// Save writes the settings to the file, creating the parent directory if needed.
func (manager *Manager) Save(settings Settings) error {
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
