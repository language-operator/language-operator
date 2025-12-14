export interface ToolCatalogEntry {
  name: string
  displayName: string
  description: string
  image: string
  deploymentMode: 'service' | 'job' | 'cronjob'
  port?: number
  type: 'mcp' | 'openapi' | 'grpc'
  authRequired?: boolean
  rbac?: {
    clusterRole?: {
      rules: Array<{
        apiGroups: string[]
        resources: string[]
        verbs: string[]
      }>
    }
  }
  egress?: Array<{
    description: string
    dns?: string[]
    ports?: Array<{
      port: number
      protocol: string
    }>
  }>
  configSchema?: any
  tags?: string[]
  version?: string
  maintainer?: string
  homepage?: string
  repository?: string
}

export interface ToolCatalog {
  version: string
  generated: string
  tools: Record<string, ToolCatalogEntry>
}

export interface InstalledTool {
  name: string
  namespace: string
  catalogName?: string
  status: {
    phase: 'Pending' | 'Ready' | 'Failed'
    message?: string
    lastUpdated: string
  }
  spec: any
}