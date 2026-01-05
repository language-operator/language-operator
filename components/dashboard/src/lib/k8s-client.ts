import * as k8s from '@kubernetes/client-node'

interface RequestOptions {
  timeout?: number
  signal?: AbortSignal
}

class KubernetesClient {
  private static instance: KubernetesClient
  private kc: k8s.KubeConfig
  private coreV1Api: k8s.CoreV1Api | null
  private customObjectsApi: k8s.CustomObjectsApi | null
  private batchV1Api: k8s.BatchV1Api | null
  private readonly DEFAULT_TIMEOUT = 10000 // 10 seconds

  private constructor() {
    this.kc = new k8s.KubeConfig()
    this.coreV1Api = null
    this.customObjectsApi = null
    this.batchV1Api = null

    try {
      // Load config based on environment
      if (process.env.KUBERNETES_SERVER_URL && process.env.KUBERNETES_TOKEN) {
        // Environment variable configuration with token
        const cluster = {
          name: 'env-cluster',
          server: process.env.KUBERNETES_SERVER_URL,
          skipTLSVerify: process.env.KUBERNETES_SKIP_TLS_VERIFY === 'true',
        }
        const user = {
          name: 'env-user',
          token: process.env.KUBERNETES_TOKEN,
        }
        const context = {
          name: 'env-context',
          user: user.name,
          cluster: cluster.name,
        }
        this.kc.loadFromOptions({
          clusters: [cluster],
          users: [user],
          contexts: [context],
          currentContext: context.name,
        })
      } else if (process.env.KUBERNETES_SERVER_URL && process.env.KUBERNETES_SERVER_URL.includes('kubectl-proxy')) {
        // Docker Compose mode: use kubectl proxy (no token needed)
        const cluster = {
          name: 'kubectl-proxy-cluster',
          server: process.env.KUBERNETES_SERVER_URL,
          skipTLSVerify: true, // kubectl proxy doesn't use TLS
        }
        const user = {
          name: 'kubectl-proxy-user',
          // No token needed for kubectl proxy
        }
        const context = {
          name: 'kubectl-proxy-context',
          user: user.name,
          cluster: cluster.name,
        }
        this.kc.loadFromOptions({
          clusters: [cluster],
          users: [user],
          contexts: [context],
          currentContext: context.name,
        })
      } else if (process.env.NODE_ENV === 'development') {
        // Local development: use ~/.kube/config if available
        this.kc.loadFromDefault()
      } else {
        // Production: use in-cluster service account
        this.kc.loadFromCluster()
      }

      this.coreV1Api = this.kc.makeApiClient(k8s.CoreV1Api)
      this.customObjectsApi = this.kc.makeApiClient(k8s.CustomObjectsApi)
      this.batchV1Api = this.kc.makeApiClient(k8s.BatchV1Api)
    } catch (error) {
      console.error('❌ Failed to configure Kubernetes client:', error instanceof Error ? error.message : String(error))
      console.error('❌ Full error details:', error)
      console.error('❌ Environment variables:')
      console.error('  KUBERNETES_SERVER_URL:', process.env.KUBERNETES_SERVER_URL)
      console.error('  KUBERNETES_TOKEN:', process.env.KUBERNETES_TOKEN ? '***SET***' : 'NOT_SET')
      console.error('  NODE_ENV:', process.env.NODE_ENV)
      console.error('  KUBECONFIG:', process.env.KUBECONFIG)
      throw new Error('Kubernetes configuration is required')
    }
  }

  public static getInstance(): KubernetesClient {
    if (!KubernetesClient.instance) {
      KubernetesClient.instance = new KubernetesClient()
    }
    return KubernetesClient.instance
  }

  /**
   * Wrap Kubernetes API calls with timeout and cancellation support
   */
  private async withTimeout<T>(
    operation: () => Promise<T>,
    options: RequestOptions = {}
  ): Promise<T> {
    const timeout = options.timeout || this.DEFAULT_TIMEOUT
    
    return Promise.race([
      operation(),
      new Promise<never>((_, reject) => {
        const timeoutId = setTimeout(
          () => reject(new Error(`Operation timed out after ${timeout}ms`)),
          timeout
        )
        
        // Cancel timeout if request is aborted
        if (options.signal) {
          options.signal.addEventListener('abort', () => {
            clearTimeout(timeoutId)
            reject(new Error('Request was cancelled'))
          })
        }
      })
    ])
  }


