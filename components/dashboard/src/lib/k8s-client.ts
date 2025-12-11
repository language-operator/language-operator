import * as k8s from '@kubernetes/client-node'

class KubernetesClient {
  private static instance: KubernetesClient
  private kc: k8s.KubeConfig
  private coreV1Api: k8s.CoreV1Api
  private customObjectsApi: k8s.CustomObjectsApi

  private constructor() {
    this.kc = new k8s.KubeConfig()

    // Load config based on environment
    if (process.env.NODE_ENV === 'development') {
      // Local development: use ~/.kube/config
      this.kc.loadFromDefault()
    } else {
      // Production: use in-cluster service account
      this.kc.loadFromCluster()
    }

    this.coreV1Api = this.kc.makeApiClient(k8s.CoreV1Api)
    this.customObjectsApi = this.kc.makeApiClient(k8s.CustomObjectsApi)
  }

  public static getInstance(): KubernetesClient {
    if (!KubernetesClient.instance) {
      KubernetesClient.instance = new KubernetesClient()
    }
    return KubernetesClient.instance
  }

  // Core V1 API methods

  async listNamespaces() {
    return await this.coreV1Api.listNamespace({})
  }

  async getNamespace(name: string) {
    return await this.coreV1Api.readNamespace({ name })
  }

  async createNamespace(name: string, labels?: Record<string, string>) {
    const namespace: k8s.V1Namespace = {
      metadata: {
        name,
        labels,
      },
    }
    return await this.coreV1Api.createNamespace({ body: namespace })
  }

  async getPodLogs(namespace: string, podName: string, tailLines: number = 100) {
    return await this.coreV1Api.readNamespacedPodLog({
      name: podName,
      namespace,
      tailLines,
    })
  }

  // Custom Resource methods for language-operator CRDs

