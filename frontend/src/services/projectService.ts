import {
  ListProjects,
  CreateProject,
  RenameProject,
  DeleteProject,
  ListFolders,
  CreateFolder,
  RenameFolder,
  DeleteFolder,
  MoveFolder,
  ListRequests,
  CreateRequestItem,
  SaveRequestItem,
  DeleteRequestItem,
  MoveRequestItem,
  GetCollectionData,
  MoveCollectionNode,
  RenameRequest,
  DuplicateRequest,
  DuplicateFolder,
  ExportProjectJSON,
  ImportFromFile,
  ImportFromURL,
  ImportCollectionWithStrategy,
  PreviewImportFromFile,
  PreviewImportFromURL,
  UpdateProjectDescription,
  UpdateProjectTheme,
} from "../../wailsjs/go/main/App"
import type { model } from "../../wailsjs/go/models"

type CollectionNodeType = "folder" | "request"

export const projectService = {
  listProjects: () => ListProjects(),
  createProject: (name: string) => CreateProject(name),
  renameProject: (id: string, name: string) => RenameProject(id, name),
  updateProjectDescription: (id: string, description: string) => UpdateProjectDescription(id, description),
  updateProjectTheme: (id: string, color: string) => UpdateProjectTheme(id, color),
  deleteProject: (id: string) => DeleteProject(id),
  exportProjectJSON: (projectID: string) => ExportProjectJSON(projectID),
}

export const collectionService = {
  getCollectionData: (projectID: string) => GetCollectionData(projectID),
  moveCollectionNode: (
    projectID: string,
    nodeID: string,
    nodeType: CollectionNodeType,
    targetParentFolderID: string,
    targetIndex: number
  ) => MoveCollectionNode(projectID, nodeID, nodeType, targetParentFolderID, targetIndex),
  importFromFile: (projectID: string, format: string, content: string) =>
    ImportFromFile(projectID, format, content),
  importFromURL: (projectID: string, format: string, sourceURL: string) =>
    ImportFromURL(projectID, format, sourceURL),
  previewImportFromFile: (projectID: string, format: string, content: string) =>
    PreviewImportFromFile(projectID, format, content),
  previewImportFromURL: (projectID: string, format: string, sourceURL: string) =>
    PreviewImportFromURL(projectID, format, sourceURL),
  importWithStrategy: (projectID: string, format: string, content: string, sourceURL: string, strategy: string) =>
    ImportCollectionWithStrategy(projectID, format, content, sourceURL, strategy),
}

export const folderService = {
  listFolders: (projectID: string) => ListFolders(projectID),
  createFolder: (projectID: string, parentID: string, name: string) =>
    CreateFolder(projectID, parentID, name),
  renameFolder: (projectID: string, folderID: string, name: string) =>
    RenameFolder(projectID, folderID, name),
  deleteFolder: (projectID: string, folderID: string) =>
    DeleteFolder(projectID, folderID),
  moveFolder: (projectID: string, folderID: string, targetParentID: string, targetIndex: number) =>
    MoveFolder(projectID, folderID, targetParentID, targetIndex),
  duplicateFolder: (projectID: string, folderID: string) =>
    DuplicateFolder(projectID, folderID),
}

export const requestItemService = {
  listRequests: (projectID: string) => ListRequests(projectID),
  createRequest: (projectID: string, folderID: string, name: string) =>
    CreateRequestItem(projectID, folderID, name),
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  saveRequest: (request: model.RequestItem) => SaveRequestItem(request as any),
  deleteRequest: (projectID: string, requestID: string) =>
    DeleteRequestItem(projectID, requestID),
  moveRequest: (projectID: string, requestID: string, targetFolderID: string, targetIndex: number) =>
    MoveRequestItem(projectID, requestID, targetFolderID, targetIndex),
  renameRequest: (projectID: string, requestID: string, name: string) =>
    RenameRequest(projectID, requestID, name),
  duplicateRequest: (projectID: string, requestID: string) =>
    DuplicateRequest(projectID, requestID),
}
