import {
  ListEnvironments,
  CreateEnvironment,
  SaveEnvironment,
  DeleteEnvironment,
} from "../../wailsjs/go/main/App"
import type { model } from "../../wailsjs/go/models"

export const environmentService = {
  listEnvironments: (projectID: string) => ListEnvironments(projectID),
  createEnvironment: (projectID: string, name: string) =>
    CreateEnvironment(projectID, name),
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  saveEnvironment: (env: model.Environment) => SaveEnvironment(env as any),
  deleteEnvironment: (projectID: string, envID: string) =>
    DeleteEnvironment(projectID, envID),
}
