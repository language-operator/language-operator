/**
 * Service URL resolver that adapts between Docker Compose and Kubernetes environments
 */

interface ServiceResolverOptions {
  serviceName: string
  namespace: string
  port: number
  path?: string
}

/**
 * Resolves service URLs based on the deployment environment
 * 
 * - In Kubernetes: Returns direct service URL (my-service.namespace.svc.cluster.local)
 * - In Docker Compose: Returns kubectl proxy URL for service access
 */
export class ServiceResolver {
  private isKubernetesEnvironment(): boolean {
    // Check if we're running inside a Kubernetes pod
    return !!(
      process.env.KUBERNETES_SERVICE_HOST ||
      process.env.KUBECONFIG?.includes('/var/run/secrets/kubernetes.io') ||
      process.env.NODE_ENV === 'production' // Assume production = K8s
    )
  }

  private isDockerComposeEnvironment(): boolean {
    // Check if we're using kubectl proxy (Docker Compose setup)
    return process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy') || false
  }

  /**
   * Resolve a service URL for the current environment
   */
  public resolveServiceUrl({ serviceName, namespace, port, path = '' }: ServiceResolverOptions): string {
    const cleanPath = path.startsWith('/') ? path : `/${path}`
    
    if (this.isKubernetesEnvironment() && !this.isDockerComposeEnvironment()) {
      // Running in Kubernetes cluster - use direct service DNS
      return `http://${serviceName}.${namespace}.svc.cluster.local:${port}${cleanPath}`
    } else {
      // Running in Docker Compose - use kubectl proxy
      const proxyHost = process.env.KUBERNETES_SERVER_URL?.replace(/https?:\/\//, '') || 'kubectl-proxy:8001'
      return `http://${proxyHost}/api/v1/namespaces/${namespace}/services/${serviceName}:${port}/proxy${cleanPath}`
    }
  }

  /**
   * Helper method specifically for chat completion endpoints
   */
  public resolveAgentChatUrl(agentName: string, namespace: string, port: number = 8080): string {
    return this.resolveServiceUrl({
      serviceName: agentName,
      namespace,
      port,
      path: '/v1/chat/completions'
    })
  }

  /**
   * Helper method for model endpoints (LiteLLM proxy)
   */
  public resolveModelUrl(modelName: string, namespace: string, port: number = 8000): string {
    return this.resolveServiceUrl({
      serviceName: modelName,
      namespace,
      port,
      path: '/v1/chat/completions'
    })
  }

  /**
   * Get environment info for debugging
   */
  public getEnvironmentInfo(): {
    environment: 'kubernetes' | 'docker-compose' | 'unknown'
    kubernetesServiceHost?: string
    kubernetesServerUrl?: string
    isProduction: boolean
  } {
    return {
      environment: this.isKubernetesEnvironment() && !this.isDockerComposeEnvironment() 
        ? 'kubernetes' 
        : this.isDockerComposeEnvironment() 
        ? 'docker-compose'
        : 'unknown',
      kubernetesServiceHost: process.env.KUBERNETES_SERVICE_HOST,
      kubernetesServerUrl: process.env.KUBERNETES_SERVER_URL,
      isProduction: process.env.NODE_ENV === 'production'
    }
  }
}

// Export singleton instance
export const serviceResolver = new ServiceResolver()

/**
 * Legacy function for backward compatibility
 * @deprecated Use serviceResolver.resolveServiceUrl() instead
 */
export function resolveServiceUrl(serviceName: string, namespace: string, port: number, path?: string): string {
  return serviceResolver.resolveServiceUrl({ serviceName, namespace, port, path })
}