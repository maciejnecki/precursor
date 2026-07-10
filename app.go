package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"precursor/internal/config"
	"precursor/internal/model"
	"precursor/internal/service"
)

// App is the Wails-bound surface of the application. It keeps the runtime context
// for native dialogs and delegates all real work to the service layer.
type App struct {
	ctx       context.Context
	service   *service.Service
	version   string
	initError error
}

// NewApp creates a new App carrying the build version; the rest of the state is
// initialised at startup.
func NewApp(version string) *App {
	return &App{version: version}
}

// Version returns the build version injected at compile time, or "dev" when the
// app runs without an injected version (wails dev and plain go builds).
func (app *App) Version() string {
	return app.version
}

// startup stores the runtime context and initialises the service against the
// per-user application data directory.
func (app *App) startup(ctx context.Context) {
	app.ctx = ctx
	baseDirectory, directoryError := applicationDataDirectory()
	if directoryError != nil {
		app.initError = directoryError
		return
	}
	createdService, serviceError := service.New(baseDirectory)
	if serviceError != nil {
		app.initError = serviceError
		return
	}
	app.service = createdService
}

// applicationDataDirectory returns the per-user directory that holds projects and
// settings.
func applicationDataDirectory() (string, error) {
	configDirectory, lookupError := os.UserConfigDir()
	if lookupError != nil {
		return "", lookupError
	}
	return filepath.Join(configDirectory, "precursor"), nil
}

// ready returns the initialised service or the startup error.
func (app *App) ready() (*service.Service, error) {
	if app.service == nil {
		if app.initError != nil {
			return nil, app.initError
		}
		return nil, errors.New("application is not initialised")
	}
	return app.service, nil
}

// ListProjects returns metadata for every project.
func (app *App) ListProjects() ([]model.Project, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return nil, readyError
	}
	return current.ListProjects()
}

// CreateProject creates a new project.
func (app *App) CreateProject(name, description, colour, icon string) (model.Project, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return model.Project{}, readyError
	}
	return current.CreateProject(name, description, colour, icon)
}

// UpdateProject changes a project's metadata.
func (app *App) UpdateProject(identifier, name, description, colour, icon string) (model.Project, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return model.Project{}, readyError
	}
	return current.UpdateProject(identifier, name, description, colour, icon)
}

// DeleteProject removes a project.
func (app *App) DeleteProject(identifier string) error {
	current, readyError := app.ready()
	if readyError != nil {
		return readyError
	}
	return current.DeleteProject(identifier)
}

// OpenProject makes a project active and returns its view.
func (app *App) OpenProject(identifier string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.OpenProject(identifier)
}

// CurrentView returns the active project's view.
func (app *App) CurrentView() (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.CurrentView()
}

// CreateTask adds an endpoint task to the active project.
func (app *App) CreateTask(title, body, icon string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.CreateTask(title, body, icon)
}

// CreatePrecursor adds a precursor task to the given parent.
func (app *App) CreatePrecursor(parentID, title, body, icon string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.CreatePrecursor(parentID, title, body, icon)
}

// CreateDecision adds a decision documenting the given task.
func (app *App) CreateDecision(childID, decisionType, title, body, icon string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.CreateDecision(childID, decisionType, title, body, icon)
}

// UpdateNode edits the content of an existing node.
func (app *App) UpdateNode(identifier, title, body, icon string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.UpdateNode(identifier, title, body, icon)
}

// CreateDecisionAfter inserts a decision downstream of an existing decision.
func (app *App) CreateDecisionAfter(decisionID, decisionType, title, body, icon string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.CreateDecisionAfter(decisionID, decisionType, title, body, icon)
}

// SetDecisionsCollapsed toggles whether a task hides the decisions on the link to
// its parent.
func (app *App) SetDecisionsCollapsed(identifier string, collapsed bool) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.SetDecisionsCollapsed(identifier, collapsed)
}

// DeleteNode removes a node and heals its chain.
func (app *App) DeleteNode(identifier string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.DeleteNode(identifier)
}

// CreateProximity bonds the chains of the two given nodes.
func (app *App) CreateProximity(nodeAID, nodeBID string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.CreateProximity(nodeAID, nodeBID)
}

// CreateProximityGroup bonds the chains of all the given nodes so they cluster.
func (app *App) CreateProximityGroup(nodeIDs []string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.CreateProximityGroup(nodeIDs)
}

// DeleteProximity removes a proximity bond.
func (app *App) DeleteProximity(identifier string) (service.ProjectView, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return service.ProjectView{}, readyError
	}
	return current.DeleteProximity(identifier)
}

// GetSettings returns the current user settings.
func (app *App) GetSettings() (config.Settings, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return config.Settings{}, readyError
	}
	return current.Settings()
}

// SaveSettings persists updated user settings.
func (app *App) SaveSettings(settings config.Settings) error {
	current, readyError := app.ready()
	if readyError != nil {
		return readyError
	}
	return current.SaveSettings(settings)
}

// GetCompletedMarkdown returns the completed-tasks Markdown table for clipboard use.
func (app *App) GetCompletedMarkdown(identifier string) (string, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return "", readyError
	}
	return current.CompletedMarkdown(identifier)
}

// SaveCompletedMarkdown writes the completed-tasks table to a file chosen by the
// user and returns the chosen path, or an empty string when cancelled.
func (app *App) SaveCompletedMarkdown(identifier string) (string, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return "", readyError
	}
	markdown, markdownError := current.CompletedMarkdown(identifier)
	if markdownError != nil {
		return "", markdownError
	}
	path, dialogError := runtime.SaveFileDialog(app.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "completed.md",
		Filters:         []runtime.FileFilter{{DisplayName: "Markdown", Pattern: "*.md"}},
	})
	if dialogError != nil || path == "" {
		return "", dialogError
	}
	return path, os.WriteFile(path, []byte(markdown), 0o644)
}

// ExportProject writes a project's JSON backup to a file chosen by the user and
// returns the chosen path, or an empty string when cancelled.
func (app *App) ExportProject(identifier string) (string, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return "", readyError
	}
	data, backupError := current.Backup(identifier)
	if backupError != nil {
		return "", backupError
	}
	path, dialogError := runtime.SaveFileDialog(app.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "project.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON backup", Pattern: "*.json"}},
	})
	if dialogError != nil || path == "" {
		return "", dialogError
	}
	return path, os.WriteFile(path, data, 0o644)
}

// ImportProject reads a JSON backup chosen by the user and creates a new project
// from it, returning the new project's metadata.
func (app *App) ImportProject() (model.Project, error) {
	current, readyError := app.ready()
	if readyError != nil {
		return model.Project{}, readyError
	}
	path, dialogError := runtime.OpenFileDialog(app.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{{DisplayName: "JSON backup", Pattern: "*.json"}},
	})
	if dialogError != nil || path == "" {
		return model.Project{}, dialogError
	}
	data, readError := os.ReadFile(path)
	if readError != nil {
		return model.Project{}, readError
	}
	return current.ImportBackup(data)
}