  // Core V1 API methods

  async listNamespaces() {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    return await this.coreV1Api.listNamespace({})
  }

  async getNamespace(name: string) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    return await this.coreV1Api.readNamespace({ name })
  }

  async createNamespace(name: string, labels?: Record<string, string>) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    const namespace: k8s.V1Namespace = {
      metadata: {
        name,
        labels,
      },
    }
    return await this.coreV1Api.createNamespace({ body: namespace })
  }

  async createOrganizationNamespace(name: string, organizationId: string, plan: string = 'free') {
    try {
      // Create namespace with enhanced organization labels and annotations
      if (!this.coreV1Api) {
        throw new Error('Kubernetes API not available')
      }
      
      const namespace: k8s.V1Namespace = {
        metadata: {
          name,
          labels: {
            'langop.io/organization-id': organizationId,
            'langop.io/managed-by': 'dashboard',
            'langop.io/type': 'organization',
            'langop.io/plan': plan.toLowerCase()
          },
          annotations: {
            'langop.io/created-at': new Date().toISOString(),
            'langop.io/namespace-type': 'uuid-based'
          }
        }
      }
      
      const namespaceResponse = await this.coreV1Api.createNamespace({ body: namespace })

      // Create ResourceQuota for the namespace
      try {
        await this.createResourceQuota(name, organizationId, plan)
      } catch (quotaError: any) {
        console.error(`Failed to create ResourceQuota for namespace ${name}:`, quotaError.message)
        // Don't fail if quota creation fails - namespace is still functional
      }

      return namespaceResponse
    } catch (error: any) {
      // If namespace already exists, that's okay for our use case
      if (error.response?.statusCode === 409) {
        console.log(`Namespace ${name} already exists`)
        return null
      }
      throw error
    }
  }

  async deleteOrganizationNamespace(name: string) {
    try {
      // First delete the ResourceQuota
      await this.deleteResourceQuota(name)
    } catch (quotaError: any) {
      console.error(`Failed to delete ResourceQuota for namespace ${name}:`, quotaError.message)
      // Continue with namespace deletion even if quota deletion fails
    }

    // Then delete the namespace (this will cascade delete all resources)
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    return await this.coreV1Api.deleteNamespace({ name })
  }

  async getPodLogs(namespace: string, podName: string, options?: {
    tailLines?: number
    timestamps?: boolean
    sinceSeconds?: number
    container?: string
    previous?: boolean
  }) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    return await this.coreV1Api.readNamespacedPodLog({
      name: podName,
      namespace,
      tailLines: options?.tailLines || 100,
      timestamps: options?.timestamps || false,
      sinceSeconds: options?.sinceSeconds,
      container: options?.container,
      previous: options?.previous,
    })
  }

  async listPods(namespace: string, options?: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
  }) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    return await this.coreV1Api.listNamespacedPod({
      namespace,
      labelSelector: options?.labelSelector,
      fieldSelector: options?.fieldSelector,
      limit: options?.limit,
    })
  }

  async streamPodLogs(namespace: string, podName: string, options?: {
    follow?: boolean
    tailLines?: number
    timestamps?: boolean
  }) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    
    // Use the log method that returns a stream
    return await this.coreV1Api.readNamespacedPodLog({
      name: podName,
      namespace,
      follow: options?.follow || false,
      tailLines: options?.tailLines || 10,
      timestamps: options?.timestamps || false,
    })
  }

  // ResourceQuota methods for multi-tenant resource management

  private getPlanBasedQuotaSpec(plan: string): Record<string, string> {
    switch (plan.toLowerCase()) {
      case 'free':
        return {
          'count/languageagents': '2',
          'count/languagemodels': '2',
          'count/languagetools': '5',
          'count/languagepersonas': '3',
          'count/languageclusters': '1',
          'requests.cpu': '1000m',
          'requests.memory': '2Gi',
          'limits.cpu': '2000m',
          'limits.memory': '4Gi'
        }
      case 'pro':
        return {
          'count/languageagents': '20',
          'count/languagemodels': '10',
          'count/languagetools': '50',
          'count/languagepersonas': '20',
          'count/languageclusters': '5',
          'requests.cpu': '10000m',
          'requests.memory': '20Gi',
          'limits.cpu': '20000m',
          'limits.memory': '40Gi'
        }
      case 'enterprise':
        return {
          'count/languageagents': '100',
          'count/languagemodels': '50',
          'count/languagetools': '200',
          'count/languagepersonas': '100',
          'count/languageclusters': '20',
          'requests.cpu': '50000m',
          'requests.memory': '100Gi',
          'limits.cpu': '100000m',
          'limits.memory': '200Gi'
        }
      default:
        return this.getPlanBasedQuotaSpec('free')
    }
  }

  async createResourceQuota(namespace: string, organizationId: string, plan: string) {
    const quotaName = `${namespace}-quota`
    const quotaSpec = this.getPlanBasedQuotaSpec(plan)

    const resourceQuota: k8s.V1ResourceQuota = {
      metadata: {
        name: quotaName,
        namespace,
        labels: {
          'langop.io/organization-id': organizationId,
          'langop.io/plan': plan.toLowerCase(),
          'langop.io/resource': 'quota'
        }
      },
      spec: {
        hard: quotaSpec
      }
    }

    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    return await this.coreV1Api.createNamespacedResourceQuota({
      namespace,
      body: resourceQuota
    })
  }

  async getResourceQuota(namespace: string, name?: string) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    const quotaName = name || `${namespace}-quota`
    return await this.coreV1Api.readNamespacedResourceQuota({
      name: quotaName,
      namespace
    })
  }

  async updateResourceQuota(namespace: string, plan: string, organizationId: string, name?: string) {
    const quotaName = name || `${namespace}-quota`
    const quotaSpec = this.getPlanBasedQuotaSpec(plan)

    const resourceQuota: k8s.V1ResourceQuota = {
      metadata: {
        name: quotaName,
        namespace,
        labels: {
          'langop.io/organization-id': organizationId,
          'langop.io/plan': plan.toLowerCase(),
          'langop.io/resource': 'quota'
        }
      },
      spec: {
        hard: quotaSpec
      }
    }

    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    return await this.coreV1Api.replaceNamespacedResourceQuota({
      name: quotaName,
      namespace,
      body: resourceQuota
    })
  }

  async updateResourceQuotaWithCustomSpec(
    namespace: string,
    quotaSpec: Record<string, string>,
    organizationId: string,
    name?: string
  ) {
    const quotaName = name || `${namespace}-quota`

    const resourceQuota: k8s.V1ResourceQuota = {
      metadata: {
        name: quotaName,
        namespace,
        labels: {
          'langop.io/organization-id': organizationId,
          'langop.io/plan': 'custom',
          'langop.io/resource': 'quota'
        }
      },
      spec: {
        hard: quotaSpec
      }
    }

    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }

    return await this.coreV1Api.replaceNamespacedResourceQuota({
      name: quotaName,
      namespace,
      body: resourceQuota
    })
  }

  validateQuotaSpec(quotaSpec: Record<string, string>): { valid: boolean; errors: string[] } {
    const errors: string[] = []

    // Validate count fields (must be positive integers)
    const countFields = [
      'count/languageagents',
      'count/languagemodels',
      'count/languagetools',
      'count/languagepersonas',
      'count/languageclusters'
    ]

    countFields.forEach(field => {
      const value = quotaSpec[field]
      if (value) {
        const num = parseInt(value)
        if (isNaN(num) || num < 0) {
          errors.push(`${field} must be a positive integer`)
        }
      }
    })

    // Validate resource fields (CPU: m/cores, Memory: Ki/Mi/Gi)
    const resourceFields = ['requests.cpu', 'limits.cpu', 'requests.memory', 'limits.memory']

    resourceFields.forEach(field => {
      const value = quotaSpec[field]
      if (value) {
        if (field.includes('cpu')) {
          // Validate CPU format (e.g., 100m, 1, 2000m)
          if (!/^\d+(m)?$/.test(value)) {
            errors.push(`${field} must be in format: 100m or 1 (cores)`)
          }
        } else {
          // Validate memory format (e.g., 128Mi, 2Gi, 1024Ki)
          if (!/^\d+(Ki|Mi|Gi)$/.test(value)) {
            errors.push(`${field} must be in format: 128Mi, 2Gi, or 1024Ki`)
          }
        }
      }
    })

    return {
      valid: errors.length === 0,
      errors
    }
  }

  async deleteResourceQuota(namespace: string, name?: string) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    const quotaName = name || `${namespace}-quota`
    return await this.coreV1Api.deleteNamespacedResourceQuota({
      name: quotaName,
      namespace
    })
  }

  async getResourceQuotaUsage(namespace: string, name?: string): Promise<{
    quota: Record<string, string>
    used: Record<string, string>
    available: Record<string, string>
    percentUsed: Record<string, number>
  }> {
    const quotaName = name || `${namespace}-quota`
    
    try {
      if (!this.coreV1Api) {
        throw new Error('Kubernetes API not available')
      }
      const response = await this.coreV1Api.readNamespacedResourceQuota({
        name: quotaName,
        namespace
      })

      const quota = (response as any).spec?.hard || {}
      const used = { ...(response as any).status?.used || {} }
      const available: Record<string, string> = {}
      const percentUsed: Record<string, number> = {}

      // Manually count custom resources since Kubernetes doesn't track them automatically
      if (!this.customObjectsApi) {
        throw new Error('Custom Objects API not available')
      }

      // Count LanguageModels
      if (quota['count/languagemodels'] && !used['count/languagemodels']) {
        try {
          const modelsResponse = await this.customObjectsApi.listNamespacedCustomObject({
            group: 'langop.io',
            version: 'v1alpha1',
            namespace,
            plural: 'languagemodels',
          })
          used['count/languagemodels'] = ((modelsResponse as any).items?.length || 0).toString()
        } catch (error) {
          console.error('Failed to count LanguageModels:', error)
          used['count/languagemodels'] = '0'
        }
      }

      // Count LanguageAgents
      if (quota['count/languageagents'] && !used['count/languageagents']) {
        try {
          const agentsResponse = await this.customObjectsApi.listNamespacedCustomObject({
            group: 'langop.io',
            version: 'v1alpha1',
            namespace,
            plural: 'languageagents',
          })
          used['count/languageagents'] = ((agentsResponse as any).items?.length || 0).toString()
        } catch (error) {
          console.error('Failed to count LanguageAgents:', error)
          used['count/languageagents'] = '0'
        }
      }

      // Count LanguageClusters
      if (quota['count/languageclusters'] && !used['count/languageclusters']) {
        try {
          const clustersResponse = await this.customObjectsApi.listNamespacedCustomObject({
            group: 'langop.io',
            version: 'v1alpha1',
            namespace,
            plural: 'languageclusters',
          })
          used['count/languageclusters'] = ((clustersResponse as any).items?.length || 0).toString()
        } catch (error) {
          console.error('Failed to count LanguageClusters:', error)
          used['count/languageclusters'] = '0'
        }
      }

      // Count LanguageTools
      if (quota['count/languagetools'] && !used['count/languagetools']) {
        try {
          const toolsResponse = await this.customObjectsApi.listNamespacedCustomObject({
            group: 'langop.io',
            version: 'v1alpha1',
            namespace,
            plural: 'languagetools',
          })
          used['count/languagetools'] = ((toolsResponse as any).items?.length || 0).toString()
        } catch (error) {
          console.error('Failed to count LanguageTools:', error)
          used['count/languagetools'] = '0'
        }
      }

      // Count LanguagePersonas
      if (quota['count/languagepersonas'] && !used['count/languagepersonas']) {
        try {
          const personasResponse = await this.customObjectsApi.listNamespacedCustomObject({
            group: 'langop.io',
            version: 'v1alpha1',
            namespace,
            plural: 'languagepersonas',
          })
          used['count/languagepersonas'] = ((personasResponse as any).items?.length || 0).toString()
        } catch (error) {
          console.error('Failed to count LanguagePersonas:', error)
          used['count/languagepersonas'] = '0'
        }
      }

      // Calculate available resources and usage percentages
      Object.keys(quota).forEach(resource => {
        const hardLimit = this.parseResourceQuantity(quota[resource])
        const usedAmount = this.parseResourceQuantity(used[resource] || '0')
        
        available[resource] = (hardLimit - usedAmount).toString()
        percentUsed[resource] = hardLimit > 0 ? (usedAmount / hardLimit) * 100 : 0
      })

      return {
        quota,
        used,
        available,
        percentUsed
      }
    } catch (error) {
      // Return empty usage if quota doesn't exist
      console.error(`[K8S] Failed to read ResourceQuota ${quotaName} in namespace ${namespace}:`, error)
      return {
        quota: {},
        used: {},
        available: {},
        percentUsed: {}
      }
    }
  }

  private parseResourceQuantity(quantity: string): number {
    // Simple parser for basic quantities
    if (quantity.endsWith('m')) {
      return parseInt(quantity.slice(0, -1))
    }
    if (quantity.endsWith('Gi')) {
      return parseInt(quantity.slice(0, -2)) * 1024 * 1024 * 1024
    }
    if (quantity.endsWith('Mi')) {
      return parseInt(quantity.slice(0, -2)) * 1024 * 1024
    }
    if (quantity.endsWith('Ki')) {
      return parseInt(quantity.slice(0, -2)) * 1024
    }
    return parseInt(quantity) || 0
  }

  // Custom Resource methods for language-operator CRDs

  async listLanguageAgents(namespace: string, options?: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
    continue?: string
  }, requestOptions: RequestOptions = {}) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    return await this.withTimeout(
      () => this.customObjectsApi!.listNamespacedCustomObject({
        group: 'langop.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languageagents',
        ...options,
      }),
      requestOptions
    )
  }

  async getLanguageAgent(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      name,
    })
  }

  async createLanguageAgent(namespace: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      body: spec,
    })
  }

  async updateLanguageAgent(namespace: string, name: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    // Use replaceNamespacedCustomObject instead of patch to avoid patch format issues
    return await this.customObjectsApi.replaceNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
      name,
      body: spec,
    })
  }

  async deleteLanguageAgent(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
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
  }, requestOptions: RequestOptions = {}) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    return await this.withTimeout(
      () => this.customObjectsApi!.listNamespacedCustomObject({
        group: 'langop.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languagemodels',
        ...options,
      }),
      requestOptions
    )
  }

  async getLanguageModel(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      name,
    })
  }

  async createLanguageModel(namespace: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      body: spec,
    })
  }

  async updateLanguageModel(namespace: string, name: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    // Create a merge patch for the spec field only (CRDs don't support strategic merge)
    const patch = {
      spec: spec.spec
    }
    
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      name,
      body: patch,
    })
  }

  async replaceLanguageModel(namespace: string, name: string, model: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    // For replaceNamespacedCustomObject, we need to include kind and apiVersion
    const completeModel = {
      kind: 'LanguageModel',
      apiVersion: 'langop.io/v1alpha1',
      ...model,
    }
    
    return await this.customObjectsApi.replaceNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
      name,
      body: completeModel,
    })
  }

  async deleteLanguageModel(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
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
  }, requestOptions: RequestOptions = {}) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    return await this.withTimeout(
      () => this.customObjectsApi!.listNamespacedCustomObject({
        group: 'langop.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languagetools',
        ...options,
      }),
      requestOptions
    )
  }

  async getLanguageTool(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      name,
    })
  }

  async createLanguageTool(namespace: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      body: spec,
    })
  }

  async updateLanguageTool(namespace: string, name: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.patchNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      name,
      body: spec,
    })
  }

  async replaceLanguageTool(namespace: string, name: string, tool: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.replaceNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
      name,
      body: tool,
    })
  }

  async deleteLanguageTool(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
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
  }, requestOptions: RequestOptions = {}) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    return await this.withTimeout(
      () => this.customObjectsApi!.listNamespacedCustomObject({
        group: 'langop.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languagepersonas',
        ...options,
      }),
      requestOptions
    )
  }

  async getLanguagePersona(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      name,
    })
  }

  async createLanguagePersona(namespace: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      body: spec,
    })
  }

  async updateLanguagePersona(namespace: string, name: string, persona: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    // For replaceNamespacedCustomObject, we need to include kind and apiVersion
    const completePersona = {
      kind: 'LanguagePersona',
      apiVersion: 'langop.io/v1alpha1',
      ...persona,
    }
    
    return await this.customObjectsApi.replaceNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
      name,
      body: completePersona,
    })
  }

  async deleteLanguagePersona(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
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
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      ...options,
    })
  }

  async getLanguageCluster(namespace: string, name: string, options: RequestOptions = {}) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.withTimeout(
      () => this.customObjectsApi!.getNamespacedCustomObject({
        group: 'langop.io',
        version: 'v1alpha1',
        namespace,
        plural: 'languageclusters',
        name,
      }),
      options
    )
  }

  async createLanguageCluster(namespace: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      body: spec,
    })
  }

  async updateLanguageCluster(namespace: string, name: string, updatedResource: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    // Ensure the resource has required Kubernetes fields
    const body = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageCluster',
      ...updatedResource
    }
    
    // Use replaceNamespacedCustomObject instead of patch to avoid patch format issues
    return await this.customObjectsApi.replaceNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      name,
      body,
    })
  }

  async deleteLanguageCluster(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
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
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
    })
  }

  async getLanguageAgentVersion(namespace: string, name: string) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.getNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
      name,
    })
  }

  async createLanguageAgentVersion(namespace: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.createNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
      body: spec,
    })
  }

  async updateLanguageAgentVersion(namespace: string, name: string, spec: any) {
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
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
    if (!this.customObjectsApi) {
      throw new Error('Kubernetes API not available')
    }
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagentversions',
      name,
    })
  }

  // Events API
  async listEvents(namespace: string, options: {
    labelSelector?: string
    fieldSelector?: string
    limit?: number
  } = {}) {
    if (!this.coreV1Api) {
      throw new Error('Kubernetes API not available')
    }
    
    return await this.coreV1Api.listNamespacedEvent({
      namespace,
      labelSelector: options.labelSelector,
      fieldSelector: options.fieldSelector,
      limit: options.limit || 50, // Default to last 50 events
    })
  }

  // Helper methods for common query patterns

  /**
   * List resources in namespace with organization filtering
   */
  async listByOrganization(resourceType: 'agents' | 'models' | 'tools' | 'personas' | 'clusters', namespace: string, organizationId: string) {
    // Try new label format first, fallback to old format for legacy resources
    const newLabelSelector = `langop.io/organization-id=${organizationId}`
    const oldLabelSelector = `langop.io/organization=${organizationId}`
    
    const listMethod = {
      agents: this.listLanguageAgents.bind(this),
      models: this.listLanguageModels.bind(this),
      tools: this.listLanguageTools.bind(this),
      personas: this.listLanguagePersonas.bind(this),
      clusters: this.listLanguageClusters.bind(this)
    }[resourceType]
    
    if (!listMethod) {
      throw new Error(`Unknown resource type: ${resourceType}`)
    }
    
    try {
      // First try with new label format
      const result = await listMethod(namespace, { labelSelector: newLabelSelector })
      const items = (result as any)?.body?.items || (result as any)?.data?.items || (result as any)?.items || []
      
      // If we found resources, return them
      if (items.length > 0) {
        return result
      }
      
      // If no resources found with new label, try old label format as fallback
      return await listMethod(namespace, { labelSelector: oldLabelSelector })
      
    } catch (error) {
      // If new label fails, try old label format
      return await listMethod(namespace, { labelSelector: oldLabelSelector })
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
    const labelSelector = organizationId ? `langop.io/organization-id=${organizationId}` : undefined
    const options = labelSelector ? { labelSelector } : undefined

    const [agents, models, tools, personas, clusters] = await Promise.all([
      this.listLanguageAgents(namespace, options),
      this.listLanguageModels(namespace, options),
      this.listLanguageTools(namespace, options),
      this.listLanguagePersonas(namespace, options),
      this.listLanguageClusters(namespace, options),
    ])

    // Handle different response structures from k8s client
    // Live K8s mode: { body: { items: [...] } }
    // Error fallback: { data: { items: [] } }
    const getItemsLength = (response: any): number => {
      const responseBody = response?.body
      const responseData = response?.data
      const responseItems = response?.items
      
      if (responseBody?.items && Array.isArray(responseBody.items)) {
        return responseBody.items.length
      } else if (responseData?.items && Array.isArray(responseData.items)) {
        return responseData.items.length
      } else if (responseItems && Array.isArray(responseItems)) {
        return responseItems.length
      }
      return 0
    }

    return {
      agents: getItemsLength(agents),
      models: getItemsLength(models),
      tools: getItemsLength(tools),
      personas: getItemsLength(personas),
      clusters: getItemsLength(clusters),
    }
  }

  /**
   * Search resources across all types in a namespace
   */
  async searchResources(namespace: string, query: string, organizationId?: string) {
    const baseSelector = organizationId ? `langop.io/organization-id=${organizationId}` : undefined
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

  // Jobs management
  async createJob(namespace: string, job: any) {
    if (!this.batchV1Api) {
      throw new Error('Kubernetes client not initialized')
    }

    try {
      const response = await this.batchV1Api.createNamespacedJob({
        namespace,
        body: job,
      })
      console.log(`✅ Job created successfully: ${job.metadata.name}`)
      return response
    } catch (error) {
      console.error(`❌ Failed to create Job ${job.metadata.name}:`, error)
      throw error
    }
  }

  async getCronJob(namespace: string, name: string) {
    if (!this.batchV1Api) {
      throw new Error('Kubernetes client not initialized')
    }

    try {
      const response = await this.batchV1Api.readNamespacedCronJob({
        name,
        namespace,
      })
      return response
    } catch (error) {
      console.error(`❌ Failed to get CronJob ${name}:`, error)
      throw error
    }
  }

  async createJobFromCronJob(namespace: string, cronJobName: string, jobName: string) {
    if (!this.batchV1Api) {
      throw new Error('Kubernetes client not initialized')
    }

    try {
      // Get the existing CronJob
      const cronJobResponse = await this.getCronJob(namespace, cronJobName)
      const cronJob = (cronJobResponse as any)?.body || (cronJobResponse as any)?.data || cronJobResponse
      
      if (!cronJob?.spec?.jobTemplate) {
        throw new Error(`CronJob ${cronJobName} does not have a valid job template`)
      }

      // Extract the job template and create a new Job
      const jobTemplate = cronJob.spec.jobTemplate
      const job = {
        apiVersion: 'batch/v1',
        kind: 'Job',
        metadata: {
          name: jobName,
          namespace,
          labels: {
            ...(jobTemplate.metadata?.labels || {}),
            'langop.io/execution-type': 'manual'
          },
          annotations: {
            ...(jobTemplate.metadata?.annotations || {}),
            'langop.io/manual-execution': 'true',
            'langop.io/created-from-cronjob': cronJobName
          }
        },
        spec: jobTemplate.spec
      }

      const response = await this.batchV1Api.createNamespacedJob({
        namespace,
        body: job,
      })
      console.log(`✅ Job created from CronJob: ${jobName} (from ${cronJobName})`)
      return response
    } catch (error) {
      console.error(`❌ Failed to create Job from CronJob ${cronJobName}:`, error)
      throw error
    }
  }

  async getJob(namespace: string, name: string) {
    if (!this.batchV1Api) {
      throw new Error('Kubernetes client not initialized')
    }

    try {
      const response = await this.batchV1Api.readNamespacedJob({
        namespace,
        name,
      })
      return response
    } catch (error) {
      console.error(`❌ Failed to get Job ${name}:`, error)
      throw error
    }
  }

  async listJobs(namespace: string, options?: { labelSelector?: string }) {
    if (!this.batchV1Api) {
      throw new Error('Kubernetes client not initialized')
    }

    try {
      const response = await this.batchV1Api.listNamespacedJob({
        namespace,
        ...(options?.labelSelector && { labelSelector: options.labelSelector }),
      })
      return response
    } catch (error) {
      console.error(`❌ Failed to list Jobs in namespace ${namespace}:`, error)
      throw error
    }
  }
}

// Export singleton instance
export const k8sClient = KubernetesClient.getInstance()
