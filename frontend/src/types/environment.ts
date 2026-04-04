export interface Variable {
  id: string
  key: string
  value: string
  enabled: boolean
  isSecret: boolean
}

export interface Environment {
  id: string
  name: string
  projectId: string
  variables: Variable[]
}
