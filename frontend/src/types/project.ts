export interface Project {
  id: string
  name: string
  description?: string
  themeColor?: string
  createdAt: string
  updatedAt: string
  schemaVersion: number
}

export interface Folder {
  id: string
  name: string
  projectId: string
  parentId?: string
  sortOrder: number
}
