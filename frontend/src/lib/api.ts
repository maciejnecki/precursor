// Thin typed seam over the generated Wails bindings so the rest of the frontend
// imports application functions from one place rather than reaching into wailsjs.
import * as bindings from '../../wailsjs/go/main/App'
import { config, model, service } from '../../wailsjs/go/models'

export type Project = model.Project
export type ProjectView = service.ProjectView
export type NodeView = service.NodeView
export type Settings = config.Settings
export type StatusColours = config.StatusColours

export const api = {
  listProjects: bindings.ListProjects,
  createProject: bindings.CreateProject,
  updateProject: bindings.UpdateProject,
  deleteProject: bindings.DeleteProject,
  openProject: bindings.OpenProject,
  currentView: bindings.CurrentView,
  createTask: bindings.CreateTask,
  createPrecursor: bindings.CreatePrecursor,
  createDecision: bindings.CreateDecision,
  createDecisionAfter: bindings.CreateDecisionAfter,
  updateNode: bindings.UpdateNode,
  setDecisionsCollapsed: bindings.SetDecisionsCollapsed,
  deleteNode: bindings.DeleteNode,
  createProximity: bindings.CreateProximity,
  createProximityGroup: bindings.CreateProximityGroup,
  deleteProximity: bindings.DeleteProximity,
  getSettings: bindings.GetSettings,
  saveSettings: bindings.SaveSettings,
  getCompletedMarkdown: bindings.GetCompletedMarkdown,
  saveCompletedMarkdown: bindings.SaveCompletedMarkdown,
  exportProject: bindings.ExportProject,
  importProject: bindings.ImportProject
}
