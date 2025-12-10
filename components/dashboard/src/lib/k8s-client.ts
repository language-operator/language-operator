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

  async listLanguageAgents(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageagents',
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

  async listLanguageModels(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagemodels',
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

  async listLanguageTools(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagetools',
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

  async listLanguagePersonas(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languagepersonas',
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

  async listLanguageClusters(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
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

  async deleteLanguageCluster(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject({
      group: 'langop.io',
      version: 'v1alpha1',
      namespace,
      plural: 'languageclusters',
      name,
    })
  }
}

// Export singleton instance
export const k8sClient = KubernetesClient.getInstance()
