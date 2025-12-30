import { KubeConfig, CustomObjectsApi, CoreV1Api } from '@kubernetes/client-node'

export interface LanguageAgent {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    creationTimestamp?: string
    resourceVersion?: string
  }
  spec: {
    model: string
    persona?: string
    tools?: string[]
    description?: string
  }
  status?: {
    phase: string
    conditions?: Array<{
      type: string
      status: string
      lastTransitionTime: string
      reason?: string
      message?: string
    }>
    ready: boolean
  }
}

export interface LanguageModel {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    creationTimestamp?: string
    resourceVersion?: string
  }
  spec: {
    provider: string
    version: string
    endpoint?: string
    contextWindow?: number
    costPer1kTokens?: number // Legacy field
    costTracking?: {
      enabled: boolean
      currency: string
      inputTokenCost: number
      outputTokenCost: number
    }
    secretRef?: {
      name: string
      key: string
    }
  }
  status?: {
    phase: string
    lastUsed?: string
    totalRequests?: number
    ready: boolean
  }
}

export interface LanguageTool {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    creationTimestamp?: string
    resourceVersion?: string
  }
  spec: {
    description: string
    category: string
    version: string
    parameters: Array<{
      name: string
      type: string
      required: boolean
      description?: string
    }>
    implementation: {
      type: string
      config: Record<string, any>
    }
  }
  status?: {
    phase: string
    usageCount: number
    lastUsed?: string
    ready: boolean
  }
}

export interface LanguagePersona {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    creationTimestamp?: string
    resourceVersion?: string
  }
  spec: {
    description: string
    personality: string
    tone: string
    specialization: string
    traits: string[]
    category: string
  }
  status?: {
    phase: string
    agentCount: number
    ready: boolean
  }
}

export interface LanguageCluster {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    creationTimestamp?: string
    resourceVersion?: string
  }
  spec: {
    description: string
    replicas: number
    agents: string[]
    ingress?: {
      enabled: boolean
      host: string
      tls?: boolean
    }
    resources?: {
      limits: {
        cpu: string
        memory: string
      }
      requests: {
        cpu: string
        memory: string
      }
    }
  }
  status?: {
    phase: string
    readyReplicas: number
    requests: number
    uptime: string
    lastDeploy?: string
    ready: boolean
  }
}

class KubernetesClient {
  private kc: KubeConfig
  private customApi: CustomObjectsApi
  private coreApi: CoreV1Api

  constructor() {
    this.kc = new KubeConfig()
    
    if (process.env.NODE_ENV === 'production') {
      this.kc.loadFromCluster()
    } else {
      // Check if we're running in Docker Compose with kubectl proxy
      if (process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy')) {
        // Configure for kubectl proxy access
        this.kc.loadFromString(JSON.stringify({
          apiVersion: 'v1',
          kind: 'Config',
          clusters: [{
            cluster: {
              server: process.env.KUBERNETES_SERVER_URL
            },
            name: 'docker-compose-cluster'
          }],
          contexts: [{
            context: {
              cluster: 'docker-compose-cluster',
              user: 'docker-compose-user'
            },
            name: 'docker-compose'
          }],
          users: [{
            name: 'docker-compose-user',
            user: {} // kubectl proxy handles auth
          }],
          'current-context': 'docker-compose'
        }))
      } else {
        this.kc.loadFromDefault()
      }
    }

    this.customApi = this.kc.makeApiClient(CustomObjectsApi)
    this.coreApi = this.kc.makeApiClient(CoreV1Api)
  }

  async getNamespaces(): Promise<string[]> {
    try {
      const response = await this.coreApi.listNamespace({})
      return response.items.map(ns => ns.metadata?.name || '')
    } catch (error) {
      console.error('Error fetching namespaces:', error)
      return []
    }
  }

  async getLanguageAgents(namespace: string): Promise<LanguageAgent[]> {
    try {
      const response = await this.customApi.listNamespacedCustomObject({
        group: 'language-operator.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languageagents'
      })
      
      return (response.body as any).items || []
    } catch (error) {
      console.error('Error fetching language agents:', error)
      return []
    }
  }

  async getLanguageModels(namespace: string): Promise<LanguageModel[]> {
    try {
      const response = await this.customApi.listNamespacedCustomObject({
        group: 'langop.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languagemodels'
      })
      
      return (response.body as any).items || []
    } catch (error) {
      console.error('Error fetching language models:', error)
      return []
    }
  }

  async getLanguageModel(namespace: string, name: string): Promise<LanguageModel | null> {
    try {
      const response = await this.customApi.getNamespacedCustomObject({
        group: 'langop.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languagemodels',
        name
      })
      
      return response.body as LanguageModel
    } catch (error) {
      console.error(`Error fetching language model ${name}:`, error)
      return null
    }
  }

  async getLanguageTools(namespace: string): Promise<LanguageTool[]> {
    try {
      const response = await this.customApi.listNamespacedCustomObject({
        group: 'language-operator.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languagetools'
      })
      
      return (response.body as any).items || []
    } catch (error) {
      console.error('Error fetching language tools:', error)
      return []
    }
  }

  async getLanguagePersonas(namespace: string): Promise<LanguagePersona[]> {
    try {
      const response = await this.customApi.listNamespacedCustomObject({
        group: 'language-operator.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languagepersonas'
      })
      
      return (response.body as any).items || []
    } catch (error) {
      console.error('Error fetching language personas:', error)
      return []
    }
  }

  async getLanguageClusters(namespace: string): Promise<LanguageCluster[]> {
    try {
      const response = await this.customApi.listNamespacedCustomObject({
        group: 'language-operator.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languageclusters'
      })
      
      return (response.body as any).items || []
    } catch (error) {
      console.error('Error fetching language clusters:', error)
      return []
    }
  }

  async createLanguageAgent(namespace: string, agent: Partial<LanguageAgent>): Promise<LanguageAgent> {
    try {
      const response = await this.customApi.createNamespacedCustomObject({
        group: 'language-operator.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languageagents',
        body: agent
      })
      
      return response.body as LanguageAgent
    } catch (error) {
      console.error('Error creating language agent:', error)
      throw error
    }
  }

  async updateLanguageAgent(namespace: string, name: string, agent: Partial<LanguageAgent>): Promise<LanguageAgent> {
    try {
      const response = await this.customApi.replaceNamespacedCustomObject({
        group: 'language-operator.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languageagents',
        name,
        body: agent
      })
      
      return response.body as LanguageAgent
    } catch (error) {
      console.error('Error updating language agent:', error)
      throw error
    }
  }

  async deleteLanguageAgent(namespace: string, name: string): Promise<void> {
    try {
      await this.customApi.deleteNamespacedCustomObject({
        group: 'language-operator.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languageagents',
        name
      })
    } catch (error) {
      console.error('Error deleting language agent:', error)
      throw error
    }
  }
}

let client: KubernetesClient | null = null

export function getKubernetesClient(): KubernetesClient {
  if (!client) {
    client = new KubernetesClient()
  }
  return client
}