import {
  ListProjects,
  CreateProject,
  RenameProject,
  DeleteProject,
  ListFolders,
  CreateFolder,
  RenameFolder,
  DeleteFolder,
  ListRequests,
  CreateRequestItem,
  SaveRequestItem,
  DeleteRequestItem,
} from "../../wailsjs/go/main/App"
import type { model } from "../../wailsjs/go/models"

export const projectService = {
  listProjects: () => ListProjects(),
  createProject: (name: string) => CreateProject(name),
  renameProject: (id: string, name: string) => RenameProject(id, name),
  deleteProject: (id: string) => DeleteProject(id),
}

export const folderService = {
  listFolders: (projectID: string) => ListFolders(projectID),
  createFolder: (projectID: string, parentID: string, name: string) =>
    CreateFolder(projectID, parentID, name),
  renameFolder: (projectID: string, folderID: string, name: string) =>
    RenameFolder(projectID, folderID, name),
  deleteFolder: (projectID: string, folderID: string) =>
    DeleteFolder(projectID, folderID),
}

export const requestItemService = {
  listRequests: (projectID: string) => ListRequests(projectID),
  createRequest: (projectID: string, folderID: string, name: string) =>
    CreateRequestItem(projectID, folderID, name),
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  saveRequest: (request: model.RequestItem) => SaveRequestItem(request as any),
  deleteRequest: (projectID: string, requestID: string) =>
    DeleteRequestItem(projectID, requestID),
}
