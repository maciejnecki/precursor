package service

import (
	"precursor/internal/export"
	"precursor/internal/model"
	"precursor/internal/storage"
)

// CompletedMarkdown returns a Markdown table of the project's completed tasks and
// the decisions that led to their completion.
func (service *Service) CompletedMarkdown(identifier string) (string, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	repository, openError := service.repositoryFor(identifier)
	if openError != nil {
		return "", openError
	}
	defer service.releaseIfNotActive(repository)

	nodes, nodesError := repository.Nodes()
	if nodesError != nil {
		return "", nodesError
	}
	return export.CompletedTable(nodes), nil
}

// Backup returns the JSON backup bytes for a project.
func (service *Service) Backup(identifier string) ([]byte, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	repository, openError := service.repositoryFor(identifier)
	if openError != nil {
		return nil, openError
	}
	defer service.releaseIfNotActive(repository)

	meta, metaError := repository.Meta()
	if metaError != nil {
		return nil, metaError
	}
	projectGraph, graphError := repository.Graph()
	if graphError != nil {
		return nil, graphError
	}
	return export.MarshalBackup(meta, projectGraph)
}

// ImportBackup creates a brand-new project from JSON backup bytes and returns its
// metadata. The project always receives a fresh identifier so imports never
// overwrite an existing project.
func (service *Service) ImportBackup(data []byte) (model.Project, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	backup, parseError := export.UnmarshalBackup(data)
	if parseError != nil {
		return model.Project{}, parseError
	}

	project, creationError := service.store.CreateProject(
		backup.Project.Name,
		backup.Project.Description,
		backup.Project.Colour,
		backup.Project.Icon,
	)
	if creationError != nil {
		return model.Project{}, creationError
	}

	repository, openError := service.store.Open(project.ID)
	if openError != nil {
		return model.Project{}, openError
	}
	defer repository.Close()

	// Insert every node and bond in one transaction so a malformed backup cannot
	// leave a partially populated project behind.
	importError := repository.WithinTransaction(func(transactional *storage.Repository) error {
		for _, node := range backup.Graph.Nodes {
			if insertError := transactional.InsertNode(node); insertError != nil {
				return insertError
			}
		}
		for _, bond := range backup.Graph.ProximityBonds {
			if insertError := transactional.InsertProximityBond(bond); insertError != nil {
				return insertError
			}
		}
		return nil
	})
	if importError != nil {
		return model.Project{}, importError
	}
	return project, nil
}
