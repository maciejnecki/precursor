// Package service is the orchestration layer between the Wails-bound application
// and the lower-level packages. It owns the project store and settings manager,
// keeps the currently open project, and returns render-ready views so the UI layer
// stays thin. All public methods are safe for the serial calls a desktop UI makes.
package service

import (
	"errors"
	"path/filepath"
	"sync"

	"precursor/internal/config"
	"precursor/internal/model"
	"precursor/internal/storage"
)

// errNoActiveProject is returned when a graph operation is attempted with no
// project open.
var errNoActiveProject = errors.New("no project is open")

// Service holds the long-lived state shared by every application method.
type Service struct {
	store    *storage.Store
	settings *config.Manager

	mutex  sync.Mutex
	active *storage.Repository
}

// New builds a service rooted at the given base directory, placing project
// databases under a projects subdirectory and settings in config.toml beside them.
func New(baseDirectory string) (*Service, error) {
	store, storeError := storage.NewStore(filepath.Join(baseDirectory, "projects"))
	if storeError != nil {
		return nil, storeError
	}
	settings := config.NewManager(filepath.Join(baseDirectory, "config.toml"))
	return &Service{store: store, settings: settings}, nil
}

// ListProjects returns the metadata of every stored project.
func (service *Service) ListProjects() ([]model.Project, error) {
	return service.store.ListProjects()
}

// CreateProject creates a new empty project and returns its metadata.
func (service *Service) CreateProject(name, description, colour, icon string) (model.Project, error) {
	return service.store.CreateProject(name, description, colour, icon)
}

// UpdateProject changes the metadata of an existing project.
func (service *Service) UpdateProject(identifier, name, description, colour, icon string) (model.Project, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	repository, openError := service.repositoryFor(identifier)
	if openError != nil {
		return model.Project{}, openError
	}
	defer service.releaseIfNotActive(repository)

	if updateError := repository.UpdateMeta(name, description, colour, icon); updateError != nil {
		return model.Project{}, updateError
	}
	return repository.Meta()
}

// DeleteProject removes a project, closing it first if it is currently open.
func (service *Service) DeleteProject(identifier string) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.active != nil && service.active.ID() == identifier {
		service.active.Close()
		service.active = nil
	}
	return service.store.DeleteProject(identifier)
}

// OpenProject makes the given project active and returns its full view.
func (service *Service) OpenProject(identifier string) (ProjectView, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.active != nil {
		service.active.Close()
		service.active = nil
	}
	repository, openError := service.store.Open(identifier)
	if openError != nil {
		return ProjectView{}, openError
	}
	service.active = repository
	return service.activeView()
}

// CurrentView returns the view of the active project, refreshed from storage.
func (service *Service) CurrentView() (ProjectView, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	return service.activeView()
}

// Settings returns the current user settings.
func (service *Service) Settings() (config.Settings, error) {
	return service.settings.Load()
}

// SaveSettings persists updated user settings.
func (service *Service) SaveSettings(settings config.Settings) error {
	return service.settings.Save(settings)
}

// activeView builds the view for the active project, requiring one to be open.
func (service *Service) activeView() (ProjectView, error) {
	if service.active == nil {
		return ProjectView{}, errNoActiveProject
	}
	meta, metaError := service.active.Meta()
	if metaError != nil {
		return ProjectView{}, metaError
	}
	projectGraph, graphError := service.active.Graph()
	if graphError != nil {
		return ProjectView{}, graphError
	}
	return buildView(meta, projectGraph), nil
}

// repositoryFor returns the active repository when it matches the identifier, or
// opens a temporary one otherwise so project edits work without opening a project.
func (service *Service) repositoryFor(identifier string) (*storage.Repository, error) {
	if service.active != nil && service.active.ID() == identifier {
		return service.active, nil
	}
	return service.store.Open(identifier)
}

// releaseIfNotActive closes a repository that was opened temporarily, leaving the
// active repository untouched.
func (service *Service) releaseIfNotActive(repository *storage.Repository) {
	if repository != service.active {
		repository.Close()
	}
}