  async listLanguageAgents(namespace: string, options?: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
    continue?: string
  }) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      ...options,
    })
  }

  async getLanguageAgent(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      name,
    })
  }

  async createLanguageAgent(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      body: spec,
    })
  }

  async updateLanguageAgent(namespace: string, name: string, spec: any) {
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      name,
      body: spec,
    })
  }

  async deleteLanguageAgent(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      name,
    })
  }

  // LanguageModel methods

  async listLanguageModels(namespace: string, options?: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
    continue?: string
  }) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      ...options,
    })
  }

  async getLanguageModel(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      name,
    })
  }

  async createLanguageModel(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      body: spec,
    })
  }

  async updateLanguageModel(namespace: string, name: string, spec: any) {
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      name,
      body: spec,
    })
  }

  async deleteLanguageModel(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      name,
    })
  }

  // LanguageTool methods

  async listLanguageTools(namespace: string, options?: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
    continue?: string
  }) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      ...options,
    })
  }

  async getLanguageTool(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      name,
    })
  }

  async createLanguageTool(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      body: spec,
    })
  }

  async updateLanguageTool(namespace: string, name: string, spec: any) {
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      name,
      body: spec,
    })
  }

  async deleteLanguageTool(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      name,
    })
  }

  // LanguagePersona methods

  async listLanguagePersonas(namespace: string, options?: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
    continue?: string
  }) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      ...options,
    })
  }

  async getLanguagePersona(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      name,
    })
  }

  async createLanguagePersona(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      body: spec,
    })
  }

  async updateLanguagePersona(namespace: string, name: string, spec: any) {
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      name,
      body: spec,
    })
  }

  async deleteLanguagePersona(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      name,
    })
  }

  // LanguageCluster methods

  async listLanguageClusters(namespace: string, options?: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
    continue?: string
  }) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      ...options,
    })
  }

  async getLanguageCluster(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      name,
    })
  }

  async createLanguageCluster(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      body: spec,
    })
  }

  async updateLanguageCluster(namespace: string, name: string, spec: any) {
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      name,
      body: spec,
    })
  }

  async deleteLanguageCluster(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      name,
    })
  }

  // LanguageAgentVersion methods

  async listLanguageAgentVersions(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
    })
  }

  async getLanguageAgentVersion(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
      name,
    })
  }

  async createLanguageAgentVersion(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
      body: spec,
    })
  }

  async updateLanguageAgentVersion(namespace: string, name: string, spec: any) {
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
      name,
      body: spec,
    })
  }

  async deleteLanguageAgentVersion(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
      name,
    })
  }

  // Helper methods for common query patterns

  /**
   * List resources in namespace with organization filtering
   */
  async listByOrganization(resourceType: 'agents' | 'models' | 'tools' | 'personas' | 'clusters', namespace: string, organizationId: string) {
    const labelSelector = `langop.io/organization=${organizationId}`
    
    switch (resourceType) {
      case 'agents':
        return this.listLanguageAgents(namespace, { labelSelector })
      case 'models':
        return this.listLanguageModels(namespace, { labelSelector })
      case 'tools':
        return this.listLanguageTools(namespace, { labelSelector })
      case 'personas':
        return this.listLanguagePersonas(namespace, { labelSelector })
      case 'clusters':
        return this.listLanguageClusters(namespace, { labelSelector })
      default:
        throw new Error(`Unknown resource type: ${resourceType}`)
    }
  }

  /**
   * List resources by status phase
   */
  async listByPhase(resourceType: 'agents' | 'models' | 'tools' | 'personas' | 'clusters', namespace: string, phase: string) {
    const fieldSelector = `status.phase=${phase}`
    
    switch (resourceType) {
      case 'agents':
        return this.listLanguageAgents(namespace, { fieldSelector })
      case 'models':
        return this.listLanguageModels(namespace, { fieldSelector })
      case 'tools':
        return this.listLanguageTools(namespace, { fieldSelector })
      case 'personas':
        return this.listLanguagePersonas(namespace, { fieldSelector })
      case 'clusters':
        return this.listLanguageClusters(namespace, { fieldSelector })
      default:
        throw new Error(`Unknown resource type: ${resourceType}`)
    }
  }

  /**
   * List resources created by a specific user
   */
  async listByCreator(resourceType: 'agents' | 'models' | 'tools' | 'personas' | 'clusters', namespace: string, userId: string) {
    const labelSelector = `langop.io/created-by=${userId}`
    
    switch (resourceType) {
      case 'agents':
        return this.listLanguageAgents(namespace, { labelSelector })
      case 'models':
        return this.listLanguageModels(namespace, { labelSelector })
      case 'tools':
        return this.listLanguageTools(namespace, { labelSelector })
      case 'personas':
        return this.listLanguagePersonas(namespace, { labelSelector })
      case 'clusters':
        return this.listLanguageClusters(namespace, { labelSelector })
      default:
        throw new Error(`Unknown resource type: ${resourceType}`)
    }
  }

  /**
   * Get resource counts for namespace dashboard
   */
  async getNamespaceResourceCounts(namespace: string, organizationId?: string) {
    const labelSelector = organizationId ? `langop.io/organization=${organizationId}` : undefined
    const options = labelSelector ? { labelSelector } : undefined

    const [agents, models, tools, personas, clusters] = await Promise.all([
      this.listLanguageAgents(namespace, options),
      this.listLanguageModels(namespace, options),
      this.listLanguageTools(namespace, options),
      this.listLanguagePersonas(namespace, options),
      this.listLanguageClusters(namespace, options),
    ])

    return {
      agents: (agents.body as any)?.items?.length || 0,
      models: (models.body as any)?.items?.length || 0,
      tools: (tools.body as any)?.items?.length || 0,
      personas: (personas.body as any)?.items?.length || 0,
      clusters: (clusters.body as any)?.items?.length || 0,
    }
  }

  /**
   * Search resources across all types in a namespace
   */
  async searchResources(namespace: string, query: string, organizationId?: string) {
    const baseSelector = organizationId ? `langop.io/organization=${organizationId}` : undefined
    const options = baseSelector ? { labelSelector: baseSelector } : undefined

    const [agents, models, tools, personas, clusters] = await Promise.all([
      this.listLanguageAgents(namespace, options),
      this.listLanguageModels(namespace, options),
      this.listLanguageTools(namespace, options),
      this.listLanguagePersonas(namespace, options),
      this.listLanguageClusters(namespace, options),
    ])

    const queryLower = query.toLowerCase()
    const results: Array<{
      type: string
      name: string
      namespace: string
      resource: any
    }> = []

    // Filter agents by name
    const agentItems = (agents.body as any)?.items || []
    agentItems.forEach((agent: any) => {
      if (agent.metadata?.name?.toLowerCase().includes(queryLower)) {
        results.push({
          type: 'agent',
          name: agent.metadata.name,
          namespace: agent.metadata.namespace,
          resource: agent,
        })
      }
    })

    // Filter models by name and provider
    const modelItems = (models.body as any)?.items || []
    modelItems.forEach((model: any) => {
      if (
        model.metadata?.name?.toLowerCase().includes(queryLower) ||
        model.spec?.provider?.toLowerCase().includes(queryLower) ||
        model.spec?.modelName?.toLowerCase().includes(queryLower)
      ) {
        results.push({
          type: 'model',
          name: model.metadata.name,
          namespace: model.metadata.namespace,
          resource: model,
        })
      }
    })

    // Filter tools by name and type
    const toolItems = (tools.body as any)?.items || []
    toolItems.forEach((tool: any) => {
      if (
        tool.metadata?.name?.toLowerCase().includes(queryLower) ||
        tool.spec?.type?.toLowerCase().includes(queryLower)
      ) {
        results.push({
          type: 'tool',
          name: tool.metadata.name,
          namespace: tool.metadata.namespace,
          resource: tool,
        })
      }
    })

    // Filter personas by name and tone
    const personaItems = (personas.body as any)?.items || []
    personaItems.forEach((persona: any) => {
      if (
        persona.metadata?.name?.toLowerCase().includes(queryLower) ||
        persona.spec?.tone?.toLowerCase().includes(queryLower)
      ) {
        results.push({
          type: 'persona',
          name: persona.metadata.name,
          namespace: persona.metadata.namespace,
          resource: persona,
        })
      }
    })

    // Filter clusters by name and domain
    const clusterItems = (clusters.body as any)?.items || []
    clusterItems.forEach((cluster: any) => {
      if (
        cluster.metadata?.name?.toLowerCase().includes(queryLower) ||
        cluster.spec?.domain?.toLowerCase().includes(queryLower)
      ) {
        results.push({
          type: 'cluster',
          name: cluster.metadata.name,
          namespace: cluster.metadata.namespace,
          resource: cluster,
        })
      }
    })

    return results
  }
}

// Export singleton instance
export const k8sClient = KubernetesClient.getInstance()
