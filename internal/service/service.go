// Package service is the orchestration layer between the Wails-bound application
// and the lower-level packages. It owns the project store and settings manager,
// keeps the currently open project, and returns render-ready views so the UI layer
// stays thin. All public methods are safe for the serial calls a desktop UI makes.
package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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

	mutex   sync.Mutex
	active  *storage.Repository
	history history
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

// ListProjects returns the metadata of every stored project in the user's saved
// sidebar order.
func (service *Service) ListProjects() ([]model.Project, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	return service.orderedProjects()
}

// Sidebar returns the ordered project list together with the stored groups, which
// is everything the sidebar needs to draw itself.
func (service *Service) Sidebar() (SidebarState, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	return service.sidebarState()
}

// SaveSidebar stores a new project order and set of groups, and returns the
// resulting state so the caller can apply it without a second round trip. Both are
// normalised first, so identifiers of deleted projects and groups left empty by a
// drag are dropped rather than stored.
func (service *Service) SaveSidebar(order []string, groups []config.ProjectGroup) (SidebarState, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	projects, listError := service.store.ListProjects()
	if listError != nil {
		return SidebarState{}, listError
	}
	known := make(map[string]bool, len(projects))
	for _, project := range projects {
		known[project.ID] = true
	}
	cleanOrder, cleanGroups := normaliseSidebar(order, groups, known)
	saveError := service.settings.SetSidebar(cleanOrder, cleanGroups)
	if saveError != nil {
		return SidebarState{}, saveError
	}
	return service.sidebarState()
}

// sidebarState assembles the ordered projects and the stored groups. The caller
// holds the mutex.
func (service *Service) sidebarState() (SidebarState, error) {
	projects, listError := service.orderedProjects()
	if listError != nil {
		return SidebarState{}, listError
	}
	_, groups, settingsError := service.settings.Sidebar()
	if settingsError != nil {
		return SidebarState{}, settingsError
	}
	if groups == nil {
		groups = []config.ProjectGroup{}
	}
	return SidebarState{Projects: projects, Groups: groups}, nil
}

// orderedProjects lists the stored projects sorted by the saved sidebar order.
// Projects the order does not mention — newly created or imported ones — keep the
// store's name ordering and follow the ones it does. The caller holds the mutex.
func (service *Service) orderedProjects() ([]model.Project, error) {
	projects, listError := service.store.ListProjects()
	if listError != nil {
		return nil, listError
	}
	order, _, orderError := service.settings.Sidebar()
	if orderError != nil {
		return nil, orderError
	}

	positions := make(map[string]int, len(order))
	for position, identifier := range order {
		positions[identifier] = position
	}
	sort.SliceStable(projects, func(first, second int) bool {
		firstPosition, firstKnown := positions[projects[first].ID]
		secondPosition, secondKnown := positions[projects[second].ID]
		if firstKnown != secondKnown {
			return firstKnown
		}
		if !firstKnown {
			return false
		}
		return firstPosition < secondPosition
	})
	return projects, nil
}

// CreateProject creates a new empty project and returns its metadata.
func (service *Service) CreateProject(name, description, colour, icon string) (model.Project, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
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
		activeRepository := service.active
		service.active = nil
		if closeError := activeRepository.Close(); closeError != nil {
			return fmt.Errorf("close project before delete: %w", closeError)
		}
	}
	return service.store.DeleteProject(identifier)
}

// OpenProject makes the given project active and returns its full view.
func (service *Service) OpenProject(identifier string) (ProjectView, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.active != nil {
		activeRepository := service.active
		service.active = nil
		if closeError := activeRepository.Close(); closeError != nil {
			return ProjectView{}, fmt.Errorf("close previous project: %w", closeError)
		}
	}
	repository, openError := service.store.Open(identifier)
	if openError != nil {
		return ProjectView{}, openError
	}
	service.active = repository
	// Undo snapshots describe the project that was just closed, so they are worthless
	// against the one being opened.
	service.history.clear()
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
	service.mutex.Lock()
	defer service.mutex.Unlock()
	return service.settings.Load()
}

// SaveSettings persists updated user settings.
func (service *Service) SaveSettings(settings config.Settings) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()
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
// active repository untouched. The close error is deliberately ignored: the
// repository was only read from and its result has already been returned.
func (service *Service) releaseIfNotActive(repository *storage.Repository) {
	if repository != service.active {
		repository.Close()
	}
}
