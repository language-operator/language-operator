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
    return await this.coreV1Api.listNamespace()
  }

  async getNamespace(name: string) {
    return await this.coreV1Api.readNamespace(name)
  }

  async createNamespace(name: string, labels?: Record<string, string>) {
    const namespace: k8s.V1Namespace = {
      metadata: {
        name,
        labels,
      },
    }
    return await this.coreV1Api.createNamespace(namespace)
  }

  async getPodLogs(namespace: string, podName: string, tailLines: number = 100) {
    return await this.coreV1Api.readNamespacedPodLog(
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
  }

  // Custom Resource methods for language-operator CRDs

  async listLanguageAgents(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents'
    )
  }

  async getLanguageAgent(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents',
      name
    )
  }

  async createLanguageAgent(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents',
      spec
    )
  }

  async updateLanguageAgent(namespace: string, name: string, spec: any) {
    return await this.customObjectsApi.patchNamespacedCustomObject(
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
  }

  async deleteLanguageAgent(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageagents',
      name
    )
  }

  // LanguageModel methods

  async listLanguageModels(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels'
    )
  }

  async getLanguageModel(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels',
      name
    )
  }

  async createLanguageModel(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels',
      spec
    )
  }

  async deleteLanguageModel(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagemodels',
      name
    )
  }

  // LanguageTool methods

  async listLanguageTools(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools'
    )
  }

  async getLanguageTool(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools',
      name
    )
  }

  async createLanguageTool(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools',
      spec
    )
  }

  async deleteLanguageTool(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagetools',
      name
    )
  }

  // LanguagePersona methods

  async listLanguagePersonas(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas'
    )
  }

  async getLanguagePersona(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas',
      name
    )
  }

  async createLanguagePersona(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas',
      spec
    )
  }

  async deleteLanguagePersona(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languagepersonas',
      name
    )
  }

  // LanguageCluster methods

  async listLanguageClusters(namespace: string) {
    return await this.customObjectsApi.listNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters'
    )
  }

  async getLanguageCluster(namespace: string, name: string) {
    return await this.customObjectsApi.getNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters',
      name
    )
  }

  async createLanguageCluster(namespace: string, spec: any) {
    return await this.customObjectsApi.createNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters',
      spec
    )
  }

  async deleteLanguageCluster(namespace: string, name: string) {
    return await this.customObjectsApi.deleteNamespacedCustomObject(
      'langop.io',
      'v1alpha1',
      namespace,
      'languageclusters',
      name
    )
  }
}

// Export singleton instance
export const k8sClient = KubernetesClient.getInstance()
