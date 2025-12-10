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
    const response = await this.coreV1Api.listNamespace()
    return response.body
  }

  async getNamespace(name: string) {
    const response = await this.coreV1Api.readNamespace(name)
    return response.body
  }

  async createNamespace(name: string, labels?: Record<string, string>) {
    const namespace: k8s.V1Namespace = {
      metadata: {
        name,
        labels,
      },
    }
    const response = await this.coreV1Api.createNamespace(namespace)
    return response.body
  }

  async getPodLogs(namespace: string, podName: string, tailLines: number = 100) {
    const response = await this.coreV1Api.readNamespacedPodLog(
      podName,
      namespace,
      undefined, // container
      false, // follow
      undefined, // insecureSkipTLSVerifyBackend
      undefined, // limitBytes
      undefined, // pretty
      undefined, // previous
      undefined, // sinceSeconds
      tailLines, // tailLines
      undefined  // timestamps
    )
    return response.body
  }

  // Custom Resource methods for language-operator CRDs

  async listLanguageAgents(namespace: string) {
    const response = await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents'
    )
    return response.body
  }

  async getLanguageAgent(namespace: string, name: string) {
    const response = await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents',
      name
    )
    return response.body
  }

  async createLanguageAgent(namespace: string, spec: any) {
    const response = await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents',
      spec
    )
    return response.body
  }

  async updateLanguageAgent(namespace: string, name: string, spec: any) {
    const response = await this.customObjectsApi.patchNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents',
      name,
      spec,
      undefined,
      undefined,
      undefined,
      { headers: { 'Content-Type': 'application/merge-patch+json' } }
    )
    return response.body
  }

  async deleteLanguageAgent(namespace: string, name: string) {
    const response = await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents',
      name
    )
    return response.body
  }

  // LanguageModel methods

  async listLanguageModels(namespace: string) {
    const response = await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels'
    )
    return response.body
  }

  async getLanguageModel(namespace: string, name: string) {
    const response = await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels',
      name
    )
    return response.body
  }

  async createLanguageModel(namespace: string, spec: any) {
    const response = await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels',
      spec
    )
    return response.body
  }

  async deleteLanguageModel(namespace: string, name: string) {
    const response = await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels',
      name
    )
    return response.body
  }

  // LanguageTool methods

  async listLanguageTools(namespace: string) {
    const response = await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools'
    )
    return response.body
  }

  async getLanguageTool(namespace: string, name: string) {
    const response = await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools',
      name
    )
    return response.body
  }

  async createLanguageTool(namespace: string, spec: any) {
    const response = await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools',
      spec
    )
    return response.body
  }

  async deleteLanguageTool(namespace: string, name: string) {
    const response = await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools',
      name
    )
    return response.body
  }

  // LanguagePersona methods

  async listLanguagePersonas(namespace: string) {
    const response = await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas'
    )
    return response.body
  }

  async getLanguagePersona(namespace: string, name: string) {
    const response = await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas',
      name
    )
    return response.body
  }

  async createLanguagePersona(namespace: string, spec: any) {
    const response = await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas',
      spec
    )
    return response.body
  }

  async deleteLanguagePersona(namespace: string, name: string) {
    const response = await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas',
      name
    )
    return response.body
  }

  // LanguageCluster methods

  async listLanguageClusters(namespace: string) {
    const response = await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters'
    )
    return response.body
  }

  async getLanguageCluster(namespace: string, name: string) {
    const response = await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters',
      name
    )
    return response.body
  }

  async createLanguageCluster(namespace: string, spec: any) {
    const response = await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters',
      spec
    )
    return response.body
  }

  async deleteLanguageCluster(namespace: string, name: string) {
    const response = await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters',
      name
    )
    return response.body
  }
}

// Export singleton instance
export const k8sClient = KubernetesClient.getInstance()
