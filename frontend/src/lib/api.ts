// Thin typed seam over the generated Wails bindings so the rest of the frontend
// imports application functions from one place rather than reaching into wailsjs.
import * as bindings from '../../wailsjs/go/main/App'
import { config, model, service } from '../../wailsjs/go/models'

export type Project = model.Project
export type ProjectView = service.ProjectView
export type NodeView = service.NodeView
export type Settings = config.Settings
export type ProjectGroup = config.ProjectGroup
export type SidebarState = service.SidebarState

export const api = {
  listProjects: bindings.ListProjects,
  createProject: bindings.CreateProject,
  updateProject: bindings.UpdateProject,
  deleteProject: bindings.DeleteProject,
  sidebar: bindings.Sidebar,
  saveSidebar: bindings.SaveSidebar,
  openProject: bindings.OpenProject,
  lastProject: bindings.LastProject,
  createTask: bindings.CreateTask,
  createPrecursor: bindings.CreatePrecursor,
  createDecision: bindings.CreateDecision,
  createDecisionAfter: bindings.CreateDecisionAfter,
  updateNode: bindings.UpdateNode,
  setDecisionsCollapsed: bindings.SetDecisionsCollapsed,
  deleteNode: bindings.DeleteNode,
  undo: bindings.Undo,
  redo: bindings.Redo,
  createProximityGroup: bindings.CreateProximityGroup,
  getSettings: bindings.GetSettings,
  saveSettings: bindings.SaveSettings,
  exportProject: bindings.ExportProject,
  importProject: bindings.ImportProject,
  version: bindings.Version
}
